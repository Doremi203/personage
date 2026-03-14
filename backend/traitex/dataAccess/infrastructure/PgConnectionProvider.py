import pydapper
import logging
from contextlib import asynccontextmanager
from pydapper.commands import CommandsAsync
from dataAccess.infrastructure.IPgConnectionProvider import IPgConnectionProvider
from typing import AsyncIterator
from urllib.parse import quote_plus

logging.getLogger('dsnparse').setLevel(logging.WARNING)


class PgConnectionProvider(IPgConnectionProvider):
    def __init__(
            self,
            username: str,
            password: str,
            host: str,
            port: int,
            db: str,
            options: str = "",
    ):
        self.username = username
        self.password = password
        self.host = host
        self.port = port
        self.db = db
        self.options = options

    @asynccontextmanager
    async def get_connection(self) -> AsyncIterator[CommandsAsync]:
        async with pydapper.connect_async(
                PgConnectionProvider._get_connection_string(
                    user=self.username,
                    password=self.password,
                    host=self.host,
                    port=self.port,
                    dbname=self.db,
                    options=self.options,
                )
        ) as connection:
            yield connection

    @staticmethod
    def _get_connection_string(
            user: str,
            password: str,
            host: str,
            port: int,
            dbname: str,
            options: str = "",
    ) -> str:
        encoded_user = quote_plus(user)
        encoded_password = quote_plus(password)
        dsn = f"postgresql+psycopg://{encoded_user}:{encoded_password}@{host}:{port}/{dbname}"
        if options:
            dsn += f"?{options}"
        return dsn
