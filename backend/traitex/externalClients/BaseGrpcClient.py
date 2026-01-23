from grpc import aio, ssl_channel_credentials


class BaseGrpcClient:
    def __init__(
            self,
            endpoint: str,
            use_ssl: bool = False
    ):
        if use_ssl:
            self._channel = aio.secure_channel(endpoint, ssl_channel_credentials())
        else:
            self._channel = aio.insecure_channel(endpoint)

    async def close(self):
        await self._channel.close()
