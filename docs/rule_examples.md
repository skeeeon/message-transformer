## Rule Examples

### 1. Device Status Rule

This rule transforms device status updates into a standardized format.

```json
{
  "id": "device-status",
  "description": "Standardizes device status updates",
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

### 2. Sensor Reading Rule

This rule transforms sensor readings with metadata handling.

```json
{
  "id": "sensor-reading",
  "description": "Transforms sensor readings with metadata",
  "api": {
    "method": "POST",
    "path": "/api/v1/sensor"
  },
  "transform": {
    "template": "{\"sensorId\": \"{{.sensor_id}}\", \"measurement\": {\"type\": \"{{.type}}\", \"value\": {{num .value}}, \"timestamp\": \"{{now}}\"}, \"metadata\": {{toJSON .meta}}, \"active\": {{bool .active}}}"
  },
  "target": {
    "topic": "sensors/data",
    "qos": 2,
    "retain": false
  }
}
```

Example Input:
```json
{
  "sensor_id": "TEMP001",
  "type": "temperature",
  "value": 23.6,
  "meta": {
    "location": "room_1",
    "floor": "ground"
  },
  "active": true
}
```

Example Output (Published to MQTT):
```json
{
  "sensorId": "TEMP001",
  "measurement": {
    "type": "temperature",
    "value": 23.6,
    "timestamp": "2025-01-31T15:30:00Z"
  },
  "metadata": {
    "location": "room_1",
    "floor": "ground"
  },
  "active": true
}
```

### 3. Alert Processing Rule

This rule processes alerts with JSON string parsing.

```json
{
  "id": "alert-processor",
  "description": "Processes alerts with JSON details",
  "api": {
    "method": "POST",
    "path": "/api/v1/alerts"
  },
  "transform": {
    "template": "{\"alertId\": \"{{.id}}\", \"type\": \"{{.type}}\", \"details\": {{fromJSON .details_json}}, \"timestamp\": \"{{now}}\", \"critical\": {{bool .is_critical}}}"
  },
  "target": {
    "topic": "system/alerts",
    "qos": 1,
    "retain": true
  }
}
```

Example Input:
```json
{
  "id": "ALT_123",
  "type": "system_error",
  "details_json": "{\"code\": \"ERR_001\", \"message\": \"Disk space low\"}",
  "is_critical": true
}
```

Example Output (Published to MQTT):
```json
{
  "alertId": "ALT_123",
  "type": "system_error",
  "details": {
    "code": "ERR_001",
    "message": "Disk space low"
  },
  "timestamp": "2025-01-31T15:30:00Z",
  "critical": true
}
```

### Currently Implemented Template Functions

1. `{{now}}` - Generates current UTC timestamp in RFC3339 format
   - Example: `"timestamp": "{{now}}"`
   - Output: `"timestamp": "2025-01-31T15:30:00Z"`

2. `{{num .field}}` - Safe number handling
   - Example: `"value": {{num .reading}}`
   - Handles integers and floating-point numbers
   - Returns "0" for invalid numbers

3. `{{bool .field}}` - Safe boolean handling
   - Example: `"active": {{bool .status}}`
   - Converts various inputs to true/false
   - Returns "false" for invalid values

4. `{{toJSON .field}}` - Convert object to JSON string
   - Example: `"metadata": {{toJSON .meta}}`
   - Preserves object structure
   - Returns "null" for invalid JSON

5. `{{fromJSON .field}}` - Parse JSON string to object
   - Example: `"details": {{fromJSON .details_json}}`
   - Converts JSON string to object
   - Returns null for invalid JSON

### Important Notes

1. Template Formatting:
   - Template must be a valid JSON string with escaped quotes
   - Use `\"` for quotes within the template

2. Static Elements:
   - MQTT topics are static strings
   - API paths are static (no URL parameters)

3. Data Types:
   - Use `num` for any numeric fields
   - Use `bool` for any boolean fields
   - Use `toJSON`/`fromJSON` for object handling
