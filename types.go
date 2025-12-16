package traefik_token_injector

import "encoding/json"

// GraphQL Query/Response Types

// GraphQLRequest represents a GraphQL query request
type GraphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

// GraphQLResponse represents a GraphQL query response
type GraphQLResponse struct {
	Data   *InstanceData  `json:"data"`
	Errors []GraphQLError `json:"errors,omitempty"`
}

// GraphQLError represents a GraphQL error
type GraphQLError struct {
	Message string   `json:"message"`
	Path    []string `json:"path,omitempty"`
}

// InstanceData wraps the getInstances query response
type InstanceData struct {
	GetInstances *InstanceConnection `json:"getInstances"`
}

// InstanceConnection represents the paginated connection
type InstanceConnection struct {
	Edges []InstanceEdge `json:"edges"`
}

// InstanceEdge represents an edge in the connection
type InstanceEdge struct {
	Node *InstanceType `json:"node"`
}

// InstanceType represents the instance data from GraphQL
type InstanceType struct {
	ID          string           `json:"_id"`
	Name        string           `json:"name"`
	Type        string           `json:"type"`
	ServiceHost string           `json:"service_host"`
	ServicePath string           `json:"service_path"`
	RemoteHost  string           `json:"remote_host"`
	RemotePath  string           `json:"remote_path"`
	VersionID   string           `json:"version_id"`
	Operations  []string         `json:"operations"`
	Headers     []HeaderType     `json:"headers"`
	Credentials *CredentialsType `json:"credentials"`
	CreatedAt   int              `json:"created_at"`
	UpdatedAt   int              `json:"updated_at"`
}

// HeaderType represents a key-value header pair
type HeaderType struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// CredentialsType represents authentication credentials
type CredentialsType struct {
	AuthType       string                `json:"authType"`       // BASIC, LOGIN, NONE, APITOKEN
	EndpointType   string                `json:"endpointType"`   // REST, GRAPHQL
	CredentialData []CredentialsPairType `json:"credentialData"` // Key-value pairs for credentials
	Token          *string               `json:"token"`          // Pre-existing token (nullable)
	TokenLocation  string                `json:"tokenLocation"`  // Path to token in response (e.g., "data.login.token")
	TokenTtl       *int                  `json:"tokenTtl"`       // Token TTL in seconds (nullable)
	ApiKey         string                `json:"apiKey"`         // API key for APITOKEN auth
	EndpointData   *EndpointConnection   `json:"endpointData"`   // Authentication endpoint data
}

// CredentialsPairType represents a key-value credential pair
type CredentialsPairType struct {
	Key   string `json:"key"`   // Supports nested paths like "user.credentials.username"
	Value string `json:"value"` // The credential value
}

// EndpointConnection represents the endpoint data connection
type EndpointConnection struct {
	Edges []EndpointEdge `json:"edges"`
}

// EndpointEdge represents an edge in the endpoint connection
type EndpointEdge struct {
	Node EndpointNode `json:"node"`
}

// EndpointNode is a union type that can be EndpointType or GqlOperationType
type EndpointNode struct {
	EndpointType     *EndpointType     `json:"-"` // REST endpoint
	GqlOperationType *GqlOperationType `json:"-"` // GraphQL operation
}

// UnmarshalJSON handles the union type for EndpointNode
func (e *EndpointNode) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as EndpointType first
	var endpoint EndpointType
	if err := json.Unmarshal(data, &endpoint); err == nil && endpoint.Method != "" {
		e.EndpointType = &endpoint
		return nil
	}

	// Try to unmarshal as GqlOperationType
	var gqlOp GqlOperationType
	if err := json.Unmarshal(data, &gqlOp); err == nil && gqlOp.OperationType != "" {
		e.GqlOperationType = &gqlOp
		return nil
	}

	return nil
}

// EndpointType represents a REST endpoint
type EndpointType struct {
	ID           string                 `json:"_id"`
	Method       string                 `json:"method"`
	Path         string                 `json:"path"`
	Description  string                 `json:"description"`
	Tags         []string               `json:"tags"`
	Parameters   []ContentAttributeType `json:"parameters"`
	ResponseBody *ContentType           `json:"responseBody"`
	RequestBody  *ContentType           `json:"requestBody"`
}

// GqlOperationType represents a GraphQL operation
type GqlOperationType struct {
	ID            string                 `json:"_id"`
	Name          string                 `json:"name"`
	OperationType string                 `json:"operationType"` // query, mutation, subscription
	Description   string                 `json:"description"`
	Arguments     map[string]interface{} `json:"arguments"`
	Result        string                 `json:"result"`
}

// ContentAttributeType represents a parameter or attribute
type ContentAttributeType struct {
	Type        string `json:"type"`
	Value       string `json:"value"`
	Required    bool   `json:"required"`
	Location    string `json:"location"` // query, header, body, path
	Description string `json:"description"`
	Default     string `json:"default"`
}

// ContentType represents request/response body content
type ContentType struct {
	ContentType   string `json:"contentType"`
	ContentSchema string `json:"contentSchema"`
	Description   string `json:"description"`
	Required      bool   `json:"required"`
}

// CachedToken represents a cached authentication token
type CachedToken struct {
	Token     string
	ExpiresAt *int64 // Unix timestamp, nil if no expiration
	RefreshAt *int64 // Unix timestamp when to refresh (TTL - buffer)
}
