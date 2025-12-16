package traefik_token_injector

import (
	"sync"
	"time"
)

// TokenCache manages cached authentication tokens with TTL support
type TokenCache struct {
	mu     sync.RWMutex
	tokens map[string]*CachedToken
}

// NewTokenCache creates a new token cache
func NewTokenCache() *TokenCache {
	return &TokenCache{
		tokens: make(map[string]*CachedToken),
	}
}

// Get retrieves a token from the cache
// Returns the token and a boolean indicating if refresh is needed
func (c *TokenCache) Get(serviceId string, refreshBuffer int) (token string, needsRefresh bool, exists bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cached, ok := c.tokens[serviceId]
	if !ok {
		return "", false, false
	}

	now := time.Now().Unix()

	// Check if token has expired
	if cached.ExpiresAt != nil && *cached.ExpiresAt <= now {
		return "", false, false
	}

	// Check if token needs refresh (within refresh buffer)
	if cached.RefreshAt != nil && *cached.RefreshAt <= now {
		return cached.Token, true, true
	}

	return cached.Token, false, true
}

// Set stores a token in the cache with optional TTL
func (c *TokenCache) Set(serviceId string, token string, ttl *int, refreshBuffer int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	cached := &CachedToken{
		Token: token,
	}

	// If TTL is provided (not null), calculate expiration and refresh times
	if ttl != nil && *ttl > 0 {
		now := time.Now().Unix()
		expiresAt := now + int64(*ttl)
		cached.ExpiresAt = &expiresAt

		// Calculate refresh time (TTL - buffer seconds)
		refreshAt := now + int64(*ttl) - int64(refreshBuffer)
		// Ensure refresh time is not in the past
		if refreshAt > now {
			cached.RefreshAt = &refreshAt
		} else {
			// If TTL is less than refresh buffer, refresh immediately
			cached.RefreshAt = &now
		}
	}
	// If TTL is null, ExpiresAt and RefreshAt remain nil (no expiration)

	c.tokens[serviceId] = cached
}

// Delete removes a token from the cache
func (c *TokenCache) Delete(serviceId string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.tokens, serviceId)
}

// Clear removes all tokens from the cache
func (c *TokenCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.tokens = make(map[string]*CachedToken)
}
