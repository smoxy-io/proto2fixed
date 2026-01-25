package proto2fixed

import (
	"os/exec"
	"strings"
	"testing"
)

// TestMessageIdValidationEnforcement verifies that all message_id validation rules are enforced
func TestMessageIdValidationEnforcement(t *testing.T) {
	tests := []struct {
		name        string
		protoFile   string
		shouldError bool
		shouldWarn  bool
		errorMsg    string
		warnMsg     string
	}{
		{
			name:        "valid message IDs",
			protoFile:   "testdata/validation/valid_message_ids.proto",
			shouldError: false,
			shouldWarn:  false,
		},
		{
			name:        "duplicate message IDs",
			protoFile:   "testdata/validation/duplicate_message_ids.proto",
			shouldError: true,
			errorMsg:    "duplicate message_id",
		},
		{
			name:        "message ID overflow",
			protoFile:   "testdata/validation/message_id_overflow.proto",
			shouldError: true,
			errorMsg:    "exceeds maximum",
		},
		{
			name:        "invalid message_id_size",
			protoFile:   "testdata/validation/invalid_message_id_size.proto",
			shouldError: true,
			errorMsg:    "must be 1, 2, 4, or 8",
		},
		{
			name:        "nested message with message_id",
			protoFile:   "testdata/validation/nested_with_message_id.proto",
			shouldError: false,
			shouldWarn:  true,
			warnMsg:     "nested and should not have message_id",
		},
		{
			name:        "missing message_id",
			protoFile:   "testdata/validation/missing_message_id.proto",
			shouldError: false,
			shouldWarn:  true,
			warnMsg:     "does not have a message_id option",
		},
		{
			name:        "message_id zero treated as no ID",
			protoFile:   "testdata/validation/message_id_zero.proto",
			shouldError: false,
			shouldWarn:  true,
			warnMsg:     "does not have a message_id option",
		},
		{
			name:        "no message IDs defined",
			protoFile:   "testdata/validation/no_message_ids.proto",
			shouldError: true,
			errorMsg:    "at least one top-level message with a message_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Run proto2fixed --validate
			cmd := exec.Command("go", "run", "cmd/proto2fixed/main.go", "--validate", tt.protoFile)
			output, err := cmd.CombinedOutput()
			outputStr := string(output)

			// Check error expectation
			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected validation to fail, but it succeeded")
				}
				if !strings.Contains(outputStr, tt.errorMsg) {
					t.Errorf("Expected error message containing '%s', got:\n%s", tt.errorMsg, outputStr)
				}
			} else {
				if err != nil {
					t.Errorf("Expected validation to succeed, but it failed with:\n%s", outputStr)
				}
			}

			// Check warning expectation
			if tt.shouldWarn {
				if !strings.Contains(outputStr, "Warning:") {
					t.Errorf("Expected warning, but none found in:\n%s", outputStr)
				}
				if tt.warnMsg != "" && !strings.Contains(outputStr, tt.warnMsg) {
					t.Errorf("Expected warning containing '%s', got:\n%s", tt.warnMsg, outputStr)
				}
			}
		})
	}
}

// TestMessageIdCodeGenerationBlocked verifies that code generation fails when validation errors exist
func TestMessageIdCodeGenerationBlocked(t *testing.T) {
	tests := []struct {
		name      string
		protoFile string
		lang      string
	}{
		{"duplicate IDs - JSON", "testdata/validation/duplicate_message_ids.proto", "json"},
		{"duplicate IDs - Arduino", "testdata/validation/duplicate_message_ids.proto", "arduino"},
		{"duplicate IDs - Go", "testdata/validation/duplicate_message_ids.proto", "go"},
		{"overflow - JSON", "testdata/validation/message_id_overflow.proto", "json"},
		{"invalid size - Go", "testdata/validation/invalid_message_id_size.proto", "go"},
		{"no message IDs - JSON", "testdata/validation/no_message_ids.proto", "json"},
		{"no message IDs - Arduino", "testdata/validation/no_message_ids.proto", "arduino"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command("go", "run", "cmd/proto2fixed/main.go", "--lang="+tt.lang, tt.protoFile)
			output, err := cmd.CombinedOutput()

			if err == nil {
				t.Errorf("Expected code generation to fail, but it succeeded with output:\n%s", string(output))
			}

			outputStr := string(output)
			if !strings.Contains(outputStr, "Error:") {
				t.Errorf("Expected error message, got:\n%s", outputStr)
			}
		})
	}
}

// TestMessageIdCodeGenerationSuccess verifies that valid schemas generate code successfully
func TestMessageIdCodeGenerationSuccess(t *testing.T) {
	tests := []struct {
		name      string
		protoFile string
		lang      string
		contains  []string
	}{
		{
			name:      "valid IDs - JSON",
			protoFile: "testdata/validation/valid_message_ids.proto",
			lang:      "json",
			contains:  []string{"messageIdSize", "messageHeader", "\"messageId\":"},
		},
		{
			name:      "valid IDs - Arduino",
			protoFile: "testdata/validation/valid_message_ids.proto",
			lang:      "arduino",
			contains:  []string{"MessageId", "MSGID_COMMAND", "encodeCommand"},
		},
		{
			name:      "valid IDs - Go",
			protoFile: "testdata/validation/valid_message_ids.proto",
			lang:      "go",
			contains:  []string{"MessageIdSize", "CommandMessageId" /*, "MessageRegistry"*/},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command("go", "run", "cmd/proto2fixed/main.go", "--lang="+tt.lang, tt.protoFile)
			output, err := cmd.CombinedOutput()

			if err != nil {
				t.Errorf("Expected code generation to succeed, but it failed with:\n%s", string(output))
				return
			}

			outputStr := string(output)
			for _, expectedStr := range tt.contains {
				if !strings.Contains(outputStr, expectedStr) {
					t.Errorf("Expected generated code to contain '%s', but it didn't", expectedStr)
				}
			}
		})
	}
}
