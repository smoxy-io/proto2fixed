# # proto2fixed

A command-line tool that compiles Protocol Buffer v3 syntax into fixed-size binary layouts optimized for embedded systems and real-time communication.

## Overview

Unlike standard Protocol Buffers which use variable-length encoding (varint), `proto2fixed` generates fixed-width binary structures ideal for:

- **Embedded Systems**: ESP32, Arduino, STM32
- **Real-time Communication**: Predictable message sizes
- **Low-latency Applications**: No parsing overhead (direct memory access)
- **Resource-constrained Devices**: Minimal memory footprint

This tool was built to ease the development of AHC2 and AHSR compliant software for AI <--> Hardware communication that relies on auto-discovery of the communication schema at runtime and binary encoding/decoding of messages for real-time communication across low-speed links.

## Features

- ✅ Familiar `.proto` syntax for schema definition
- ✅ Fixed-size binary layouts (no varint encoding)
- ✅ Auto-calculated field offsets and padding
- ✅ Multiple output targets: JSON schema, Arduino/C++, Go
- ✅ Dynamic schema discovery via embedded JSON
- ✅ Compile-time validation and size assertions
- ✅ Support for strings, arrays, nested messages, enums, and unions
- ✅ Message IDs for runtime type identification and dispatching
- ✅ Registry-based message routing (Go) and dynamic codecs
- ✅ Automatic discriminator headers for union/oneof type identification

## Installation

### Prebuilt Binaries

Prebuilt binaries for Linux, MacOS, and Windows are provided with each release.

### From Source

```*bash*
git clone https://github.com/smoxy-io/proto2fixed
mage buildlocal
mage install
```

## Quick Start

### 1. Define Your Schema

Create a `.proto` file using standard proto3 syntax:

```*protobuf*
syntax = "proto3";

import "proto2fixed/binary.proto";

package protocol;

message StatusReport {
  option (binary.message_id) = 1;

  uint32 timestamp = 1;
  float temperature = 2;
  bool active = 3;
}
```

**That's it!** proto2fixed works with standard proto3 files that import proto2fixed’s custom binary options file.  `package` must be defined and at least one message must define a message id using the `binary.message_id` option.

### 2. Generate Code

```*bash*
# Validate schema
proto2fixed --validate status.proto

# Generate JSON schema
proto2fixed --lang=json status.proto > status_schema.json

# Generate Arduino/C++ header
proto2fixed --lang=arduino --output=./ status.proto

# Generate Go decoder/encoder
proto2fixed --lang=go --output=./ status.proto
```

### 3. Use Generated Code

**Arduino/ESP32:**

```*cpp*
#include "status.h"

StatusReport msg;
msg.timestamp = millis();
msg.temperature = 25.5f;
msg.active = true;

uint8_t buffer[sizeof(StatusReport)];
encodeStatusReport(&msg, buffer);
Serial.write(buffer, sizeof(buffer));
```

**Go (Orin Nano):**

```*go*
import "yourproject/protocol"

decoder := protocol.NewStatusReportDecoder()
jsonStr, err := decoder.Decode(binaryData)
// jsonStr contains: {"timestamp":12345,"temperature":25.5,"active":true}
```

## Command-Line Interface

```
Usage:
  proto2fixed [flags] <input.proto> [<input.proto>] ...

Flags:
  --import-paths=<pathList>    OS specific path-list-separator separated
                               list of import paths (linux: colon-separated,
                               windows: semicolon-separated)
  --lang=<target>              Output language (json|arduino|go)
  --output=<dir>               Output directory (default: stdout)
  --validate                   Validate schema only (no code generation)
  --version                    Show version information
  --help                       Show help message

Examples:
  proto2fixed --lang=json status.proto
  proto2fixed --lang=arduino --output=./ status.proto
  proto2fixed --lang=go status.proto
  proto2fixed --validate status.proto
```

## Schema Options

### File-Level Options

```*protobuf*
option (binary.fixed) = true;          // Enable fixed binary mode (default: true)
option (binary.endian) = "little";     // Endianness: "little" or "big" (default: little)
option (binary.version) = "v1.0.0";    // Schema version (default: v1.0.0)
option (binary.message_id_size) = 1;   // Message ID header size: 1, 2, 4, or 8 bytes (default: 1)
```

### Message Options

```*protobuf*
message MyMessage {
  option (binary.size) = 64;           // Optional: Validate calculated size
  option (binary.align) = 4;           // Alignment boundary (default: natural)
  option (binary.union) = true;        // Fields overlay (union-like)
  option (binary.message_id) = 1;      // Message identifier for top-level messages
}
```

**Message IDs:** Top-level messages can have a unique `message_id` which enables:
- Message type identification at runtime
- Automatic message dispatching (dynamic codec)
- Header-prefixed binary encoding: `[message_id][message_data]`

Only top-level messages should have message IDs. Nested messages will generate a warning if they include a `message_id` option.

At least one top-level message must define a message id.

When code is generated, only messages with message IDs will have encoders and decoders generated for them.

### Field Options

```*protobuf*
// Fixed-size arrays (REQUIRED for repeated fields)
repeated float values = 1 [(binary.array_size) = 16];

// Fixed-size strings (REQUIRED for string fields)
string name = 2 [(binary.string_size) = 32];
```

**Strings:** a `string` field is represented as a null terminated array of bytes. This means that a `string` field with `string_size = 8` can hold a maximum of 7 single byte characters (the 8th byte will be the null byte `\0`).

**Important:** Arrays and strings require size specifications. The validator will error if these are missing.

### Enum Options

```*protobuf*
enum Status {
  option (binary.enum_size) = 1;       // Size: 1, 2, or 4 bytes (default: 1)

  *UNKNOWN* = 0;
  *ACTIVE* = 1;
}
```

## Supported Types

| Proto Type | Size | Notes |
|------------|------|-------|
| `bool` | 1 byte | |
| `int32`, `uint32` | 4 bytes | Fixed-width (not varint) |
| `int64`, `uint64` | 8 bytes | Fixed-width (not varint) |
| `float` | 4 bytes | IEEE 754 |
| `double` | 8 bytes | IEEE 754 |
| `string` | Fixed | Requires `(binary.string_size)` |
| `bytes` | Fixed | Requires `(binary.array_size)` |
| `repeated` | Fixed | Requires `(binary.array_size)` |
| Nested messages | Calculated | Inline structs |
| Enums | 1/2/4 bytes | Configurable with `(binary.enum_size)` |
| `oneof` | Calculated | 1-byte discriminator + largest variant |

## Binary Layout Rules

### Field Ordering

Fields are laid out in **field number order** (ascending):

```*protobuf*
message Example {
  uint32 field_c = 3;  // Would be offset: 8 (not 0!)
  uint32 field_a = 1;  // Would be offset: 0 (not 4!)
  uint32 field_b = 2;  // Would be offset: 4 (not 8!)
}
```

### Alignment and Padding

- Fields are automatically aligned to their natural boundaries
- Padding is inserted where needed
- Explicit padding fields are generated in output

```*protobuf*
message Aligned {
  bool flag = 1;      // Offset: 0, Size: 1
  // Automatic padding: 3 bytes
  uint32 value = 2;    // Offset: 4, Size: 4 (4-byte aligned)
}
```

### Union Messages

Messages marked with `(binary.union) = true` have overlapping fields with a 1-byte discriminator header:

```*protobuf*
message CommandPayload {
  option (binary.union) = true;

  ServoCommand servo = 1;     // Field number: 1
  GaitCommand gait = 2;        // Field number: 2
  StopCommand stop = 3;        // Field number: 3
}
```

**Binary Layout:**
- Byte 0: Discriminator (uint8) containing the field number of the active variant
- Byte 1+: The active field data (all fields overlay at offset 1)
- Size = 1 (discriminator) + size of largest field
- Maximum field number: 255 (uint8 limit)

The discriminator enables runtime identification of which field is active in the union.

### Oneof Fields

Oneof fields are similar to unions but at the field level. They also use a 1-byte discriminator:

```*protobuf*
message Notification {
  uint32 id = 1;

  oneof payload {
    string text = 2;          // Field number: 2
    bytes image = 3;          // Field number: 3
    uint32 code = 4;          // Field number: 4
  }
}
```

**Binary Layout:**
- The oneof region includes a 1-byte discriminator at the start
- Discriminator (uint8) contains the field number of the active variant
- All variants overlay at offset +1 within the oneof region
- Oneof size = 1 (discriminator) + size of largest variant
- Maximum variant field number: 255 (uint8 limit)

The discriminator is automatically handled during encoding/decoding using O(1) lookup maps for optimal performance.

## Examples

See the `examples/` directory for complete examples:

- **status.proto**: Telemetry message with arrays, nested messages, and strings
- **command.proto**: Command message with unions and enums

Generate all examples:

```*bash*
mage examples:generate
```

## Message IDs and Runtime Dispatching

Message IDs enable runtime message type identification and automatic dispatching. This is particularly useful for:

- **Protocol multiplexing**: Send different message types over a single channel
- **Dynamic routing**: Decode and route messages without knowing type ahead of time
- **Auto-discovery**: Clients can query available message types from schema

### Example: Command/Response Protocol

```*protobuf*
syntax = "proto3";

package protocol;

import "proto2fixed/binary.proto";

option (binary.fixed) = true;
option (binary.endian) = "little";
option (binary.message_id_size) = 2;  // Use 2-byte message IDs

message Command {
  option (binary.message_id) = 1;

  uint32 id = 1;
  string action = 2 [(binary.string_size) = 32];
  uint32 value = 3;
}

message Response {
  option (binary.message_id) = 2;

  uint32 id = 1;
  int32 status = 2;
  string message = 3 [(binary.string_size) = 64];
}
```

### Generated Code Usage

**Arduino: Encoding with Message ID**

```*cpp*
#include "protocol.h"

// Encode Command
Command cmd;
cmd.id = 42;
strcpy(cmd.action, "MOVE");
cmd.value = 100;

uint8_t buffer[sizeof(MessageId) + sizeof(Command)];  // Header + body
encodeCommand(&cmd, buffer);  // Prepends message ID automatically
Serial.write(buffer, sizeof(buffer));
```

**Go: Runtime Dispatching**

```*go*
import "generated/protocol"

// Create registry
registry := protocol.NewMessageRegistry()

// Decode incoming binary data by message ID
data := <-channel  // Receive binary message
msgName, jsonData, err := registry.DecodeById(data)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Received %s: %s\n", msgName, jsonData)
// Output: "Received Command: {\"id\":42,\"action\":\"MOVE\",\"value\":100}"

// Lookup message info
id, _ := registry.GetIdByName("Command")   // Returns: 1
name, _ := registry.GetNameById(2)          // Returns: "Response"
```

**Dynamic Codec: Schema-driven Encoding/Decoding**

```*go*
import (
    "github.com/smoxy-io/proto2fixed/pkg/api"
)

// Load JSON schema (could be from embedded JSON or API)
schema := loadJSONSchema()  // Contains messageIdSize, messageHeader, etc.

// Create codec
codec, _ := api.NewCodec(schema)

// Encode JSON to binary
jsonInput := `{"Command": {"id": 42, "action": "MOVE", "value": 100}}`
binary, _ := codec.Encode([]byte(jsonInput))
// binary = [0x01, 0x00, 0x2A, 0x00, 0x00, 0x00, ...]
//          ^^^^^^^^ Message ID (1 in 2 bytes, little-endian)

// Decode binary to JSON (automatic message type detection)
jsonOutput, _ := codec.Decode(binary)
// jsonOutput = `{"Command":{"id":42,"action":"MOVE","value":100}}`

// Union and oneof discriminators are handled automatically
// Encoding: sets discriminator to active field number
// Decoding: uses O(1) lookup maps for fast variant identification
```

### Validation

The validator ensures:
- ✅ Message IDs are unique across all top-level messages
- ✅ Message IDs fit within the configured size (1/2/4/8 bytes)
- ✅ Warnings for nested messages with message IDs (they're ignored)
- ✅ Warnings for top-level messages without message IDs
- ✅ Union message field numbers ≤ 255 (discriminator is uint8)
- ✅ Oneof variant field numbers ≤ 255 (discriminator is uint8)

```*bash*
$ proto2fixed --validate protocol.proto
Warning: Message 'NestedData' is nested and should not have message_id option
Error: Message 'Response' has duplicate message_id 1 (already used by 'Command')
```

## Building and Testing

`mage` is used instead of `make`.  It is preferred to use the mage commands instead of directly using  `go` commands or using shell scripts.

### Using Mage

```*bash*
# Build proto2fixed for all platforms
mage build

# Build proto2fixed for local platform
mage buildlocal

# Run tests
mage test

# Format code
mage fmt

# Run benchmarks
mage bench

# Validate example schemas
mage examples:validate

# Generate all example outputs
mage examples:generate

# Clean generated example files
mage examples:clean
```

### Performance Benchmarks

Detailed performance benchmarks for all components (parser, analyzer, generators, codecs) are available in [BENCHMARKS.md](*BENCHMARKS.md*). Key highlights:

- **Union/Oneof decoding**: O(1) discriminator lookups (~601-841 ns)
- **Simple message encoding**: ~939 ns with 952 B/op
- **Parser performance**: 65-916 μs depending on schema complexity
- **Code generation**: 4-2000 μs for JSON/Arduino/Go output

Run benchmarks yourself:
```*bash*
mage bench
```

## Project Structure

```
proto2fixed/
├─ cmd/proto2fixed/        # CLI entry point
├─ pkg/
│  ├─ api/                 # Library entry point
│  ├─ codecs/
│  │  └─ dynamic/          # Dynamic codec implementation
│  ├─ parser/              # Proto file parser
│  ├─ analyzer/            # Layout calculation & validation
│  └─ generator/           # Code generators (JSON, Arduino, Go)
├─ proto2fixed/            # Custom options definition
├─ examples/               # Example proto schemas
├─ build/mage/             # Mage build targets
└─ README.md
```

## Contributing

Contributions are welcome! Please:

1\. Fork the repository
2\. Create a feature branch
3\. Add tests for new functionality
4\. Run `mage fmt` and `mage test`
5\. Submit a pull request

## License

MIT License - see LICENSE file for details.

## Acknowledgments

- Built with [jhump/protoreflect](*https://github.com/jhump/protoreflect*) for proto parsing
- Inspired by the need for efficient binary protocols in embedded robotics

## Support

- Issues: https://github.com/smoxy-io/proto2fixed/issues
- Documentation: See `examples/README.md` for detailed usage examples

