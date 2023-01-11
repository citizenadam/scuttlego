package commands_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/planetary-social/scuttlego/di"
	"github.com/planetary-social/scuttlego/fixtures"
	"github.com/planetary-social/scuttlego/service/adapters/mocks"
	"github.com/planetary-social/scuttlego/service/app/commands"
	"github.com/planetary-social/scuttlego/service/app/common"
	"github.com/planetary-social/scuttlego/service/domain/feeds/message"
	"github.com/planetary-social/scuttlego/service/domain/refs"
	"github.com/stretchr/testify/require"
	gossbrefs "go.mindeco.de/ssb-refs"
)

func TestMigrationHandlerImportDataFromGoSSB_MessageReturnedFromRepoReaderIsSaved(t *testing.T) {
	tc, err := di.BuildTestCommands(t)
	require.NoError(t, err)

	directory := fixtures.SomeString()
	receiveLogSequence := common.MustNewReceiveLogSequence(123)
	saveResumeFromSequenceFn := newSaveResumeFromSequenceFnMock()

	cmd, err := commands.NewImportDataFromGoSSB(directory, &receiveLogSequence, saveResumeFromSequenceFn.Fn)
	require.NoError(t, err)

	msgReceiveLogSequence := common.MustNewReceiveLogSequence(456)
	msg := mockGoSSBMessage(t)

	tc.GoSSBRepoReader.MockGetMessages(
		[]commands.GoSSBMessageOrError{
			{
				Value: commands.GoSSBMessage{
					ReceiveLogSequence: msgReceiveLogSequence,
					Message:            msg,
				},
				Err: nil,
			},
		},
	)

	ctx := fixtures.TestContext(t)
	result, err := tc.MigrationImportDataFromGoSSB.Handle(ctx, cmd)
	require.NoError(t, err)

	require.Equal(t,
		commands.ImportDataFromGoSSBResult{
			Successes: 1,
			Errors:    0,
		},
		result,
	)

	require.Equal(t,
		[]mocks.GoSSBRepoReaderMockGetMessagesCall{
			{
				Directory:          directory,
				ResumeFromSequence: &receiveLogSequence,
			},
		},
		tc.GoSSBRepoReader.GoSSBRepoReaderMockGetMessagesCalls,
	)

	require.Equal(
		t,
		[]mocks.FeedRepositoryMockUpdateFeedIgnoringReceiveLogCall{
			{
				Feed: refs.MustNewFeed(msg.Author().Sigil()),
				MessagesToPersist: []refs.Message{
					refs.MustNewMessage(msg.Key().String()),
				},
			},
		},
		tc.FeedRepository.UpdateFeedIgnoringReceiveLogCalls(),
	)

	require.Equal(
		t,
		[]mocks.ReceiveLogRepositoryPutUnderSpecificSequenceCall{
			{
				Id:       refs.MustNewMessage(msg.Key().String()),
				Sequence: msgReceiveLogSequence,
			},
		},
		tc.ReceiveLog.PutUnderSpecificSequenceCalls,
	)

	require.Equal(t,
		[]common.ReceiveLogSequence(nil),
		saveResumeFromSequenceFn.calls,
	)
}

func TestMigrationHandlerImportDataFromGoSSB_ErrorsWhenAppendingMessagesAreIgnored(t *testing.T) {
	tc, err := di.BuildTestCommands(t)
	require.NoError(t, err)

	directory := fixtures.SomeString()
	receiveLogSequence := fixtures.SomeReceiveLogSequence()
	saveResumeFromSequenceFn := newSaveResumeFromSequenceFnMock()

	cmd, err := commands.NewImportDataFromGoSSB(directory, &receiveLogSequence, saveResumeFromSequenceFn.Fn)
	require.NoError(t, err)

	msgReceiveLogSequence1 := common.MustNewReceiveLogSequence(1)
	id1 := fixtures.SomeRefMessage()
	msg1 := mockGoSSBMessageWithIdPreviousSequence(t, id1, nil, message.MustNewSequence(1))

	msgReceiveLogSequence2 := common.MustNewReceiveLogSequence(2)
	id2 := fixtures.SomeRefMessage()
	msg2 := mockGoSSBMessageWithIdPreviousSequence(t, id2, &id1, message.MustNewSequence(2))

	tc.GoSSBRepoReader.MockGetMessages(
		[]commands.GoSSBMessageOrError{
			{
				Value: commands.GoSSBMessage{
					ReceiveLogSequence: msgReceiveLogSequence1,
					Message:            msg1,
				},
				Err: nil,
			},
			{
				Value: commands.GoSSBMessage{
					ReceiveLogSequence: msgReceiveLogSequence2,
					Message:            msg2,
				},
				Err: nil,
			},
		},
	)

	ctx := fixtures.TestContext(t)
	result, err := tc.MigrationImportDataFromGoSSB.Handle(ctx, cmd)
	require.NoError(t, err)

	require.Equal(t,
		commands.ImportDataFromGoSSBResult{
			Successes: 1,
			Errors:    1,
		},
		result,
	)

	require.Equal(t,
		[]mocks.GoSSBRepoReaderMockGetMessagesCall{
			{
				Directory:          directory,
				ResumeFromSequence: &receiveLogSequence,
			},
		},
		tc.GoSSBRepoReader.GoSSBRepoReaderMockGetMessagesCalls,
	)

	require.Equal(
		t,
		[]mocks.FeedRepositoryMockUpdateFeedIgnoringReceiveLogCall{
			{
				Feed: refs.MustNewFeed(msg1.Author().Sigil()),
				MessagesToPersist: []refs.Message{
					refs.MustNewMessage(msg1.Key().String()),
				},
			},
			{
				Feed:              refs.MustNewFeed(msg2.Author().Sigil()),
				MessagesToPersist: nil,
			},
		},
		tc.FeedRepository.UpdateFeedIgnoringReceiveLogCalls(),
	)

	require.Equal(
		t,
		[]mocks.ReceiveLogRepositoryPutUnderSpecificSequenceCall{
			{
				Id:       refs.MustNewMessage(msg1.Key().String()),
				Sequence: msgReceiveLogSequence1,
			},
		},
		tc.ReceiveLog.PutUnderSpecificSequenceCalls,
	)

	require.Equal(t,
		[]common.ReceiveLogSequence(nil),
		saveResumeFromSequenceFn.calls,
	)
}

func TestMigrationHandlerImportDataFromGoSSB_ConflictingSequenceNumbersCauseAnErrorIfMessagesAreDifferent(t *testing.T) {
	receiveLogSequence := fixtures.SomeReceiveLogSequence()
	gossbmsg := mockGoSSBMessage(t)
	receiveLogMessage1 := someMessageWithId(refs.MustNewMessage(gossbmsg.Key().Sigil()))
	receiveLogMessage2 := someMessageWithId(fixtures.SomeRefMessage())

	testCases := []struct {
		Name              string
		ReceiveLogMessage message.Message
		ExpectedError     error
	}{
		{
			Name:              "duplicate_message_with_identical_sequence_and_identical_id",
			ReceiveLogMessage: receiveLogMessage1,
			ExpectedError:     nil,
		},
		{
			Name:              "duplicate_message_with_identical_sequence_and_different_id",
			ReceiveLogMessage: receiveLogMessage2,
			ExpectedError: fmt.Errorf(
				"error saving messages: error saving messages per feed: transaction failed: duplicate message with receive log sequence '%d', old='%s', new='%s'",
				receiveLogSequence.Int(),
				receiveLogMessage2.Id().String(),
				gossbmsg.Key().Sigil(),
			),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			tc, err := di.BuildTestCommands(t)
			require.NoError(t, err)

			directory := fixtures.SomeString()
			saveResumeFromSequenceFn := newSaveResumeFromSequenceFnMock()

			cmd, err := commands.NewImportDataFromGoSSB(directory, nil, saveResumeFromSequenceFn.Fn)
			require.NoError(t, err)

			tc.GoSSBRepoReader.MockGetMessages([]commands.GoSSBMessageOrError{
				{
					Value: commands.GoSSBMessage{
						ReceiveLogSequence: receiveLogSequence,
						Message:            gossbmsg,
					},
					Err: nil,
				},
			})

			tc.ReceiveLog.MockMessage(receiveLogSequence, testCase.ReceiveLogMessage)

			ctx := fixtures.TestContext(t)
			_, err = tc.MigrationImportDataFromGoSSB.Handle(ctx, cmd)

			require.Equal(t,
				[]common.ReceiveLogSequence{
					receiveLogSequence,
				},
				tc.ReceiveLog.GetMessageCalls,
			)

			if testCase.ExpectedError != nil {
				require.EqualError(t, err, testCase.ExpectedError.Error())

				require.Empty(
					t,
					tc.ReceiveLog.PutUnderSpecificSequenceCalls,
				)

				require.Equal(
					t,
					[]mocks.FeedRepositoryMockUpdateFeedIgnoringReceiveLogCall{
						{
							Feed: refs.MustNewFeed(gossbmsg.Author().Sigil()),
							MessagesToPersist: []refs.Message{
								refs.MustNewMessage(gossbmsg.Key().String()),
							},
						},
					},
					tc.FeedRepository.UpdateFeedIgnoringReceiveLogCalls(),
				)

				require.Empty(t,
					saveResumeFromSequenceFn.calls,
				)
			} else {
				require.NoError(t, err)

				require.Equal(
					t,
					[]mocks.ReceiveLogRepositoryPutUnderSpecificSequenceCall{
						{
							Id:       refs.MustNewMessage(gossbmsg.Key().String()),
							Sequence: receiveLogSequence,
						},
					},
					tc.ReceiveLog.PutUnderSpecificSequenceCalls,
				)

				require.Equal(
					t,
					[]mocks.FeedRepositoryMockUpdateFeedIgnoringReceiveLogCall{
						{
							Feed: refs.MustNewFeed(gossbmsg.Author().Sigil()),
							MessagesToPersist: []refs.Message{
								refs.MustNewMessage(gossbmsg.Key().String()),
							},
						},
					},
					tc.FeedRepository.UpdateFeedIgnoringReceiveLogCalls(),
				)

				require.Equal(t,
					[]common.ReceiveLogSequence{
						receiveLogSequence,
					},
					saveResumeFromSequenceFn.calls,
				)
			}
		})
	}
}

func TestMigrationHandlerImportDataFromGoSSB_SequenceIsNotSavedIfThereIsNoResume(t *testing.T) {
	tc, err := di.BuildTestCommands(t)
	require.NoError(t, err)

	directory := fixtures.SomeString()
	saveResumeFromSequenceFn := newSaveResumeFromSequenceFnMock()

	cmd, err := commands.NewImportDataFromGoSSB(directory, nil, saveResumeFromSequenceFn.Fn)
	require.NoError(t, err)

	msgReceiveLogSequence := common.MustNewReceiveLogSequence(1)
	id := fixtures.SomeRefMessage()
	msg := mockGoSSBMessageWithIdPreviousSequence(t, id, nil, message.MustNewSequence(1))

	tc.GoSSBRepoReader.MockGetMessages(
		[]commands.GoSSBMessageOrError{
			{
				Value: commands.GoSSBMessage{
					ReceiveLogSequence: msgReceiveLogSequence,
					Message:            msg,
				},
				Err: nil,
			},
		},
	)

	ctx := fixtures.TestContext(t)

	_, err = tc.MigrationImportDataFromGoSSB.Handle(ctx, cmd)
	require.NoError(t, err)

	require.Equal(t,
		[]common.ReceiveLogSequence{
			msgReceiveLogSequence,
		},
		saveResumeFromSequenceFn.calls,
	)
}

func TestMigrationHandlerImportDataFromGoSSB_SequenceIsNotSavedIfItIsNotSignificantlyHigherThanResume(t *testing.T) {
	tc, err := di.BuildTestCommands(t)
	require.NoError(t, err)

	directory := fixtures.SomeString()
	receiveLogSequence := fixtures.SomeReceiveLogSequence()
	saveResumeFromSequenceFn := newSaveResumeFromSequenceFnMock()

	cmd, err := commands.NewImportDataFromGoSSB(directory, &receiveLogSequence, saveResumeFromSequenceFn.Fn)
	require.NoError(t, err)

	msgReceiveLogSequence := common.MustNewReceiveLogSequence(1)
	id := fixtures.SomeRefMessage()
	msg := mockGoSSBMessageWithIdPreviousSequence(t, id, nil, message.MustNewSequence(1))

	tc.GoSSBRepoReader.MockGetMessages(
		[]commands.GoSSBMessageOrError{
			{
				Value: commands.GoSSBMessage{
					ReceiveLogSequence: msgReceiveLogSequence,
					Message:            msg,
				},
				Err: nil,
			},
		},
	)

	ctx := fixtures.TestContext(t)

	_, err = tc.MigrationImportDataFromGoSSB.Handle(ctx, cmd)
	require.NoError(t, err)

	require.Equal(t,
		[]common.ReceiveLogSequence(nil),
		saveResumeFromSequenceFn.calls,
	)
}

func mockGoSSBMessage(t *testing.T) gossbrefs.Message {
	key, err := gossbrefs.ParseMessageRef(fixtures.SomeRefMessage().String())
	require.NoError(t, err)

	author, err := gossbrefs.ParseFeedRef(fixtures.SomeRefIdentity().String())
	require.NoError(t, err)

	return mockMessage{
		key:    key,
		author: author,
		seq:    1,
	}
}

func mockGoSSBMessageWithIdPreviousSequence(
	t *testing.T,
	id refs.Message,
	prev *refs.Message,
	sequence message.Sequence,
) gossbrefs.Message {
	key, err := gossbrefs.ParseMessageRef(id.String())
	require.NoError(t, err)

	var previous *gossbrefs.MessageRef
	if prev != nil {
		tmp, err := gossbrefs.ParseMessageRef(prev.String())
		require.NoError(t, err)

		previous = &tmp
	}

	author, err := gossbrefs.ParseFeedRef(fixtures.SomeRefIdentity().String())
	require.NoError(t, err)

	return mockMessage{
		key:      key,
		author:   author,
		previous: previous,
		seq:      int64(sequence.Int()),
	}
}

type saveResumeFromSequenceFnMock struct {
	calls []common.ReceiveLogSequence
}

func newSaveResumeFromSequenceFnMock() *saveResumeFromSequenceFnMock {
	return &saveResumeFromSequenceFnMock{}
}

func (m *saveResumeFromSequenceFnMock) Fn(s common.ReceiveLogSequence) error {
	m.calls = append(m.calls, s)
	return nil
}

type mockMessage struct {
	key      gossbrefs.MessageRef
	author   gossbrefs.FeedRef
	previous *gossbrefs.MessageRef
	seq      int64
}

func (m mockMessage) Key() gossbrefs.MessageRef {
	return m.key
}

func (m mockMessage) Previous() *gossbrefs.MessageRef {
	return m.previous
}

func (m mockMessage) Seq() int64 {
	return m.seq
}

func (m mockMessage) Claimed() time.Time {
	return fixtures.SomeTime()
}

func (m mockMessage) Received() time.Time {
	return fixtures.SomeTime()
}

func (m mockMessage) Author() gossbrefs.FeedRef {
	return m.author
}

func (m mockMessage) ContentBytes() []byte {
	return fixtures.SomeBytes()
}

func (m mockMessage) ValueContent() *gossbrefs.Value {
	return nil
}

func (m mockMessage) ValueContentJSON() json.RawMessage {
	return fixtures.SomeRawMessage().Bytes()
}

func someMessageWithId(id refs.Message) message.Message {
	return message.MustNewMessage(
		id,
		nil,
		message.NewFirstSequence(),
		fixtures.SomeRefIdentity(),
		fixtures.SomeRefFeed(),
		fixtures.SomeTime(),
		fixtures.SomeContent(),
		fixtures.SomeRawMessage(),
	)
}
