import logging
from aiohttp import web

logger = logging.getLogger(__name__)


class HealthCheckServer:
    """Simple HTTP server for health and liveliness checks."""

    def __init__(self, host: str = "0.0.0.0", port: int = 8080):
        self.host = host
        self.port = port
        self.app = web.Application()
        self.app.add_routes([
            web.get("/health", self.health),
            web.get("/liveliness", self.liveliness),
        ])
        self.runner = web.AppRunner(self.app, access_log=None)
        self.site: web.TCPSite | None = None

    async def health(self, request: web.Request) -> web.Response:
        """Health check endpoint - returns 200 if service is healthy."""
        # TODO: Add actual health checks (DB connection, etc.)
        return web.Response(text="OK", status=200)

    async def liveliness(self, request: web.Request) -> web.Response:
        """Liveliness check endpoint - returns 200 if service is running."""
        return web.Response(text="OK", status=200)

    async def start(self) -> None:
        """Start the HTTP server."""
        await self.runner.setup()
        self.site = web.TCPSite(self.runner, self.host, self.port)
        await self.site.start()
        logger.info(f"Health check server started on http://{self.host}:{self.port}")

    async def stop(self) -> None:
        """Stop the HTTP server."""
        await self.runner.cleanup()
        logger.info("Health check server stopped")
