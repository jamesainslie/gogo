package db

// Database schema migrations
const (
	createTemplatesTable = `
CREATE TABLE IF NOT EXISTS templates (
    id              INTEGER PRIMARY KEY,
    name            TEXT NOT NULL UNIQUE,
    kind            TEXT NOT NULL,
    content         BLOB NOT NULL,
    metadata_json   TEXT NOT NULL DEFAULT '{}',
    created_at      TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);`

	createBlueprintsTable = `
CREATE TABLE IF NOT EXISTS blueprints (
    id              INTEGER PRIMARY KEY,
    name            TEXT NOT NULL UNIQUE,
    stack           TEXT NOT NULL,
    config_json     TEXT NOT NULL,
    created_at      TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);`

	createConfigsTable = `
CREATE TABLE IF NOT EXISTS configs (
    id              INTEGER PRIMARY KEY,
    scope           TEXT NOT NULL DEFAULT 'global',
    key             TEXT NOT NULL,
    value           TEXT NOT NULL,
    UNIQUE(scope, key)
);`

	createHooksTable = `
CREATE TABLE IF NOT EXISTS hooks (
    id              INTEGER PRIMARY KEY,
    name            TEXT NOT NULL,
    event           TEXT NOT NULL,
    language        TEXT NOT NULL DEFAULT 'shell',
    script          TEXT NOT NULL,
    enabled         INTEGER NOT NULL DEFAULT 1
);`

	createPluginsTable = `
CREATE TABLE IF NOT EXISTS plugins (
    id              INTEGER PRIMARY KEY,
    name            TEXT NOT NULL UNIQUE,
    version         TEXT NOT NULL,
    entrypoint      TEXT NOT NULL,
    metadata_json   TEXT NOT NULL DEFAULT '{}'
);`

	createAuditsTable = `
CREATE TABLE IF NOT EXISTS audits (
    id              INTEGER PRIMARY KEY,
    actor           TEXT NOT NULL,
    action          TEXT NOT NULL,
    entity          TEXT NOT NULL,
    details_json    TEXT NOT NULL DEFAULT '{}',
    created_at      TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);`

	createIndexes = `
CREATE INDEX IF NOT EXISTS idx_templates_kind ON templates(kind);
CREATE INDEX IF NOT EXISTS idx_blueprints_stack ON blueprints(stack);
CREATE INDEX IF NOT EXISTS idx_configs_scope_key ON configs(scope, key);
CREATE INDEX IF NOT EXISTS idx_hooks_event ON hooks(event);
CREATE INDEX IF NOT EXISTS idx_audits_action ON audits(action);
CREATE INDEX IF NOT EXISTS idx_audits_created_at ON audits(created_at);`
)
