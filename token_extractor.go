package traefik_token_injector

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ExtractTokenFromResponse extracts a token from a JSON response using a dot-notation path
// Example paths: "token", "data.login.token", "response.auth.accessToken"
func ExtractTokenFromResponse(responseBody []byte, tokenLocation string) (string, error) {
	if tokenLocation == "" {
		return "", fmt.Errorf("tokenLocation is empty")
	}

	// Parse the response as JSON
	var data interface{}
	if err := json.Unmarshal(responseBody, &data); err != nil {
		return "", fmt.Errorf("failed to parse response as JSON: %w", err)
	}

	// Split the path by dots
	pathParts := strings.Split(tokenLocation, ".")

	// Navigate through the JSON structure
	current := data
	for i, part := range pathParts {
		// Check if current is a map
		currentMap, ok := current.(map[string]interface{})
		if !ok {
			return "", fmt.Errorf("path segment '%s' (index %d) is not an object", part, i)
		}

		// Get the next value
		next, exists := currentMap[part]
		if !exists {
			return "", fmt.Errorf("path segment '%s' not found in response", part)
		}

		current = next
	}

	// The final value should be a string (the token)
	token, ok := current.(string)
	if !ok {
		return "", fmt.Errorf("token value at path '%s' is not a string", tokenLocation)
	}

	if token == "" {
		return "", fmt.Errorf("token value at path '%s' is empty", tokenLocation)
	}

	return token, nil
}
