from pydantic import BaseModel, Field
from typing import Optional, Literal


class InitiateAuthRequest(BaseModel):
    user_id: str = Field(..., description="User ID from Personage.Auth")
    phone: Optional[str] = Field(None, description="Phone number for code flow")
    method: Literal['qr', 'phone'] = Field('qr', description="Authentication method")


class InitiateAuthResponse(BaseModel):
    login_id: str
    method: str
    qr_data: Optional[str] = None
    expires_in: int
    phone_code_hash: Optional[str] = None


class VerifyCodeRequest(BaseModel):
    login_id: str
    code: str
    password: Optional[str] = None


class VerifyCodeResponse(BaseModel):
    status: Literal['success', 'password_required', 'error']
    message: Optional[str] = None


class ResendCodeRequest(BaseModel):
    login_id: str


class AuthStatusResponse(BaseModel):
    status: Literal['pending', 'completed', 'expired', 'failed']
    user_id: Optional[str] = None
    error: Optional[str] = None


class StoreSessionRequest(BaseModel):
    user_id: str
    session_string: str


class HealthResponse(BaseModel):
    status: str
    version: str
    redis: bool
    telegram_api: bool
    environment: str
