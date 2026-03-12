from fastapi import HTTPException, Security
from fastapi.security import APIKeyHeader
from starlette.status import HTTP_403_FORBIDDEN
from app.config import settings

api_key_header = APIKeyHeader(name="X-Internal-Api-Key", auto_error=False)

async def verify_internal_api_key(api_key: str = Security(api_key_header)):
    if not api_key or api_key != settings.AUTH_SERVICE_API_KEY:
        raise HTTPException(
            status_code=HTTP_403_FORBIDDEN,
            detail="Invalid internal API key"
        )
    return api_key