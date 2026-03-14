FROM python:3.14-slim AS build

WORKDIR /src

RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    libpq-dev \
    && rm -rf /var/lib/apt/lists/*

# Copy requirements and install Python dependencies
COPY traitex/requirements.txt ./
RUN pip install --no-cache-dir -r requirements.txt

# Copy traitex source code
COPY traitex/ ./

# Copy external Personage.Auth proto source files for generation
COPY Personage.Auth/Personage.Auth.Api/Protos/*.proto ./externalClients/personage_auth/proto/

# Generate protobuf Python classes for internal proto files
RUN python3 -m grpc_tools.protoc \
    -I=./proto \
    --python_out=./proto \
    --grpc_python_out=./proto \
    --mypy_out=./proto \
    proto/*.proto

# Generate protobuf Python classes for external Personage.Auth proto files
RUN python3 -m grpc_tools.protoc \
    -I=./externalClients/personage_auth/proto \
    --python_out=./externalClients/personage_auth/proto \
    --grpc_python_out=./externalClients/personage_auth/proto \
    --mypy_out=./externalClients/personage_auth/proto \
    externalClients/personage_auth/proto/*.proto

FROM alpine AS certs

RUN apk add --no-cache wget && \
    mkdir -p ~/.postgresql && \
    wget "https://storage.yandexcloud.net/cloud-certs/CA.pem" \
        --output-document ~/.postgresql/root.crt && \
    chmod 0655 ~/.postgresql/root.crt

# Runtime stage
FROM python:3.14-slim

WORKDIR /app

# Install runtime dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    libpq5 \
    && rm -rf /var/lib/apt/lists/*

# Copy certificate from certs stage
COPY --from=certs /root/.postgresql /root/.postgresql

# Copy Python dependencies from build stage
COPY --from=build /usr/local/lib/python3.14/site-packages /usr/local/lib/python3.14/site-packages
COPY --from=build /usr/local/bin /usr/local/bin

# Copy application code
COPY --from=build /src/app ./app
COPY --from=build /src/config ./config
COPY --from=build /src/dataAccess ./dataAccess
COPY --from=build /src/externalClients ./externalClients
COPY --from=build /src/messaging ./messaging
COPY --from=build /src/proto ./proto

# Set PYTHONPATH to include proto directories
ENV PYTHONPATH=/app/proto:/app/externalClients/personage_auth/proto

# Set environment to production by default
ENV APP_ENV=production

# Expose gRPC and HTTP ports
EXPOSE 5051 80

# Run the application
CMD ["python", "-m", "app.runner"]
