# Message Transformer

A Go application that transforms HTTP JSON messages to MQTT messages using configurable rules and templates.

## Core Features

### HTTP to MQTT Transformation
- Accepts JSON messages via HTTP POST endpoints
- Transforms messages using Go templates
- Publishes transformed messages to MQTT topics
- Health check endpoint for monitoring

### Template Functions
- `{{now}}` - Current UTC timestamp in RFC3339 format
- `{{num .field}}` - Type-safe number handling
- `{{bool .field}}` - Type-safe boolean handling
- `{{toJSON .field}}` - Object to JSON string conversion
- `{{fromJSON .field}}` - JSON string to object parsing

### MQTT Features
- TLS support with client certificates
- Username/password authentication
- Configurable QoS levels (0, 1, 2)
- Message retention control
- Automatic reconnection handling

### Configuration
- External application configuration (app.json)
- Rule-based message transformation (JSON files)
- Structured logging with levels
- MQTT connection settings

## Project Structure

```
message-transformer/
├── cmd/
│   └── server/
│       └── main.go                 # Application entry point
├── config/
│   ├── app.json                    # Main application configuration
│   └── rules/                      # Rule configuration directory
│       ├── device-status.json      # Example rule configuration
│       └── sensor-data.json        # Example rule configuration
├── internal/
│   ├── api/
│   │   ├── handler.go             # HTTP request handlers
│   │   ├── middleware.go          # Logging middleware
│   │   └── router.go              # Chi router setup
│   ├── config/
│   │   ├── config.go              # Configuration handling
│   │   └── rule.go                # Rule loading and validation
│   ├── mqtt/
│   │   └── client.go              # MQTT client implementation
│   └── transformer/
│       └── transformer.go          # Message transformation logic
└── pkg/
    └── logger/
        └── logger.go              # Structured logging setup
```

## Configuration

### Application Configuration (app.json)
```json
{
  "mqtt": {
    "broker": "ssl://mqtt.example.com:8883",
    "clientId": "message-transformer-1",
    "username": "service-user",
    "password": "service-password",
    "tls": {
      "enabled": true,
      "caCert": "/etc/certs/ca.crt",
      "cert": "/etc/certs/client.crt",
      "key": "/etc/certs/client.key"
    },
    "reconnect": {
      "initial": 3,
      "maxDelay": 60,
      "maxRetries": 10
    }
  },
  "api": {
    "host": "0.0.0.0",
    "port": 8080
  },
  "rules": {
    "directory": "/etc/message-transformer/rules"
  },
  "logger": {
    "level": "info",
    "outputPath": "stdout",
    "encoding": "json"
  }
}
```

### Rule Configuration Example
```json
{
  "id": "device-status",
  "description": "Transforms device status updates",
  "api": {
    "method": "POST",
    "path": "/api/v1/device-status"
  },
  "transform": {
    "template": "{\"deviceId\": \"{{.id}}\", \"status\": {\"state\": \"{{.current_state}}\", \"lastUpdated\": \"{{now}}\", \"batteryLevel\": {{num .battery}}, \"isOnline\": {{bool .online}}}}"
  },
  "target": {
    "topic": "devices/status",
    "qos": 1,
    "retain": true
  }
}
```

## Building and Running

### Prerequisites
- Go 1.21 or higher
- MQTT broker (with TLS support if needed)
- Write access to log directory (or use stdout)

### Build
```bash
go build -o message-transformer cmd/server/main.go
```

### Run
```bash
./message-transformer -config /path/to/config/app.json
```

## API Usage

### Health Check
Request:
```bash
curl http://localhost:8080/health
```

Response:
```json
{
  "status": "ok",
  "mqtt_connected": true
}
```

### Transform Message
Request:
```bash
curl -X POST http://localhost:8080/api/v1/device-status \
  -H "Content-Type: application/json" \
  -d '{
    "id": "device_123",
    "current_state": "running",
    "battery": 85.5,
    "online": true
  }'
```

Response:
```json
{
  "status": "published",
  "rule_id": "device-status",
  "transformed": {
    "deviceId": "device_123",
    "status": {
      "state": "running",
      "lastUpdated": "2025-01-31T15:30:00Z",
      "batteryLevel": 85.5,
      "isOnline": true
    }
  }
}
```

## Error Handling

### Invalid JSON Request
```json
{
  "error": "Invalid JSON in request body"
}
```

### Transform Error
```json
{
  "error": "Transform error: failed to execute template"
}
```

### MQTT Publishing Error
```json
{
  "error": "Failed to publish message"
}
```

## Current Limitations

1. Template Restrictions:
   - No array iteration support
   - No conditional logic (if/else)
   - No dynamic MQTT topics

2. API Restrictions:
   - Only supports static paths (no URL parameters)
   - POST method for transformations
   - GET method for health check

3. Validation:
   - Basic JSON syntax validation
   - Template syntax validation
   - No schema validation for messages

## Monitoring

### Health Check Endpoint
- Provides service status
- Reports MQTT connection state
- Available at /health

### Logging
- Structured JSON logging
- Configurable log levels
- Request/response logging
- Error tracking with stack traces

## Security Features

### TLS Support
- MQTT TLS connection
- Client certificate authentication
- Custom CA certificate support
- Strong cipher suites

### Authentication
- MQTT username/password
- Configurable credentials
- Secure credential handling
