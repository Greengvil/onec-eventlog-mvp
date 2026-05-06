# Пример конфигурации

## Режим ibcmd (по умолчанию)

Сервис запускает команду `ibcmd eventlog export` и читает события из stdout.

```yaml
service:
  name: onec-eventlog-service
  http_addr: "127.0.0.1:8080"

source:
  source_id: prod-main-local
  source_name: "ЖР УХ PROD"
  infobase_id: prod-main
  infobase_name: "УХ PROD"
  source_node_id: app01
  mode: "ibcmd"  # default
  ibcmd_path: "ibcmd"
  log_path: "/var/lib/1c/srvinfo/reg_1541/<infobase-id>/1Cv8Log"
  from: "2026-05-01T00:00:00"
  follow_interval_ms: 1000

buffer:
  path: "./data/agent.db"
  batch_size: 500
  retention_after_ack_hours: 24

clickhouse:
  address: "localhost:9000"
  database: "onec_eventlog"
  username: "default"
  password: ""
  table: "eventlog_events"
```

## Режим file (для разработки и тестирования)

Сервис читает события из JSONL-файла (одно событие ЖР на одной строке).

```yaml
service:
  name: onec-eventlog-service
  http_addr: "127.0.0.1:8080"

source:
  source_id: dev-local
  source_name: "ЖР Dev"
  infobase_id: dev
  infobase_name: "Dev"
  source_node_id: local
  mode: "file"
  file_path: "./testdata/events.jsonl"

buffer:
  path: "./data/agent.db"
  batch_size: 500
  retention_after_ack_hours: 24

clickhouse:
  address: "localhost:9000"
  database: "onec_eventlog"
  username: "default"
  password: ""
  table: "eventlog_events"
```

### Формат JSONL файла

Каждая строка должна содержать одно валидное JSON событие ЖР:

```jsonl
{"Date":"2026-05-01T10:00:00","User":"admin","Computer":"PC01","Application":"1CV8","Event":"","Severity":"I","TransactionStatus":"","TransactionStartTime":"","TransactionID":"","SessionNumber":"","ClientID":"","ServerID":"","MainPort":"","SecondPort":"","Message":""}
{"Date":"2026-05-01T10:01:00","User":"user1","Computer":"PC02","Application":"1CV8","Event":"","Severity":"W","TransactionStatus":"","TransactionStartTime":"","TransactionID":"","SessionNumber":"","ClientID":"","ServerID":"","MainPort":"","SecondPort":"","Message":""}
```
