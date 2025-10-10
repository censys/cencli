package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"

	storedb "github.com/censys/cencli/internal/store/db"
)

const (
	// the name of the database file
	dbName = "cencli.db"
)

//go:generate mockgen -destination=../../gen/store/mocks/store_mock.go -package=mocks github.com/censys/cencli/internal/store Store,AuthsStore,GlobalsStore
type Store interface {
	AuthsStore
	GlobalsStore
}

type dataStore struct {
	db *sql.DB
}

func New(dataDir string) (Store, error) {
	ds := &dataStore{}

	_, err := os.Stat(dataDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("directory %s does not exist", dataDir)
		}
		return nil, fmt.Errorf("failed to check if data directory exists: %w", err)
	}

	dbPath := filepath.Join(dataDir, dbName)
	ds.db, err = sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	// Apply pragmatic defaults for CLI UX
	if _, err := ds.db.Exec(`PRAGMA journal_mode=WAL;`); err != nil {
		return nil, fmt.Errorf("failed to set journal_mode WAL: %w", err)
	}
	if _, err := ds.db.Exec(`PRAGMA busy_timeout = 5000;`); err != nil {
		return nil, fmt.Errorf("failed to set busy_timeout: %w", err)
	}
	if _, err := ds.db.Exec(`PRAGMA synchronous = NORMAL;`); err != nil {
		return nil, fmt.Errorf("failed to set synchronous NORMAL: %w", err)
	}
	if _, err := ds.db.Exec(string(storedb.Schema)); err != nil {
		return nil, fmt.Errorf("failed to execute schema: %w", err)
	}

	authsStore, err := newAuthsStore(ds)
	if err != nil {
		return nil, fmt.Errorf("failed to create auths store: %w", err)
	}

	globalsStore, err := newGlobalsStore(ds)
	if err != nil {
		return nil, fmt.Errorf("failed to create globals store: %w", err)
	}

	return &struct {
		AuthsStore
		GlobalsStore
	}{
		AuthsStore:   authsStore,
		GlobalsStore: globalsStore,
	}, nil
}
