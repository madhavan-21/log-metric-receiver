```markdown
# 🛰️ Go Observability Consumers

This repository contains **Go-based consumers** that process observability data (logs and metrics) from NSQ topics.  
The logs are forwarded to **Loki** or **Quickwit**, and metrics are stored in **TimescaleDB** after optional enrichment via a memorizer.

---

## 📦 Components

| Component        | Purpose                                                                 |
|------------------|-------------------------------------------------------------------------|
| `log_consumer.go` | Consumes log batches from NSQ, forwards them to Loki or Quickwit       |
| `metric_consumer.go` | Consumes metric batches from NSQ, enriches and stores them in TimescaleDB |
| `receiver/`       | gRPC server that receives data from Go agents (not shown here)         |

---

## 🔌 NSQ + Loki + TimescaleDB Architecture

```

Go Agent
↓ (protobuf via gRPC)
Receiver (gRPC)
↓
NSQ Producer
↓
\[ NSQ Topic ]
↓
LogConsumer → Loki/Quickwit
MetricConsumer → TimescaleDB

````

---

## 🚀 Getting Started

### 1. Prerequisites

- NSQD (Port `4150`) and NSQLookupd (Port `4161`)
- Loki (Port `3100`) or Quickwit (Port `7280`)
- PostgreSQL + TimescaleDB (Port `5433`)
- Go 1.20+

---

## 🔧 Log Consumer

Consumes protobuf-encoded logs from NSQ (`logs_protobuf`) and forwards them to **Loki** or **Quickwit**.

### ✅ Configuration

Edit in `log_consumer.go`:

```go
const (
    lokiURL     = "http://localhost:3100/loki/api/v1/push"
    quickwitURL = "http://localhost:7280/api/v1/ingest"
    useLoki     = true // Switch to false to use Quickwit
)
````

### ✅ Run

```bash
go run log_consumer.go
```

### ✅ Output Example

```
📥 Received Log: [INFO] 2025-06-14 22:30:12 - main.go:34 main | Service started
📤 Log successfully sent to http://localhost:3100/loki/api/v1/push
```

---

## 📊 Metric Consumer

Consumes protobuf-encoded metrics from NSQ (`metrics_protobuf`) and stores them in **TimescaleDB** after passing through a **memorizer** for enrichment.

### ✅ DB Config

In `metric_consumer.go`:

```go
dbConnStr := "user=metrics_user password=metrics_password dbname=metrics_db sslmode=disable host=127.0.0.1 port=5433"
```

Make sure TimescaleDB has this schema:

```sql
CREATE TABLE metrics (
    name TEXT,
    type TEXT,
    value DOUBLE PRECISION,
    timestamp BIGINT,
    tags JSONB,
    project_name TEXT,
    hostname TEXT,
    os TEXT,
    unique_id TEXT,
    unit TEXT
);
```

### ✅ Run

```bash
go run metric_consumer.go
```

### ✅ Output Example

```
📥 Received 3 metrics.
✅ Successfully stored metric: cpu.usage
```

---

## 🧠 Memorizer

The `Memorizer` (from `bitbucket.org/minion/metrics-system/consumer/memorizer`) enriches incoming metrics with context or deduplication logic before storing.

You can modify it to include:

* Rate calculation
* Aggregation windows
* Derived metrics

---

## 📡 gRPC Receiver (Optional Component)

If you're running a full stack, you can deploy a **Go-based gRPC server** that receives telemetry from agents and forwards data to NSQ. Example protobuf structure:

```proto
service LogReceiver {
  rpc ReceiveLogs(LogBatch) returns (LogResponse);
}

service MetricReceiver {
  rpc ReceiveMetrics(MetricBatch) returns (MetricResponse);
}
```

Agents can send batched logs and metrics using gRPC, which is then pushed to NSQ topics `logs_protobuf` and `metrics_protobuf`.

---

## 🛑 Graceful Shutdown

Both consumers listen for `SIGINT` / `SIGTERM` and gracefully flush or disconnect:

```bash
^C
✅ NSQ Metric Consumer stopped gracefully.
```

---

## 🧪 Testing

1. Publish a test protobuf message to NSQ:

```bash
nsq_pub --topic=logs_protobuf --message="..." --nsqd-tcp-address=127.0.0.1:4150
```

2. Watch consumer log output and validate in Loki or TimescaleDB.

---

## 📁 Folder Structure

```
.
├── log_consumer.go        # NSQ to Loki/Quickwit
├── metric_consumer.go     # NSQ to TimescaleDB
├── consumer/
│   └── memorizer/         # Metric enrichment logic
├── protobuf/
│   ├── log_protobuf/      # LogBatch.proto
│   └── metric_protobuf/   # MetricBatch.proto
└── receiver/              # Optional gRPC server (not shown here)
```

---

## 📜 License

MIT © Minion Metrics Team

---
