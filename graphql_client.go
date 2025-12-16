package traefik_token_injector

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// GraphQLClient handles communication with the GraphQL API
type GraphQLClient struct {
	config     *GlobalConfig
	httpClient *http.Client
}

// NewGraphQLClient creates a new GraphQL client
func NewGraphQLClient(config *GlobalConfig) (*GraphQLClient, error) {
	timeout, err := config.GetTimeout()
	if err != nil {
		return nil, fmt.Errorf("invalid timeout: %w", err)
	}

	return &GraphQLClient{
		config: config,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

// FetchInstanceById fetches instance data by ID from the GraphQL API
func (c *GraphQLClient) FetchInstanceById(instanceId string) (*InstanceType, error) {
	// Build the GraphQL query
	query := `
		query instance($id: String!) {
			getInstances(
				queryInput: {
					search: [
						{
							field: "_id"
							value: {
								value: $id
								kind: ID
								operator: EQ
							}
						}
					]
				}
			) {
				edges {
					node {
						_id
						name
						type
						service_host
						service_path
						remote_host
						remote_path
						version_id
						operations
						headers {
							key
							value
						}
						credentials {
							apiKey
							token
							tokenLocation
							tokenTtl
							credentialData {
								key
								value
							}
							endpointType
							authType
							endpointData {
								edges {
									node {
										... on EndpointType {
											_id
											method
											path
											description
											tags
											parameters {
												type
												value
												required
												location
												description
												default
											}
											responseBody {
												contentType
												contentSchema
												description
											}
											requestBody {
												contentType
												contentSchema
												description
												required
											}
										}
										... on GqlOperationType {
											_id
											name
											operationType
											description
											arguments
											result
										}
									}
								}
							}
						}
					}
				}
			}
		}
	`

	// Create request body
	reqBody := GraphQLRequest{
		Query: query,
		Variables: map[string]interface{}{
			"id": instanceId,
		},
	}

	reqData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", c.config.GraphQLAPIURL, bytes.NewBuffer(reqData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Add authentication if configured
	if err := c.addAuthentication(req); err != nil {
		return nil, fmt.Errorf("failed to add authentication: %w", err)
	}

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GraphQL API returned status %d: %s", resp.StatusCode, string(respData))
	}

	// Parse response
	var gqlResp GraphQLResponse
	if err := json.Unmarshal(respData, &gqlResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for GraphQL errors
	if len(gqlResp.Errors) > 0 {
		return nil, fmt.Errorf("GraphQL error: %s", gqlResp.Errors[0].Message)
	}

	// Extract instance data
	if gqlResp.Data == nil || gqlResp.Data.GetInstances == nil || len(gqlResp.Data.GetInstances.Edges) == 0 {
		return nil, fmt.Errorf("no instance found with ID: %s", instanceId)
	}

	instance := gqlResp.Data.GetInstances.Edges[0].Node
	if instance == nil {
		return nil, fmt.Errorf("instance node is nil")
	}

	return instance, nil
}

// addAuthentication adds authentication headers to the request based on config
func (c *GraphQLClient) addAuthentication(req *http.Request) error {
	switch c.config.GraphQLAuthType {
	case "basic":
		// Basic authentication
		auth := c.config.GraphQLUsername + ":" + c.config.GraphQLPassword
		encoded := base64.StdEncoding.EncodeToString([]byte(auth))
		req.Header.Set("Authorization", "Basic "+encoded)

	case "apitoken":
		// API token authentication
		req.Header.Set(c.config.GraphQLTokenHeader, c.config.GraphQLAPIToken)

	case "none":
		// No authentication
		return nil

	default:
		return fmt.Errorf("unsupported auth type: %s", c.config.GraphQLAuthType)
	}

	return nil
}
