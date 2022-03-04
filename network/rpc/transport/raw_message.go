package transport

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/boreq/errors"
)

const (
	MessageHeaderSize = 9

	headerFlagsMaskStream         byte = 1 << 3
	headerFlagsMaskEndOrError     byte = 1 << 2
	headerFlagsMaskBodyTypeFirst  byte = 1 << 1
	headerFlagsMaskBodyTypeSecond byte = 1 << 0
)

type Message struct {
	Header MessageHeader
	Body   []byte
}

func NewMessage(header MessageHeader, body []byte) (Message, error) {
	if len(body) != int(header.bodyLength) {
		return Message{}, errors.New("invalid body length")
	}

	return Message{
		Header: header,
		Body:   body,
	}, nil
}

type MessageHeader struct {
	flags         MessageHeaderFlags
	bodyLength    uint32
	requestNumber int32
}

// NewMessageHeaderFromBytes loads a message header from the wire format. The provided slice must be exactly
// MessageHeaderSize bytes long.
func NewMessageHeaderFromBytes(header []byte) (MessageHeader, error) {
	if l := len(header); l != MessageHeaderSize {
		return MessageHeader{}, fmt.Errorf("invalid header length %d", l)
	}

	flags, err := NewMessageHeaderFlagsFromByte(header[0])
	if err != nil {
		return MessageHeader{}, errors.Wrap(err, "could not parse header flags")
	}

	buffer := bytes.NewBuffer(header[1:])

	var bodyLength uint32
	if err = binary.Read(buffer, binary.BigEndian, &bodyLength); err != nil {
		return MessageHeader{}, errors.Wrap(err, "could not read body length")
	}

	var requestNumber int32
	if err = binary.Read(buffer, binary.BigEndian, &requestNumber); err != nil {
		return MessageHeader{}, errors.Wrap(err, "could not read request number")
	}

	return NewMessageHeader(flags, bodyLength, requestNumber)
}

func NewMessageHeader(flags MessageHeaderFlags, bodyLength uint32, requestNumber int32) (MessageHeader, error) {
	header := MessageHeader{
		flags:         flags,
		bodyLength:    bodyLength,
		requestNumber: requestNumber,
	}

	if requestNumber == 0 {
		return MessageHeader{}, errors.New("request number can not be set to zero")
	}

	if header.IsRequest() {
		if flags.BodyType != MessageBodyTypeJSON {
			return MessageHeader{}, errors.New("requests should have body type set to JSON")
		}

		if flags.EndOrError {
			return MessageHeader{}, errors.New("requests should not have the end or error bit set")
		}
	}

	return header, nil
}

func (m MessageHeader) Flags() MessageHeaderFlags {
	return m.flags
}

func (m MessageHeader) BodyLength() int {
	return int(m.bodyLength)
}

func (m MessageHeader) RequestNumber() int {
	return int(m.requestNumber)
}

func (m MessageHeader) IsRequest() bool {
	return m.requestNumber > 0
}

func (m MessageHeader) Bytes() ([]byte, error) {
	flags, err := m.Flags().Marshal()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal flags")
	}

	var buf bytes.Buffer

	buf.WriteByte(flags)

	if err = binary.Write(&buf, binary.BigEndian, m.bodyLength); err != nil {
		return nil, errors.Wrap(err, "could not read body length")
	}

	if err = binary.Write(&buf, binary.BigEndian, m.requestNumber); err != nil {
		return nil, errors.Wrap(err, "could not read request number")
	}

	return buf.Bytes(), nil
}

type MessageHeaderFlags struct {
	Stream     bool
	EndOrError bool
	BodyType   MessageBodyType
}

func NewMessageHeaderFlagsFromByte(headerFlags byte) (MessageHeaderFlags, error) {
	bodyType, err := NewMessageBodyTypeFromByte(headerFlags)
	if err != nil {
		return MessageHeaderFlags{}, errors.Wrap(err, "could not parse the message body type")
	}

	flags := MessageHeaderFlags{
		BodyType: bodyType,
	}

	if headerFlags&headerFlagsMaskStream != 0 {
		flags.Stream = true
	}

	if headerFlags&headerFlagsMaskEndOrError != 0 {
		flags.EndOrError = true
	}

	return flags, nil
}

func (f MessageHeaderFlags) Marshal() (byte, error) {
	var flags byte

	if err := f.BodyType.Marshal(&flags); err != nil {
		return 0, errors.Wrap(err, "could not marshal the body type")
	}

	if f.Stream {
		flags |= headerFlagsMaskStream
	}

	if f.EndOrError {
		flags |= headerFlagsMaskEndOrError
	}

	return flags, nil
}

type MessageBodyType struct {
	s string
}

func NewMessageBodyTypeFromByte(headerFlags byte) (MessageBodyType, error) {
	if headerFlags&headerFlagsMaskBodyTypeFirst == 0 && headerFlags&headerFlagsMaskBodyTypeSecond == 0 {
		return MessageBodyTypeBinary, nil
	}

	if headerFlags&headerFlagsMaskBodyTypeFirst == 0 && headerFlags&headerFlagsMaskBodyTypeSecond != 0 {
		return MessageBodyTypeString, nil
	}

	if headerFlags&headerFlagsMaskBodyTypeFirst != 0 && headerFlags&headerFlagsMaskBodyTypeSecond == 0 {
		return MessageBodyTypeJSON, nil
	}

	return MessageBodyType{}, errors.New("unknown message body type")
}

func (t MessageBodyType) Marshal(b *byte) error {
	switch t {
	case MessageBodyTypeBinary:
		return nil
	case MessageBodyTypeString:
		*b |= headerFlagsMaskBodyTypeSecond
		return nil
	case MessageBodyTypeJSON:
		*b |= headerFlagsMaskBodyTypeFirst
		return nil
	default:
		return fmt.Errorf("unsupported type '%s'", t.s)
	}
}

func (t MessageBodyType) IsZero() bool {
	return t == MessageBodyType{}
}

var (
	MessageBodyTypeBinary = MessageBodyType{"binary"}
	MessageBodyTypeString = MessageBodyType{"string"}
	MessageBodyTypeJSON   = MessageBodyType{"json"}
)
