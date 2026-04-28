#!/bin/bash
set -e

# Скрипт для деплоя фронтенда на Yandex Object Storage
# Требует: AWS CLI и переменные окружения AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY

BUCKET_NAME="persomanage.ru"
DIST_DIR="./dist"
ENDPOINT_URL="https://storage.yandexcloud.net"

# Проверка наличия dist
if [ ! -d "$DIST_DIR" ]; then
    echo "Error: $DIST_DIR directory not found. Run 'npm run build' first."
    exit 1
fi

# Проверка AWS credentials
if [ -z "$AWS_ACCESS_KEY_ID" ] || [ -z "$AWS_SECRET_ACCESS_KEY" ]; then
    echo "Error: AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY must be set"
    echo "Run: source ../secrets.env"
    exit 1
fi

echo "Deploying frontend to s3://$BUCKET_NAME..."

# Sync всех файлов с правильными content-type
# --delete удаляет файлы которых нет в dist
aws s3 sync "$DIST_DIR" "s3://$BUCKET_NAME" \
    --endpoint-url "$ENDPOINT_URL" \
    --delete \
    --cache-control "max-age=31536000" \
    --exclude "index.html" \
    --exclude "*.json" \
    --exclude "sw.js" \
    --exclude "sw.js.map"

# index.html, json, and service worker files must revalidate on every request
# (so the SW update check and bundle entrypoint never go stale).
# Yandex CDN's edge_cache_settings.default_value (24h) kicks in whenever the
# origin Cache-Control has no max-age directive — including no-cache/no-store —
# so we must use max-age=0 explicitly to force per-request revalidation at the
# edge. See: https://cloud.yandex.com/en/docs/cdn/concepts/caching
aws s3 sync "$DIST_DIR" "s3://$BUCKET_NAME" \
    --endpoint-url "$ENDPOINT_URL" \
    --cache-control "max-age=0, must-revalidate" \
    --exclude "*" \
    --include "index.html" \
    --include "*.json" \
    --include "sw.js" \
    --include "sw.js.map"

echo "✅ Frontend deployed successfully!"
echo "🌐 https://$BUCKET_NAME"
