# Message Transformer

A configurable service that accepts JSON messages via REST API endpoints and transforms them to MQTT messages based on customizable rules.

## Project Structure

```
message-transformer/
├── cmd/
│   └── server/
│       └── main.go                 # Application entry point and component orchestration
├── config/
│   ├── app.json                    # Main application configuration
│   └── rules/                      # Rule configuration directory
│       ├── sensor_data.json
│       └── device_status.json
├── internal/
│   ├── api/
│   │   ├── handler.go             # HTTP request handlers and error responses
│   │   ├── middleware.go          # Structured logging middleware
│   │   └── router.go              # Chi router setup and server configuration
│   ├── config/
│   │   ├── config.go              # Application configuration structures
│   │   └── rule.go                # Rule configuration and loading
│   ├── mqtt/
│   │   └── client.go              # MQTT client with TLS and reconnection support
│   ├── transformer/
│   │   └── transformer.go         # Message transformation with type-safe template functions
│   └── validator/
│       └── validator.go           # Basic configuration and MQTT settings validation
├── pkg/
│   └── logger/
│       └── logger.go              # Structured logging with Zap
├── go.mod
└── README.md
```

## Features

### Currently Implemented

- **HTTP API**
  - Dynamic endpoints based on rule configuration
  - Health check endpoint with MQTT connection status
  - JSON request/response handling
  - Structured error responses

- **Message Transformation**
  - JSON to JSON transformation using Go templates
  - Type-safe template functions
  - Template output validation

- **Template Functions**
  - `{{now}}` - Current UTC timestamp in RFC3339 format
  - `{{num .value}}` - Safe number handling with type conversion
  - `{{bool .value}}` - Safe boolean handling with type conversion
  - `{{toJSON .object}}` - Convert object to JSON string
  - `{{fromJSON .string}}` - Parse JSON string to object

- **MQTT Integration**
  - Secure connection with TLS support
  - Client certificate authentication
  - Username/password authentication
  - Configurable QoS levels (0, 1, 2)
  - Message retention control
  - Automatic reconnection handling

- **Configuration**
  - External application configuration
  - Directory-based rule loading
  - MQTT connection settings
  - Logging configuration

- **Validation**
  - JSON syntax validation
  - Basic rule configuration validation
  - MQTT settings validation (QoS, topic format)

- **Logging**
  - Structured logging with Zap
  - Configurable output paths and formats
  - Request/response logging
  - Error tracking

### Current Limitations

- No dynamic URL parameters in API paths (e.g., cannot use /api/v1/devices/{id})
- No array handling in templates (e.g., cannot iterate over lists)
- No conditional logic in templates (no if/else statements)
- No validation of input data against required template fields
- No schema validation for incoming messages
- Static MQTT topics only (no dynamic topic generation)

## Configuration

### Application Configuration (app.json)
```json
{
    "mqtt": {
        "broker": "ssl://mqtt.example.com:8883",
        "clientId": "message-transformer-prod-1",
        "username": "transformer-service",
        "password": "your-secure-password",
        "tls": {
            "enabled": true,
            "caCert": "/etc/message-transformer/certs/ca.crt",
            "cert": "/etc/message-transformer/certs/client.crt",
            "key": "/etc/message-transformer/certs/client.key"
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
        "outputPath": "/var/log/message-transformer/service.log",
        "encoding": "json"
    }
}
```

### Rule Configuration Example
```json
{
    "sensor_data_transform": {
        "id": "sensor-data-transform",
        "description": "Transforms sensor data to new format",
        "api": {
            "method": "POST",
            "path": "/api/v1/sensor-data"
        },
        "transform": {
            "template": {
                "deviceId": "{{.device_id}}",
                "reading": {
                    "type": "{{.sensor_type}}",
                    "value": {{num .reading}},
                    "unit": "{{.unit}}",
                    "timestamp": "{{now}}"
                },
                "metadata": {
                    "batteryLevel": {{num .battery_level}},
                    "enabled": {{bool .enabled}}
                }
            }
        },
        "target": {
            "topic": "devices/sensor-data/transformed",
            "qos": 1,
            "retain": false
        }
    }
}
```

## Building and Running

1. Build the application:
```bash
go build -o message-transformer cmd/server/main.go
```

2. Run with custom config:
```bash
./message-transformer -config /path/to/config/app.json
```

## API Usage

### Health Check
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

### Send Message
```bash
curl -X POST http://localhost:8080/api/v1/sensor-data \
  -H "Content-Type: application/json" \
  -d '{
    "device_id": "sensor123",
    "sensor_type": "temperature",
    "reading": 23.5,
    "unit": "celsius",
    "battery_level": 85,
    "enabled": true
  }'
```
Success Response:
```json
{
    "status": "published",
    "rule_id": "sensor-data-transform",
    "transformed": {
        "deviceId": "sensor123",
        "reading": {
            "type": "temperature",
            "value": 23.5,
            "unit": "celsius",
            "timestamp": "2025-01-31T15:30:00Z"
        },
        "metadata": {
            "batteryLevel": 85,
            "enabled": true
        }
    }
}
```

## Error Responses

- **400 Bad Request** - Invalid JSON or request body
```json
{
    "error": "Invalid JSON in request body"
}
```

- **422 Unprocessable Entity** - Transform error
```json
{
    "error": "Transform error: failed to execute template"
}
```

- **503 Service Unavailable** - MQTT publishing failed
```json
{
    "error": "Failed to publish message"
}
```

## Development

### Required Dependencies
- Go 1.21 or higher
- MQTT Broker for testing (e.g., Mosquitto)
- Access to write configuration files

### Future Development

Areas identified for future improvement:
1. Input validation against template requirements
2. Dynamic URL parameter support
3. Array handling in templates
4. Conditional logic in templates
5. Dynamic MQTT topic generation
6. Message schema validation
