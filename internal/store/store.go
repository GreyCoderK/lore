// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/greycoderk/lore/internal/domain"

	_ "modernc.org/sqlite"
)

// Compile-time check: SQLiteStore implements LoreStore.
var _ domain.LoreStore = (*SQLiteStore)(nil)

// SQLiteStore is the SQLite-backed implementation of domain.LoreStore.
type SQLiteStore struct {
	db *sql.DB
}

// Open creates or opens a SQLite database at dbPath with WAL mode,
// foreign keys ON, and busy_timeout 5000ms. Applies schema migration if needed.
func Open(dbPath string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("store: open: %w", err)
	}

	// Set PRAGMAs
	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA foreign_keys=ON",
		"PRAGMA busy_timeout=5000",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("store: pragma %q: %w", p, err)
		}
	}

	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(0)

	s := &SQLiteStore{db: db}
	if err := s.Migrate(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("store: migrate: %w", err)
	}

	return s, nil
}

// Close closes the database connection.
func (s *SQLiteStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	_, _ = s.db.Exec("PRAGMA optimize")
	return s.db.Close()
}

// migration represents a single schema migration step.
type migration struct {
	version     int
	description string
	ddl         string
}

// migrations is the ordered list of schema migrations.
// To add a new migration: append to this slice with the next version number.
var migrations = []migration{
	{version: 1, description: "Initial LKS schema", ddl: schemaV1DDL},
	{version: 2, description: "Add composite index for scope+date queries", ddl: schemaV2DDL},
}

const schemaV2DDL = `CREATE INDEX IF NOT EXISTS idx_commits_scope_date ON commits(scope, date DESC) WHERE scope != '';`

// Migrate applies all pending schema migrations incrementally. Idempotent.
func (s *SQLiteStore) Migrate() error {
	// Ensure schema_version table exists (bootstrapping)
	if _, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS schema_version (
		version     INTEGER PRIMARY KEY,
		applied_at  INTEGER NOT NULL,
		description TEXT NOT NULL
	)`); err != nil {
		return fmt.Errorf("schema: create version table: %w", err)
	}

	// Get current version
	var currentVersion int
	err := s.db.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM schema_version`).Scan(&currentVersion)
	if err != nil {
		return fmt.Errorf("schema: read version: %w", err)
	}

	// Apply each pending migration in order
	for _, m := range migrations {
		if m.version <= currentVersion {
			continue
		}
		tx, txErr := s.db.Begin()
		if txErr != nil {
			return fmt.Errorf("schema v%d: begin tx: %w", m.version, txErr)
		}
		if _, err := tx.Exec(m.ddl); err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				return fmt.Errorf("schema v%d: %w (rollback failed: %v)", m.version, err, rbErr)
			}
			return fmt.Errorf("schema v%d: %w", m.version, err)
		}
		if _, err := tx.Exec(
			`INSERT INTO schema_version (version, applied_at, description) VALUES (?, ?, ?)`,
			m.version, time.Now().Unix(), m.description,
		); err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				return fmt.Errorf("schema v%d: %w (rollback failed: %v)", m.version, err, rbErr)
			}
			return fmt.Errorf("schema v%d version record: %w", m.version, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("schema v%d: commit: %w", m.version, err)
		}
	}
	return nil
}

const schemaV1DDL = `
CREATE TABLE IF NOT EXISTS schema_version (
    version     INTEGER PRIMARY KEY,
    applied_at  INTEGER NOT NULL,
    description TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS commits (
    hash            TEXT PRIMARY KEY,
    date            INTEGER NOT NULL,
    branch          TEXT NOT NULL DEFAULT '',
    scope           TEXT NOT NULL DEFAULT '',
    conv_type       TEXT NOT NULL DEFAULT '',
    subject         TEXT NOT NULL DEFAULT '',
    message         TEXT NOT NULL,
    files_changed   INTEGER NOT NULL DEFAULT 0,
    lines_added     INTEGER NOT NULL DEFAULT 0,
    lines_deleted   INTEGER NOT NULL DEFAULT 0,
    doc_id          TEXT,
    decision        TEXT NOT NULL CHECK(decision IN (
        'documented','skipped','pending','auto-skipped','merge-skipped','unknown'
    )),
    decision_score  INTEGER,
    decision_confidence REAL,
    skip_reason     TEXT,
    question_mode   TEXT DEFAULT 'full' CHECK(question_mode IN ('full','reduced','confirm','none'))
);
CREATE INDEX IF NOT EXISTS idx_commits_date ON commits(date);
CREATE INDEX IF NOT EXISTS idx_commits_branch ON commits(branch) WHERE branch != '';
CREATE INDEX IF NOT EXISTS idx_commits_scope ON commits(scope) WHERE scope != '';
CREATE INDEX IF NOT EXISTS idx_commits_conv_type ON commits(conv_type);
CREATE INDEX IF NOT EXISTS idx_commits_decision ON commits(decision);
CREATE INDEX IF NOT EXISTS idx_commits_doc ON commits(doc_id) WHERE doc_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS code_signatures (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    commit_hash TEXT NOT NULL REFERENCES commits(hash) ON DELETE CASCADE,
    file_path   TEXT NOT NULL,
    entity_name TEXT NOT NULL,
    entity_type TEXT NOT NULL CHECK(entity_type IN (
        'func','method','type','struct','interface','class','trait','enum','const_block'
    )),
    sig_hash    TEXT NOT NULL,
    lang        TEXT NOT NULL,
    line_start  INTEGER,
    change_type TEXT NOT NULL DEFAULT 'context' CHECK(change_type IN (
        'added','deleted','modified','moved','context'
    ))
);
CREATE INDEX IF NOT EXISTS idx_sig_hash ON code_signatures(sig_hash);
CREATE INDEX IF NOT EXISTS idx_sig_commit ON code_signatures(commit_hash);
CREATE INDEX IF NOT EXISTS idx_sig_entity ON code_signatures(entity_name, lang);
CREATE INDEX IF NOT EXISTS idx_sig_file ON code_signatures(file_path);

CREATE TABLE IF NOT EXISTS doc_index (
    filename          TEXT PRIMARY KEY,
    type              TEXT NOT NULL,
    date              TEXT NOT NULL,
    commit_hash       TEXT,
    branch            TEXT NOT NULL DEFAULT '',
    scope             TEXT NOT NULL DEFAULT '',
    status            TEXT NOT NULL DEFAULT 'draft',
    tags              TEXT NOT NULL DEFAULT '',
    related           TEXT NOT NULL DEFAULT '',
    generated_by      TEXT NOT NULL DEFAULT '',
    angela_mode       TEXT NOT NULL DEFAULT '',
    consolidated_into TEXT NOT NULL DEFAULT '',
    content_hash      TEXT NOT NULL,
    summary_why       TEXT NOT NULL DEFAULT '',
    summary_what      TEXT NOT NULL DEFAULT '',
    title_extracted   TEXT NOT NULL DEFAULT '',
    word_count        INTEGER NOT NULL DEFAULT 0,
    updated_at        INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_doc_type ON doc_index(type);
CREATE INDEX IF NOT EXISTS idx_doc_scope ON doc_index(scope) WHERE scope != '';
CREATE INDEX IF NOT EXISTS idx_doc_branch ON doc_index(branch) WHERE branch != '';
CREATE INDEX IF NOT EXISTS idx_doc_status ON doc_index(status);
CREATE INDEX IF NOT EXISTS idx_doc_date ON doc_index(date);
CREATE INDEX IF NOT EXISTS idx_doc_commit ON doc_index(commit_hash) WHERE commit_hash IS NOT NULL;

CREATE TABLE IF NOT EXISTS ai_usage (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp       INTEGER NOT NULL,
    mode            TEXT NOT NULL CHECK(mode IN ('polish','review','render','ask','consult','merge')),
    provider        TEXT NOT NULL,
    model           TEXT NOT NULL,
    tokens_in       INTEGER NOT NULL,
    tokens_out      INTEGER NOT NULL,
    cached_in       INTEGER NOT NULL DEFAULT 0,
    cost_usd        REAL NOT NULL DEFAULT 0.0,
    latency_ms      INTEGER NOT NULL,
    commit_hash     TEXT,
    doc_id          TEXT,
    prompt_version  INTEGER NOT NULL DEFAULT 1
);
CREATE INDEX IF NOT EXISTS idx_ai_timestamp ON ai_usage(timestamp);
CREATE INDEX IF NOT EXISTS idx_ai_mode ON ai_usage(mode);

CREATE TABLE IF NOT EXISTS review_cache (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    review_date     INTEGER NOT NULL,
    corpus_hash     TEXT NOT NULL,
    corpus_count    INTEGER NOT NULL,
    findings_json   TEXT NOT NULL,
    tokens_in       INTEGER NOT NULL,
    tokens_out      INTEGER NOT NULL,
    provider        TEXT NOT NULL,
    model           TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_review_corpus ON review_cache(corpus_hash);
CREATE INDEX IF NOT EXISTS idx_review_date ON review_cache(review_date);

CREATE TABLE IF NOT EXISTS commit_patterns (
    conv_type           TEXT NOT NULL,
    scope               TEXT NOT NULL DEFAULT '',
    total_count         INTEGER NOT NULL DEFAULT 0,
    documented_count    INTEGER NOT NULL DEFAULT 0,
    skipped_count       INTEGER NOT NULL DEFAULT 0,
    auto_skipped_count  INTEGER NOT NULL DEFAULT 0,
    avg_diff_lines      INTEGER NOT NULL DEFAULT 0,
    avg_score           INTEGER NOT NULL DEFAULT 0,
    last_updated        INTEGER NOT NULL,
    PRIMARY KEY (conv_type, scope)
);

CREATE TABLE IF NOT EXISTS commit_relations (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    source_hash  TEXT NOT NULL REFERENCES commits(hash),
    target_hash  TEXT NOT NULL REFERENCES commits(hash),
    relation     TEXT NOT NULL CHECK(relation IN (
        'reverts','amends','same_branch','same_scope','co_change'
    )),
    detected_at  INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_rel_source ON commit_relations(source_hash);
CREATE INDEX IF NOT EXISTS idx_rel_target ON commit_relations(target_hash);

CREATE TABLE IF NOT EXISTS doc_edges (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    source_doc   TEXT NOT NULL REFERENCES doc_index(filename),
    target_doc   TEXT NOT NULL REFERENCES doc_index(filename),
    edge_type    TEXT NOT NULL CHECK(edge_type IN (
        'related','consolidates','supersedes','contradicts','extends','references'
    )),
    confidence   REAL NOT NULL DEFAULT 1.0,
    detected_at  INTEGER NOT NULL,
    detected_by  TEXT NOT NULL DEFAULT 'manual'
);
CREATE INDEX IF NOT EXISTS idx_edge_source ON doc_edges(source_doc);
CREATE INDEX IF NOT EXISTS idx_edge_target ON doc_edges(target_doc);
CREATE INDEX IF NOT EXISTS idx_edge_type ON doc_edges(edge_type);
`
