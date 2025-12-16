package traefik_token_injector

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// AuthHandler handles authentication for different auth types
type AuthHandler struct {
	client *http.Client
	cache  *TokenCache
	config *GlobalConfig
}

// NewAuthHandler creates a new authentication handler
func NewAuthHandler(cache *TokenCache, config *GlobalConfig) *AuthHandler {
	return &AuthHandler{
		client: &http.Client{},
		cache:  cache,
		config: config,
	}
}

// GetAuthToken retrieves or generates an authentication token based on the auth type
func (h *AuthHandler) GetAuthToken(serviceId string, credentials *CredentialsType) (string, error) {
	if credentials == nil {
		return "", fmt.Errorf("credentials are nil")
	}

	switch credentials.AuthType {
	case "BASIC":
		return h.handleBasicAuth(credentials)

	case "LOGIN":
		return h.handleLoginAuth(serviceId, credentials)

	case "APITOKEN":
		return h.handleAPITokenAuth(credentials)

	case "NONE":
		return "", nil

	default:
		return "", fmt.Errorf("unsupported auth type: %s", credentials.AuthType)
	}
}

// handleBasicAuth creates a Basic Authentication header value
func (h *AuthHandler) handleBasicAuth(credentials *CredentialsType) (string, error) {
	// Find username and password in credential data
	var username, password string

	for _, pair := range credentials.CredentialData {
		if pair.Key == "username" || pair.Key == "user" {
			username = pair.Value
		}
		if pair.Key == "password" || pair.Key == "pass" {
			password = pair.Value
		}
	}

	if username == "" || password == "" {
		return "", fmt.Errorf("username or password not found in credential data")
	}

	// Create Basic Auth header value
	auth := username + ":" + password
	encoded := base64.StdEncoding.EncodeToString([]byte(auth))
	return "Basic " + encoded, nil
}

// handleLoginAuth calls the authentication endpoint to obtain a token
func (h *AuthHandler) handleLoginAuth(serviceId string, credentials *CredentialsType) (string, error) {
	// Check cache first
	if h.config.CacheEnabled {
		token, needsRefresh, exists := h.cache.Get(serviceId, h.config.TokenRefreshBuffer)
		if exists && !needsRefresh {
			return token, nil
		}
	}

	// If token exists but doesn't need refresh, use it
	if credentials.Token != nil && *credentials.Token != "" {
		// Cache the pre-existing token
		if h.config.CacheEnabled {
			h.cache.Set(serviceId, *credentials.Token, credentials.TokenTtl, h.config.TokenRefreshBuffer)
		}
		return *credentials.Token, nil
	}

	// Need to fetch a new token from the authentication endpoint
	if credentials.EndpointData == nil || len(credentials.EndpointData.Edges) == 0 {
		return "", fmt.Errorf("no authentication endpoint configured")
	}

	// Get the first endpoint
	endpointNode := credentials.EndpointData.Edges[0].Node

	var token string
	var err error

	// Determine endpoint type and call accordingly
	if credentials.EndpointType == "REST" && endpointNode.EndpointType != nil {
		token, err = h.callRESTAuthEndpoint(endpointNode.EndpointType, credentials)
	} else if credentials.EndpointType == "GRAPHQL" && endpointNode.GqlOperationType != nil {
		token, err = h.callGraphQLAuthEndpoint(endpointNode.GqlOperationType, credentials)
	} else {
		return "", fmt.Errorf("invalid endpoint configuration")
	}

	if err != nil {
		return "", fmt.Errorf("failed to obtain token: %w", err)
	}

	// Cache the token
	if h.config.CacheEnabled {
		h.cache.Set(serviceId, token, credentials.TokenTtl, h.config.TokenRefreshBuffer)
	}

	return token, nil
}

// callRESTAuthEndpoint calls a REST authentication endpoint
func (h *AuthHandler) callRESTAuthEndpoint(endpoint *EndpointType, credentials *CredentialsType) (string, error) {
	// Build the request
	method, url, body, headers, err := BuildRESTRequest(endpoint, credentials.CredentialData, "")
	if err != nil {
		return "", fmt.Errorf("failed to build REST request: %w", err)
	}

	// Create HTTP request
	var req *http.Request
	if body != nil {
		req, err = http.NewRequest(method, url, bytes.NewBuffer(body))
	} else {
		req, err = http.NewRequest(method, url, nil)
	}
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Execute request
	resp, err := h.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("authentication endpoint returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Extract token from response
	token, err := ExtractTokenFromResponse(respBody, credentials.TokenLocation)
	if err != nil {
		return "", fmt.Errorf("failed to extract token: %w", err)
	}

	return token, nil
}

// callGraphQLAuthEndpoint calls a GraphQL authentication endpoint
func (h *AuthHandler) callGraphQLAuthEndpoint(operation *GqlOperationType, credentials *CredentialsType) (string, error) {
	// Build the GraphQL request
	query, variables, err := BuildGraphQLRequest(operation, credentials.CredentialData)
	if err != nil {
		return "", fmt.Errorf("failed to build GraphQL request: %w", err)
	}

	// Create request body
	reqBody := GraphQLRequest{
		Query:     query,
		Variables: variables,
	}

	reqData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Determine the GraphQL endpoint URL
	// For now, we'll use a placeholder - this should be configured
	graphqlURL := "" // TODO: Get from endpoint configuration

	// Create HTTP request
	req, err := http.NewRequest("POST", graphqlURL, bytes.NewBuffer(reqData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := h.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GraphQL endpoint returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Extract token from response
	token, err := ExtractTokenFromResponse(respBody, credentials.TokenLocation)
	if err != nil {
		return "", fmt.Errorf("failed to extract token: %w", err)
	}

	return token, nil
}

// handleAPITokenAuth returns the API key directly
func (h *AuthHandler) handleAPITokenAuth(credentials *CredentialsType) (string, error) {
	if credentials.ApiKey == "" {
		return "", fmt.Errorf("apiKey is empty")
	}
	return credentials.ApiKey, nil
}
