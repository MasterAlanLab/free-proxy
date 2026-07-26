-- +goose Up
CREATE TABLE jobs (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    status      TEXT NOT NULL,
    created_at  TEXT NOT NULL,
    started_at  TEXT,
    finished_at TEXT,
    result      TEXT,
    error       TEXT
);
CREATE INDEX idx_jobs_name    ON jobs(name);
CREATE INDEX idx_jobs_status  ON jobs(status);
CREATE INDEX idx_jobs_created ON jobs(created_at);

CREATE TABLE probe_results (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    node_id    TEXT NOT NULL,
    available  INTEGER NOT NULL,
    latency_ms INTEGER NOT NULL,
    probed_at  TEXT NOT NULL,
    result     TEXT NOT NULL
);
CREATE INDEX idx_probe_node      ON probe_results(node_id);
CREATE INDEX idx_probe_available ON probe_results(available);
CREATE INDEX idx_probe_probed    ON probe_results(probed_at);

-- +goose Down
DROP TABLE probe_results;
DROP TABLE jobs;
