FROM golang:1.26.1-alpine AS build
ARG service
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
WORKDIR /src/$service/cmd/app
RUN go build -ldflags="-s -w" -o /src/app

FROM alpine
ARG service
WORKDIR /app

# Yandex Cloud CA certificate is vendored at backend/certs/root.crt
# (see backend/certs/README.md) instead of being fetched over the network,
# which used to make builds flaky/fail on registry connectivity issues.
COPY certs/root.crt /root/.postgresql/root.crt

COPY --from=build /src/app ./
COPY --from=build /src/$service/configs configs
COPY --from=build /src/$service/swagger swagger
CMD ["./app"]
