# Personage Traitex

A background service that ingests data from various sources (messengers, mail, calendar) and extracts traits using NLP to create enriched events for task processing.

---
## Purpose

- **Listens** to multiple data sources for incoming messages/events
- **Extracts traits** like locations, deadlines, importance, people attribution using NLP
- **Enriches** raw data with extracted traits
- **Publishes** enriched events to message queue for downstream task processing
---

## Quick Start

### Activate venv

#### windows
```cmd
python -m venv .venv
.venv\Scripts\activate.bat
```

#### unix
```bash
python3 -m venv .venv
source .venv/bin/activate
```
---
### Install dependencies
```bash
pip install -r requirements.txt
```
---
### Set PYTHONPATH
#### windows
```cmd
set PYTHONPATH=%cd%
```

#### unix
```bash
export PYTHONPATH=$(pwd)/externalClient/personage_auth/proto
```

### Generate classes for .proto files

#### traitex api files
```bash
python3 -m grpc_tools.protoc -I=./proto --python_out=./proto --grpc_python_out=./proto --mypy_out=./proto proto/*.proto
```

#### External dependencies
```bash
python3 -m grpc_tools.protoc --proto_path=../Personage.Auth/Personage.Auth.Api/Protos --python_out=./externalClients/personage_auth/proto --grpc_python_out=./externalClients/personage_auth/proto --mypy_out=./externalClients/personage_auth/proto ../Personage.Auth/Personage.Auth.Api/Protos/*.proto
```

```bash
# Run the service
python -m app.runner
```
---

## Architecture
This is an API-less service that runs as a continuous background process, consuming messages from datasources and publishing enriched events.