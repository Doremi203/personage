import redis.asyncio as redis
from redis.asyncio.sentinel import Sentinel
import json
from tenacity import retry, stop_after_attempt, wait_exponential
import structlog
from app.config import settings

logger = structlog.get_logger()


class RedisClient:
    def __init__(self):
        self.client = None

    def _build_direct_url(self) -> str:
        """Build a direct redis:// connection URL."""
        if settings.REDIS_PASSWORD:
            import urllib.parse
            encoded_password = urllib.parse.quote(settings.REDIS_PASSWORD, safe='')
            url = f"redis://:{encoded_password}@{settings.REDIS_HOST}:{settings.REDIS_PORT}/{settings.REDIS_DB}"
        else:
            url = f"redis://{settings.REDIS_HOST}:{settings.REDIS_PORT}/{settings.REDIS_DB}"

        if settings.REDIS_SSL:
            url = url.replace("redis://", "rediss://")

        return url

    @retry(
        stop=stop_after_attempt(3),
        wait=wait_exponential(multiplier=1, min=2, max=10)
    )
    async def connect(self):
        """Establish Redis/Valkey connection with retry logic.

        Supports two modes:
        - Direct: connects to a single Redis/Valkey instance via URL
        - Sentinel: discovers master through Sentinel nodes (for HA setups)
        """
        try:
            if settings.REDIS_USE_SENTINEL:
                await self._connect_via_sentinel()
            else:
                await self._connect_direct()

            await self.client.ping()
            logger.info(
                "Connected to Redis/Valkey successfully",
                mode="sentinel" if settings.REDIS_USE_SENTINEL else "direct",
            )
        except Exception as e:
            logger.error("Failed to connect to Redis/Valkey", error=str(e))
            raise

    async def _connect_direct(self):
        """Connect directly to a Redis/Valkey instance."""
        redis_url = self._build_direct_url()
        self.client = await redis.from_url(
            redis_url,
            decode_responses=True,
            health_check_interval=30,
        )

    async def _connect_via_sentinel(self):
        """Connect to Redis/Valkey master discovered via Sentinel.

        Sentinel provides automatic failover: if the current master goes down,
        Sentinel promotes a replica and subsequent master_for() calls return
        the new master transparently.
        """
        sentinel_hosts = settings.get_sentinel_hosts()
        if not sentinel_hosts:
            raise ValueError(
                "REDIS_USE_SENTINEL is true but REDIS_SENTINEL_HOSTS is empty. "
                "Set REDIS_SENTINEL_HOSTS='host1:port1,host2:port2,...'"
            )

        connection_kwargs = {
            "decode_responses": True,
            "health_check_interval": 30,
            "db": settings.REDIS_DB,
        }
        if settings.REDIS_PASSWORD:
            connection_kwargs["password"] = settings.REDIS_PASSWORD
        if settings.REDIS_SSL:
            connection_kwargs["ssl"] = True

        # sentinel_kwargs authenticate to the Sentinel nodes themselves
        # (required when auth_sentinel=true on the cluster).
        sentinel_kwargs = {}
        if settings.REDIS_PASSWORD:
            sentinel_kwargs["password"] = settings.REDIS_PASSWORD
        if settings.REDIS_SSL:
            sentinel_kwargs["ssl"] = True

        sentinel = Sentinel(
            sentinel_hosts,
            socket_timeout=5,
            sentinel_kwargs=sentinel_kwargs or None,
        )

        self.client = sentinel.master_for(
            settings.REDIS_SENTINEL_SERVICE_NAME,
            redis_class=redis.Redis,
            **connection_kwargs,
        )

        logger.info(
            "Sentinel discovery complete",
            service_name=settings.REDIS_SENTINEL_SERVICE_NAME,
            sentinel_hosts=[f"{h}:{p}" for h, p in sentinel_hosts],
        )

    async def set_login_session(self, login_id: str, data: dict):
        """Store login session data with expiration"""
        ttl = settings.LOGIN_TIMEOUT_SECONDS
        key = f"login:{login_id}"
        await self.client.setex(key, ttl, json.dumps(data))
        logger.debug("Stored login session", login_id=login_id, ttl=ttl)

    async def get_login_session(self, login_id: str) -> dict | None:
        """Retrieve login session data"""
        key = f"login:{login_id}"
        data = await self.client.get(key)
        if data:
            return json.loads(data)
        return None

    async def delete_login_session(self, login_id: str):
        """Delete login session"""
        key = f"login:{login_id}"
        await self.client.delete(key)
        logger.debug("Deleted login session", login_id=login_id)

    async def close(self):
        """Close Redis/Valkey connection"""
        if self.client:
            await self.client.close()
            logger.info("Redis/Valkey connection closed")


redis_client = RedisClient()
