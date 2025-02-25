package ebt

import (
	"context"
	"time"

	"github.com/boreq/errors"
	"github.com/planetary-social/scuttlego/logging"
	"github.com/planetary-social/scuttlego/service/domain/feeds/message"
	"github.com/planetary-social/scuttlego/service/domain/identity"
	"github.com/planetary-social/scuttlego/service/domain/messages"
	"github.com/planetary-social/scuttlego/service/domain/refs"
	"github.com/planetary-social/scuttlego/service/domain/replication"
)

type MessageWriter interface {
	WriteMessage(msg message.Message) error
}

type Stream interface {
	RemoteIdentity() identity.Public
	IncomingMessages(ctx context.Context) <-chan IncomingMessage
	SendNotes(notes messages.EbtReplicateNotes) error
	SendMessage(msg *message.Message) error
}

type IncomingMessage struct {
	notes *messages.EbtReplicateNotes
	msg   *message.RawMessage
	err   error
}

func NewIncomingMessageWithNotes(notes messages.EbtReplicateNotes) IncomingMessage {
	return IncomingMessage{
		notes: &notes,
	}
}

func NewIncomingMessageWithMessage(msg message.RawMessage) IncomingMessage {
	return IncomingMessage{
		msg: &msg,
	}
}

func NewIncomingMessageWithErr(err error) IncomingMessage {
	return IncomingMessage{
		err: err,
	}
}

func (i IncomingMessage) Notes() (messages.EbtReplicateNotes, bool) {
	if i.notes != nil {
		return *i.notes, true
	}
	return messages.EbtReplicateNotes{}, false
}

func (i IncomingMessage) Msg() (message.RawMessage, bool) {
	if i.msg != nil {
		return *i.msg, true
	}
	return message.RawMessage{}, false
}

func (i IncomingMessage) Err() error {
	return i.err
}

type SessionRunner struct {
	logger            logging.Logger
	rawMessageHandler replication.RawMessageHandler
	contactsStorage   replication.ContactsStorage
	streamer          MessageStreamer
}

func NewSessionRunner(
	logger logging.Logger,
	rawMessageHandler replication.RawMessageHandler,
	contactsStorage replication.ContactsStorage,
	streamer MessageStreamer,
) *SessionRunner {
	return &SessionRunner{
		logger:            logger,
		rawMessageHandler: rawMessageHandler,
		contactsStorage:   contactsStorage,
		streamer:          streamer,
	}
}

func (s *SessionRunner) HandleStream(ctx context.Context, stream Stream) error {
	rf := NewRequestedFeeds(s.streamer, stream)
	session := NewSession(ctx, stream, s.logger, s.rawMessageHandler, s.contactsStorage, rf)
	go session.SendNotesLoop()
	return session.HandleIncomingMessagesLoop()
}

type FeedRequester interface {
	Request(ctx context.Context, ref refs.Feed, seq *message.Sequence)
	Cancel(ref refs.Feed)
}

type Session struct {
	ctx    context.Context
	cancel context.CancelFunc

	stream Stream

	sentNotes            *SentNotes
	sentNotesAtLeastOnce bool
	feedRequester        FeedRequester

	logger            logging.Logger
	rawMessageHandler replication.RawMessageHandler
	contactsStorage   replication.ContactsStorage
}

func NewSession(
	ctx context.Context,
	stream Stream,
	logger logging.Logger,
	rawMessageHandler replication.RawMessageHandler,
	contactsStorage replication.ContactsStorage,
	feedRequester FeedRequester,
) *Session {
	ctx, cancel := context.WithCancel(ctx)

	return &Session{
		ctx:    ctx,
		cancel: cancel,

		stream: stream,

		sentNotes:     NewSentNotes(),
		feedRequester: feedRequester,

		logger:            logger.New("session").WithCtx(ctx),
		rawMessageHandler: rawMessageHandler,
		contactsStorage:   contactsStorage,
	}
}

func (s *Session) HandleIncomingMessagesLoop() error {
	defer s.cancel()

	for incoming := range s.stream.IncomingMessages(s.ctx) {
		if err := s.handleIncomingMessage(s.ctx, incoming); err != nil {
			return errors.Wrap(err, "error handling incoming message")
		}
	}

	return nil
}

func (s *Session) SendNotesLoop() {
	for {
		if err := s.SendNotes(); err != nil {
			s.logger.Debug().WithError(err).Message("error sending our notes")
		}

		select {
		case <-time.After(10 * time.Second):
			continue
		case <-s.ctx.Done():
			return
		}
	}
}

func (s *Session) SendNotes() error {
	contacts, err := s.contactsStorage.GetContacts(s.stream.RemoteIdentity())
	if err != nil {
		return errors.Wrap(err, "could not get the contacts")
	}

	notesToSend, err := s.sentNotes.Update(contacts)
	if err != nil {
		return errors.Wrap(err, "could not create the notes")
	}

	if notesToSend.Empty() && s.sentNotesAtLeastOnce {
		return nil
	}

	s.sentNotesAtLeastOnce = true

	s.logger.
		Trace().
		WithField("number_of_notes", len(notesToSend.Notes())).
		Message("sending notes")

	return s.stream.SendNotes(notesToSend)
}

func (s *Session) handleIncomingMessage(ctx context.Context, incoming IncomingMessage) error {
	if err := incoming.Err(); err != nil {
		return errors.Wrap(err, "error receiving messages")
	}

	notes, ok := incoming.Notes()
	if ok {
		s.logger.Trace().WithField("number_of_notes", len(notes.Notes())).Message("received notes")
		return s.handleIncomingNotes(ctx, notes)
	}

	msg, ok := incoming.Msg()
	if ok {
		if err := s.rawMessageHandler.Handle(s.stream.RemoteIdentity(), msg); err != nil {
			// todo ban this feed somehow
			s.logger.Debug().WithError(err).Message("error handling a raw message")
			return nil
		}
		return nil
	}

	return errors.New("logic error")
}

func (s *Session) handleIncomingNotes(ctx context.Context, notes messages.EbtReplicateNotes) error {
	for _, note := range notes.Notes() {
		if !note.Replicate() || !note.Receive() {
			s.feedRequester.Cancel(note.Ref())
		} else {
			seq, err := s.parseSeq(note.Sequence())
			if err != nil {
				return errors.Wrap(err, "error parsing sequence")
			}
			s.feedRequester.Request(ctx, note.Ref(), seq)
		}
	}

	return nil
}

func (s *Session) parseSeq(seq int) (*message.Sequence, error) {
	if seq <= 0 {
		return nil, nil
	}
	sequence, err := message.NewSequence(seq)
	if err != nil {
		return nil, errors.Wrap(err, "new sequence error")
	}
	return &sequence, nil
}

type StreamMessageWriter struct {
	stream Stream
}

func NewStreamMessageWriter(stream Stream) *StreamMessageWriter {
	return &StreamMessageWriter{
		stream: stream,
	}
}

func (s StreamMessageWriter) WriteMessage(msg message.Message) error {
	return s.stream.SendMessage(&msg)
}
