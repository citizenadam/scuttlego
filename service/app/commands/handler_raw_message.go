package commands

import (
	"github.com/boreq/errors"
	"github.com/planetary-social/go-ssb/logging"
	"github.com/planetary-social/go-ssb/service/domain/feeds/message"
)

type RawMessageIdentifier interface {
	IdentifyRawMessage(raw message.RawMessage) (message.Message, error)
}

type RawMessageHandler struct {
	identifier RawMessageIdentifier
	buffer     *MessageBuffer
	logger     logging.Logger
}

func NewRawMessageHandler(
	identifier RawMessageIdentifier,
	buffer *MessageBuffer,
	logger logging.Logger,
) *RawMessageHandler {
	return &RawMessageHandler{
		identifier: identifier,
		buffer:     buffer,
		logger:     logger.New("raw_message_handler"),
	}
}

func (h *RawMessageHandler) Handle(rawMsg message.RawMessage) error {
	msg, err := h.identifier.IdentifyRawMessage(rawMsg)
	if err != nil {
		return errors.Wrap(err, "failed to identify the raw message")
	}

	if err := h.buffer.Handle(msg); err != nil {
		return errors.Wrap(err, "failed to put the message in the buffer")
	}

	return nil
}
