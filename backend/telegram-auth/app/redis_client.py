import redis.asyncio as redis
import json
from tenacity import retry, stop_after_attempt, wait_exponential
import structlog
from app.config import settings

logger = structlog.get_logger()


class RedisClient:
    def __init__(self):
        self.redis_url = f"redis://:{settings.REDIS_PASSWORD}@{settings.REDIS_HOST}:{settings.REDIS_PORT}/{settings.REDIS_DB}" if settings.REDIS_PASSWORD \
            else f"redis://{settings.REDIS_HOST}:{settings.REDIS_PORT}/{settings.REDIS_DB}"

        if settings.REDIS_SSL:
            self.redis_url = self.redis_url.replace("redis://", "rediss://")

        self.client = None

    @retry(
        stop=stop_after_attempt(3),
        wait=wait_exponential(multiplier=1, min=2, max=10)
    )
    async def connect(self):
        """Establish Redis connection with retry logic"""
        try:
            self.client = await redis.from_url(
                self.redis_url,
                decode_responses=True,
                health_check_interval=30
            )
            await self.client.ping()
            logger.info("Connected to Redis successfully")
        except Exception as e:
            logger.error("Failed to connect to Redis", error=str(e))
            raise

    async def set_login_session(self, login_id: str, data: dict, ttl: int = None):
        ttl = ttl or settings.LOGIN_TIMEOUT_SECONDS
        key = f"login:{login_id}"
        await self.client.setex(key, ttl, json.dumps(data))
        logger.debug("Stored login session", login_id=login_id, ttl=ttl)

    async def get_login_session(self, login_id: str) -> dict | None:
        key = f"login:{login_id}"
        data = await self.client.get(key)
        if data:
            return json.loads(data)
        return None

    async def delete_login_session(self, login_id: str):
        key = f"login:{login_id}"
        await self.client.delete(key)
        logger.debug("Deleted login session", login_id=login_id)

    async def set_user_active_login(self, user_id: str, login_id: str):
        key = f"user:{user_id}:active_logins"
        await self.client.sadd(key, login_id)
        await self.client.expire(key, settings.LOGIN_TIMEOUT_SECONDS)

    async def get_user_active_logins(self, user_id: str) -> list:
        key = f"user:{user_id}:active_logins"
        return await self.client.smembers(key) or []

    async def remove_user_active_login(self, user_id: str, login_id: str):
        key = f"user:{user_id}:active_logins"
        await self.client.srem(key, login_id)

    async def cleanup_user_sessions(self, user_id: str):
        active_logins = await self.get_user_active_logins(user_id)
        for login_id in active_logins:
            await self.delete_login_session(login_id)
        key = f"user:{user_id}:active_logins"
        await self.client.delete(key)

    async def close(self):
        if self.client:
            await self.client.close()
            logger.info("Redis connection closed")


redis_client = RedisClient()
