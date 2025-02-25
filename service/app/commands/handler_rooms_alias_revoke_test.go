package commands_test

import (
	"context"
	"testing"
	"time"

	"github.com/planetary-social/scuttlego/internal/fixtures"
	"github.com/planetary-social/scuttlego/internal/mocks"
	"github.com/planetary-social/scuttlego/service/app/commands"
	"github.com/planetary-social/scuttlego/service/di"
	"github.com/planetary-social/scuttlego/service/domain/messages"
	"github.com/planetary-social/scuttlego/service/domain/network"
	"github.com/planetary-social/scuttlego/service/domain/transport/rpc"
	"github.com/stretchr/testify/require"
)

func TestRoomsAliasRevokeHandler_RemoteReturnsSomeData(t *testing.T) {
	c, err := di.BuildTestCommands(t)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(fixtures.TestContext(t), 5*time.Second)
	defer cancel()

	roomIdentityRef := fixtures.SomeRefIdentity()
	roomAddress := network.NewAddress(fixtures.SomeString())

	alias := fixtures.SomeAlias()

	connection := mocks.NewConnectionMock(ctx)
	connection.Mock(
		func(req *rpc.Request) []rpc.ResponseWithError {
			require.Equal(t, messages.RoomRevokeAliasProcedure.Typ(), req.Type())
			require.Equal(t, messages.RoomRevokeAliasProcedure.Name(), req.Name())
			require.Contains(t, string(req.Arguments()), alias.String())

			return []rpc.ResponseWithError{
				{
					Value: rpc.NewResponse(fixtures.SomeBytes()),
					Err:   nil,
				},
			}
		},
	)

	c.Dialer.MockPeer(roomIdentityRef.Identity(), roomAddress, connection)

	cmd, err := commands.NewRoomsAliasRevoke(
		roomIdentityRef,
		roomAddress,
		alias,
	)
	require.NoError(t, err)

	err = c.RoomsAliasRevoke.Handle(ctx, cmd)
	require.NoError(t, err)
}

func TestRoomsAliasRevokeHandler_RemoteTerminatesWithAnError(t *testing.T) {
	c, err := di.BuildTestCommands(t)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(fixtures.TestContext(t), 5*time.Second)
	defer cancel()

	roomIdentityRef := fixtures.SomeRefIdentity()
	roomAddress := network.NewAddress(fixtures.SomeString())

	alias := fixtures.SomeAlias()

	connection := mocks.NewConnectionMock(ctx)
	connection.Mock(
		func(req *rpc.Request) []rpc.ResponseWithError {
			require.Equal(t, messages.RoomRevokeAliasProcedure.Typ(), req.Type())
			require.Equal(t, messages.RoomRevokeAliasProcedure.Name(), req.Name())
			require.Contains(t, string(req.Arguments()), alias.String())

			return []rpc.ResponseWithError{
				{
					Value: nil,
					Err:   rpc.NewRemoteError(nil),
				},
			}
		},
	)

	c.Dialer.MockPeer(roomIdentityRef.Identity(), roomAddress, connection)

	cmd, err := commands.NewRoomsAliasRevoke(
		roomIdentityRef,
		roomAddress,
		alias,
	)
	require.NoError(t, err)

	err = c.RoomsAliasRevoke.Handle(ctx, cmd)
	require.EqualError(t, err, "received an error: remote returned an error")
}

func TestRoomsAliasRevokeHandler_RemoteTerminatesCleanly(t *testing.T) {
	c, err := di.BuildTestCommands(t)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(fixtures.TestContext(t), 5*time.Second)
	defer cancel()

	roomIdentityRef := fixtures.SomeRefIdentity()
	roomAddress := network.NewAddress(fixtures.SomeString())

	alias := fixtures.SomeAlias()

	connection := mocks.NewConnectionMock(ctx)
	connection.Mock(
		func(req *rpc.Request) []rpc.ResponseWithError {
			require.Equal(t, messages.RoomRevokeAliasProcedure.Typ(), req.Type())
			require.Equal(t, messages.RoomRevokeAliasProcedure.Name(), req.Name())
			require.Contains(t, string(req.Arguments()), alias.String())

			return []rpc.ResponseWithError{
				{
					Value: nil,
					Err:   rpc.ErrRemoteEnd,
				},
			}
		},
	)

	c.Dialer.MockPeer(roomIdentityRef.Identity(), roomAddress, connection)

	cmd, err := commands.NewRoomsAliasRevoke(
		roomIdentityRef,
		roomAddress,
		alias,
	)
	require.NoError(t, err)

	err = c.RoomsAliasRevoke.Handle(ctx, cmd)
	require.EqualError(t, err, "received an error: remote end")
}
