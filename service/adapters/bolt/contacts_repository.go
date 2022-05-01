package bolt

import (
	"github.com/boreq/errors"
	"github.com/planetary-social/go-ssb/service/domain/refs"
	"github.com/planetary-social/go-ssb/service/domain/replication"
	"go.etcd.io/bbolt"
)

type BoltContactsRepository struct {
	db      *bbolt.DB
	factory TxRepositoriesFactory
}

func NewBoltContactsRepository(db *bbolt.DB, factory TxRepositoriesFactory) *BoltContactsRepository {
	return &BoltContactsRepository{db: db, factory: factory}
}

func (b BoltContactsRepository) GetContacts() ([]replication.Contact, error) {
	var result []replication.Contact

	if err := b.db.View(func(tx *bbolt.Tx) error {
		r, err := b.factory(tx)
		if err != nil {
			return errors.Wrap(err, "could not call the factory")
		}

		graph, err := r.Graph.GetSocialGraph()
		if err != nil {
			return errors.Wrap(err, "could not get contacts")
		}

		for _, contact := range graph.Contacts() {
			f := contact.Id.MainFeed()

			feedState, err := b.getFeedState(r.Feed, f)
			if err != nil {
				return errors.Wrap(err, "could not get feed state")
			}

			result = append(result, replication.Contact{
				Who:       f,
				Hops:      contact.Hops,
				FeedState: feedState,
			})
		}

		return nil
	}); err != nil {
		return nil, errors.Wrap(err, "transaction failed")
	}

	return result, nil
}

func (b BoltContactsRepository) getFeedState(repository *FeedRepository, feed refs.Feed) (replication.FeedState, error) {
	f, err := repository.GetFeed(feed)
	if err != nil {
		if errors.Is(err, ErrFeedNotFound) {
			return replication.NewEmptyFeedState(), nil
		}
		return replication.FeedState{}, errors.Wrap(err, "could not get a feed")
	}
	return replication.NewFeedState(f.Sequence())
}
