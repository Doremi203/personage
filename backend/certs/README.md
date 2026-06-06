# Yandex Cloud CA certificate

`root.crt` — корневой CA-сертификат Yandex Cloud, нужный для TLS-подключения
к managed-PostgreSQL. Раньше каждый Docker-образ тянул его по сети
(`wget https://storage.yandexcloud.net/cloud-certs/CA.pem`), что периодически
ломало сборки из-за сетевых сбоев. Теперь сертификат закоммичен в репозиторий
и просто копируется в образ.

Используется в:
- `backend/go.Dockerfile` (контекст сборки — `backend/`)
- `backend/traitex/Dockerfile` (контекст сборки — `backend/`)
- `backend/Personage.Auth/certs/root.crt` — отдельная копия, т.к. auth
  собирается из контекста `backend/Personage.Auth/` и не видит `backend/certs/`.

## Обновление

Сертификат публичный и меняется крайне редко. Чтобы обновить обе копии:

```bash
curl -fsSL https://storage.yandexcloud.net/cloud-certs/CA.pem \
    -o backend/certs/root.crt
cp backend/certs/root.crt backend/Personage.Auth/certs/root.crt
```
