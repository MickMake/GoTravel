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

CREATE TABLE IF NOT EXISTS route_match_runs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    provider TEXT NOT NULL,
    profile TEXT NOT NULL,
    status TEXT NOT NULL,
    source_point_count INTEGER NOT NULL,
    geometry TEXT,
    geometry_format TEXT,
    distance_meters REAL,
    duration_seconds REAL,
    confidence REAL,
    warnings_json TEXT,
    raw_response BLOB,
    matched_at TEXT NOT NULL,
    created_at TEXT NOT NULL,
    source_filter_start TEXT,
    source_filter_end TEXT
);

CREATE TABLE IF NOT EXISTS route_match_points (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    route_match_run_id INTEGER NOT NULL,
    point_id INTEGER NOT NULL,
    sequence INTEGER NOT NULL,
    FOREIGN KEY(route_match_run_id) REFERENCES route_match_runs(id),
    FOREIGN KEY(point_id) REFERENCES points(id),
    UNIQUE(route_match_run_id, sequence)
);

CREATE TABLE IF NOT EXISTS trips (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    start_time TEXT NOT NULL,
    end_time TEXT NOT NULL,
    source_point_count INTEGER NOT NULL,
    first_point_id INTEGER NOT NULL,
    last_point_id INTEGER NOT NULL,
    duration_seconds INTEGER NOT NULL,
    gap_seconds INTEGER NOT NULL,
    created_at TEXT NOT NULL,
    FOREIGN KEY(first_point_id) REFERENCES points(id),
    FOREIGN KEY(last_point_id) REFERENCES points(id)
);

CREATE TABLE IF NOT EXISTS trip_points (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    trip_id INTEGER NOT NULL,
    point_id INTEGER NOT NULL,
    sequence INTEGER NOT NULL,
    FOREIGN KEY(trip_id) REFERENCES trips(id) ON DELETE CASCADE,
    FOREIGN KEY(point_id) REFERENCES points(id),
    UNIQUE(trip_id, sequence)
);

CREATE INDEX IF NOT EXISTS idx_points_dt ON points(dt);
CREATE INDEX IF NOT EXISTS idx_points_source ON points(source_file, source_line);
CREATE INDEX IF NOT EXISTS idx_route_match_runs_matched_at ON route_match_runs(matched_at);
CREATE INDEX IF NOT EXISTS idx_route_match_runs_provider ON route_match_runs(provider, profile);
CREATE INDEX IF NOT EXISTS idx_route_match_points_run ON route_match_points(route_match_run_id, sequence);
CREATE INDEX IF NOT EXISTS idx_trips_start_time ON trips(start_time);
CREATE INDEX IF NOT EXISTS idx_trip_points_trip ON trip_points(trip_id, sequence);
CREATE INDEX IF NOT EXISTS idx_trip_points_point ON trip_points(point_id);
`
