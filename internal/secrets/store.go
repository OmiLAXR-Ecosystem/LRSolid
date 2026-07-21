package secrets

import (
	"context"
	"errors"
)

var ErrNotFound = errors.New("secret not found")

// Store persists sensitive values outside their owning domain tables.
type Store interface {
	Put(ctx context.Context, ref string, plaintext []byte) error
	Get(ctx context.Context, ref string) ([]byte, error)
	Delete(ctx context.Context, ref string) error
}
