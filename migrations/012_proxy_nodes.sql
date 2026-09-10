CREATE TABLE proxy_nodes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    node_type TEXT NOT NULL CHECK (node_type IN ('proxy', 'relay')),
    name TEXT NOT NULL COLLATE NOCASE UNIQUE CHECK (length(name) BETWEEN 1 AND 128 AND name = trim(name)),
    region TEXT NOT NULL CHECK (length(region) BETWEEN 1 AND 128 AND region = trim(region)),
    host TEXT NOT NULL CHECK (length(host) BETWEEN 1 AND 255),
    public_host TEXT NOT NULL CHECK (length(public_host) BETWEEN 1 AND 255),
    mtproto_port INTEGER NOT NULL CHECK (mtproto_port BETWEEN 1 AND 65535),
    internal_api_endpoint TEXT NOT NULL CHECK (length(internal_api_endpoint) BETWEEN 1 AND 1024),
    enabled INTEGER NOT NULL DEFAULT 1 CHECK (enabled IN (0, 1)),
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE INDEX idx_proxy_nodes_type_enabled
    ON proxy_nodes(node_type, enabled, id);
