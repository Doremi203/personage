FROM python:3.14-slim AS build

WORKDIR /build

RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    gcc \
    && rm -rf /var/lib/apt/lists/*

COPY requirements.txt .
RUN pip install --no-cache-dir --prefix=/install -r requirements.txt

FROM build AS protogen

WORKDIR /proto-gen

COPY Personage.Auth/Personage.Auth.Api/Protos/*.proto ./externalClients/personage_auth/proto/
RUN pip install grpcio-tools

RUN python3 -m grpc_tools.protoc \
    -I=./externalClients/personage_auth/proto \
    --python_out=./externalClients/personage_auth/proto \
    --grpc_python_out=./externalClients/personage_auth/proto \
    --mypy_out=./externalClients/personage_auth/proto \
    externalClients/personage_auth/proto/*.proto


FROM python:3.14-slim

WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends \
    && rm -rf /var/lib/apt/lists/*

RUN useradd -m -u 1000 appuser && \
    mkdir -p /app/logs && \
    chown -R appuser:appuser /app

COPY --from=build /install /usr/local

COPY --from=protogen /proto-gen/externalClients /app/externalClients

COPY app/ /app/app/
COPY requirements.txt /app/

ENV PYTHONPATH=/app

USER appuser

ENV REDIS_HOST=redis
ENV REDIS_PORT=6379
ENV AUTH_SERVICE_GRPC_HOST=grpc-auth.persomanage.ru
ENV LOGIN_TIMEOUT_SECONDS=300
ENV ENVIRONMENT=production


EXPOSE 8000

CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8000", "--workers", "4"]