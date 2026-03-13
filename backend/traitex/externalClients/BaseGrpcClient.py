from grpc import aio, ssl_channel_credentials


class BaseGrpcClient:
    def __init__(
            self,
            endpoint: str,
            use_ssl: bool | None = None
    ):
        # Auto-detect SSL based on port 443 if not explicitly specified
        if use_ssl is None:
            use_ssl = self._should_use_ssl(endpoint)
        
        if use_ssl:
            self._channel = aio.secure_channel(endpoint, ssl_channel_credentials())
        else:
            self._channel = aio.insecure_channel(endpoint)

    @staticmethod
    def _should_use_ssl(endpoint: str) -> bool:
        """Auto-detect SSL based on port 443."""
        # Extract port from endpoint (e.g., "host:443" or "host:port")
        if ":" in endpoint:
            # Split by last colon to handle IPv6 addresses
            parts = endpoint.rsplit(":", 1)
            if len(parts) == 2:
                port_str = parts[1]
                try:
                    port = int(port_str)
                    return port == 443
                except ValueError:
                    pass
        return False

    async def close(self):
        await self._channel.close()
