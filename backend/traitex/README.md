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

```bash
# Install dependencies
pip install -r requirements.txt

# Run the service
python -m app.runner
```
---

## Architecture
This is an API-less service that runs as a continuous background process, consuming messages from datasources and publishing enriched events.