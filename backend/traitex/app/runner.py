import asyncio
import logging
import signal
import sys
import os
import grpc
from app.core.configuration.config import Configuration
from app.core.containers.ApplicationContainer import create_application_container
from app.core.containers.GrpcServiceContainer import register_grpc_services
from app.http_server import HealthCheckServer

logger = logging.getLogger(__name__)

GRACE_PERIOD_IN_SECOND = 15

class ApplicationRunner:
    def __init__(self, config: Configuration):
        self.config = config
        self.container = create_application_container(config)
        self.shutdown_event = asyncio.Event()
        self.server: grpc.Server | None = None
        self.http_server: HealthCheckServer | None = None

    async def start(self) -> None:
        # Start HTTP health check server
        http_port = self.config.get("Hosting.HttpPort", 8080)
        self.http_server = HealthCheckServer(port=http_port)
        await self.http_server.start()

        # Start gRPC server
        self.server = grpc.aio.server()
        register_grpc_services(self.container, self.server)
        grpc_port = self.config.get("Hosting.ServerGrpcPort")
        if not grpc_port:
            logger.error("No grpc-server port configured")
            return

        try:
            self.server.add_insecure_port(f'[::]:{grpc_port}')
            logger.info(f"Server started on [::]:{grpc_port}")

        except Exception as e:
            logger.error(f"Failed to start server: {str(e)}")
            raise

        await self.server.start()
        logger.info("Server is running...")


        try:
            logger.info("Starting Personage Traitex service...")
            await self._start_consumers()
            logger.info("Service started successfully")
            await self.shutdown_event.wait()

        except Exception as e:
            logger.error(f"Failed to start service: {e}")
            raise

    async def _start_consumers(self):
        logger.info("Starting consumers...")
        for consumer in self.container.consumers.all_consumers():
            await consumer.start()
        logger.info("All consumers started")

    async def stop(self):
        logger.info("Shutting down service...")
        for consumer in self.container.consumers.all_consumers():
            await consumer.stop()

        if self.http_server:
            await self.http_server.stop()

        await self.server.stop(GRACE_PERIOD_IN_SECOND)
        logger.info("Service shutdown complete")


def setup_signal_handlers(runner: ApplicationRunner) -> None:
    def signal_handler(signum, _):
        logger.info(f"Received signal {signum}, initiating shutdown...")
        runner.shutdown_event.set()

    signal.signal(signal.SIGINT, signal_handler)
    signal.signal(signal.SIGTERM, signal_handler)


async def serve() -> None:
    try:
        config = Configuration()

        logging_config = config.get_section("Logging")
        logging.basicConfig(
            level=getattr(logging, logging_config.get("Level", "INFO")),
            format=logging_config.get("Format", '%(asctime)s - %(name)s - %(levelname)s - %(message)s'),
            stream=sys.stdout
        )

        logger.info(f"Application started. Environment: {os.getenv('APP_ENV', 'development')}")

        runner = ApplicationRunner(config)
        setup_signal_handlers(runner)

        try:
            await runner.start()
        except KeyboardInterrupt:
            logger.info("Received keyboard interrupt")
        except Exception as e:
            logger.error(f"Application error: {e}")
            raise
        finally:
            await runner.stop()

    except Exception as e:
        logger.error(f"Failed to start application: {e}", exc_info=True)
        sys.exit(1)

if __name__ == '__main__':
    if os.name == 'nt':
        asyncio.set_event_loop_policy(asyncio.WindowsSelectorEventLoopPolicy())

    if 'pydevd' in sys.modules:
        loop = asyncio.new_event_loop()
        asyncio.set_event_loop(loop)
        try:
            loop.run_until_complete(serve())
        finally:
            loop.close()
    else:
        asyncio.run(serve())