# Message ID Validation Rules

This document describes all validation rules enforced for the `binary.message_id` and `binary.message_id_size` options in proto2fixed.

## Enforced Rules

### 1. At Least One Message ID Required ❌ ERROR
**Rule:** The schema must have at least one top-level message with a `message_id` option.

**Example (Invalid):**
```protobuf
option (binary.fixed) = true;

message Command {
  // ERROR: No message_id defined
  uint32 id = 1;
}

message Response {
  // ERROR: No message_id defined
  uint32 status = 1;
}
```

**Validation:**
- File: `pkg/analyzer/validator.go:418-422`
- Test: `testdata/validation/no_message_ids.proto`
- Error: "Schema must have at least one top-level message with a message_id option"

**Rationale:** Message IDs are fundamental to the proto2fixed protocol for runtime message identification and dispatching. Every schema must define at least one message ID.

---

### 2. Valid message_id_size Values ❌ ERROR
**Rule:** The file-level `message_id_size` option must be 1, 2, 4, or 8 bytes.

**Example (Invalid):**
```protobuf
option (binary.message_id_size) = 3;  // ERROR: must be 1, 2, 4, or 8
```

**Validation:**
- File: `pkg/analyzer/validator.go:345-350`
- Test: `testdata/validation/invalid_message_id_size.proto`
- Error: "File option message_id_size (N) must be 1, 2, 4, or 8"

---

### 3. Unique Message IDs ❌ ERROR
**Rule:** Each message must have a unique `message_id`. No two messages can share the same ID.

**Example (Invalid):**
```protobuf
message Command {
  option (binary.message_id) = 1;
  uint32 id = 1;
}

message Response {
  option (binary.message_id) = 1;  // ERROR: duplicate
  uint32 status = 1;
}
```

**Validation:**
- File: `pkg/analyzer/validator.go:392-400`
- Test: `testdata/validation/duplicate_message_ids.proto`
- Error: "Message 'Response' has duplicate message_id 1 (already used by 'Command')"

---

### 4. Message ID Within Range ❌ ERROR
**Rule:** Message ID values must fit within the size specified by `message_id_size`.

| Size | Max Value |
|------|-----------|
| 1 byte | 255 |
| 2 bytes | 65,535 |
| 4 bytes | 4,294,967,295 |
| 8 bytes | 18,446,744,073,709,551,615 |

**Example (Invalid):**
```protobuf
option (binary.message_id_size) = 1;  // 1 byte, max = 255

message Command {
  option (binary.message_id) = 300;  // ERROR: exceeds max
  uint32 id = 1;
}
```

**Validation:**
- File: `pkg/analyzer/validator.go:384-390`
- Test: `testdata/validation/message_id_overflow.proto`
- Error: "Message 'Command' message_id (300) exceeds maximum for size 1 bytes (max: 255)"

---

### 5. Nested Messages Should Not Have IDs ⚠️ WARNING
**Rule:** Only top-level messages should have `message_id`. Nested messages (used as field types) should not have IDs, as they will be ignored.

**Example (Warning):**
```protobuf
message Nested {
  option (binary.message_id) = 99;  // WARNING: ignored
  uint32 value = 1;
}

message Parent {
  option (binary.message_id) = 1;
  Nested nested = 1;  // Nested is used as a field type
}
```

**Validation:**
- File: `pkg/analyzer/validator.go:376-382`
- Test: `testdata/validation/nested_with_message_id.proto`
- Warning: "Message 'Nested' is nested and should not have message_id option (will be ignored)"

---

### 6. Top-Level Messages Should Have IDs ⚠️ WARNING
**Rule:** When `message_id_size` is configured, all top-level messages should have a `message_id` for consistency.

**Example (Warning):**
```protobuf
option (binary.message_id_size) = 4;

message Command {
  option (binary.message_id) = 1;
  uint32 id = 1;
}

message Response {
  // WARNING: missing message_id
  uint32 status = 1;
}
```

**Validation:**
- File: `pkg/analyzer/validator.go:401-406`
- Test: `testdata/validation/missing_message_id.proto`
- Warning: "Top-level message 'Response' does not have a message_id option"

---

## Special Cases

### Message ID = 0
Setting `message_id = 0` is equivalent to not setting it at all. Zero is treated as "no message ID assigned."

**Example:**
```protobuf
message Command {
  option (binary.message_id) = 0;  // Same as not having the option
  uint32 id = 1;
}
```

**Result:** Warning for missing message_id (if message_id_size is configured)

---

### Default message_id_size
If `message_id_size` is not specified, it defaults to 4 bytes.

**Example:**
```protobuf
// No message_id_size specified → defaults to 4
message Command {
  option (binary.message_id) = 1;  // Will use 4 bytes
  uint32 id = 1;
}
```

---

## CLI Enforcement

All validation rules are enforced before code generation:

```bash
# Validation-only mode
$ proto2fixed --validate schema.proto
Warning: Message 'Response' does not have a message_id option
✓ Schema validation passed for schema.proto

# Code generation (validation runs automatically)
$ proto2fixed --lang=json schema.proto
Error: Message 'Response' has duplicate message_id 1
exit status 1
```

**Behavior:**
- **Errors:** Code generation fails and exits with status 1
- **Warnings:** Code generation succeeds but warnings are printed to stderr

---

## Test Coverage

All validation rules are comprehensively tested:

| Rule | Unit Tests | Integration Tests | Test Files |
|------|-----------|-------------------|-----------|
| At least one message ID | ✅ `TestValidator_NoMessageIds` | ✅ `TestMessageIdValidationEnforcement` | `testdata/validation/no_message_ids.proto` |
| Valid message_id_size | ✅ `TestValidator_InvalidMessageIdSize` | ✅ `TestMessageIdValidationEnforcement` | `testdata/validation/invalid_message_id_size.proto` |
| Unique message IDs | ✅ `TestValidator_DuplicateMessageIds` | ✅ `TestMessageIdCodeGenerationBlocked` | `testdata/validation/duplicate_message_ids.proto` |
| Message ID range | ✅ `TestValidator_MessageIdExceedsMax` | ✅ `TestMessageIdValidationEnforcement` | `testdata/validation/message_id_overflow.proto` |
| Nested message warning | ✅ `TestValidator_NestedMessageWithId` | ✅ `TestMessageIdValidationEnforcement` | `testdata/validation/nested_with_message_id.proto` |
| Missing ID warning | ✅ `TestValidator_MissingMessageId` | ✅ `TestMessageIdValidationEnforcement` | `testdata/validation/missing_message_id.proto` |
| Valid schemas | ✅ `TestValidator_MessageIds` | ✅ `TestMessageIdCodeGenerationSuccess` | `testdata/validation/valid_message_ids.proto` |

Run all validation tests:
```bash
go test -v -run TestMessageId ./...
```

---

## Implementation Files

| Component | File | Lines |
|-----------|------|-------|
| Validation Logic | `pkg/analyzer/validator.go` | 337-412 |
| Unit Tests | `pkg/analyzer/validator_test.go` | 556-772 |
| Parser Tests | `pkg/parser/parser_test.go` | 390-446 |
| Integration Tests | `validation_enforcement_test.go` | 1-169 |
| CLI Enforcement | `cmd/proto2fixed/main.go` | 69-90 |

---

## Examples

### ✅ Valid Configuration
```protobuf
syntax = "proto3";
package protocol;

import "proto2fixed/binary.proto";

option (binary.fixed) = true;
option (binary.message_id_size) = 2;

message Command {
  option (binary.message_id) = 1;
  uint32 id = 1;
}

message Response {
  option (binary.message_id) = 2;
  uint32 status = 1;
}
```

### ❌ Invalid Configuration
```protobuf
syntax = "proto3";
package protocol;

import "proto2fixed/binary.proto";

option (binary.fixed) = true;
option (binary.message_id_size) = 3;  // ERROR: must be 1,2,4,8

message Command {
  option (binary.message_id) = 1;
  uint32 id = 1;
}

message Response {
  option (binary.message_id) = 1;  // ERROR: duplicate ID
  uint32 status = 1;
}
```
