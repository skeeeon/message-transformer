## Rule Examples

### 1. Basic Device Status

This rule transforms device status updates into a standardized format.

```json
{
    "device_status": {
        "id": "device-status",
        "description": "Standardizes device status updates",
        "api": {
            "method": "POST",
            "path": "/api/v1/device-status"
        },
        "transform": {
            "template": {
                "deviceId": "{{.id}}",
                "status": {
                    "state": "{{.current_state}}",
                    "lastUpdated": "{{now}}",
                    "batteryLevel": {{num .battery}},
                    "isOnline": {{bool .online}}
                }
            }
        },
        "target": {
            "topic": "devices/status",
            "qos": 1,
            "retain": true
        }
    }
}
```

Example Input:
```json
{
    "id": "device_123",
    "current_state": "running",
    "battery": 85.5,
    "online": true
}
```

Example Output (Published to MQTT):
```json
{
    "deviceId": "device_123",
    "status": {
        "state": "running",
        "lastUpdated": "2025-01-31T15:30:00Z",
        "batteryLevel": 85.5,
        "isOnline": true
    }
}
```

### 2. Sensor Reading Transformation

This rule transforms legacy sensor readings into a new format with metadata.

```json
{
    "sensor_reading": {
        "id": "sensor-reading",
        "description": "Transforms legacy sensor readings to new format",
        "api": {
            "method": "POST",
            "path": "/api/v1/sensor"
        },
        "transform": {
            "template": {
                "sensorId": "{{.sensor_id}}",
                "measurement": {
                    "type": "{{.type}}",
                    "value": {{num .value}},
                    "unit": "{{.unit}}",
                    "timestamp": "{{now}}"
                },
                "metadata": {{toJSON .meta}},
                "quality": {
                    "isValid": {{bool .valid}},
                    "signalStrength": {{num .signal_strength}}
                }
            }
        },
        "target": {
            "topic": "sensors/data",
            "qos": 2,
            "retain": false
        }
    }
}
```

Example Input:
```json
{
    "sensor_id": "TEMP001",
    "type": "temperature",
    "value": 23.6,
    "unit": "celsius",
    "meta": {
        "location": "room_1",
        "floor": "ground"
    },
    "valid": true,
    "signal_strength": 92
}
```

Example Output (Published to MQTT):
```json
{
    "sensorId": "TEMP001",
    "measurement": {
        "type": "temperature",
        "value": 23.6,
        "unit": "celsius",
        "timestamp": "2025-01-31T15:30:00Z"
    },
    "metadata": {
        "location": "room_1",
        "floor": "ground"
    },
    "quality": {
        "isValid": true,
        "signalStrength": 92
    }
}
```

### 3. Alert Normalization

This rule normalizes alerts from different systems into a standard format.

```json
{
    "alert_normalizer": {
        "id": "alert-normalizer",
        "description": "Normalizes alerts from various systems",
        "api": {
            "method": "POST",
            "path": "/api/v1/alerts"
        },
        "transform": {
            "template": {
                "alertId": "{{.alert_id}}",
                "source": "{{.system}}",
                "severity": "{{.level}}",
                "details": {{toJSON .details}},
                "timestamp": "{{now}}",
                "acknowledgement": {
                    "required": {{bool .needs_ack}},
                    "timeout": {{num .ack_timeout}}
                }
            }
        },
        "target": {
            "topic": "alerts/normalized",
            "qos": 2,
            "retain": true
        }
    }
}
```

Example Input:
```json
{
    "alert_id": "ALT_456",
    "system": "HVAC",
    "level": "critical",
    "details": {
        "code": "HIGH_TEMP",
        "message": "Temperature exceeded threshold",
        "location": "Server Room"
    },
    "needs_ack": true,
    "ack_timeout": 300
}
```

Example Output (Published to MQTT):
```json
{
    "alertId": "ALT_456",
    "source": "HVAC",
    "severity": "critical",
    "details": {
        "code": "HIGH_TEMP",
        "message": "Temperature exceeded threshold",
        "location": "Server Room"
    },
    "timestamp": "2025-01-31T15:30:00Z",
    "acknowledgement": {
        "required": true,
        "timeout": 300
    }
}
```

### Template Function Usage Notes

1. **String Values** (`{{.field_name}}`)
   - Direct string substitution
   - Used for text fields that don't need type conversion
   - Example: `"deviceId": "{{.id}}"`

2. **Numeric Values** (`{{num .field_name}}`)
   - Safely handles integers and floating-point numbers
   - Converts string numbers to proper JSON numbers
   - Example: `"temperature": {{num .temp}}`

3. **Boolean Values** (`{{bool .field_name}}`)
   - Converts various inputs to true/false
   - Handles "true"/"false" strings
   - Example: `"active": {{bool .is_active}}`

4. **JSON Objects** (`{{toJSON .field_name}}`)
   - Preserves object structure
   - Useful for nested data
   - Example: `"metadata": {{toJSON .meta}}`

5. **Timestamps** (`{{now}}`)
   - Inserts current UTC timestamp
   - Format: RFC3339
   - Example: `"timestamp": "{{now}}"`

### Error Handling in Templates

Common error scenarios and their handling:

1. **Missing Fields**
   - String fields: Empty string
   - Numeric fields: 0
   - Boolean fields: false
   - Object fields: null

2. **Type Mismatches**
   - Invalid numbers: 0
   - Invalid booleans: false
   - Invalid JSON: null

3. **JSON Syntax**
   - Invalid template output: Transform error response
   - Invalid input JSON: 400 Bad Request
