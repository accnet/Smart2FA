package db

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func Init(path string) {
	var err error
	DB, err = sql.Open("sqlite3", path+"?_journal_mode=WAL&_synchronous=NORMAL&_temp_store=MEMORY&_mmap_size=268435456&_cache_size=-20000")
	if err != nil {
		log.Fatalf("db open: %v", err)
	}

	if err := DB.Ping(); err != nil {
		log.Fatalf("db ping: %v", err)
	}

	migrate()
}

func migrate() {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS vaults (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			vault_hash BLOB NOT NULL UNIQUE,
			salt BLOB NOT NULL,
			entries_count INTEGER DEFAULT 0,
			last_access_at INTEGER NOT NULL,
			created_at INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS vault_blobs (
			vault_id INTEGER PRIMARY KEY,
			encrypted_blob BLOB NOT NULL
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_vault_hash ON vaults(vault_hash)`,
	}

	for _, s := range stmts {
		if _, err := DB.Exec(s); err != nil {
			log.Fatalf("migrate: %v", err)
		}
	}
}
