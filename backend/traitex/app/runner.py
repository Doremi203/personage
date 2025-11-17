import asyncio
import logging
import signal
import sys
from app.core.containers import ApplicationContainer

logger = logging.getLogger(__name__)


class ApplicationRunner:
    def __init__(self):
        self.container = ApplicationContainer()
        self.shutdown_event = asyncio.Event()

    async def start(self) -> None:
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
        logger.info("Service shutdown complete")


def setup_signal_handlers(runner: ApplicationRunner) -> None:
    def signal_handler(signum, frame):
        logger.info(f"Received signal {signum}, initiating shutdown...")
        runner.shutdown_event.set()

    signal.signal(signal.SIGINT, signal_handler)
    signal.signal(signal.SIGTERM, signal_handler)


async def main() -> None:
    """Main application entry point"""
    logging.basicConfig(
        level=logging.INFO,
        format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
        stream=sys.stdout
    )

    runner = ApplicationRunner()
    setup_signal_handlers(runner)

    try:
        await runner.start()
    except KeyboardInterrupt:
        logger.info("Received keyboard interrupt")
    except Exception as e:
        logger.error(f"Application error: {e}")
    finally:
        await runner.stop()


if __name__ == "__main__":
    asyncio.run(main())
