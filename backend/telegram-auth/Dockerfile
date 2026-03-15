FROM python:3.14-slim AS builder

WORKDIR /build

RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc \
    && rm -rf /var/lib/apt/lists/*

COPY requirements.txt .
RUN pip install --no-cache-dir --prefix=/install -r requirements.txt

FROM python:3.14-slim

WORKDIR /app

# Copy pre-built Python packages from builder
COPY --from=builder /install /usr/local

# Create non-root user
RUN useradd -m -u 1000 appuser && \
    mkdir -p /app/logs && \
    chown -R appuser:appuser /app

# Copy application code
COPY app/ /app/app/
COPY proto/ /app/proto/

ENV PYTHONPATH=/app/proto
ENV PYTHONUNBUFFERED=1

# Switch to non-root user
USER appuser

ENV REDIS_HOST=valkey \
    REDIS_PORT=6379 \
    REDIS_USE_SENTINEL=true \
    REDIS_SENTINEL_HOSTS="rc1d-ga172rk461nqc3a3.mdb.yandexcloud.net" \
    REDIS_SENTINEL_SERVICE_NAME=main \
    REDIS_SENTINEL_PORT=26379 \
    AUTH_SERVICE_GRPC_HOST=grpc-auth.persomanage.ru \
    LOGIN_TIMEOUT_SECONDS=300 \
    ENVIRONMENT=production

# Expose port
EXPOSE 8000

# Run with single worker for stateful service (Redis, gRPC, Telegram clients)
CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8000"]
