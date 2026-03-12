from fastapi import FastAPI, HTTPException, Depends, Request
from fastapi.middleware.cors import CORSMiddleware
from contextlib import asynccontextmanager
import uuid
import structlog
import httpx
import datetime
import asyncio

from starlette.responses import JSONResponse
from telethon.errors import SessionPasswordNeededError
from telethon.sessions import StringSession

from app.config import settings
from app.models import *
from app.redis_client import redis_client
from app.telegram_client import client_manager
from app.dependencies import verify_internal_api_key

logger = structlog.get_logger()


@asynccontextmanager
async def lifespan(app: FastAPI):
    logger.info("Starting Telegram Auth Service", environment=settings.ENVIRONMENT)
    await redis_client.connect()
    asyncio.create_task(cleanup_task())

    yield
    logger.info("Shutting down Telegram Auth Service")
    await redis_client.close()

    for login_id in list(client_manager.active_clients.keys()):
        await client_manager.close_client(login_id)


app = FastAPI(
    title="Telegram Auth Service",
    description="Minimal production-ready Telegram authentication service",
    version="1.0.0",
    lifespan=lifespan
)

# CORS
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)


async def cleanup_task():
    while True:
        try:
            await asyncio.sleep(60)
        except Exception as e:
            logger.error("Error in cleanup task", error=str(e))


@app.get("/health", response_model=HealthResponse)
async def health_check():
    redis_ok = False
    telegram_ok = False

    try:
        await redis_client.client.ping()
        redis_ok = True
    except:
        pass

    try:
        from telethon import TelegramClient
        client = TelegramClient(StringSession(), settings.TELEGRAM_API_ID, settings.TELEGRAM_API_HASH)
        await client.connect()
        telegram_ok = await client.is_user_authorized() is not None
        await client.disconnect()
    except:
        pass

    return HealthResponse(
        status="healthy" if redis_ok and telegram_ok else "degraded",
        version="1.0.0",
        redis=redis_ok,
        telegram_api=telegram_ok,
        environment=settings.ENVIRONMENT
    )


@app.post("/v1/auth/initiate", response_model=InitiateAuthResponse)
async def initiate_auth(request: InitiateAuthRequest):
    login_id = str(uuid.uuid4())
    active_logins = await redis_client.get_user_active_logins(request.user_id)
    if len(active_logins) >= settings.MAX_ACTIVE_LOGINS_PER_USER:
        raise HTTPException(
            status_code=429,
            detail=f"Too many active login attempts. Maximum: {settings.MAX_ACTIVE_LOGINS_PER_USER}"
        )

    client = await client_manager.create_client(login_id)

    try:
        if request.method == 'phone' and request.phone:
            sent_code = await client.send_code_request(request.phone)

            login_data = {
                'user_id': request.user_id,
                'type': 'phone',
                'phone': request.phone,
                'phone_code_hash': sent_code.phone_code_hash,
                'created_at': datetime.datetime.now(datetime.UTC).isoformat()
            }

            await redis_client.set_login_session(login_id, login_data)
            await redis_client.set_user_active_login(request.user_id, login_id)

            return InitiateAuthResponse(
                login_id=login_id,
                method='phone',
                expires_in=settings.LOGIN_TIMEOUT_SECONDS,
                phone_code_hash=sent_code.phone_code_hash
            )

        else:
            qr_login = await client.qr_login()

            login_data = {
                'user_id': request.user_id,
                'type': 'qr',
                'created_at': datetime.datetime.now(datetime.UTC).isoformat()
            }

            await redis_client.set_login_session(login_id, login_data)
            await redis_client.set_user_active_login(request.user_id, login_id)

            asyncio.create_task(wait_for_qr_login(login_id))

            return InitiateAuthResponse(
                login_id=login_id,
                method='qr',
                qr_data=qr_login.url,
                expires_in=settings.LOGIN_TIMEOUT_SECONDS
            )

    except Exception as e:
        await client_manager.close_client(login_id)
        logger.error("Failed to initiate auth", error=str(e), login_id=login_id)
        raise HTTPException(status_code=500, detail=str(e))


async def wait_for_qr_login(login_id: str):
    client = await client_manager.get_client(login_id)
    if not client:
        return

    login_data = await redis_client.get_login_session(login_id)
    if not login_data:
        await client_manager.close_client(login_id)
        return

    try:
        qr_login = await client.qr_login()
        await asyncio.wait_for(
            qr_login.wait(),
            timeout=settings.LOGIN_TIMEOUT_SECONDS
        )

        session_string = client.session.save()

        await store_session_in_auth_service(
            user_id=login_data['user_id'],
            session_string=session_string
        )

        login_data['status'] = 'completed'
        login_data['session_stored'] = True
        await redis_client.set_login_session(login_id, login_data)

        logger.info("QR login completed", user_id=login_data['user_id'], login_id=login_id)

    except asyncio.TimeoutError:
        logger.info("QR login timeout", login_id=login_id)
    except Exception as e:
        logger.error("QR login failed", error=str(e), login_id=login_id)
    finally:
        await client_manager.close_client(login_id)
        await redis_client.remove_user_active_login(login_data['user_id'], login_id)

@app.post("/v1/auth/verify", response_model=VerifyCodeResponse)
async def verify_code(request: VerifyCodeRequest):
    """
    Step 2: Verify code (for phone flow)
    """
    login_data = await redis_client.get_login_session(request.login_id)
    if not login_data:
        raise HTTPException(status_code=404, detail="Login session not found or expired")

    if login_data['type'] != 'phone':
        raise HTTPException(status_code=400, detail="This endpoint is for phone verification only")

    client = await client_manager.get_client(request.login_id)
    if not client:
        raise HTTPException(status_code=500, detail="Telegram client not found")

    try:
        if request.password:
            await client.sign_in(password=request.password)
        else:
            await client.sign_in(
                login_data['phone'],
                code=request.code,
                phone_code_hash=login_data['phone_code_hash']
            )

        session_string = client.session.save()

        await store_session_in_auth_service(
            user_id=login_data['user_id'],
            session_string=session_string
        )

        await redis_client.delete_login_session(request.login_id)
        await redis_client.remove_user_active_login(login_data['user_id'], request.login_id)
        await client_manager.close_client(request.login_id)

        logger.info("Phone login completed", user_id=login_data['user_id'])

        return VerifyCodeResponse(status='success')

    except SessionPasswordNeededError:
        return VerifyCodeResponse(
            status='password_required',
            message='Two-factor authentication required'
        )
    except Exception as e:
        logger.error("Verification failed", error=str(e), login_id=request.login_id)
        return VerifyCodeResponse(status='error', message=str(e))


@app.post("/v1/auth/resend-code")
async def resend_code(request: ResendCodeRequest):
    """
    Resend verification code for phone flow
    """
    login_data = await redis_client.get_login_session(request.login_id)
    if not login_data or login_data['type'] != 'phone':
        raise HTTPException(status_code=404, detail="Valid phone login session not found")

    client = await client_manager.get_client(request.login_id)
    if not client:
        raise HTTPException(status_code=500, detail="Telegram client not found")

    try:
        sent_code = await client.send_code_request(login_data['phone'])

        login_data['phone_code_hash'] = sent_code.phone_code_hash
        await redis_client.set_login_session(request.login_id, login_data)

        return {"status": "success", "message": "Code resent"}

    except Exception as e:
        logger.error("Failed to resend code", error=str(e), login_id=request.login_id)
        raise HTTPException(status_code=500, detail=str(e))


@app.get("/v1/auth/status/{login_id}", response_model=AuthStatusResponse)
async def get_auth_status(login_id: str):
    """
    Check authentication status (useful for polling QR flow)
    """
    login_data = await redis_client.get_login_session(login_id)

    if not login_data:
        return AuthStatusResponse(status='expired')

    if login_data.get('session_stored'):
        return AuthStatusResponse(
            status='completed',
            user_id=login_data['user_id']
        )

    return AuthStatusResponse(status='pending')


@app.post("/internal/sessions", response_model=dict)
async def internal_store_session(
        request: StoreSessionRequest,
        _: bool = Depends(verify_internal_api_key)
):
    """
    Internal endpoint for Personage.Auth to store session strings
    """
    logger.info("Session stored for user", user_id=request.user_id)
    return {"status": "success"}


async def store_session_in_auth_service(user_id: str, session_string: str):
    """Call Personage.Auth to store the session string"""
    try:
        async with httpx.AsyncClient() as client:
            response = await client.post(
                f"{settings.AUTH_SERVICE_URL}/internal/telegram-sessions",
                json={"user_id": user_id, "session_string": session_string},
                headers={"X-Internal-Api-Key": settings.AUTH_SERVICE_API_KEY},
                timeout=10.0
            )
            response.raise_for_status()
            logger.info("Session stored in Personage.Auth", user_id=user_id)
    except Exception as e:
        logger.error("Failed to store session in Personage.Auth", error=str(e), user_id=user_id)


@app.exception_handler(Exception)
async def global_exception_handler(request: Request, exc: Exception):
    logger.error("Unhandled exception", error=str(exc), path=request.url.path)
    return JSONResponse(
        status_code=500,
        content={"detail": "Internal server error"}
    )