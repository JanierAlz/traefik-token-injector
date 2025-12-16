# Traefik Token Injector Plugin

A Traefik middleware plugin that automatically injects authentication tokens into requests by fetching credentials from a GraphQL API.

## Features

- **Multiple Authentication Types**: Supports BASIC, LOGIN, APITOKEN, and NONE authentication
- **Smart Token Caching**: Caches tokens with automatic refresh 10 seconds before expiration
- **GraphQL API Integration**: Fetches instance credentials from a configurable GraphQL endpoint
- **Flexible Endpoint Support**: Works with both REST and GraphQL authentication endpoints
- **Configurable**: Easy configuration via YAML files

## Installation

### Local Mode (Development)

1. Clone this repository to your Traefik plugins directory
2. Configure the plugin in your Traefik configuration

### Traefik Pilot Mode

Add the plugin to your Traefik static configuration:

```yaml
experimental:
  plugins:
    token-injector:
      moduleName: github.com/your-username/traefik-token-injector
      version: v1.0.0
```

## Configuration

### Global Configuration

Create a configuration file at `instance/etc/config.yml`:

```yaml
# GraphQL API endpoint that provides instance data
graphql_api_url: "https://api.example.com/graphql"

# Authentication for the GraphQL API (optional)
graphql_auth_type: "none"  # Options: "none", "basic", "apitoken"

# Basic auth credentials (if graphql_auth_type is "basic")
# graphql_username: "your-username"
# graphql_password: "your-password"

# API token (if graphql_auth_type is "apitoken")
# graphql_api_token: "your-token"
# graphql_token_header: "Authorization"

# HTTP client settings
timeout: "10s"

# Token caching settings
cache_enabled: true
token_refresh_buffer: 10  # Refresh tokens 10 seconds before expiration
```

See `instance/etc/config.example.yml` for a complete example with all options.

### Plugin Configuration

Configure the middleware in your Traefik dynamic configuration:

```yaml
http:
  middlewares:
    my-auth:
      plugin:
        tokenInjectorPlugin:
          serviceId: "693ae3a02956967b201ce9b8"  # Instance ID from GraphQL API
```

### Router Configuration

Apply the middleware to your routers:

```yaml
http:
  routers:
    my-router:
      rule: "Host(`example.com`)"
      service: my-service
      middlewares:
        - my-auth
```

## Authentication Types

### BASIC Authentication

Uses username and password from credential data to create a Basic Auth header.

```json
{
  "authType": "BASIC",
  "credentialData": [
    {"key": "username", "value": "user"},
    {"key": "password", "value": "pass"}
  ]
}
```

### LOGIN Authentication

Calls an authentication endpoint to obtain a token, then caches it with TTL.

```json
{
  "authType": "LOGIN",
  "endpointType": "REST",
  "tokenLocation": "data.token",
  "tokenTtl": 3600,
  "credentialData": [
    {"key": "user.username", "value": "user"},
    {"key": "user.password", "value": "pass"}
  ],
  "endpointData": {
    "edges": [{
      "node": {
        "method": "POST",
        "path": "/auth/login",
        "requestBody": {"contentType": "application/json", "required": true}
      }
    }]
  }
}
```

### APITOKEN Authentication

Uses a pre-configured API key directly.

```json
{
  "authType": "APITOKEN",
  "apiKey": "your-api-key-here"
}
```

### NONE Authentication

No authentication is applied.

```json
{
  "authType": "NONE"
}
```

## Token Caching

The plugin implements intelligent token caching:

- **Automatic Refresh**: Tokens are refreshed 10 seconds before expiration to prevent mid-flight errors
- **TTL Support**: Respects the `tokenTtl` field (in seconds) from the GraphQL API
- **Null TTL**: If `tokenTtl` is null, tokens are cached indefinitely
- **Thread-Safe**: Cache operations are safe for concurrent access

## How It Works

1. **Fetch Instance Data**: The middleware fetches instance configuration from the GraphQL API using the `serviceId`
2. **Check Cache**: If caching is enabled, check for a valid cached token
3. **Authenticate**: Based on the `authType`, the plugin:
   - For BASIC: Creates a Basic Auth header
   - For LOGIN: Calls the authentication endpoint and extracts the token
   - For APITOKEN: Uses the configured API key
   - For NONE: Skips authentication
4. **Inject Headers**: Adds the authentication header (and any custom headers) to the request
5. **Forward Request**: Passes the authenticated request to the target service

## Troubleshooting

### "Failed to fetch instance data"

- Check that the GraphQL API URL is correct in `instance/etc/config.yml`
- Verify the `serviceId` matches an existing instance in the GraphQL API
- Check GraphQL API authentication settings if required

### "Failed to authenticate"

- Verify the credential data is correctly configured in the GraphQL API
- Check the authentication endpoint is accessible
- Verify the `tokenLocation` path matches the response structure

### Token refresh issues

- Check that `tokenTtl` is set correctly (in seconds)
- Verify `token_refresh_buffer` is less than `tokenTtl`
- Check logs for token refresh attempts

## Development

### Building

```bash
go build
```

### Testing

```bash
go test ./...
```

## License

MIT License
