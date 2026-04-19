from typing import Any
import logging
import aioboto3
from botocore.config import Config

from app.domain.interfaces.messaging.IQueueClient import IQueueClient

logger = logging.getLogger(__name__)


class YMQClient(IQueueClient):
    QUEUE_URL_PREFIX = "https://message-queue.api.cloud.yandex.net"

    def __init__(
            self,
            access_key: str,
            secret_key: str,
            endpoint_url: str,
            default_queue_url: str,
            region: str = "ru-central1",
    ):
        self.endpoint_url = self._normalize_service_endpoint(endpoint_url)
        self.default_queue_url = default_queue_url.rstrip('/')
        self.session = aioboto3.Session(
            region_name=region,
            aws_access_key_id=access_key,
            aws_secret_access_key=secret_key,
        )

        logger.info(f"Async YMQ client initialized for region: {region}")


    async def send_single(
            self,
            deduplication_id: str,
            message_group_id: str,
            message_body: str,
            message_attributes: dict[str, Any] | None = None,
            target_queue_url: str | None = None
    ) -> dict[str, Any]:
        async with self.session.client(
                service_name='sqs',
                endpoint_url=self.endpoint_url,
                config=Config(
                    signature_version='v4',
                    retries={'max_attempts': 3}
                )
        ) as client:
            queue_url = target_queue_url.strip() if target_queue_url and target_queue_url.strip() else self.default_queue_url
            params: dict[str, Any] = {
                'QueueUrl': queue_url,
                'MessageBody': message_body,
                'MessageGroupId': message_group_id,
                'MessageDeduplicationId': deduplication_id,
            }

            if message_attributes:
                params['MessageAttributes'] = self._format_attributes(message_attributes)

            response = await client.send_message(**params)
            logger.debug(f"Sent message to queue: {response['MessageId']}")
            return response

    @classmethod
    def _normalize_service_endpoint(cls, endpoint_url: str) -> str:
        normalized = endpoint_url.rstrip('/')
        if normalized.startswith(f"{cls.QUEUE_URL_PREFIX}/"):
            return cls.QUEUE_URL_PREFIX
        return normalized


    @staticmethod
    def _format_attributes(attributes: dict[str, Any]) -> dict[str, dict]:
        formatted = {}
        for key, value in attributes.items():
            if isinstance(value, str):
                formatted[key] = {
                    'StringValue': value,
                    'DataType': 'String'
                }
            elif isinstance(value, (int, float)):
                formatted[key] = {
                    'StringValue': str(value),
                    'DataType': 'Number'
                }
            elif isinstance(value, bool):
                formatted[key] = {
                    'StringValue': str(value).lower(),
                    'DataType': 'String'
                }
            else:
                formatted[key] = {
                    'StringValue': str(value),
                    'DataType': 'String'
                }
        return formatted

    async def close(self):
        """Close the client connection."""
        pass
