from pathlib import Path
from pydantic_settings import BaseSettings
from pydantic import Field
from typing import Optional, List
import requests
import logging
import os
from yc_lockbox import YandexLockboxClient
from dotenv import load_dotenv

logger = logging.getLogger(__name__)


class Settings(BaseSettings):
    AUTH_SECRETS_LOCKBOX_ID: str = "e6qk3le6lts8r8nos42j"

    REDIS_HOST: str = Field('valkey', env='REDIS_HOST')
    REDIS_PORT: int = Field(6379, env='REDIS_PORT')

    REDIS_PASSWORD: Optional[str] = Field(None, env='REDIS_PASSWORD')
    REDIS_PASSWORD_SECRET_ID: str = "e6qjundo7djfkiccb0s0"

    REDIS_DB: int = Field(0, env='REDIS_DB')
    REDIS_SSL: bool = Field(False, env='REDIS_SSL')

    # Sentinel configuration for Valkey HA (default: enabled)
    REDIS_USE_SENTINEL: bool = Field(True, env='REDIS_USE_SENTINEL')
    REDIS_SENTINEL_HOSTS: str = Field('', env='REDIS_SENTINEL_HOSTS')
    REDIS_SENTINEL_SERVICE_NAME: str = Field('mymaster', env='REDIS_SENTINEL_SERVICE_NAME')
    REDIS_SENTINEL_PORT: int = Field(26379, env='REDIS_SENTINEL_PORT')
    REDIS_SENTINEL_PASSWORD: Optional[str] = Field(None, env='REDIS_SENTINEL_PASSWORD')

    LOGIN_TIMEOUT_SECONDS: int = Field(300, env='LOGIN_TIMEOUT_SECONDS')
    ENVIRONMENT: str = Field('development', env='ENVIRONMENT')

    LOG_LEVEL: str = Field('INFO', env='LOG_LEVEL')
    LOG_FORMAT: str = Field('console', env='LOG_FORMAT')

    AUTH_SERVICE_GRPC_HOST: str = Field('localhost', env='AUTH_SERVICE_GRPC_HOST')
    AUTH_SERVICE_GRPC_PORT: int = Field(50051, env='AUTH_SERVICE_GRPC_PORT')
    AUTH_SERVICE_GRPC_TLS: bool = Field(False, env='AUTH_SERVICE_GRPC_TLS')

    TELEGRAM_API_HASH: str | None = None
    TELEGRAM_API_ID: str | None = None

    class Config:
        env_file = '.env'
        case_sensitive = True

    def get_sentinel_hosts(self) -> List[tuple]:
        """
        Parse REDIS_SENTINEL_HOSTS into a list of (host, port) tuples.
        Expected format: "host1:port1,host2:port2,host3:port3"
        If port is omitted, REDIS_SENTINEL_PORT is used as default.
        """
        if not self.REDIS_SENTINEL_HOSTS:
            return []

        hosts = []
        for entry in self.REDIS_SENTINEL_HOSTS.split(','):
            entry = entry.strip()
            if not entry:
                continue
            if ':' in entry:
                host, port_str = entry.rsplit(':', 1)
                hosts.append((host, int(port_str)))
            else:
                hosts.append((entry, self.REDIS_SENTINEL_PORT))
        return hosts

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

        lockbox_redis_password_key = "password"
        redis_secrets_payload = lockbox.get_secret_payload(self.REDIS_PASSWORD_SECRET_ID)
        raw_redis_password = redis_secrets_payload.get(lockbox_redis_password_key).text_value.get_secret_value()
        # Store the raw password — URL encoding is applied only when building
        # a redis:// connection URL in RedisClient._build_direct_url().
        self.REDIS_PASSWORD = raw_redis_password


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
