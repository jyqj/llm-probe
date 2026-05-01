package stream

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"io"
)

// EventStreamDecoder decodes AWS event-stream binary format.
// Used for reading Bedrock streaming responses.
type EventStreamDecoder struct {
	r io.Reader
}

// EventStreamMessage represents a single event-stream message.
type EventStreamMessage struct {
	Headers map[string]string
	Payload []byte
}

// NewEventStreamDecoder creates a new decoder.
func NewEventStreamDecoder(r io.Reader) *EventStreamDecoder {
	return &EventStreamDecoder{r: r}
}

// Decode reads the next message from the event stream.
// Returns io.EOF when the stream ends.
func (d *EventStreamDecoder) Decode() (*EventStreamMessage, error) {
	// Read prelude: total_length(4) + headers_length(4)
	prelude := make([]byte, 8)
	if _, err := io.ReadFull(d.r, prelude); err != nil {
		return nil, err
	}

	totalLen := binary.BigEndian.Uint32(prelude[0:4])
	headersLen := binary.BigEndian.Uint32(prelude[4:8])

	// Read prelude CRC (4 bytes)
	preludeCRC := make([]byte, 4)
	if _, err := io.ReadFull(d.r, preludeCRC); err != nil {
		return nil, fmt.Errorf("read prelude CRC: %w", err)
	}

	// Verify prelude CRC
	expectedCRC := crc32.ChecksumIEEE(prelude)
	actualCRC := binary.BigEndian.Uint32(preludeCRC)
	if expectedCRC != actualCRC {
		return nil, fmt.Errorf("prelude CRC mismatch: expected %d, got %d", expectedCRC, actualCRC)
	}

	// Read headers
	headersData := make([]byte, headersLen)
	if _, err := io.ReadFull(d.r, headersData); err != nil {
		return nil, fmt.Errorf("read headers: %w", err)
	}

	headers, err := parseHeaders(headersData)
	if err != nil {
		return nil, fmt.Errorf("parse headers: %w", err)
	}

	// Calculate payload length: total - prelude(8) - prelude_crc(4) - headers - message_crc(4)
	payloadLen := int(totalLen) - 8 - 4 - int(headersLen) - 4
	if payloadLen < 0 {
		return nil, fmt.Errorf("invalid payload length: %d", payloadLen)
	}

	payload := make([]byte, payloadLen)
	if payloadLen > 0 {
		if _, err := io.ReadFull(d.r, payload); err != nil {
			return nil, fmt.Errorf("read payload: %w", err)
		}
	}

	// Read message CRC (4 bytes)
	msgCRC := make([]byte, 4)
	if _, err := io.ReadFull(d.r, msgCRC); err != nil {
		return nil, fmt.Errorf("read message CRC: %w", err)
	}

	// Verify message CRC (over prelude + preludeCRC + headers + payload)
	crcData := make([]byte, 0, 8+4+int(headersLen)+payloadLen)
	crcData = append(crcData, prelude...)
	crcData = append(crcData, preludeCRC...)
	crcData = append(crcData, headersData...)
	crcData = append(crcData, payload...)
	expectedMsgCRC := crc32.ChecksumIEEE(crcData)
	actualMsgCRC := binary.BigEndian.Uint32(msgCRC)
	if expectedMsgCRC != actualMsgCRC {
		return nil, fmt.Errorf("message CRC mismatch: expected %d, got %d", expectedMsgCRC, actualMsgCRC)
	}

	return &EventStreamMessage{
		Headers: headers,
		Payload: payload,
	}, nil
}

// parseHeaders parses the binary header format.
func parseHeaders(data []byte) (map[string]string, error) {
	headers := make(map[string]string)
	offset := 0

	for offset < len(data) {
		if offset >= len(data) {
			break
		}

		// Header name length (1 byte)
		nameLen := int(data[offset])
		offset++

		if offset+nameLen > len(data) {
			return nil, fmt.Errorf("header name overflow")
		}

		name := string(data[offset : offset+nameLen])
		offset += nameLen

		if offset >= len(data) {
			return nil, fmt.Errorf("missing header value type")
		}

		// Value type (1 byte)
		valueType := data[offset]
		offset++

		switch valueType {
		case 7: // STRING
			if offset+2 > len(data) {
				return nil, fmt.Errorf("string value length overflow")
			}
			valueLen := int(binary.BigEndian.Uint16(data[offset : offset+2]))
			offset += 2

			if offset+valueLen > len(data) {
				return nil, fmt.Errorf("string value overflow")
			}
			headers[name] = string(data[offset : offset+valueLen])
			offset += valueLen

		case 6: // BYTES
			if offset+2 > len(data) {
				return nil, fmt.Errorf("bytes value length overflow")
			}
			valueLen := int(binary.BigEndian.Uint16(data[offset : offset+2]))
			offset += 2
			offset += valueLen // skip bytes

		case 0: // BOOL_TRUE
			headers[name] = "true"
		case 1: // BOOL_FALSE
			headers[name] = "false"
		case 2: // BYTE
			offset += 1
		case 3: // SHORT (INT16)
			offset += 2
		case 4: // INT
			offset += 4
		case 5: // LONG
			offset += 8
		case 8: // TIMESTAMP
			offset += 8
		case 9: // UUID
			offset += 16
		default:
			return nil, fmt.Errorf("unknown header value type: %d", valueType)
		}
	}

	return headers, nil
}

// IsException checks if the message is an error/exception.
func (m *EventStreamMessage) IsException() bool {
	msgType := m.Headers[":message-type"]
	return msgType == "exception" || msgType == "error"
}

// EventType returns the :event-type header value.
func (m *EventStreamMessage) EventType() string {
	return m.Headers[":event-type"]
}

// ExceptionType returns the :exception-type header value.
func (m *EventStreamMessage) ExceptionType() string {
	return m.Headers[":exception-type"]
}

// PayloadJSON unmarshals the payload as JSON into the target.
func (m *EventStreamMessage) PayloadJSON(target any) error {
	if len(m.Payload) == 0 {
		return nil
	}
	return json.Unmarshal(m.Payload, target)
}
