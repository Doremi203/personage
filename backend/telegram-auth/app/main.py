from fastapi import FastAPI, HTTPException, Request
from fastapi.middleware.cors import CORSMiddleware
from contextlib import asynccontextmanager
import uuid
import structlog
import datetime
import asyncio

from starlette.responses import JSONResponse
from telethon.errors import SessionPasswordNeededError
from app.auth_service_grpc_client import auth_service_grpc_client
from app.config import settings
from app.logging import setup_logging
from app.models import *
from app.redis_client import redis_client
from app.telegram_client import client_manager

# Initialise logging before any logger is created.
# In production (LOG_FORMAT=json) this produces JSON matching the Go slog /
# traitex JSONFormatter shape consumed by Fluent Bit.
setup_logging(log_level=settings.LOG_LEVEL, log_format=settings.LOG_FORMAT)

logger = structlog.get_logger()


@asynccontextmanager
async def lifespan(_: FastAPI):
    logger.info("Starting Telegram Auth Service", environment=settings.ENVIRONMENT)
    await redis_client.connect()
    asyncio.create_task(cleanup_task())

    try:
        await auth_service_grpc_client.connect()
    except Exception as e:
        logger.error("Failed to connect to gRPC auth service", error=str(e))
        raise

    yield
    logger.info("Shutting down Telegram Auth Service")
    await redis_client.close()
    await auth_service_grpc_client.close()

    for login_id in list(client_manager.active_clients.keys()):
        await client_manager.close_client(login_id)


app = FastAPI(
    title="Telegram Auth Service",
    description="Minimal production-ready Telegram authentication service",
    version="1.0.0",
    lifespan=lifespan
)


app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)


async def cleanup_task():
    """Background task for any periodic cleanup"""
    while True:
        try:
            await asyncio.sleep(60)
        except Exception as e:
            logger.error("Error in cleanup task", error=str(e))


@app.get("/liveliness")
async def health_check():
    return {'status': 'ok'}


@app.get("/health")
async def health_check():
    return {'status': 'ok'}


@app.post("/v1/auth/initiate", response_model=InitiateAuthResponse)
async def initiate_auth(request: InitiateAuthRequest):
    """
    Step 1: Initiate Telegram authentication
    - For phone method: sends code to provided phone number
    - For QR method: returns QR code URL
    """
    login_id = str(uuid.uuid4())
    client = await client_manager.create_client(login_id)

    try:
        if request.method == 'phone' and request.phone:
            sent_code = await client.send_code_request(request.phone)

            login_data = {
                'user_id': request.user_id,
                'type': 'phone',
                'phone': request.phone,
                'phone_code_hash': sent_code.phone_code_hash,
                'created_at': datetime.datetime.now(datetime.UTC).isoformat(),
                'status': 'pending'
            }

            await redis_client.set_login_session(login_id, login_data)

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
                'created_at': datetime.datetime.now(datetime.UTC).isoformat(),
                'status': 'pending'
            }

            await redis_client.set_login_session(login_id, login_data)

            asyncio.create_task(wait_for_qr_login(login_id, qr_login))

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


async def wait_for_qr_login(login_id: str, qr_login):
    """
    Background task that waits for QR code to be scanned
    """
    client = await client_manager.get_client(login_id)
    if not client:
        return

    try:
        await asyncio.wait_for(
            qr_login.wait(),
            timeout=settings.LOGIN_TIMEOUT_SECONDS
        )

        session_string = client.session.save()

        login_data = await redis_client.get_login_session(login_id)
        if not login_data:
            logger.warning("Login session expired before QR completion", login_id=login_id)
            return

        await store_session_in_auth_service(
            user_id=login_data['user_id'],
            session_string=session_string
        )

        login_data['status'] = 'completed'
        login_data['session_stored'] = True
        await redis_client.set_login_session(login_id, login_data)

        logger.info("QR login completed",
                    user_id=login_data['user_id'],
                    login_id=login_id)

    except asyncio.TimeoutError:
        logger.info("QR login timeout", login_id=login_id)
    except Exception as e:
        logger.error("QR login failed", error=str(e), login_id=login_id)
    finally:
        await client_manager.close_client(login_id)


@app.post("/v1/auth/verify", response_model=VerifyCodeResponse)
async def verify_code(request: VerifyCodeRequest):
    """
    Step 2: Verify code for phone flow
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

    if login_data.get('status') == 'completed' or login_data.get('session_stored'):
        return AuthStatusResponse(
            status='completed',
            user_id=login_data['user_id']
        )

    return AuthStatusResponse(status='pending')


async def store_session_in_auth_service(user_id: str, session_string: str):
    """
    Store Telegram session in Personage.Auth via gRPC
    """
    try:
        await auth_service_grpc_client.store_session(
            user_id=user_id,
            session_string=session_string
        )

    except Exception as e:
        logger.error("Failed to store session via gRPC", error=str(e), user_id=user_id)
        raise


@app.exception_handler(Exception)
async def global_exception_handler(request: Request, exc: Exception):
    logger.error("Unhandled exception", error=str(exc), path=request.url.path)
    return JSONResponse(
        status_code=500,
        content={"detail": "Internal server error"}
    )
