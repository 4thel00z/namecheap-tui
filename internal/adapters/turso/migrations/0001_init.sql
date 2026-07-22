CREATE TABLE IF NOT EXISTS profiles (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    name       TEXT NOT NULL UNIQUE,
    api_user   TEXT NOT NULL,
    username   TEXT NOT NULL,
    client_ip  TEXT NOT NULL,
    endpoint   TEXT NOT NULL,
    is_default INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS secrets (
    profile_id INTEGER PRIMARY KEY REFERENCES profiles(id) ON DELETE CASCADE,
    api_key    TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS cache (
    profile    TEXT NOT NULL,
    kind       TEXT NOT NULL,
    key        TEXT NOT NULL,
    payload    BLOB NOT NULL,
    fetched_at INTEGER NOT NULL,
    PRIMARY KEY (profile, kind, key)
);
