from typing import Any
import logging
import aioboto3
from botocore.config import Config

from app.domain.interfaces.messaging.IQueueClient import (
    BatchResult,
    BatchSendFailure,
    IQueueClient,
    QueueEntry,
)

logger = logging.getLogger(__name__)


class YMQClient(IQueueClient):
    QUEUE_URL_PREFIX = "https://message-queue.api.cloud.yandex.net"
    BATCH_CHUNK_SIZE = 10

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
            queue_url = self._resolve_queue_url(target_queue_url)
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

    async def send_batch(
            self,
            entries: list[QueueEntry],
            target_queue_url: str | None = None
    ) -> BatchResult:
        result = BatchResult()
        if not entries:
            return result

        queue_url = self._resolve_queue_url(target_queue_url)
        async with self.session.client(
                service_name='sqs',
                endpoint_url=self.endpoint_url,
                config=Config(
                    signature_version='v4',
                    retries={'max_attempts': 3}
                )
        ) as client:
            for chunk_start in range(0, len(entries), self.BATCH_CHUNK_SIZE):
                chunk = entries[chunk_start:chunk_start + self.BATCH_CHUNK_SIZE]
                sqs_entries = [self._to_sqs_entry(entry) for entry in chunk]

                try:
                    response = await client.send_message_batch(
                        QueueUrl=queue_url,
                        Entries=sqs_entries,
                    )
                except Exception as exc:
                    logger.exception(
                        "send_message_batch call failed for chunk of %s entries (ids=%s)",
                        len(chunk),
                        [entry.id for entry in chunk],
                    )
                    for entry in chunk:
                        result.failed.append(BatchSendFailure(
                            id=entry.id,
                            code='BatchCallFailed',
                            message=str(exc),
                            sender_fault=False,
                        ))
                    continue

                for ok in response.get('Successful', []):
                    result.successful_ids.append(ok['Id'])

                for failed in response.get('Failed', []):
                    result.failed.append(BatchSendFailure(
                        id=failed['Id'],
                        code=failed.get('Code', 'Unknown'),
                        message=failed.get('Message', ''),
                        sender_fault=bool(failed.get('SenderFault', False)),
                    ))

        if result.failed:
            logger.warning(
                "send_batch finished with %s successes and %s failures",
                len(result.successful_ids),
                len(result.failed),
            )

        return result

    def _resolve_queue_url(self, target_queue_url: str | None) -> str:
        if target_queue_url and target_queue_url.strip():
            return target_queue_url.strip()
        return self.default_queue_url

    def _to_sqs_entry(self, entry: QueueEntry) -> dict[str, Any]:
        sqs_entry: dict[str, Any] = {
            'Id': entry.id,
            'MessageBody': entry.message_body,
            'MessageGroupId': entry.message_group_id,
            'MessageDeduplicationId': entry.deduplication_id,
        }
        if entry.message_attributes:
            sqs_entry['MessageAttributes'] = self._format_attributes(entry.message_attributes)
        return sqs_entry

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
