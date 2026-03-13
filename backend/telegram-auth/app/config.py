from pathlib import Path
import urllib.parse
from pydantic_settings import BaseSettings
from pydantic import Field
from typing import Optional
import requests
import logging
import os
from yc_lockbox import YandexLockboxClient
from dotenv import load_dotenv

logger = logging.getLogger(__name__)


class Settings(BaseSettings):
    AUTH_SECRETS_LOCKBOX_ID: str = "e6qk3le6lts8r8nos42j"

    REDIS_HOST: str = Field('localhost', env='REDIS_HOST')
    REDIS_PORT: int = Field(6379, env='REDIS_PORT')

    REDIS_PASSWORD: Optional[str] = Field(None, env='REDIS_PASSWORD')
    REDIS_PASSWORD_SECRET_ID: str = "e6qvfihd6jhni4pugu4a"

    REDIS_DB: int = Field(0, env='REDIS_DB')
    REDIS_SSL: bool = Field(False, env='REDIS_SSL')

    LOGIN_TIMEOUT_SECONDS: int = Field(300, env='LOGIN_TIMEOUT_SECONDS')
    ENVIRONMENT: str = Field('development', env='ENVIRONMENT')

    AUTH_SERVICE_GRPC_HOST: str = Field('localhost', env='AUTH_SERVICE_GRPC_HOST')
    AUTH_SERVICE_GRPC_PORT: int = Field(50051, env='AUTH_SERVICE_GRPC_PORT')

    TELEGRAM_API_HASH: str | None = None
    TELEGRAM_API_ID: str | None = None

    class Config:
        env_file = '.env'
        case_sensitive = True

    def load_lockbox_secrets(self):
        lockbox_telegram_id_key = "telegram_api_id"
        lockbox_telegram_hash_key = "telegram_api_hash"

        iam_token = Settings._get_iam_token_on_vm()
        if not iam_token:
            logger.warning("Using fallback iam token from environment")
            iam_token = Settings._get_iam_token_from_env()

        lockbox = YandexLockboxClient(iam_token)
        auth_secrets_payload = lockbox.get_secret_payload(self.AUTH_SECRETS_LOCKBOX_ID)

        self.TELEGRAM_API_ID = auth_secrets_payload.get(lockbox_telegram_id_key).text_value.get_secret_value()
        self.TELEGRAM_API_HASH = auth_secrets_payload.get(lockbox_telegram_hash_key).text_value.get_secret_value()

        lockbox_redis_password_key = "redis_pass"
        redis_secrets_payload = lockbox.get_secret_payload(self.REDIS_PASSWORD_SECRET_ID)
        raw_redis_password = redis_secrets_payload.get(lockbox_redis_password_key).text_value.get_secret_value()
        encoded_password = urllib.parse.quote(raw_redis_password, safe='')
        self.REDIS_PASSWORD = encoded_password


    @staticmethod
    def _get_iam_token_on_vm() -> str | None:
        GET_TOKEN_FROM_VM_METADATA_URL = "http://169.254.169.254/computeMetadata/v1/instance/service-accounts/default/token"
        try:
            headers = {"Metadata-Flavor": "Google"}
            response = requests.get(GET_TOKEN_FROM_VM_METADATA_URL, headers=headers, timeout=2)
            response.raise_for_status()
            return response.json()["access_token"]
        except:
            logger.warning("Unable to get IAM token on VM")
            return None

    @staticmethod
    def _get_iam_token_from_env() -> str:
        personage_root = Path(__file__).parent.parent.parent.parent
        personage_secrets = personage_root / 'secrets.env'
        load_dotenv(personage_secrets)


        yc_token = os.environ.get("YC_TOKEN", None)
        if not yc_token:
            raise Exception("YC_TOKEN environment variable not set")

        return yc_token


settings = Settings()
settings.load_lockbox_secrets()
