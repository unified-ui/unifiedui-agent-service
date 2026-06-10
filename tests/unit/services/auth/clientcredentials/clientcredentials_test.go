package clientcredentials_test

import (
	"context"
	"crypto/rand"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/unifiedui/agent-service/internal/services/auth/clientcredentials"
)

func newTestServer(t *testing.T, status int, body string, hits *int32) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hits != nil {
			atomic.AddInt32(hits, 1)
		}
		_ = r.ParseForm()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}))
}

func TestTokenClient_Acquire_Success(t *testing.T) {
	srv := newTestServer(t, http.StatusOK, `{"access_token":"AAA","expires_in":3600}`, nil)
	defer srv.Close()

	c := clientcredentials.NewTokenClient(srv.Client())
	c.SetAuthority(srv.URL)

	tok, err := c.Acquire(context.Background(), clientcredentials.Credentials{
		TenantID: "t", ClientID: "c", ClientSecret: "s",
	}, "scope/.default")
	require.NoError(t, err)
	assert.Equal(t, "AAA", tok.AccessToken)
	assert.False(t, tok.IsExpired())
}

func TestTokenClient_Acquire_InvalidCredentials(t *testing.T) {
	srv := newTestServer(t, http.StatusUnauthorized, `{"error":"invalid_client","error_description":"bad secret"}`, nil)
	defer srv.Close()

	c := clientcredentials.NewTokenClient(srv.Client())
	c.SetAuthority(srv.URL)

	_, err := c.Acquire(context.Background(), clientcredentials.Credentials{
		TenantID: "t", ClientID: "c", ClientSecret: "wrong",
	}, "scope/.default")
	require.Error(t, err)
	assert.True(t, errors.Is(err, clientcredentials.ErrInvalidCredentials))
}

func TestTokenClient_Acquire_MissingFields(t *testing.T) {
	c := clientcredentials.NewTokenClient(nil)
	_, err := c.Acquire(context.Background(), clientcredentials.Credentials{}, "scope")
	assert.ErrorIs(t, err, clientcredentials.ErrInvalidCredentials)
}

func TestTokenClient_Acquire_MissingScope(t *testing.T) {
	c := clientcredentials.NewTokenClient(nil)
	_, err := c.Acquire(context.Background(), clientcredentials.Credentials{
		TenantID: "t", ClientID: "c", ClientSecret: "s",
	}, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "scope is required")
}

// inMemoryCache implements CacheBackend for tests.
type inMemoryCache struct {
	store map[string][]byte
}

func newInMemoryCache() *inMemoryCache { return &inMemoryCache{store: map[string][]byte{}} }

func (m *inMemoryCache) Get(_ context.Context, key string) ([]byte, error) {
	v, ok := m.store[key]
	if !ok {
		return nil, nil
	}
	return v, nil
}
func (m *inMemoryCache) Set(_ context.Context, key string, value []byte, _ time.Duration) error {
	m.store[key] = value
	return nil
}
func (m *inMemoryCache) Delete(_ context.Context, key string) (bool, error) {
	if _, ok := m.store[key]; !ok {
		return false, nil
	}
	delete(m.store, key)
	return true, nil
}

func makeKey(t *testing.T) []byte {
	t.Helper()
	k := make([]byte, 32)
	_, err := io.ReadFull(rand.Reader, k)
	require.NoError(t, err)
	return k
}

func TestCachedTokenClient_HitMissAndEncryption(t *testing.T) {
	hits := int32(0)
	srv := newTestServer(t, http.StatusOK, `{"access_token":"BBB","expires_in":3600}`, &hits)
	defer srv.Close()

	inner := clientcredentials.NewTokenClient(srv.Client())
	inner.SetAuthority(srv.URL)
	cache := newInMemoryCache()

	cc, err := clientcredentials.NewCachedTokenClient(inner, cache, makeKey(t), "test")
	require.NoError(t, err)

	creds := clientcredentials.Credentials{TenantID: "tnt", ClientID: "cli", ClientSecret: "sec"}

	tok1, err := cc.Acquire(context.Background(), creds, "scope/.default")
	require.NoError(t, err)
	assert.Equal(t, "BBB", tok1.AccessToken)
	assert.Equal(t, int32(1), atomic.LoadInt32(&hits))

	stored := cache.store[cc.CacheKey(creds, "scope/.default")]
	require.NotEmpty(t, stored)
	assert.NotContains(t, string(stored), "BBB", "cached token must be encrypted")

	tok2, err := cc.Acquire(context.Background(), creds, "scope/.default")
	require.NoError(t, err)
	assert.Equal(t, "BBB", tok2.AccessToken)
	assert.Equal(t, int32(1), atomic.LoadInt32(&hits), "second Acquire must hit the cache")
}

func TestCachedTokenClient_Invalidate(t *testing.T) {
	hits := int32(0)
	srv := newTestServer(t, http.StatusOK, `{"access_token":"CCC","expires_in":3600}`, &hits)
	defer srv.Close()

	inner := clientcredentials.NewTokenClient(srv.Client())
	inner.SetAuthority(srv.URL)
	cache := newInMemoryCache()
	cc, err := clientcredentials.NewCachedTokenClient(inner, cache, makeKey(t), "test")
	require.NoError(t, err)

	creds := clientcredentials.Credentials{TenantID: "t", ClientID: "c", ClientSecret: "s"}

	_, err = cc.Acquire(context.Background(), creds, "s/.default")
	require.NoError(t, err)
	cc.Invalidate(context.Background(), creds, "s/.default")
	_, err = cc.Acquire(context.Background(), creds, "s/.default")
	require.NoError(t, err)
	assert.Equal(t, int32(2), atomic.LoadInt32(&hits))
}

func TestCachedTokenClient_RejectsBadKeyLength(t *testing.T) {
	inner := clientcredentials.NewTokenClient(nil)
	cache := newInMemoryCache()
	_, err := clientcredentials.NewCachedTokenClient(inner, cache, []byte("short"), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "32 bytes")
}

func TestCachedTokenClient_NoCacheStillWorks(t *testing.T) {
	hits := int32(0)
	srv := newTestServer(t, http.StatusOK, `{"access_token":"DDD","expires_in":3600}`, &hits)
	defer srv.Close()

	inner := clientcredentials.NewTokenClient(srv.Client())
	inner.SetAuthority(srv.URL)
	cc, err := clientcredentials.NewCachedTokenClient(inner, nil, nil, "")
	require.NoError(t, err)

	tok, err := cc.Acquire(context.Background(), clientcredentials.Credentials{TenantID: "t", ClientID: "c", ClientSecret: "s"}, "scope/.default")
	require.NoError(t, err)
	assert.Equal(t, "DDD", tok.AccessToken)
}

func TestCachedTokenClient_CacheKeyDeterministic(t *testing.T) {
	cc, err := clientcredentials.NewCachedTokenClient(clientcredentials.NewTokenClient(nil), nil, nil, "p")
	require.NoError(t, err)
	creds := clientcredentials.Credentials{TenantID: "t", ClientID: "c", ClientSecret: "ignored"}
	k1 := cc.CacheKey(creds, "scope/.default")
	k2 := cc.CacheKey(creds, "scope/.default")
	assert.Equal(t, k1, k2)
	assert.True(t, strings.HasPrefix(k1, "p:"))
}

func TestToken_IsExpired(t *testing.T) {
	t1 := &clientcredentials.Token{AccessToken: "x", ExpiresAt: time.Now().Add(1 * time.Hour)}
	assert.False(t, t1.IsExpired())
	t2 := &clientcredentials.Token{AccessToken: "x", ExpiresAt: time.Now().Add(10 * time.Second)}
	assert.True(t, t2.IsExpired(), "30s safety margin")
	t3 := &clientcredentials.Token{AccessToken: "x", ExpiresAt: time.Now().Add(-1 * time.Hour)}
	assert.True(t, t3.IsExpired())
}
