# Smoke-проверка source.mode=file

Этот сценарий проверяет минимальный путь `file` режима: JSONL-файл -> SQLite-буфер -> parser -> ClickHouse.

Тестовый файл: `testdata/eventlog_sample.jsonl`.

Конфиг для запуска: `config.file.example.yaml`.

## 1. Поднять ClickHouse

Из корня проекта:

```powershell
docker compose -f deployments/docker-compose/docker-compose.yml up -d
```

Применить миграцию:

```powershell
Get-Content migrations\clickhouse\001_eventlog_events.sql | docker exec -i onec-eventlog-clickhouse clickhouse-client --multiquery
```

Проверить, что ClickHouse отвечает:

```powershell
docker exec -i onec-eventlog-clickhouse clickhouse-client --query "SELECT 1"
```

## 2. Запустить сервис в file-режиме

В первом терминале:

```powershell
go run ./cmd/eventlog-service -config ./config.file.example.yaml
```

В `file` режиме сервис читает файл до конца, затем ждёт, пока локальный SQLite-буфер будет обработан ClickHouse worker-ом. После опустошения буфера процесс завершается штатно.

## 3. Проверить /health и /status

Пока сервис ждёт опустошения буфера, во втором терминале:

```powershell
Invoke-WebRequest http://localhost:8080/health
Invoke-RestMethod http://localhost:8080/status
```

В `/status` ожидаемые признаки:

- `current_source_id` равен `file-smoke-local`;
- `current_infobase_id` равен `file-smoke`;
- `read_events` становится `4`;
- `buffered_events` становится `4`;
- после записи в ClickHouse `parsed_events` и `clickhouse_writes` становятся `4`;
- после drain-завершения `/health` и `/status` больше не отвечают, потому что процесс штатно остановился.

## 4. Проверить данные в ClickHouse

```powershell
docker exec -i onec-eventlog-clickhouse clickhouse-client --query "SELECT count() FROM onec_eventlog.eventlog_events WHERE source_id = 'file-smoke-local'"
```

Ожидаемый результат для чистой базы:

```text
4
```

Посмотреть несколько записей:

```powershell
docker exec -i onec-eventlog-clickhouse clickhouse-client --query "SELECT event_time, level, event_name, user_name, comment FROM onec_eventlog.eventlog_events WHERE source_id = 'file-smoke-local' ORDER BY event_time FORMAT PrettyCompact"
```

Посмотреть основные нормализованные поля ЖР:

```powershell
docker exec -i onec-eventlog-clickhouse clickhouse-client --query "SELECT event_time, event_name, user_name, metadata_names AS metadata_name, data_presentation, transaction_status, transaction_id, connection, session, server_name FROM onec_eventlog.eventlog_events WHERE source_id = 'file-smoke-local' ORDER BY event_time FORMAT PrettyCompact"
```

Для реальной выгрузки `ibcmd eventlog export` проверить заполнение alias-полей:

```powershell
docker exec -i onec-eventlog-clickhouse clickhouse-client --query "SELECT event_time, application, event_name, user_name, metadata_names, data_presentation FROM onec_eventlog.eventlog_events WHERE source_id = 'real-jr-local' ORDER BY event_time LIMIT 10 FORMAT PrettyCompact"
```

Проверить отпечатки событий:

```powershell
docker exec -i onec-eventlog-clickhouse clickhouse-client --query "SELECT event_fingerprint, event_time, event_name, user_name FROM onec_eventlog.eventlog_events WHERE source_id = 'file-smoke-local' ORDER BY event_time FORMAT PrettyCompact"
```

Для проверки повторного запуска выполните тот же запуск ещё раз, не удаляя `data/file-mode-smoke.db`:

```powershell
go run ./cmd/eventlog-service -config ./config.file.example.yaml
```

После повторного запуска на том же JSONL-файле количество уникальных `event_fingerprint` должно остаться тем же:

```powershell
docker exec -i onec-eventlog-clickhouse clickhouse-client --query "SELECT count() AS rows, uniqExact(event_fingerprint) AS unique_fingerprints FROM onec_eventlog.eventlog_events WHERE source_id = 'file-smoke-local'"
```

## 5. Повторная проверка

Если smoke запускается повторно на той же базе, количество строк может быть больше `4`. Для чистого повторного прогона можно удалить локальный SQLite-буфер и строки smoke-источника:

```powershell
Remove-Item .\data\file-mode-smoke.db -ErrorAction SilentlyContinue
docker exec -i onec-eventlog-clickhouse clickhouse-client --query "ALTER TABLE onec_eventlog.eventlog_events DELETE WHERE source_id = 'file-smoke-local'"
```

После этого снова запустить сервис:

```powershell
go run ./cmd/eventlog-service -config ./config.file.example.yaml
```
