package clientcredentials

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/unifiedui/agent-service/internal/core/cache"
)

// CacheBackend is the subset of cache.Cache required by CachedTokenClient.
type CacheBackend interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) (bool, error)
}

// CachedTokenClient wraps a TokenClient with a Redis-backed AES-256-GCM cache.
type CachedTokenClient struct {
	inner       *TokenClient
	cacheClient CacheBackend
	encKey      []byte
	keyPrefix   string
}

// NewCachedTokenClient creates a cached token client.
//
// encryptionKey must be exactly 32 bytes (AES-256). cacheClient may be nil to
// disable caching (every Acquire then hits AAD directly).
func NewCachedTokenClient(inner *TokenClient, cacheClient CacheBackend, encryptionKey []byte, keyPrefix string) (*CachedTokenClient, error) {
	if inner == nil {
		return nil, errors.New("clientcredentials: inner token client is required")
	}
	if cacheClient != nil && len(encryptionKey) != 32 {
		return nil, errors.New("clientcredentials: encryption key must be 32 bytes (AES-256)")
	}
	if keyPrefix == "" {
		keyPrefix = "foundry:token"
	}
	return &CachedTokenClient{
		inner:       inner,
		cacheClient: cacheClient,
		encKey:      encryptionKey,
		keyPrefix:   keyPrefix,
	}, nil
}

// CacheKey computes the deterministic cache key for given credentials and scope.
func (c *CachedTokenClient) CacheKey(creds Credentials, scope string) string {
	h := sha256.Sum256([]byte(creds.TenantID + "|" + creds.ClientID + "|" + scope))
	return fmt.Sprintf("%s:%s", c.keyPrefix, hex.EncodeToString(h[:]))
}

// Acquire returns a token for the given credentials/scope, using the cache when possible.
func (c *CachedTokenClient) Acquire(ctx context.Context, creds Credentials, scope string) (*Token, error) {
	if c.cacheClient != nil {
		if tok, ok := c.tryGet(ctx, creds, scope); ok {
			return tok, nil
		}
	}

	tok, err := c.inner.Acquire(ctx, creds, scope)
	if err != nil {
		return nil, err
	}

	if c.cacheClient != nil {
		_ = c.tryStore(ctx, creds, scope, tok)
	}

	return tok, nil
}

// Invalidate deletes a cached token (used after a 401 from the resource server).
func (c *CachedTokenClient) Invalidate(ctx context.Context, creds Credentials, scope string) {
	if c.cacheClient == nil {
		return
	}
	_, _ = c.cacheClient.Delete(ctx, c.CacheKey(creds, scope))
}

func (c *CachedTokenClient) tryGet(ctx context.Context, creds Credentials, scope string) (*Token, bool) {
	raw, err := c.cacheClient.Get(ctx, c.CacheKey(creds, scope))
	if err != nil || len(raw) == 0 {
		return nil, false
	}
	plaintext, err := c.decrypt(raw)
	if err != nil {
		return nil, false
	}
	var tok Token
	if err := json.Unmarshal(plaintext, &tok); err != nil {
		return nil, false
	}
	if tok.IsExpired() {
		return nil, false
	}
	return &tok, true
}

func (c *CachedTokenClient) tryStore(ctx context.Context, creds Credentials, scope string, tok *Token) error {
	plaintext, err := json.Marshal(tok)
	if err != nil {
		return err
	}
	ciphertext, err := c.encrypt(plaintext)
	if err != nil {
		return err
	}
	ttl := time.Until(tok.ExpiresAt) - 30*time.Second
	if ttl <= 0 {
		return nil
	}
	return c.cacheClient.Set(ctx, c.CacheKey(creds, scope), ciphertext, ttl)
}

func (c *CachedTokenClient) encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(c.encKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func (c *CachedTokenClient) decrypt(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(c.encKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(ciphertext) < gcm.NonceSize() {
		return nil, errors.New("clientcredentials: ciphertext too short")
	}
	nonce := ciphertext[:gcm.NonceSize()]
	data := ciphertext[gcm.NonceSize():]
	return gcm.Open(nil, nonce, data, nil)
}

var _ CacheBackend = (cache.Cache)(nil)
