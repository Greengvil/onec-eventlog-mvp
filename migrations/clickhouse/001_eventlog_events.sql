CREATE DATABASE IF NOT EXISTS onec_eventlog;

CREATE TABLE IF NOT EXISTS onec_eventlog.eventlog_events
(
    event_fingerprint String,

    source_id LowCardinality(String),
    source_node_id LowCardinality(String),
    infobase_id LowCardinality(String),
    infobase_name String,

    event_time DateTime64(3, 'UTC'),
    level LowCardinality(String),
    application LowCardinality(String),
    application_presentation String,
    event_name LowCardinality(String),
    event_presentation String,

    user_id String,
    user_name String,
    metadata_names Array(String),
    metadata_presentations Array(String),

    comment String,
    data_presentation String,
    data_json String,

    transaction_status LowCardinality(String),
    transaction_id String,
    connection Int64,
    session Int64,
    server_name LowCardinality(String),
    port Int64,
    sync_port Int64,

    session_data_separation_json String,
    session_data_separation_presentation_json String,

    raw_payload String,
    raw_hash String,
    parser_version LowCardinality(String),
    ingested_at DateTime64(3, 'UTC')
)
ENGINE = ReplacingMergeTree(ingested_at)
PARTITION BY toYYYYMM(event_time)
ORDER BY (infobase_id, event_time, event_name, user_name, session, connection, transaction_id, event_fingerprint);

CREATE TABLE IF NOT EXISTS onec_eventlog.eventlog_events_errors
(
    created_at DateTime64(3, 'UTC'),
    source_id String,
    infobase_id String,
    payload_hash String,
    raw_payload String,
    error_text String
)
ENGINE = MergeTree
PARTITION BY toYYYYMM(created_at)
ORDER BY (created_at, source_id, infobase_id);
