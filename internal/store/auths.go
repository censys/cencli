package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	db "github.com/censys/cencli/gen/db"
	authdom "github.com/censys/cencli/internal/pkg/domain/auth"
)

type AuthsStore interface {
	// AddValueForAuth inserts a new value for an auth defined in the config.
	AddValueForAuth(ctx context.Context, name, description, value string) (*ValueForAuth, error)
	// DeleteValueForAuth deletes an added auth value.
	DeleteValueForAuth(ctx context.Context, id int64) (*ValueForAuth, error)
	// GetValuesForAuth returns all added values for a given auth from the config.
	GetValuesForAuth(ctx context.Context, name string) ([]*ValueForAuth, error)
	// UpdateAuthLastUsedAtToNow updates the last used at timestamp for a given auth value.
	// This is used to "select" the auth value to be used for an API request.
	UpdateAuthLastUsedAtToNow(ctx context.Context, id int64) error
	// GetLastUsedAuthByName returns the last used auth value for a given auth from the config.
	GetLastUsedAuthByName(ctx context.Context, name string) (*ValueForAuth, error)
}

type ValueForAuth struct {
	ID          int64
	Name        string // identifies which auth this value is for
	Description string
	Value       string // the value of the auth
	CreatedAt   time.Time
	LastUsedAt  time.Time
}

// String prints a one-line description of the auth value.
func (v *ValueForAuth) String(showName bool) string {
	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf("id=%d ", v.ID))
	if showName {
		sb.WriteString(fmt.Sprintf("name=%s ", v.Name))
	}
	valLen := 10
	if len(v.Value) < 10 {
		valLen = len(v.Value)
	}
	sb.WriteString(fmt.Sprintf("value=%s... ", v.Value[:valLen]))
	sb.WriteString(fmt.Sprintf("description=%s ", v.Description))
	sb.WriteString(fmt.Sprintf("created_at=%s ", v.CreatedAt.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("last_used_at=%s", v.LastUsedAt.Format(time.RFC3339)))
	return sb.String()
}

// Backwards-compatible sentinel; prefer auth.ErrAuthNotFound
var ErrAuthNotFound = authdom.ErrAuthNotFound

type authsStore struct {
	*dataStore
}

var _ AuthsStore = &authsStore{}

func newAuthsStore(ds *dataStore) (*authsStore, error) {
	return &authsStore{
		dataStore: ds,
	}, nil
}

func (s *authsStore) AddValueForAuth(ctx context.Context, name, description, value string) (*ValueForAuth, error) {
	now := time.Now()
	auth := &ValueForAuth{
		Name:        name,
		Description: description,
		Value:       value,
		CreatedAt:   now,
		LastUsedAt:  now,
	}
	q := db.New(s.db)
	id, err := q.InsertAuth(ctx, s.authToDb(auth))
	if err != nil {
		return nil, fmt.Errorf("failed to insert auth: %w", err)
	}
	auth.ID = id
	return auth, nil
}

func (s *authsStore) DeleteValueForAuth(ctx context.Context, id int64) (*ValueForAuth, error) {
	q := db.New(s.db)
	row, err := q.DeleteAuth(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAuthNotFound
		}
		return nil, fmt.Errorf("failed to delete auth: %w", err)
	}
	auth := s.authFromDb(&row)
	return auth, nil
}

func (s *authsStore) UpdateAuthLastUsedAtToNow(ctx context.Context, id int64) error {
	q := db.New(s.db)
	_, err := q.UpdateAuthLastUsedAt(ctx, db.UpdateAuthLastUsedAtParams{
		ID:         id,
		LastUsedAt: toZulu(time.Now()),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrAuthNotFound
		}
		return fmt.Errorf("failed to update auth last used at: %w", err)
	}
	return nil
}

func (s *authsStore) GetValuesForAuth(ctx context.Context, name string) ([]*ValueForAuth, error) {
	q := db.New(s.db)
	rows, err := q.GetAuthsByName(ctx, name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAuthNotFound
		}
		return nil, fmt.Errorf("failed to get all auths: %w", err)
	}
	auths := make([]*ValueForAuth, len(rows))
	for i, row := range rows {
		auths[i] = s.authFromDb(&row)
	}
	return auths, nil
}

func (s *authsStore) GetLastUsedAuthByName(ctx context.Context, name string) (*ValueForAuth, error) {
	q := db.New(s.db)
	row, err := q.GetLastUsedAuthByName(ctx, name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAuthNotFound
		}
		return nil, fmt.Errorf("failed to get last used auth: %w", err)
	}
	return s.authFromDb(&row), nil
}

func (*authsStore) authFromDb(row *db.Auth) *ValueForAuth {
	return &ValueForAuth{
		ID:          row.ID,
		Name:        row.Name,
		Description: row.Description,
		Value:       row.Value,
		CreatedAt:   fromZulu(row.CreatedAt),
		LastUsedAt:  fromZulu(row.LastUsedAt),
	}
}

func (*authsStore) authToDb(auth *ValueForAuth) db.InsertAuthParams {
	return db.InsertAuthParams{
		Name:        auth.Name,
		Description: auth.Description,
		Value:       auth.Value,
		CreatedAt:   toZulu(auth.CreatedAt),
		LastUsedAt:  toZulu(auth.LastUsedAt),
	}
}
