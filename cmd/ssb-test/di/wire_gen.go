// Code generated by Wire. DO NOT EDIT.

//go:generate go run github.com/google/wire/cmd/wire
//go:build !wireinject
// +build !wireinject

package di

import (
	"context"
	"path"
	"time"

	"github.com/boreq/errors"
	"github.com/google/wire"
	"github.com/planetary-social/go-ssb/fixtures"
	"github.com/planetary-social/go-ssb/logging"
	"github.com/planetary-social/go-ssb/service/adapters/bolt"
	"github.com/planetary-social/go-ssb/service/adapters/mocks"
	"github.com/planetary-social/go-ssb/service/adapters/pubsub"
	"github.com/planetary-social/go-ssb/service/app"
	"github.com/planetary-social/go-ssb/service/app/commands"
	"github.com/planetary-social/go-ssb/service/app/queries"
	"github.com/planetary-social/go-ssb/service/domain"
	"github.com/planetary-social/go-ssb/service/domain/feeds"
	"github.com/planetary-social/go-ssb/service/domain/feeds/content/transport"
	"github.com/planetary-social/go-ssb/service/domain/feeds/formats"
	"github.com/planetary-social/go-ssb/service/domain/graph"
	"github.com/planetary-social/go-ssb/service/domain/identity"
	"github.com/planetary-social/go-ssb/service/domain/network"
	"github.com/planetary-social/go-ssb/service/domain/network/local"
	"github.com/planetary-social/go-ssb/service/domain/replication"
	transport2 "github.com/planetary-social/go-ssb/service/domain/transport"
	"github.com/planetary-social/go-ssb/service/domain/transport/boxstream"
	rpc2 "github.com/planetary-social/go-ssb/service/domain/transport/rpc"
	network2 "github.com/planetary-social/go-ssb/service/ports/network"
	pubsub2 "github.com/planetary-social/go-ssb/service/ports/pubsub"
	"github.com/planetary-social/go-ssb/service/ports/rpc"
	"go.etcd.io/bbolt"
)

// Injectors from wire.go:

func BuildTxAdaptersForTest(tx *bbolt.Tx) (TxTestAdapters, error) {
	messageContentMappings := transport.DefaultMappings()
	logger := fixtures.SomeLogger()
	marshaler, err := transport.NewMarshaler(messageContentMappings, logger)
	if err != nil {
		return TxTestAdapters{}, err
	}
	messageHMAC := formats.NewDefaultMessageHMAC()
	scuttlebutt := formats.NewScuttlebutt(marshaler, messageHMAC)
	v := newFormats(scuttlebutt)
	rawMessageIdentifier := formats.NewRawMessageIdentifier(v)
	messageRepository := bolt.NewMessageRepository(tx, rawMessageIdentifier)
	private, err := identity.NewPrivate()
	if err != nil {
		return TxTestAdapters{}, err
	}
	public := privateIdentityToPublicIdentity(private)
	graphHops := _wireHopsValue
	socialGraphRepository := bolt.NewSocialGraphRepository(tx, public, graphHops)
	receiveLogRepository := bolt.NewReceiveLogRepository(tx, messageRepository)
	pubRepository := bolt.NewPubRepository(tx)
	feedRepository := bolt.NewFeedRepository(tx, socialGraphRepository, receiveLogRepository, messageRepository, pubRepository, scuttlebutt)
	txTestAdapters := TxTestAdapters{
		MessageRepository: messageRepository,
		FeedRepository:    feedRepository,
		ReceiveLog:        receiveLogRepository,
	}
	return txTestAdapters, nil
}

var (
	_wireHopsValue = hops
)

func BuildAdaptersForTest(db *bbolt.DB) (TestAdapters, error) {
	private, err := identity.NewPrivate()
	if err != nil {
		return TestAdapters{}, err
	}
	public := privateIdentityToPublicIdentity(private)
	logger := fixtures.SomeLogger()
	messageHMAC := formats.NewDefaultMessageHMAC()
	txRepositoriesFactory := newTxRepositoriesFactory(public, logger, messageHMAC)
	readMessageRepository := bolt.NewReadMessageRepository(db, txRepositoriesFactory)
	readFeedRepository := bolt.NewReadFeedRepository(db, txRepositoriesFactory)
	readReceiveLogRepository := bolt.NewReadReceiveLogRepository(db, txRepositoriesFactory)
	testAdapters := TestAdapters{
		MessageRepository: readMessageRepository,
		FeedRepository:    readFeedRepository,
		ReceiveLog:        readReceiveLogRepository,
	}
	return testAdapters, nil
}

func BuildApplicationForTests() (TestApplication, error) {
	feedRepositoryMock := mocks.NewFeedRepositoryMock()
	messagePubSubMock := mocks.NewMessagePubSubMock()
	createHistoryStreamHandler := queries.NewCreateHistoryStreamHandler(feedRepositoryMock, messagePubSubMock)
	receiveLogRepositoryMock := mocks.NewReceiveLogRepositoryMock()
	receiveLogHandler := queries.NewReceiveLogHandler(receiveLogRepositoryMock)
	messageRepositoryMock := mocks.NewMessageRepositoryMock()
	peerManagerMock := mocks.NewPeerManagerMock()
	statusHandler := queries.NewStatusHandler(messageRepositoryMock, feedRepositoryMock, peerManagerMock)
	private, err := identity.NewPrivate()
	if err != nil {
		return TestApplication{}, err
	}
	public := privateIdentityToPublicIdentity(private)
	publishedMessagesHandler, err := queries.NewPublishedMessagesHandler(feedRepositoryMock, public)
	if err != nil {
		return TestApplication{}, err
	}
	appQueries := app.Queries{
		CreateHistoryStream: createHistoryStreamHandler,
		ReceiveLog:          receiveLogHandler,
		Status:              statusHandler,
		PublishedMessages:   publishedMessagesHandler,
	}
	testApplication := TestApplication{
		Queries:           appQueries,
		FeedRepository:    feedRepositoryMock,
		MessagePubSub:     messagePubSubMock,
		MessageRepository: messageRepositoryMock,
		PeerManager:       peerManagerMock,
	}
	return testApplication, nil
}

func BuildTransactableAdapters(tx *bbolt.Tx, public identity.Public, logger logging.Logger, config Config) (commands.Adapters, error) {
	graphHops := _wireGraphHopsValue
	socialGraphRepository := bolt.NewSocialGraphRepository(tx, public, graphHops)
	messageContentMappings := transport.DefaultMappings()
	marshaler, err := transport.NewMarshaler(messageContentMappings, logger)
	if err != nil {
		return commands.Adapters{}, err
	}
	messageHMAC := extractMessageHMACFromConfig(config)
	scuttlebutt := formats.NewScuttlebutt(marshaler, messageHMAC)
	v := newFormats(scuttlebutt)
	rawMessageIdentifier := formats.NewRawMessageIdentifier(v)
	messageRepository := bolt.NewMessageRepository(tx, rawMessageIdentifier)
	receiveLogRepository := bolt.NewReceiveLogRepository(tx, messageRepository)
	pubRepository := bolt.NewPubRepository(tx)
	feedRepository := bolt.NewFeedRepository(tx, socialGraphRepository, receiveLogRepository, messageRepository, pubRepository, scuttlebutt)
	adapters := commands.Adapters{
		Feed:        feedRepository,
		SocialGraph: socialGraphRepository,
	}
	return adapters, nil
}

var (
	_wireGraphHopsValue = hops
)

func BuildTxRepositories(tx *bbolt.Tx, public identity.Public, logger logging.Logger, messageHMAC formats.MessageHMAC) (bolt.TxRepositories, error) {
	graphHops := _wireHopsValue2
	socialGraphRepository := bolt.NewSocialGraphRepository(tx, public, graphHops)
	messageContentMappings := transport.DefaultMappings()
	marshaler, err := transport.NewMarshaler(messageContentMappings, logger)
	if err != nil {
		return bolt.TxRepositories{}, err
	}
	scuttlebutt := formats.NewScuttlebutt(marshaler, messageHMAC)
	v := newFormats(scuttlebutt)
	rawMessageIdentifier := formats.NewRawMessageIdentifier(v)
	messageRepository := bolt.NewMessageRepository(tx, rawMessageIdentifier)
	receiveLogRepository := bolt.NewReceiveLogRepository(tx, messageRepository)
	pubRepository := bolt.NewPubRepository(tx)
	feedRepository := bolt.NewFeedRepository(tx, socialGraphRepository, receiveLogRepository, messageRepository, pubRepository, scuttlebutt)
	txRepositories := bolt.TxRepositories{
		Feed:       feedRepository,
		Graph:      socialGraphRepository,
		ReceiveLog: receiveLogRepository,
		Message:    messageRepository,
	}
	return txRepositories, nil
}

var (
	_wireHopsValue2 = hops
)

// BuildService creates a new service which uses the provided context as a long-term context used as a base context for
// e.g. established connections.
func BuildService(contextContext context.Context, private identity.Private, config Config) (Service, error) {
	networkKey := extractNetworkKeyFromConfig(config)
	handshaker, err := boxstream.NewHandshaker(private, networkKey)
	if err != nil {
		return Service{}, err
	}
	requestPubSub := pubsub.NewRequestPubSub()
	logger := extractLoggerFromConfig(config)
	peerInitializer := transport2.NewPeerInitializer(handshaker, requestPubSub, logger)
	dialer, err := network.NewDialer(peerInitializer, logger)
	if err != nil {
		return Service{}, err
	}
	db, err := newBolt(config)
	if err != nil {
		return Service{}, err
	}
	public := privateIdentityToPublicIdentity(private)
	adaptersFactory := newAdaptersFactory(config, public, logger)
	transactionProvider := bolt.NewTransactionProvider(db, adaptersFactory)
	messageContentMappings := transport.DefaultMappings()
	marshaler, err := transport.NewMarshaler(messageContentMappings, logger)
	if err != nil {
		return Service{}, err
	}
	redeemInviteHandler := commands.NewRedeemInviteHandler(dialer, transactionProvider, networkKey, private, requestPubSub, marshaler, logger)
	followHandler := commands.NewFollowHandler(transactionProvider, private, marshaler, logger)
	publishRawHandler := commands.NewPublishRawHandler(transactionProvider, private, logger)
	peerManagerConfig := extractPeerManagerConfigFromConfig(config)
	messageHMAC := extractMessageHMACFromConfig(config)
	txRepositoriesFactory := newTxRepositoriesFactory(public, logger, messageHMAC)
	boltContactsRepository := bolt.NewBoltContactsRepository(db, txRepositoriesFactory)
	manager := replication.NewManager(logger, boltContactsRepository)
	scuttlebutt := formats.NewScuttlebutt(marshaler, messageHMAC)
	v := newFormats(scuttlebutt)
	rawMessageIdentifier := formats.NewRawMessageIdentifier(v)
	messageBuffer := commands.NewMessageBuffer(transactionProvider, logger)
	rawMessageHandler := commands.NewRawMessageHandler(rawMessageIdentifier, messageBuffer, logger)
	gossipReplicator, err := replication.NewGossipReplicator(manager, rawMessageHandler, logger)
	if err != nil {
		return Service{}, err
	}
	peerManager := domain.NewPeerManager(contextContext, peerManagerConfig, gossipReplicator, dialer, logger)
	connectHandler := commands.NewConnectHandler(peerManager, logger)
	establishNewConnectionsHandler := commands.NewEstablishNewConnectionsHandler(peerManager)
	acceptNewPeerHandler := commands.NewAcceptNewPeerHandler(peerManager)
	processNewLocalDiscoveryHandler := commands.NewProcessNewLocalDiscoveryHandler(peerManager)
	appCommands := app.Commands{
		RedeemInvite:             redeemInviteHandler,
		Follow:                   followHandler,
		PublishRaw:               publishRawHandler,
		Connect:                  connectHandler,
		EstablishNewConnections:  establishNewConnectionsHandler,
		AcceptNewPeer:            acceptNewPeerHandler,
		ProcessNewLocalDiscovery: processNewLocalDiscoveryHandler,
	}
	readFeedRepository := bolt.NewReadFeedRepository(db, txRepositoriesFactory)
	messagePubSub := pubsub.NewMessagePubSub()
	createHistoryStreamHandler := queries.NewCreateHistoryStreamHandler(readFeedRepository, messagePubSub)
	readReceiveLogRepository := bolt.NewReadReceiveLogRepository(db, txRepositoriesFactory)
	receiveLogHandler := queries.NewReceiveLogHandler(readReceiveLogRepository)
	readMessageRepository := bolt.NewReadMessageRepository(db, txRepositoriesFactory)
	statusHandler := queries.NewStatusHandler(readMessageRepository, readFeedRepository, peerManager)
	publishedMessagesHandler, err := queries.NewPublishedMessagesHandler(readFeedRepository, public)
	if err != nil {
		return Service{}, err
	}
	appQueries := app.Queries{
		CreateHistoryStream: createHistoryStreamHandler,
		ReceiveLog:          receiveLogHandler,
		Status:              statusHandler,
		PublishedMessages:   publishedMessagesHandler,
	}
	application := app.Application{
		Commands: appCommands,
		Queries:  appQueries,
	}
	listener, err := newListener(peerInitializer, application, config, logger)
	if err != nil {
		return Service{}, err
	}
	discoverer, err := local.NewDiscoverer(public, logger)
	if err != nil {
		return Service{}, err
	}
	networkDiscoverer := network2.NewDiscoverer(discoverer, application, logger)
	connectionEstablisher := network2.NewConnectionEstablisher(application, logger)
	handlerCreateHistoryStream := rpc.NewHandlerCreateHistoryStream(createHistoryStreamHandler)
	handlerBlobsGet := rpc.NewHandlerBlobsGet()
	v2 := rpc.NewMuxHandlers(handlerCreateHistoryStream, handlerBlobsGet)
	mux, err := rpc.NewMux(logger, v2)
	if err != nil {
		return Service{}, err
	}
	pubSub := pubsub2.NewPubSub(requestPubSub, mux)
	advertiser, err := newAdvertiser(public, config)
	if err != nil {
		return Service{}, err
	}
	service := NewService(application, listener, networkDiscoverer, connectionEstablisher, pubSub, advertiser, messageBuffer)
	return service, nil
}

// wire.go:

var replicatorSet = wire.NewSet(replication.NewManager, wire.Bind(new(replication.ReplicationManager), new(*replication.Manager)), replication.NewGossipReplicator, wire.Bind(new(domain.Replicator), new(*replication.GossipReplicator)))

var formatsSet = wire.NewSet(
	newFormats, formats.NewScuttlebutt, transport.NewMarshaler, wire.Bind(new(formats.Marshaler), new(*transport.Marshaler)), transport.DefaultMappings, formats.NewRawMessageIdentifier, wire.Bind(new(commands.RawMessageIdentifier), new(*formats.RawMessageIdentifier)), wire.Bind(new(bolt.RawMessageIdentifier), new(*formats.RawMessageIdentifier)),
)

var portsSet = wire.NewSet(rpc.NewMux, rpc.NewMuxHandlers, rpc.NewHandlerBlobsGet, rpc.NewHandlerCreateHistoryStream, pubsub2.NewPubSub, local.NewDiscoverer, network2.NewDiscoverer, network2.NewConnectionEstablisher)

var requestPubSubSet = wire.NewSet(pubsub.NewRequestPubSub, wire.Bind(new(rpc2.RequestHandler), new(*pubsub.RequestPubSub)))

var messagePubSubSet = wire.NewSet(pubsub.NewMessagePubSub, wire.Bind(new(queries.MessageSubscriber), new(*pubsub.MessagePubSub)))

var hops = graph.MustNewHops(3)

type TxTestAdapters struct {
	MessageRepository *bolt.MessageRepository
	FeedRepository    *bolt.FeedRepository
	ReceiveLog        *bolt.ReceiveLogRepository
}

type TestAdapters struct {
	MessageRepository *bolt.ReadMessageRepository
	FeedRepository    *bolt.ReadFeedRepository
	ReceiveLog        *bolt.ReadReceiveLogRepository
}

type TestApplication struct {
	Queries app.Queries

	FeedRepository    *mocks.FeedRepositoryMock
	MessagePubSub     *mocks.MessagePubSubMock
	MessageRepository *mocks.MessageRepositoryMock
	PeerManager       *mocks.PeerManagerMock
}

func newAdvertiser(l identity.Public, config Config) (*local.Advertiser, error) {
	return local.NewAdvertiser(l, config.ListenAddress)
}

func newListener(
	initializer network2.ServerPeerInitializer, app2 app.Application,

	config Config,
	logger logging.Logger,
) (*network2.Listener, error) {
	return network2.NewListener(initializer, app2, config.ListenAddress, logger)
}

func newAdaptersFactory(config Config, local2 identity.Public, logger logging.Logger) bolt.AdaptersFactory {
	return func(tx *bbolt.Tx) (commands.Adapters, error) {
		return BuildTransactableAdapters(tx, local2, logger, config)
	}
}

func newBolt(config Config) (*bbolt.DB, error) {
	filename := path.Join(config.DataDirectory, "database.bolt")
	b, err := bbolt.Open(filename, 0600, &bbolt.Options{Timeout: 5 * time.Second})
	if err != nil {
		return nil, errors.Wrap(err, "could not open the database, is something else reading it?")
	}
	return b, nil
}

func privateIdentityToPublicIdentity(p identity.Private) identity.Public {
	return p.Public()
}

func newFormats(
	s *formats.Scuttlebutt,
) []feeds.FeedFormat {
	return []feeds.FeedFormat{
		s,
	}
}

func extractNetworkKeyFromConfig(config Config) boxstream.NetworkKey {
	return config.NetworkKey
}

func extractMessageHMACFromConfig(config Config) formats.MessageHMAC {
	return config.MessageHMAC
}

func extractLoggerFromConfig(config Config) logging.Logger {
	return config.Logger
}

func extractPeerManagerConfigFromConfig(config Config) domain.PeerManagerConfig {
	return config.PeerManagerConfig
}
