# onec-eventlog-mvp

Open-core MVP для удобного разбора журнала регистрации 1С, загрузки событий в ClickHouse и поиска через Event Explorer.

Базовый профиль MVP — односерверный. На сервере, где доступен каталог ЖР, запускается один сервис ЖР. Он вызывает штатную утилиту `ibcmd eventlog export`, читает JSON из stdout, сохраняет сырые события в локальный SQLite-буфер, нормализует их и пишет в ClickHouse.

## Что входит в MVP

- чтение ЖР через `ibcmd eventlog export --format=json --skip-root --follow`;
- локальный надёжный буфер на SQLite;
- встроенный разборщик ЖР;
- запись нормализованных событий в ClickHouse;
- HTTP-статус сервиса;
- базовая схема для Event Explorer.

## Что не входит в MVP

- кластерный сбор с нескольких серверов 1С;
- центральные агенты и контур управления;
- автоматическая корреляция ЖР + ТЖ;
- расследования, рекомендации и ИИ-разбор;
- управление сервером 1С, очистка или сокращение ЖР.

## Быстрый запуск инфраструктуры

```bash
cd deployments/docker-compose
docker compose up -d
```

Применить миграцию ClickHouse:

```bash
docker exec -i onec-eventlog-clickhouse clickhouse-client --multiquery < ../../migrations/clickhouse/001_eventlog_events.sql
```

Запуск сервиса:

```bash
go run ./cmd/eventlog-service -config ./config.example.yaml
```

## Логическая схема

```text
ibcmd eventlog export stdout
  ↓
сервис ЖР
  ↓
SQLite-буфер
  ↓
встроенный разборщик ЖР
  ↓
ClickHouse
  ↓
Event Explorer
```

## Подготовка Go-зависимостей

После клонирования репозитория выполните:

```bash
go mod tidy
```

В текущем каркасе используются:

- `modernc.org/sqlite` — SQLite без отдельной установки сервера;
- `clickhouse-go/v2` — запись в ClickHouse;
- `yaml.v3` — чтение конфигурации;
- `google/uuid` — идентификаторы запусков чтения.

## Состояние каркаса

Это первый технический инкремент. Он фиксирует структуру проекта, SQLite-буфер, запуск `ibcmd`, потоковое чтение JSON, встроенный разборщик и запись в ClickHouse. Event Explorer пока представлен как место под будущий web-интерфейс.
