package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	db "github.com/censys/cencli/gen/db"
)

type GlobalsStore interface {
	// AddValueForGlobal inserts a new value for a global defined in the config.
	AddValueForGlobal(ctx context.Context, name, description, value string) (*ValueForGlobal, error)
	// DeleteValueForGlobal deletes an added global value.
	DeleteValueForGlobal(ctx context.Context, id int64) (*ValueForGlobal, error)
	// GetValuesForGlobal returns all added values for a given global from the config.
	GetValuesForGlobal(ctx context.Context, name string) ([]*ValueForGlobal, error)
	// UpdateGlobalLastUsedAtToNow updates the last used at timestamp for a given global value.
	// This is used to "select" the global value to be used for an API request.
	UpdateGlobalLastUsedAtToNow(ctx context.Context, id int64) error
	// GetLastUsedGlobalByName returns the last used global value for a given global from the config.
	GetLastUsedGlobalByName(ctx context.Context, name string) (*ValueForGlobal, error)
}

type ValueForGlobal struct {
	ID          int64
	Name        string // identifies which global this value is for
	Description string
	Value       string // the value of the global
	CreatedAt   time.Time
	LastUsedAt  time.Time
}

// String prints a one-line description of the global value.
func (v *ValueForGlobal) String(showName bool) string {
	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf("id=%d ", v.ID))
	if showName {
		sb.WriteString(fmt.Sprintf("name=%s ", v.Name))
	}
	sb.WriteString(fmt.Sprintf("value=%s ", v.Value))
	sb.WriteString(fmt.Sprintf("description=%s ", v.Description))
	sb.WriteString(fmt.Sprintf("created_at=%s ", v.CreatedAt.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("last_used_at=%s", v.LastUsedAt.Format(time.RFC3339)))
	return sb.String()
}

// ErrGlobalNotFound is returned when no global value is found.
var ErrGlobalNotFound = errors.New("global not found")

type globalsStore struct {
	*dataStore
}

var _ GlobalsStore = &globalsStore{}

func newGlobalsStore(ds *dataStore) (*globalsStore, error) {
	return &globalsStore{
		dataStore: ds,
	}, nil
}

func (r *globalsStore) AddValueForGlobal(ctx context.Context, name, description, value string) (*ValueForGlobal, error) {
	now := time.Now()
	global := &ValueForGlobal{
		Name:        name,
		Description: description,
		Value:       value,
		CreatedAt:   now,
		LastUsedAt:  now,
	}
	q := db.New(r.db)
	id, err := q.InsertGlobal(ctx, r.globalToDb(global))
	if err != nil {
		return nil, fmt.Errorf("failed to insert global: %w", err)
	}
	global.ID = id
	return global, nil
}

func (r *globalsStore) DeleteValueForGlobal(ctx context.Context, id int64) (*ValueForGlobal, error) {
	q := db.New(r.db)
	row, err := q.DeleteGlobal(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrGlobalNotFound
		}
		return nil, fmt.Errorf("failed to delete global: %w", err)
	}
	global := r.globalFromDb(&row)
	return global, nil
}

func (r *globalsStore) UpdateGlobalLastUsedAtToNow(ctx context.Context, id int64) error {
	q := db.New(r.db)
	_, err := q.UpdateGlobalLastUsedAt(ctx, db.UpdateGlobalLastUsedAtParams{
		ID:         id,
		LastUsedAt: toZulu(time.Now()),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrGlobalNotFound
		}
		return fmt.Errorf("failed to update global last used at: %w", err)
	}
	return nil
}

func (r *globalsStore) GetValuesForGlobal(ctx context.Context, name string) ([]*ValueForGlobal, error) {
	q := db.New(r.db)
	rows, err := q.GetGlobalsByName(ctx, name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrGlobalNotFound
		}
		return nil, fmt.Errorf("failed to get all globals: %w", err)
	}
	globals := make([]*ValueForGlobal, len(rows))
	for i, row := range rows {
		globals[i] = r.globalFromDb(&row)
	}
	return globals, nil
}

func (r *globalsStore) GetLastUsedGlobalByName(ctx context.Context, name string) (*ValueForGlobal, error) {
	q := db.New(r.db)
	row, err := q.GetLastUsedGlobalByName(ctx, name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrGlobalNotFound
		}
		return nil, fmt.Errorf("failed to get last used global: %w", err)
	}
	return r.globalFromDb(&row), nil
}

func (*globalsStore) globalFromDb(row *db.Global) *ValueForGlobal {
	return &ValueForGlobal{
		ID:          row.ID,
		Name:        row.Name,
		Description: row.Description,
		Value:       row.Value,
		CreatedAt:   fromZulu(row.CreatedAt),
		LastUsedAt:  fromZulu(row.LastUsedAt),
	}
}

func (*globalsStore) globalToDb(global *ValueForGlobal) db.InsertGlobalParams {
	return db.InsertGlobalParams{
		Name:        global.Name,
		Description: global.Description,
		Value:       global.Value,
		CreatedAt:   toZulu(global.CreatedAt),
		LastUsedAt:  toZulu(global.LastUsedAt),
	}
}
