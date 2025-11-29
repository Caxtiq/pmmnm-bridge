# PMMNM MQTT Bridge - Testing with Mosquitto

## Install Mosquitto Client Tools

### Windows
Download from: https://mosquitto.org/download/
Or use Chocolatey: `choco install mosquitto`

### Linux
```bash
sudo apt-get install mosquitto-clients
```

## Test Scenarios

### 1. Test Payload-Based Mode (Recommended)

**Publish with sensor ID in payload:**
```bash
mosquitto_pub -h localhost -t "sensors/data" \
  -m '{"sensorId":"sensor-abc123","value":6.5,"timestamp":1701234567890}'
```

**Subscribe to see messages:**
```bash
mosquitto_sub -h localhost -t "sensors/#" -v
```

### 2. Test Topic-Based Mode

**Publish with sensor ID in topic:**
```bash
mosquitto_pub -h localhost -t "sensors/flood/sensor-xyz789" \
  -m '{"value":7.2,"timestamp":1701234567890}'
```

### 3. Test Multiple Sensors

**Simulate multiple sensors:**
```bash
# Sensor 1
mosquitto_pub -h localhost -t "sensors/data" \
  -m '{"sensorId":"sensor-001","value":3.2}'

# Sensor 2
mosquitto_pub -h localhost -t "sensors/data" \
  -m '{"sensorId":"sensor-002","value":5.8}'

# Sensor 3 (threshold exceeded)
mosquitto_pub -h localhost -t "sensors/data" \
  -m '{"sensorId":"sensor-003","value":10.5}'
```

### 4. Test with Authentication

```bash
mosquitto_pub -h localhost -t "sensors/data" \
  -u "username" -P "password" \
  -m '{"sensorId":"sensor-abc123","value":4.5}'
```

### 5. Continuous Testing

**Create a bash script (test-sensor.sh):**
```bash
#!/bin/bash
while true; do
  VALUE=$(awk -v min=0 -v max=10 'BEGIN{srand(); print min+rand()*(max-min)}')
  mosquitto_pub -h localhost -t "sensors/data" \
    -m "{\"sensorId\":\"sensor-test\",\"value\":$VALUE}"
  echo "Sent: $VALUE"
  sleep 5
done
```

**Run it:**
```bash
chmod +x test-sensor.sh
./test-sensor.sh
```

### 6. PowerShell Testing Script (Windows)

**test-sensor.ps1:**
```powershell
while ($true) {
    $value = Get-Random -Minimum 0 -Maximum 10
    $timestamp = [DateTimeOffset]::Now.ToUnixTimeMilliseconds()
    $json = @{
        sensorId = "sensor-test"
        value = $value
        timestamp = $timestamp
    } | ConvertTo-Json -Compress
    
    mosquitto_pub -h localhost -t "sensors/data" -m $json
    Write-Host "Sent: $json"
    Start-Sleep -Seconds 5
}
```

**Run it:**
```powershell
.\test-sensor.ps1
```

## Monitor Bridge Logs

**Watch for incoming data:**
```bash
# Run bridge with debug logging
go run . -config config.yaml

# Or if built:
./pmmnm-bridge -config config.yaml
```

You should see output like:
```
INFO[0005] Processing sensor data sensor_id=sensor-abc123 topic=sensors/data value=6.5
INFO[0005] Successfully forwarded sensor data to API sensor_id=sensor-abc123
```

## Verify API Reception

**Check API directly:**
```bash
curl http://localhost:3001/api/sensor-data?limit=10
```

You should see your test data in the response.
