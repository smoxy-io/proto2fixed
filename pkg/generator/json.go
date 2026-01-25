package generator

import (
	"encoding/json"
	"fmt"

	"github.com/smoxy-io/proto2fixed/pkg/analyzer"
	"github.com/smoxy-io/proto2fixed/pkg/parser"
)

const (
	LangJSON Language = "json"
)

// JSONGenerator generates JSON schema for fixed binary layouts
type JSONGenerator struct{}

// NewJSONGenerator creates a new JSON generator
func NewJSONGenerator() *JSONGenerator {
	return &JSONGenerator{}
}

// JSONSchema represents the JSON output format
type JSONSchema struct {
	Protocol      string                  `json:"protocol"`
	Version       string                  `json:"version"`
	Endian        string                  `json:"endian"`
	MessageIdSize uint                    `json:"messageIdSize,omitempty"`
	MessageHeader *JSONMessageHeader      `json:"messageHeader,omitempty"`
	Messages      map[string]*JSONMessage `json:"messages"`
	Enums         map[string]*JSONEnum    `json:"enums,omitempty"`
}

// JSONMessageHeader describes the message ID header structure
type JSONMessageHeader struct {
	Description string            `json:"description"`
	Size        uint              `json:"size"`
	Structure   []JSONHeaderField `json:"structure"`
}

// JSONHeaderField represents a field in the message header
type JSONHeaderField struct {
	Name   string `json:"name"`
	Offset uint   `json:"offset"`
	Size   uint   `json:"size"`
	Type   string `json:"type"`
}

// JSONMessage represents a message in JSON format
type JSONMessage struct {
	TotalSize           uint         `json:"totalSize"`
	Union               bool         `json:"union,omitempty"`
	MessageId           uint         `json:"messageId,omitempty"`
	MessageTotalSize    uint         `json:"messageTotalSize,omitempty"`
	Structure           []*JSONField `json:"structure"`
	Oneofs              []*JSONOneof `json:"oneofs,omitempty"`
	HasDiscriminator    bool         `json:"hasDiscriminator,omitempty"`
	DiscriminatorOffset uint         `json:"discriminatorOffset,omitempty"`
	DiscriminatorSize   uint         `json:"discriminatorSize,omitempty"`
}

// JSONOneof represents a oneof group in JSON format
type JSONOneof struct {
	Name                string       `json:"name"`
	Offset              uint         `json:"offset"`
	Size                uint         `json:"size"`
	Variants            []*JSONField `json:"variants"`
	DiscriminatorOffset uint         `json:"discriminatorOffset"`
	DiscriminatorSize   uint         `json:"discriminatorSize"`
}

// JSONField represents a field in JSON format
type JSONField struct {
	Name        string       `json:"name"`
	FieldNumber int          `json:"fieldNumber"`
	Offset      uint         `json:"offset"`
	Type        string       `json:"type"`
	Size        uint         `json:"size"`
	Count       uint         `json:"count,omitempty"`       // For arrays
	ElementSize uint         `json:"elementSize,omitempty"` // For arrays
	Encoding    string       `json:"encoding,omitempty"`    // For strings
	Structure   []*JSONField `json:"structure,omitempty"`   // For nested messages
}

// JSONEnum represents an enum in JSON format
type JSONEnum struct {
	Size   uint           `json:"size"`
	Values map[string]int `json:"values"`
}

// Generate generates JSON schema output
func (g *JSONGenerator) Generate(schema *parser.Schema, layouts map[string]*analyzer.MessageLayout) (string, error) {
	output := &JSONSchema{
		Protocol: "fixed-binary",
		Version:  schema.Version,
		Endian:   schema.Endian,
		Messages: make(map[string]*JSONMessage),
		Enums:    make(map[string]*JSONEnum),
	}

	// Check if any message has an ID and populate message header
	hasMessageIds := false
	for _, layout := range layouts {
		if layout.HasMessageId {
			hasMessageIds = true
			break
		}
	}

	if hasMessageIds {
		output.MessageIdSize = uint(schema.MessageIdSize)
		output.MessageHeader = &JSONMessageHeader{
			Description: "All messages with messageId are prefixed with this header",
			Size:        uint(schema.MessageIdSize),
			Structure: []JSONHeaderField{
				{
					Name:   "messageId",
					Offset: 0,
					Size:   uint(schema.MessageIdSize),
					Type:   fmt.Sprintf("uint%d", schema.MessageIdSize*8),
				},
			},
		}
	}

	// Generate enums
	for _, enum := range schema.Enums {
		values := make(map[string]int)
		for _, val := range enum.Values {
			values[val.Name] = int(val.Number)
		}
		output.Enums[enum.Name] = &JSONEnum{
			Size:   uint(enum.Size),
			Values: values,
		}
	}

	// Generate messages
	for _, msg := range schema.Messages {
		layout, exists := layouts[msg.Name]
		if !exists {
			return "", fmt.Errorf("layout not found for message: %s", msg.Name)
		}

		jsonMsg := &JSONMessage{
			TotalSize: uint(layout.TotalSize),
			Union:     msg.Union,
			Structure: make([]*JSONField, 0),
			Oneofs:    make([]*JSONOneof, 0),
		}

		// Add message ID fields if present
		if layout.HasMessageId {
			jsonMsg.MessageId = uint(layout.MessageId)
			jsonMsg.MessageTotalSize = uint(layout.MessageTotalSize)
		}

		// Add discriminator info for unions
		if layout.HasDiscriminator {
			jsonMsg.HasDiscriminator = true
			jsonMsg.DiscriminatorOffset = uint(layout.DiscriminatorOffset)
			jsonMsg.DiscriminatorSize = uint(layout.DiscriminatorSize)
		}

		for _, fieldLayout := range layout.Fields {
			jsonField := g.generateField(fieldLayout, layouts)
			jsonMsg.Structure = append(jsonMsg.Structure, jsonField)
		}

		for _, oneofLayout := range layout.Oneofs {
			jsonOneof := &JSONOneof{
				Name:                oneofLayout.Oneof.Name,
				Offset:              uint(oneofLayout.Offset),
				Size:                uint(oneofLayout.Size),
				Variants:            make([]*JSONField, 0),
				DiscriminatorOffset: uint(oneofLayout.DiscriminatorOffset),
				DiscriminatorSize:   uint(oneofLayout.DiscriminatorSize),
			}

			for _, variantLayout := range oneofLayout.Fields {
				jsonField := g.generateField(variantLayout, layouts)
				jsonOneof.Variants = append(jsonOneof.Variants, jsonField)
			}

			jsonMsg.Oneofs = append(jsonMsg.Oneofs, jsonOneof)
		}

		output.Messages[msg.Name] = jsonMsg
	}

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return string(data), nil
}

func (g *JSONGenerator) generateField(fieldLayout *analyzer.FieldLayout, layouts map[string]*analyzer.MessageLayout) *JSONField {
	field := fieldLayout.Field

	jsonField := &JSONField{
		Name:        field.Name,
		FieldNumber: int(field.Number),
		Offset:      uint(fieldLayout.Offset),
		Type:        getJSONTypeName(field),
		Size:        uint(fieldLayout.Size),
	}

	// Handle arrays
	if field.Repeated {
		jsonField.Count = uint(field.ArraySize)
		jsonField.ElementSize = uint(fieldLayout.Size / field.ArraySize)
		if field.Type == parser.TypeMessage && field.MessageType != nil {
			// Add nested structure for array elements
			nestedLayout, exists := layouts[field.MessageType.Name]
			if exists {
				jsonField.Structure = make([]*JSONField, 0)
				for _, nestedFieldLayout := range nestedLayout.Fields {
					nestedJsonField := g.generateField(nestedFieldLayout, layouts)
					jsonField.Structure = append(jsonField.Structure, nestedJsonField)
				}
			}
		}
	} else if field.Type == parser.TypeString {
		jsonField.Encoding = "null-terminated"
	} else if field.Type == parser.TypeMessage && field.MessageType != nil {
		// Add nested structure
		nestedLayout, exists := layouts[field.MessageType.Name]
		if exists {
			jsonField.Structure = make([]*JSONField, 0)
			for _, nestedFieldLayout := range nestedLayout.Fields {
				nestedJsonField := g.generateField(nestedFieldLayout, layouts)
				jsonField.Structure = append(jsonField.Structure, nestedJsonField)
			}
		}
	}

	return jsonField
}
