package dynamic

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"

	"github.com/smoxy-io/proto2fixed/pkg/codecs"
	"github.com/smoxy-io/proto2fixed/pkg/generator"
)

// codec implements the codecs.Codec interface using a JSON schema
type codec struct {
	schema        generator.JSONSchema
	endian        binary.ByteOrder
	messages      map[string]*generator.JSONMessage
	messageIdSize uint32
	idToName      map[uint]string // Message ID -> name
	nameToId      map[string]uint // Message name -> ID

	// Discriminator lookup maps
	// For unions: message name -> discriminator -> field
	unionDiscriminatorToField map[string]map[uint8]*generator.JSONField
	// For oneofs: message name + "." + oneof name -> discriminator -> variant
	oneofDiscriminatorToField map[string]map[uint8]*generator.JSONField
	// Reverse maps for encoding (field name to discriminator)
	unionFieldToDiscriminator map[string]map[string]uint8
	oneofFieldToDiscriminator map[string]map[string]uint8
}

// New creates a new dynamic codec from a JSON schema
func New(schema generator.JSONSchema) (codecs.Codec, error) {
	// Version is optional - set a default if not provided
	if schema.Version == "" {
		schema.Version = "v1.0.0"
	}

	var endian binary.ByteOrder
	switch schema.Endian {
	case "little":
		endian = binary.LittleEndian
	case "big":
		endian = binary.BigEndian
	default:
		return nil, fmt.Errorf("invalid endian: %s (must be 'little' or 'big')", schema.Endian)
	}

	if len(schema.Messages) == 0 {
		return nil, fmt.Errorf("schema must contain at least one message")
	}

	// Build message ID registry
	idToName := make(map[uint]string)
	nameToId := make(map[string]uint)

	// Build discriminator lookup maps
	unionDiscriminatorToField := make(map[string]map[uint8]*generator.JSONField)
	oneofDiscriminatorToField := make(map[string]map[uint8]*generator.JSONField)
	unionFieldToDiscriminator := make(map[string]map[string]uint8)
	oneofFieldToDiscriminator := make(map[string]map[string]uint8)

	for msgName, msg := range schema.Messages {
		// Build message ID registry
		if msg.MessageId > 0 {
			idToName[msg.MessageId] = msgName
			nameToId[msgName] = msg.MessageId
		}

		// Build union discriminator maps
		if msg.HasDiscriminator && msg.Union {
			unionDiscriminatorToField[msgName] = make(map[uint8]*generator.JSONField)
			unionFieldToDiscriminator[msgName] = make(map[string]uint8)

			for _, field := range msg.Structure {
				discriminator := uint8(field.FieldNumber)
				unionDiscriminatorToField[msgName][discriminator] = field
				unionFieldToDiscriminator[msgName][field.Name] = discriminator
			}
		}

		// Build oneof discriminator maps
		for _, oneof := range msg.Oneofs {
			key := msgName + "." + oneof.Name
			oneofDiscriminatorToField[key] = make(map[uint8]*generator.JSONField)
			oneofFieldToDiscriminator[key] = make(map[string]uint8)

			for _, variant := range oneof.Variants {
				discriminator := uint8(variant.FieldNumber)
				oneofDiscriminatorToField[key][discriminator] = variant
				oneofFieldToDiscriminator[key][variant.Name] = discriminator
			}
		}
	}

	return &codec{
		schema:                    schema,
		endian:                    endian,
		messages:                  schema.Messages,
		messageIdSize:             uint32(schema.MessageIdSize),
		idToName:                  idToName,
		nameToId:                  nameToId,
		unionDiscriminatorToField: unionDiscriminatorToField,
		oneofDiscriminatorToField: oneofDiscriminatorToField,
		unionFieldToDiscriminator: unionFieldToDiscriminator,
		oneofFieldToDiscriminator: oneofFieldToDiscriminator,
	}, nil
}

func NewFromJSON(schema []byte) (codecs.Codec, error) {
	sc := generator.JSONSchema{}

	if err := json.Unmarshal(schema, &sc); err != nil {
		return nil, fmt.Errorf("failed to parse JSON schema: %w", err)
	}

	return New(sc)
}

// Encode encodes JSON data to binary format
func (c *codec) Encode(data []byte) ([]byte, error) {
	// Parse JSON input
	var input map[string]any

	if err := json.Unmarshal(data, &input); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Determine which message to encode
	if len(input) != 1 {
		return nil, fmt.Errorf("input must contain exactly one message")
	}

	var messageName string
	var messageData map[string]any

	for name, d := range input {
		messageName = name

		if m, ok := d.(map[string]any); ok {
			messageData = m
		} else {
			return nil, fmt.Errorf("message data must be an object")
		}

		break
	}

	// Get message schema
	msg, exists := c.messages[messageName]

	if !exists {
		return nil, fmt.Errorf("message %s not found in schema", messageName)
	}

	// msg must have a message ID
	if msg.MessageId == 0 {
		return nil, fmt.Errorf("cannot encode message %s. missing message ID", messageName)
	}

	// Determine total size (with message ID header if present)
	totalSize := msg.TotalSize

	if msg.MessageTotalSize != 0 {
		totalSize = msg.MessageTotalSize
	}

	// Allocate buffer
	buffer := make([]byte, totalSize)

	// Write message ID header
	offset := uint32(0)

	c.writeMessageId(buffer, msg.MessageId)
	offset = c.messageIdSize

	// Encode fields (after message ID header)
	if err := c.encodeFields(buffer[offset:], messageData, msg, messageName); err != nil {
		return nil, fmt.Errorf("failed to encode message %s: %w", messageName, err)
	}

	return buffer, nil
}

// Decode decodes binary data to JSON format
func (c *codec) Decode(data []byte) ([]byte, error) {
	var msg *generator.JSONMessage
	var bodyData []byte

	// check if messages with IDs are defined
	if c.messageIdSize == 0 || len(c.idToName) == 0 {
		return nil, fmt.Errorf("message IDs are required for decoding")
	}

	// Check if data is large enough for message ID header
	if len(data) < int(c.messageIdSize) {
		return nil, fmt.Errorf("data too short for message ID header")
	}

	messageId := c.readMessageId(data)

	msgName, mnExists := c.idToName[messageId]

	if !mnExists {
		return nil, fmt.Errorf("no message found matching message id %d", messageId)
	}

	msg = c.messages[msgName]
	bodyData = data[c.messageIdSize:]

	// Decode fields
	result := make(map[string]any)

	if err := c.decodeFields(bodyData, result, msg, msgName); err != nil {
		return nil, fmt.Errorf("failed to decode message %s: %w", msgName, err)
	}

	// Wrap in message name
	output := map[string]any{
		msgName: result,
	}

	// Marshal to JSON
	return json.Marshal(output)
}

// Schema returns the JSON schema
func (c *codec) Schema() generator.JSONSchema {
	return c.schema
}

// encodeFields encodes all fields in a message
func (c *codec) encodeFields(buffer []byte, data map[string]any, msg *generator.JSONMessage, messageName string) error {
	// Handle union discriminator
	if msg.HasDiscriminator && msg.Union {
		// Use lookup map to find active field
		fieldToDisc := c.unionFieldToDiscriminator[messageName]

		if len(data) > 1 {
			return fmt.Errorf("only one field can be set in a union message")
		}

		for fieldName := range data {
			disc, exists := fieldToDisc[fieldName]

			if !exists {
				return fmt.Errorf("unknown union field %s", fieldName)
			}

			buffer[msg.DiscriminatorOffset] = disc

			break
		}
	}

	// Encode regular fields
	for _, field := range msg.Structure {
		value, exists := data[field.Name]

		if !exists {
			// Field not present in input - leave as zero
			continue
		}

		if err := c.encodeField(buffer, value, field); err != nil {
			return fmt.Errorf("field %s: %w", field.Name, err)
		}
	}

	// Encode oneofs
	for _, oneof := range msg.Oneofs {
		oneofData, exists := data[oneof.Name]

		if !exists {
			continue
		}

		oneofMap, ok := oneofData.(map[string]any)

		if !ok {
			return fmt.Errorf("oneof %s must be an object", oneof.Name)
		}

		key := messageName + "." + oneof.Name
		fieldToDisc := c.oneofFieldToDiscriminator[key]

		// Find which variant is set and encode it
		for variantName, value := range oneofMap {
			disc, dExists := fieldToDisc[variantName]

			if !dExists {
				continue
			}

			// Get variant field from discriminator map
			variant := c.oneofDiscriminatorToField[key][disc]

			if err := c.encodeField(buffer, value, variant); err != nil {
				return fmt.Errorf("oneof %s variant %s: %w", oneof.Name, variantName, err)
			}

			// Write discriminator
			buffer[oneof.DiscriminatorOffset] = disc

			break // Only one variant should be set
		}
	}

	return nil
}

// encodeField encodes a single field
func (c *codec) encodeField(buffer []byte, value any, field *generator.JSONField) error {
	offset := int(field.Offset)

	switch field.Type {
	case "bool":
		if b, ok := value.(bool); ok {
			if b {
				buffer[offset] = 1
			} else {
				buffer[offset] = 0
			}
		}

	case "int32":
		var v int32
		switch val := value.(type) {
		case float64:
			v = int32(val)
		case int:
			v = int32(val)
		case int32:
			v = val
		}
		c.endian.PutUint32(buffer[offset:offset+4], uint32(v))

	case "uint32":
		var v uint32
		switch val := value.(type) {
		case float64:
			v = uint32(val)
		case int:
			v = uint32(val)
		case uint32:
			v = val
		}
		c.endian.PutUint32(buffer[offset:offset+4], v)

	case "int64":
		var v int64
		switch val := value.(type) {
		case float64:
			v = int64(val)
		case int:
			v = int64(val)
		case int64:
			v = val
		}
		c.endian.PutUint64(buffer[offset:offset+8], uint64(v))

	case "uint64":
		var v uint64
		switch val := value.(type) {
		case float64:
			v = uint64(val)
		case int:
			v = uint64(val)
		case uint64:
			v = val
		}
		c.endian.PutUint64(buffer[offset:offset+8], v)

	case "float", "float32":
		if f, ok := value.(float64); ok {
			bits := math.Float32bits(float32(f))
			c.endian.PutUint32(buffer[offset:offset+4], bits)
		}

	case "double", "float64":
		if f, ok := value.(float64); ok {
			bits := math.Float64bits(f)
			c.endian.PutUint64(buffer[offset:offset+8], bits)
		}

	case "string":
		if s, ok := value.(string); ok {
			size := int(field.Size)
			copy(buffer[offset:offset+size], []byte(s))
			// Null terminate if there's room
			if len(s) < size {
				buffer[offset+len(s)] = 0
			}
		}

	case "bytes":
		if s, ok := value.(string); ok {
			// Assume base64 or hex encoded
			// For now, treat as string
			size := int(field.Size)
			copy(buffer[offset:offset+size], []byte(s))
		}

	case "enum":
		var v int32
		switch val := value.(type) {
		case float64:
			v = int32(val)
		case int:
			v = int32(val)
		case int32:
			v = val
		case string:
			// Look up enum value by name
			return fmt.Errorf("enum encoding by name not yet implemented")
		}
		// Encode based on enum size
		switch field.Size {
		case 1:
			buffer[offset] = byte(v)
		case 2:
			c.endian.PutUint16(buffer[offset:offset+2], uint16(v))
		case 4:
			c.endian.PutUint32(buffer[offset:offset+4], uint32(v))
		}

	case "message":
		if m, ok := value.(map[string]any); ok {
			// For nested messages, the structure is inline in the field
			// We need to encode the nested fields directly
			nestedBuffer := buffer[offset : offset+int(field.Size)]
			if err := c.encodeNestedMessage(nestedBuffer, m, field.Structure); err != nil {
				return fmt.Errorf("nested message: %w", err)
			}
		}

	default:
		return fmt.Errorf("unsupported field type: %s", field.Type)
	}

	return nil
}

// encodeNestedMessage encodes a nested message from its structure definition
func (c *codec) encodeNestedMessage(buffer []byte, data map[string]any, structure []*generator.JSONField) error {
	for _, field := range structure {
		value, exists := data[field.Name]

		if !exists {
			continue
		}

		if err := c.encodeField(buffer, value, field); err != nil {
			return fmt.Errorf("field %s: %w", field.Name, err)
		}
	}

	return nil
}

// decodeFields decodes all fields from binary data
func (c *codec) decodeFields(data []byte, result map[string]any, msg *generator.JSONMessage, messageName string) error {
	// Handle union discriminator
	if msg.HasDiscriminator && msg.Union {
		discriminator := data[msg.DiscriminatorOffset]

		if discriminator == 0 {
			return nil // No field set
		}

		// Use lookup map for O(1) field lookup
		discToField := c.unionDiscriminatorToField[messageName]

		field, exists := discToField[discriminator]

		if !exists {
			return fmt.Errorf("msg %s: unknown discriminator value: %d", messageName, discriminator)
		}

		value, err := c.decodeField(data, field)

		if err != nil {
			return fmt.Errorf("field %s: %w", field.Name, err)
		}

		result[field.Name] = value

		return nil // Only decode active field
	}

	// Decode regular fields
	for _, field := range msg.Structure {
		value, err := c.decodeField(data, field)

		if err != nil {
			return fmt.Errorf("field %s: %w", field.Name, err)
		}

		result[field.Name] = value
	}

	// Decode oneofs
	for _, oneof := range msg.Oneofs {
		discriminator := data[oneof.DiscriminatorOffset]

		if discriminator == 0 {
			continue // oneof not set
		}

		// Use lookup map for O(1) variant lookup
		key := messageName + "." + oneof.Name
		discToField := c.oneofDiscriminatorToField[key]

		variant, exists := discToField[discriminator]

		if !exists {
			return fmt.Errorf("oneof %s: unknown discriminator value %d", oneof.Name, discriminator)
		}

		value, vErr := c.decodeField(data, variant)

		if vErr != nil {
			return fmt.Errorf("oneof %s variant %s: %w", oneof.Name, variant.Name, vErr)
		}

		result[oneof.Name] = map[string]any{variant.Name: value}
	}

	return nil
}

// decodeField decodes a single field
func (c *codec) decodeField(data []byte, field *generator.JSONField) (any, error) {
	offset := int(field.Offset)

	switch field.Type {
	case "bool":
		return data[offset] != 0, nil

	case "int32":
		v := c.endian.Uint32(data[offset : offset+4])
		return int32(v), nil

	case "uint32":
		return c.endian.Uint32(data[offset : offset+4]), nil

	case "int64":
		v := c.endian.Uint64(data[offset : offset+8])
		return int64(v), nil

	case "uint64":
		return c.endian.Uint64(data[offset : offset+8]), nil

	case "float", "float32":
		bits := c.endian.Uint32(data[offset : offset+4])
		return math.Float32frombits(bits), nil

	case "double", "float64":
		bits := c.endian.Uint64(data[offset : offset+8])
		return math.Float64frombits(bits), nil

	case "string":
		size := int(field.Size)
		bytes := data[offset : offset+size]
		// Find null terminator
		end := 0
		for i, b := range bytes {
			if b == 0 {
				end = i
				break
			}
		}
		if end > 0 {
			return string(bytes[:end]), nil
		}
		return string(bytes), nil

	case "bytes":
		size := int(field.Size)
		return data[offset : offset+size], nil

	case "enum":
		var v int32
		switch field.Size {
		case 1:
			v = int32(data[offset])
		case 2:
			v = int32(c.endian.Uint16(data[offset : offset+2]))
		case 4:
			v = int32(c.endian.Uint32(data[offset : offset+4]))
		}
		return v, nil

	case "message":
		// Decode nested message from inline structure
		nestedData := data[offset : offset+int(field.Size)]
		nestedResult := make(map[string]any)

		if err := c.decodeNestedMessage(nestedData, nestedResult, field.Structure); err != nil {
			return nil, fmt.Errorf("nested message: %w", err)
		}
		return nestedResult, nil

	default:
		return nil, fmt.Errorf("unsupported field type: %s", field.Type)
	}
}

// decodeNestedMessage decodes a nested message from its structure definition
func (c *codec) decodeNestedMessage(data []byte, result map[string]any, structure []*generator.JSONField) error {
	for _, field := range structure {
		value, vErr := c.decodeField(data, field)

		if vErr != nil {
			return fmt.Errorf("field %s: %w", field.Name, vErr)
		}

		result[field.Name] = value
	}

	return nil
}

// writeMessageId writes a message ID to the buffer based on messageIdSize
func (c *codec) writeMessageId(buffer []byte, messageId uint) {
	switch c.messageIdSize {
	case 1:
		buffer[0] = byte(messageId)
	case 2:
		c.endian.PutUint16(buffer[0:2], uint16(messageId))
	case 4:
		c.endian.PutUint32(buffer[0:4], uint32(messageId))
	case 8:
		c.endian.PutUint64(buffer[0:8], uint64(messageId))
	}
}

// readMessageId reads a message ID from the buffer based on messageIdSize
func (c *codec) readMessageId(buffer []byte) uint {
	switch c.messageIdSize {
	case 1:
		return uint(buffer[0])
	case 2:
		return uint(c.endian.Uint16(buffer[0:2]))
	case 4:
		return uint(c.endian.Uint32(buffer[0:4]))
	case 8:
		return uint(c.endian.Uint64(buffer[0:8]))
	default:
		return 0
	}
}
