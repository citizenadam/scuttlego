package commands

import (
	"context"
	"time"

	"github.com/boreq/errors"
	"github.com/planetary-social/scuttlego/service/domain/bans"
	"github.com/planetary-social/scuttlego/service/domain/feeds"
	"github.com/planetary-social/scuttlego/service/domain/graph"
	"github.com/planetary-social/scuttlego/service/domain/identity"
	"github.com/planetary-social/scuttlego/service/domain/network"
	"github.com/planetary-social/scuttlego/service/domain/refs"
	"github.com/planetary-social/scuttlego/service/domain/transport"
)

type NewPeerHandler interface {
	HandleNewPeer(peer transport.Peer)
}

type PeerManager interface {
	// Connect instructs the peer manager that it should establish
	// communications with the specified node. The peer manager may ignore this
	// request under specific circumstances e.g. it may avoid establishing
	// duplicate connections to a single identity.
	Connect(remote identity.Public, address network.Address) error

	// ConnectViaRoom instructs the peer manager that it should establish
	// communications with the specified node using a room as a relay. Behaves
	// like Connect.
	ConnectViaRoom(portal transport.Peer, target identity.Public) error

	// EstablishNewConnections instructs the peer manager that it is time to
	// establish new connections so that the specific connections quotas are
	// met.
	EstablishNewConnections() error

	// ProcessNewLocalDiscovery informs the peer manager about a new local
	// discovery.
	ProcessNewLocalDiscovery(remote identity.Public, address network.Address) error

	// DisconnectAll disconnects all peers.
	DisconnectAll() error
}

type TransactionProvider interface {
	Transact(func(adapters Adapters) error) error
}

type Adapters struct {
	Feed         FeedRepository
	SocialGraph  SocialGraphRepository
	BlobWantList BlobWantListRepository
	FeedWantList FeedWantListRepository
	BanList      BanListRepository
}

type UpdateFeedFn func(feed *feeds.Feed) error

type FeedRepository interface {
	// UpdateFeed updates the specified feed by calling the provided function on
	// it. Feed is never nil.
	UpdateFeed(ref refs.Feed, f UpdateFeedFn) error

	// DeleteFeed removes the feed with all associated data.
	DeleteFeed(ref refs.Feed) error
}

type SocialGraphRepository interface {
	GetSocialGraph() (graph.SocialGraph, error)
}

type BlobWantListRepository interface {
	// Add puts the blob in the want list. If the blob can't be retrieved before
	// the specified point of time it will be removed from the want list.
	Add(id refs.Blob, until time.Time) error
}

// FeedWantListRepository adds a way to temporarily add feeds to the list of
// replicated feeds. Those feeds will be replicated even if they are not in the
// social graph.
type FeedWantListRepository interface {
	// Add puts the feed in the want list. The entry is removed after the
	// specified amount of time.
	Add(id refs.Feed, until time.Time) error
}

type CurrentTimeProvider interface {
	Get() time.Time
}

// BannableRef wraps a feed ref.
type BannableRef struct {
	v any
}

func NewBannableRef(v any) (BannableRef, error) {
	switch v.(type) {
	case refs.Feed:
	default:
		return BannableRef{}, errors.New("must carry a feed ref")
	}
	return BannableRef{v: v}, nil
}

func (b *BannableRef) Value() any {
	return b.v
}

type BanListRepository interface {
	// Add adds a hash to the ban list.
	Add(hash bans.Hash) error

	// Remove removes a hash from the ban list. If a hash isn't in the ban list
	// no errors are returned.
	Remove(hash bans.Hash) error

	// ContainsFeed checks if the particular feed is banned.
	ContainsFeed(feed refs.Feed) (bool, error)

	// LookupMapping returns ErrBanListMappingNotFound error if a ref can not be
	// found.
	LookupMapping(hash bans.Hash) (BannableRef, error)
}

var ErrBanListMappingNotFound = errors.New("ban list mapping not found")

type Dialer interface {
	DialWithInitializer(ctx context.Context, initializer network.ClientPeerInitializer, remote identity.Public, addr network.Address) (transport.Peer, error)
	Dial(ctx context.Context, remote identity.Public, address network.Address) (transport.Peer, error)
}
