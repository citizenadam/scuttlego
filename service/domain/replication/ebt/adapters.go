package ebt

import (
	"context"

	"github.com/boreq/errors"
	"github.com/hashicorp/go-multierror"
	"github.com/planetary-social/scuttlego/service/domain/feeds/message"
	"github.com/planetary-social/scuttlego/service/domain/identity"
	"github.com/planetary-social/scuttlego/service/domain/messages"
	"github.com/planetary-social/scuttlego/service/domain/replication"
	"github.com/planetary-social/scuttlego/service/domain/transport/rpc"
	"github.com/planetary-social/scuttlego/service/domain/transport/rpc/mux"
	"github.com/planetary-social/scuttlego/service/domain/transport/rpc/transport"
)

type OutgoingStreamAdapter struct {
	remoteIdentity identity.Public
	stream         rpc.ResponseStream
}

func NewOutgoingStreamAdapter(remoteIdentity identity.Public, stream rpc.ResponseStream) *OutgoingStreamAdapter {
	return &OutgoingStreamAdapter{remoteIdentity: remoteIdentity, stream: stream}
}

func (r *OutgoingStreamAdapter) RemoteIdentity() identity.Public {
	return r.remoteIdentity
}

func (r *OutgoingStreamAdapter) IncomingMessages(ctx context.Context) <-chan IncomingMessage {
	ch := make(chan IncomingMessage)
	go func() {
		defer close(ch)

		for resp := range r.stream.Channel() {
			if err := r.parseErr(resp.Err); err != nil {
				select {
				case <-ctx.Done():
					return
				case ch <- NewIncomingMessageWithErr(err):
					return
				}
			}

			incomingMessage, err := parseIncomingMsg(resp.Value.Bytes())
			if err != nil {
				select {
				case <-ctx.Done():
					return
				case ch <- NewIncomingMessageWithErr(err):
					return
				}
			}

			select {
			case <-ctx.Done():
				return
			case ch <- incomingMessage:
			}
		}
	}()
	return ch
}

func (r *OutgoingStreamAdapter) parseErr(err error) error {
	if err != nil {
		if errors.Is(err, rpc.RemoteError{}) {
			return replication.ErrPeerDoesNotSupportEBT
		}
		return errors.Wrap(err, "stream returned an error")
	}
	return nil
}

func (r *OutgoingStreamAdapter) SendNotes(notes messages.EbtReplicateNotes) error {
	j, err := notes.MarshalJSON()
	if err != nil {
		return errors.Wrap(err, "json marshal error")
	}
	return r.stream.WriteMessage(j, transport.MessageBodyTypeJSON)
}

func (r *OutgoingStreamAdapter) SendMessage(msg *message.Message) error {
	return r.stream.WriteMessage(msg.Raw().Bytes(), transport.MessageBodyTypeJSON)
}

type IncomingStreamAdapter struct {
	remoteIdentity identity.Public
	stream         mux.Stream
}

func NewIncomingStreamAdapter(remoteIdentity identity.Public, stream mux.Stream) IncomingStreamAdapter {
	return IncomingStreamAdapter{remoteIdentity: remoteIdentity, stream: stream}
}

func (s IncomingStreamAdapter) RemoteIdentity() identity.Public {
	return s.remoteIdentity
}

func (s IncomingStreamAdapter) IncomingMessages(ctx context.Context) <-chan IncomingMessage {
	ch := make(chan IncomingMessage)
	go func() {
		defer close(ch)

		incomingMessages, err := s.stream.IncomingMessages()
		if err != nil {
			select {
			case <-ctx.Done():
				return
			case ch <- NewIncomingMessageWithErr(err):
				return
			}
		}

		for msg := range incomingMessages {
			incomingMessage, err := parseIncomingMsg(msg.Body)
			if err != nil {
				select {
				case <-ctx.Done():
					return
				case ch <- NewIncomingMessageWithErr(err):
					return
				}
			}

			select {
			case <-ctx.Done():
				return
			case ch <- incomingMessage:
			}
		}
	}()
	return ch
}

func (s IncomingStreamAdapter) SendNotes(notes messages.EbtReplicateNotes) error {
	j, err := notes.MarshalJSON()
	if err != nil {
		return errors.Wrap(err, "json marshal error")
	}
	return s.stream.WriteMessage(j, transport.MessageBodyTypeJSON)
}

func (s IncomingStreamAdapter) SendMessage(msg *message.Message) error {
	return s.stream.WriteMessage(msg.Raw().Bytes(), transport.MessageBodyTypeJSON)
}

func parseIncomingMsg(b []byte) (IncomingMessage, error) {
	var returnErr error

	note, err := messages.NewEbtReplicateNotesFromBytes(b)
	if err == nil {
		return NewIncomingMessageWithNotes(note), nil
	}
	returnErr = multierror.Append(returnErr, errors.Wrap(err, "could not create a new note"))

	rawMessage, err := message.NewRawMessage(b)
	if err == nil {
		return NewIncomingMessageWithMessage(rawMessage), nil
	}
	returnErr = multierror.Append(returnErr, errors.Wrap(err, "could not create a new raw message"))

	return IncomingMessage{}, returnErr
}
