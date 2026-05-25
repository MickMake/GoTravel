package storage

const schemaSQL = `
CREATE TABLE IF NOT EXISTS points (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    dt TEXT NOT NULL,
    lat REAL NOT NULL,
    lng REAL NOT NULL,
    altitude REAL,
    angle REAL,
    speed REAL,
    params TEXT,
    format TEXT NOT NULL,
    source_file TEXT NOT NULL,
    source_line INTEGER NOT NULL,
    imported_at TEXT NOT NULL,
    point_hash TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS import_runs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    format TEXT NOT NULL,
    source_file TEXT NOT NULL,
    started_at TEXT NOT NULL,
    finished_at TEXT,
    rows_seen INTEGER DEFAULT 0,
    rows_imported INTEGER DEFAULT 0,
    rows_skipped INTEGER DEFAULT 0,
    status TEXT NOT NULL,
    error TEXT
);

CREATE TABLE IF NOT EXISTS import_errors (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    import_run_id INTEGER,
    source_file TEXT NOT NULL,
    source_line INTEGER,
    format TEXT NOT NULL,
    raw_row TEXT,
    error TEXT NOT NULL,
    created_at TEXT NOT NULL,
    FOREIGN KEY(import_run_id) REFERENCES import_runs(id)
);

CREATE INDEX IF NOT EXISTS idx_points_dt ON points(dt);
CREATE INDEX IF NOT EXISTS idx_points_source ON points(source_file, source_line);
`
