### Generate classes for .proto files

#### traitex api files

#### External dependencies

```bash
python -m grpc_tools.protoc \
  --proto_path=../Personage.Auth/Personage.Auth.Api/Protos \
  --python_out=./proto --grpc_python_out=./proto --mypy_out=./proto \
  ../Personage.Auth/Personage.Auth.Api/Protos/telegram.proto \
  ../Personage.Auth/Personage.Auth.Api/Protos/common.proto \
  ../Personage.Auth/Personage.Auth.Api/Protos/telegram_chats.proto
```