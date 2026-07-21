package secrets

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	keyVersion = 1
	nonceSize  = 12
)

type PostgresStore struct {
	db  *pgxpool.Pool
	aes cipher.AEAD
}

func NewPostgresStore(db *pgxpool.Pool, keyMaterial string) (*PostgresStore, error) {
	key, err := parseKey(keyMaterial)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("init secret cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("init secret gcm: %w", err)
	}
	return &PostgresStore{db: db, aes: aead}, nil
}

func (s *PostgresStore) Put(ctx context.Context, ref string, plaintext []byte) error {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return errors.New("secret ref required")
	}
	nonce := make([]byte, nonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return fmt.Errorf("generate secret nonce: %w", err)
	}
	ciphertext := s.aes.Seal(nil, nonce, plaintext, []byte(ref))

	_, err := s.db.Exec(ctx, `
		INSERT INTO secrets (ref, ciphertext, nonce, key_version)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (ref) DO UPDATE SET
		  ciphertext = EXCLUDED.ciphertext,
		  nonce = EXCLUDED.nonce,
		  key_version = EXCLUDED.key_version,
		  updated_at = now()
	`, ref, ciphertext, nonce, keyVersion)
	if err != nil {
		return fmt.Errorf("store secret: %w", err)
	}
	return nil
}

func (s *PostgresStore) Get(ctx context.Context, ref string) ([]byte, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return nil, ErrNotFound
	}

	var ciphertext []byte
	var nonce []byte
	var version int
	err := s.db.QueryRow(ctx, `
		SELECT ciphertext, nonce, key_version
		FROM secrets
		WHERE ref = $1
	`, ref).Scan(&ciphertext, &nonce, &version)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("load secret: %w", err)
	}
	if version != keyVersion {
		return nil, fmt.Errorf("unsupported secret key version %d", version)
	}
	plaintext, err := s.aes.Open(nil, nonce, ciphertext, []byte(ref))
	if err != nil {
		return nil, fmt.Errorf("decrypt secret: %w", err)
	}
	return plaintext, nil
}

func (s *PostgresStore) Delete(ctx context.Context, ref string) error {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return nil
	}
	_, err := s.db.Exec(ctx, `DELETE FROM secrets WHERE ref = $1`, ref)
	if err != nil {
		return fmt.Errorf("delete secret: %w", err)
	}
	return nil
}

func parseKey(in string) ([]byte, error) {
	in = strings.TrimSpace(in)
	if in == "" {
		return nil, errors.New("data store secret key required")
	}
	if strings.HasPrefix(in, "base64:") {
		in = strings.TrimPrefix(in, "base64:")
		key, err := base64.StdEncoding.DecodeString(in)
		if err != nil {
			return nil, fmt.Errorf("decode base64 secret key: %w", err)
		}
		return validateKey(key)
	}
	if strings.HasPrefix(in, "hex:") {
		in = strings.TrimPrefix(in, "hex:")
		key, err := hex.DecodeString(in)
		if err != nil {
			return nil, fmt.Errorf("decode hex secret key: %w", err)
		}
		return validateKey(key)
	}
	if key, err := base64.StdEncoding.DecodeString(in); err == nil {
		if validKeyLen(len(key)) {
			return key, nil
		}
	}
	if key, err := base64.RawStdEncoding.DecodeString(in); err == nil {
		if validKeyLen(len(key)) {
			return key, nil
		}
	}
	if key, err := base64.RawURLEncoding.DecodeString(in); err == nil {
		if validKeyLen(len(key)) {
			return key, nil
		}
	}
	if key, err := hex.DecodeString(in); err == nil {
		if validKeyLen(len(key)) {
			return key, nil
		}
	}
	return validateKey([]byte(in))
}

func validateKey(key []byte) ([]byte, error) {
	if !validKeyLen(len(key)) {
		return nil, fmt.Errorf("data store secret key must decode to 32 bytes, got %d", len(key))
	}
	return key, nil
}

func validKeyLen(n int) bool {
	return n == 32
}
