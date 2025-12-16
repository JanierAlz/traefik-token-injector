package traefik_token_injector

import (
	"context"
	"fmt"
	"log"
	"net/http"
)

// TokenInjector is the main middleware struct
type TokenInjector struct {
	next         http.Handler
	name         string
	config       *Config
	globalConfig *GlobalConfig
	gqlClient    *GraphQLClient
	authHandler  *AuthHandler
	cache        *TokenCache
}

// New creates a new TokenInjector middleware instance
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	// Validate plugin config
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid plugin configuration: %w", err)
	}

	// Load global configuration
	globalConfig, err := LoadGlobalConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load global configuration: %w", err)
	}

	// Validate global config
	if err := globalConfig.Validate(); err != nil {
		return nil, fmt.Errorf("invalid global configuration: %w", err)
	}

	// Create GraphQL client
	gqlClient, err := NewGraphQLClient(globalConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create GraphQL client: %w", err)
	}

	// Create token cache
	cache := NewTokenCache()

	// Create auth handler
	authHandler := NewAuthHandler(cache, globalConfig)

	log.Printf("[TokenInjector] Initialized for service ID: %s", config.ServiceId)

	return &TokenInjector{
		next:         next,
		name:         name,
		config:       config,
		globalConfig: globalConfig,
		gqlClient:    gqlClient,
		authHandler:  authHandler,
		cache:        cache,
	}, nil
}

// ServeHTTP implements the http.Handler interface
func (t *TokenInjector) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	// Fetch instance data from GraphQL API
	instance, err := t.gqlClient.FetchInstanceById(t.config.ServiceId)
	if err != nil {
		log.Printf("[TokenInjector] Failed to fetch instance data: %v", err)
		http.Error(rw, "Failed to fetch instance data", http.StatusInternalServerError)
		return
	}

	// Check if credentials are configured
	if instance.Credentials == nil {
		log.Printf("[TokenInjector] No credentials configured for service ID: %s", t.config.ServiceId)
		// No authentication required, pass through
		t.next.ServeHTTP(rw, req)
		return
	}

	// Get authentication token based on auth type
	token, err := t.authHandler.GetAuthToken(t.config.ServiceId, instance.Credentials)
	if err != nil {
		log.Printf("[TokenInjector] Failed to get auth token: %v", err)
		http.Error(rw, "Failed to authenticate", http.StatusUnauthorized)
		return
	}

	// Inject authentication header if token is not empty
	if token != "" {
		// Determine the header name based on auth type
		headerName := "Authorization"

		// For BASIC auth, the token already includes "Basic " prefix
		// For other types, we might need to add "Bearer " prefix
		if instance.Credentials.AuthType != "BASIC" && instance.Credentials.AuthType != "APITOKEN" {
			// For LOGIN type, add Bearer prefix if not already present
			if len(token) > 7 && token[:7] != "Bearer " {
				token = "Bearer " + token
			}
		}

		req.Header.Set(headerName, token)
		log.Printf("[TokenInjector] Injected %s auth token for service ID: %s", instance.Credentials.AuthType, t.config.ServiceId)
	}

	// Add any custom headers from instance configuration
	if len(instance.Headers) > 0 {
		for _, header := range instance.Headers {
			req.Header.Set(header.Key, header.Value)
		}
		log.Printf("[TokenInjector] Added %d custom headers", len(instance.Headers))
	}

	// Forward the request to the next handler
	t.next.ServeHTTP(rw, req)
}
