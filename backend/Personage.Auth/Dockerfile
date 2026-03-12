FROM alpine AS certs

RUN apk add --no-cache wget && \
    mkdir -p /certs && \
    wget "https://storage.yandexcloud.net/cloud-certs/CA.pem" \
        --output-document /certs/root.crt && \
    chmod 0655 /certs/root.crt

# ============================================================================
# Stage 1: restore — cache NuGet packages in a separate layer
# ============================================================================
FROM mcr.microsoft.com/dotnet/sdk:9.0 AS restore

WORKDIR /src

# Copy solution and all project files first for layer caching.
# NuGet restore only re-runs when project references change.
COPY Personage.Auth.sln ./
COPY Personage.Auth.Api/Personage.Auth.Api.csproj         Personage.Auth.Api/
COPY Personage.Auth.Bll/Personage.Auth.Bll.csproj         Personage.Auth.Bll/
COPY Personage.Auth.DataAccess/Personage.Auth.DataAccess.csproj Personage.Auth.DataAccess/
COPY Personage.Auth.Domain/Personage.Auth.Domain.csproj   Personage.Auth.Domain/

RUN dotnet restore Personage.Auth.Api/Personage.Auth.Api.csproj

# ============================================================================
# Stage 2: build + publish
# ============================================================================
FROM restore AS publish

# Copy all source code (tests excluded via .dockerignore)
COPY . .

RUN dotnet publish Personage.Auth.Api/Personage.Auth.Api.csproj \
    -c Release \
    --no-restore \
    -o /app/publish

# ============================================================================
# Stage 3: runtime
# ============================================================================
FROM mcr.microsoft.com/dotnet/aspnet:9.0 AS runtime

WORKDIR /app

# Copy Yandex Cloud CA certificate from the certs stage
COPY --from=certs /certs/root.crt /root/.postgresql/root.crt

COPY --from=publish /app/publish .

ENV ASPNETCORE_ENVIRONMENT=Production

ENTRYPOINT ["dotnet", "Personage.Auth.Api.dll"]
