# proto2fixed Examples

This directory contains example `.proto` schemas demonstrating various features of proto2fixed.

## Example Files

### status.proto

A comprehensive telemetry example demonstrating:

- **Nested messages**: IMU, Battery, Servo structs
- **Fixed-size arrays**: `repeated Servo servos = 2 [(proto2fixed.array_size) = 12]`
- **Fixed-size strings**: `string robot_id = 1 [(proto2fixed.string_size) = 64]`
- **Primitive types**: uint32, float, bool
- **Automatic layout calculation**: Offsets and padding computed automatically

**Binary Layout:**
- StatusReport: 431 bytes
- Servo: 30 bytes each
- IMU: 40 bytes
- Battery: 27 bytes
- RobotInfo: 84 bytes

### command.proto

A command protocol example demonstrating:

- **Union messages**: `CommandPayload` with `(proto2fixed.union) = true`
- **Enum types**: ActionType, GaitType with 1-byte encoding
- **Discriminated unions**: ActionType field indicates which payload variant is active
- **Multiple command variants**: ServoCommand, GaitCommand, StopCommand, CalibrateCommand, ConfigCommand

**Binary Layout:**
- Command: 128 bytes
- CommandPayload (union): 120 bytes (size of largest variant)

## Usage

### Validate Schemas

```bash
# Validate a single schema
proto2fixed --validate status.proto

# Validate all examples
mage schemas:validate
```

### Generate JSON Schema

```bash
proto2fixed --lang=json status.proto > status_schema.json
proto2fixed --lang=json command.proto > command_schema.json

# Or use mage
mage schemas:json
```

### Generate Arduino/C++ Code

```bash
proto2fixed --lang=arduino --output=status.h status.proto
proto2fixed --lang=arduino --output=command.h command.proto

# Or use mage
mage schemas:arduino
```

### Generate Go Code

```bash
proto2fixed --lang=go --package=protocol --output=status.go status.proto
proto2fixed --lang=go --package=protocol --output=command.go command.proto

# Or use mage
mage schemas:go
```

### Generate All Outputs

```bash
mage schemas:generate
```

This creates:
- `generated/*.json` - JSON schemas
- `generated/arduino/*.h` - C/C++ headers
- `generated/go/*.go` - Go decoders/encoders

## Integration Example

### ESP32 Firmware (Arduino)

```cpp
#include "status.h"
#include "command.h"

// Create status message
StatusReport status;
status.timestamp = millis();

// Fill servo data
for (int i = 0; i < 12; i++) {
  status.servos[i].position = servo_positions[i];
  status.servos[i].target_position = servo_targets[i];
  status.servos[i].moving = true;
}

// Fill IMU data
status.imu.accel_x = imu.getAccelX();
status.imu.accel_y = imu.getAccelY();
status.imu.accel_z = imu.getAccelZ();

// Encode and send
uint8_t buffer[sizeof(StatusReport)];
encodeStatusReport(&status, buffer);
Serial.write(buffer, sizeof(buffer));

// Receive and decode command
if (Serial.available() >= sizeof(Command)) {
  uint8_t cmdBuffer[sizeof(Command)];
  Serial.readBytes(cmdBuffer, sizeof(Command));

  Command cmd;
  decodeCommand(cmdBuffer, &cmd);

  switch (cmd.action) {
    case ACTION_SERVO:
      handleServoCommand(&cmd.payload.servo);
      break;
    case ACTION_GAIT:
      handleGaitCommand(&cmd.payload.gait);
      break;
    case ACTION_STOP:
      handleStopCommand(&cmd.payload.stop);
      break;
  }
}
```

### Orin Nano (Go)

```go
package main

import (
    "fmt"
    "yourproject/protocol"
)

func main() {
    // Decode status from ESP32
    statusDecoder := protocol.NewStatusReportDecoder()

    binaryData := readFromUART() // Read 431 bytes
    jsonStr, err := statusDecoder.Decode(binaryData)
    if err != nil {
        panic(err)
    }

    fmt.Println("Status:", jsonStr)
    // {"timestamp":12345,"servos":[...],"imu":{...},"battery":{...}}

    // Encode command to ESP32
    cmdEncoder := protocol.NewCommandEncoder()

    commandJSON := `{
      "id": 42,
      "action": 1,
      "payload": {
        "servo": {
          "servoId": 5,
          "targetPosition": 45.0,
          "speed": 50.0,
          "torqueLimit": 80.0
        }
      }
    }`

    binaryCmd, err := cmdEncoder.Encode(commandJSON)
    if err != nil {
        panic(err)
    }

    writeToUART(binaryCmd) // Write 128 bytes
}
```

## Tips and Best Practices

### 1. Use Sequential Field Numbers

```protobuf
// Good: Sequential numbering
message Good {
  uint32 field_a = 1;  // Offset: 0
  uint32 field_b = 2;  // Offset: 4
  uint32 field_c = 3;  // Offset: 8
}

// Avoid: Gaps in numbering create padding
message Avoid {
  uint32 field_a = 1;  // Offset: 0
  uint32 field_b = 5;  // Offset: 4 (warning: gap in field numbers)
  uint32 field_c = 10; // Offset: 8 (warning: gap in field numbers)
}
```

### 2. Group Fields by Size for Better Packing

```protobuf
// Better packing: largest first
message BetterPacking {
  uint64 large = 1;    // 8 bytes, offset: 0
  uint32 medium = 2;   // 4 bytes, offset: 8
  uint16 small = 3;    // 2 bytes, offset: 12
  uint8 tiny = 4;      // 1 byte, offset: 14
}
```

### 3. Always Specify Array and String Sizes

```protobuf
message Sizes {
  // ✗ Error: missing size
  repeated float bad = 1;

  // ✓ Correct: size specified
  repeated float good = 2 [(proto2fixed.array_size) = 10];

  // ✗ Error: missing size
  string bad_str = 3;

  // ✓ Correct: size specified (includes null terminator)
  string good_str = 4 [(proto2fixed.string_size) = 32];
}
```

### 4. Use Unions for Command Payloads

```protobuf
// Discriminator enum
enum CommandType {
  option (proto2fixed.enum_size) = 1;
  CMD_A = 0;
  CMD_B = 1;
}

message Command {
  CommandType type = 1;           // Discriminator
  CommandPayload payload = 2;     // Union
}

message CommandPayload {
  option (proto2fixed.union) = true;
  CommandA cmd_a = 1;
  CommandB cmd_b = 2;
}
```

### 5. Validate Schemas Early

```bash
# Validate during development
proto2fixed --validate myschema.proto

# Integrate into build pipeline
mage schemas:validate
```

## Binary Layout Visualization

### status.proto Layout

```
StatusReport (431 bytes)
├─ timestamp (uint32)         : Offset 0-3     (4 bytes)
├─ servos[12] (Servo)         : Offset 4-363   (360 bytes)
│  └─ Each Servo:
│     ├─ position (float)     : +0-3   (4 bytes)
│     ├─ target_position      : +4-7   (4 bytes)
│     ├─ speed                : +8-11  (4 bytes)
│     ├─ load                 : +12-15 (4 bytes)
│     ├─ voltage              : +16-19 (4 bytes)
│     ├─ temperature          : +20-23 (4 bytes)
│     ├─ moving (bool)        : +24    (1 byte)
│     ├─ enabled (bool)       : +25    (1 byte)
│     └─ padding              : +26-29 (4 bytes)
├─ imu (IMU)                  : Offset 364-403 (40 bytes)
└─ battery (Battery)          : Offset 404-430 (27 bytes)
```

### command.proto Layout (Union)

```
Command (128 bytes)
├─ id (uint32)                : Offset 0-3     (4 bytes)
├─ action (ActionType)        : Offset 4       (1 byte)
├─ padding                    : Offset 5-7     (3 bytes)
└─ payload (CommandPayload)   : Offset 8-127   (120 bytes, union)
   ├─ servo (ServoCommand)    : Size 16 bytes
   ├─ gait (GaitCommand)      : Size 24 bytes
   ├─ stop (StopCommand)      : Size 1 byte
   ├─ calibrate (CalibrateCommand) : Size 16 bytes
   └─ config (ConfigCommand)  : Size 120 bytes (largest)

Union size = 120 bytes (size of ConfigCommand, the largest variant)
```

## Troubleshooting

### Size Mismatch Error

```
Error: Message 'StatusReport' declared size (431) != calculated size (435)
```

**Solution**: Remove the `option (proto2fixed.size)` declaration or update it to match the calculated size.

### Missing Array Size

```
Error: Field 'servos' is repeated but missing (proto2fixed.array_size) option
```

**Solution**: Add the array size option:
```protobuf
repeated Servo servos = 2 [(proto2fixed.array_size) = 12];
```

### Missing String Size

```
Error: Field 'name' is type string but missing (proto2fixed.string_size) option
```

**Solution**: Add the string size option:
```protobuf
string name = 1 [(proto2fixed.string_size) = 32];
```

### Alignment Warnings

```
Warning: Field 'value' at offset 5 is not 4-byte aligned
```

**Solution**: Reorder fields or add explicit padding fields to ensure proper alignment.

## Performance Characteristics

### Encoding/Decoding

- **Arduino (memcpy)**: ~50µs for 431-byte message
- **Go (binary encoding)**: ~200µs for 431-byte message
- **Zero allocations** in hot path (Arduino)
- **Minimal allocations** in Go (only for JSON marshal)

### Memory Usage

- **StatusReport**: 431 bytes (stack allocation)
- **Command**: 128 bytes (stack allocation)
- **No heap allocations** for encode/decode (Arduino)

### Bandwidth

- **Fixed size** enables precise bandwidth calculation
- **No framing overhead** (application handles framing)
- **Example**: 100Hz status @ 431 bytes = 43.1 KB/s = 344.8 Kbit/s

## Additional Resources

- Main README: `../README.md`
- Options Definition: `../proto2fixed/binary.proto`
- Source Code: `../pkg/`

## Questions?

File an issue: https://github.com/smoxy-io/proto2fixed/issues
