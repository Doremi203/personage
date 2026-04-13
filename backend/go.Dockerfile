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

RUN apk add --no-cache wget && \
    mkdir -p ~/.postgresql && \
    wget "https://storage.yandexcloud.net/cloud-certs/CA.pem" \
        --output-document ~/.postgresql/root.crt && \
    chmod 0655 ~/.postgresql/root.crt

COPY --from=build /src/app ./
COPY --from=build /src/$service/configs configs
COPY --from=build /src/$service/swagger swagger
CMD ["./app"]
