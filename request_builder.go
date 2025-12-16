package traefik_token_injector

import (
	"encoding/json"
	"fmt"
	"strings"
)

// BuildNestedObject creates a nested JSON object from credential data pairs
// Example: [{key: "user.name", value: "john"}, {key: "user.pass", value: "secret"}]
// Returns: {"user": {"name": "john", "pass": "secret"}}
func BuildNestedObject(credentialData []CredentialsPairType) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	for _, pair := range credentialData {
		if err := setNestedValue(result, pair.Key, pair.Value); err != nil {
			return nil, fmt.Errorf("failed to set nested value for key '%s': %w", pair.Key, err)
		}
	}

	return result, nil
}

// setNestedValue sets a value in a nested map using dot notation
// Example: setNestedValue(map, "user.credentials.username", "john")
// Creates: {"user": {"credentials": {"username": "john"}}}
func setNestedValue(obj map[string]interface{}, path string, value string) error {
	parts := strings.Split(path, ".")
	if len(parts) == 0 {
		return fmt.Errorf("empty path")
	}

	// Navigate to the parent object
	current := obj
	for i := 0; i < len(parts)-1; i++ {
		key := parts[i]

		// Check if the key already exists
		if existing, exists := current[key]; exists {
			// It should be a map
			if existingMap, ok := existing.(map[string]interface{}); ok {
				current = existingMap
			} else {
				return fmt.Errorf("path conflict: '%s' is not an object", key)
			}
		} else {
			// Create a new nested map
			newMap := make(map[string]interface{})
			current[key] = newMap
			current = newMap
		}
	}

	// Set the final value
	finalKey := parts[len(parts)-1]
	current[finalKey] = value

	return nil
}

// BuildRESTRequest builds an HTTP request for a REST authentication endpoint
func BuildRESTRequest(endpoint *EndpointType, credentialData []CredentialsPairType, baseURL string) (method string, url string, body []byte, headers map[string]string, err error) {
	if endpoint == nil {
		return "", "", nil, nil, fmt.Errorf("endpoint is nil")
	}

	method = endpoint.Method
	url = baseURL + endpoint.Path
	headers = make(map[string]string)

	// Build request body if needed
	if endpoint.RequestBody != nil && endpoint.RequestBody.Required {
		// Build nested object from credential data
		bodyObj, err := BuildNestedObject(credentialData)
		if err != nil {
			return "", "", nil, nil, fmt.Errorf("failed to build request body: %w", err)
		}

		// Marshal to JSON
		bodyData, err := json.Marshal(bodyObj)
		if err != nil {
			return "", "", nil, nil, fmt.Errorf("failed to marshal request body: %w", err)
		}

		body = bodyData
		headers["Content-Type"] = endpoint.RequestBody.ContentType
	}

	// Handle parameters (query, header, path)
	queryParams := make(map[string]string)
	for _, param := range endpoint.Parameters {
		// Find the value in credential data
		value := findCredentialValue(credentialData, param.Value)
		if value == "" && param.Required {
			return "", "", nil, nil, fmt.Errorf("required parameter '%s' not found in credentials", param.Value)
		}

		switch param.Location {
		case "header":
			headers[param.Value] = value
		case "query":
			queryParams[param.Value] = value
		case "path":
			// Replace path parameter
			url = strings.ReplaceAll(url, "{"+param.Value+"}", value)
		}
	}

	// Add query parameters to URL
	if len(queryParams) > 0 {
		url += "?"
		first := true
		for key, value := range queryParams {
			if !first {
				url += "&"
			}
			url += key + "=" + value
			first = false
		}
	}

	return method, url, body, headers, nil
}

// BuildGraphQLRequest builds a GraphQL query/mutation for authentication
func BuildGraphQLRequest(operation *GqlOperationType, credentialData []CredentialsPairType) (query string, variables map[string]interface{}, err error) {
	if operation == nil {
		return "", nil, fmt.Errorf("operation is nil")
	}

	// Build the GraphQL query/mutation
	query = operation.OperationType + " " + operation.Name

	// Build variables from credential data
	variables, err = BuildNestedObject(credentialData)
	if err != nil {
		return "", nil, fmt.Errorf("failed to build variables: %w", err)
	}

	return query, variables, nil
}

// findCredentialValue finds a credential value by key
func findCredentialValue(credentialData []CredentialsPairType, key string) string {
	for _, pair := range credentialData {
		if pair.Key == key {
			return pair.Value
		}
	}
	return ""
}
