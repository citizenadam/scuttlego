package bolt

import (
	"github.com/boreq/errors"
	"github.com/planetary-social/go-ssb/service/domain/feeds"
	"github.com/planetary-social/go-ssb/service/domain/refs"
	"go.etcd.io/bbolt"
)

var bucketBlobs = bucketName("blobs")

type BlobRepository struct {
	tx *bbolt.Tx
}

func NewBlobRepository(
	tx *bbolt.Tx,
) *BlobRepository {
	return &BlobRepository{
		tx: tx,
	}
}

func (r BlobRepository) Put(blob feeds.BlobsToSave) error {
	for _, blobRef := range blob.Blobs() {
		bucket, err := r.createBucket(blobRef, blob.Feed())
		if err != nil {
			return errors.Wrap(err, "could not create the bucket")
		}

		if err := bucket.Put([]byte(blob.Message().String()), nil); err != nil {
			return errors.Wrap(err, "bucket put failed")
		}
	}

	return nil
}

func (r BlobRepository) List() ([]refs.Blob, error) {
	var result []refs.Blob

	bucket := r.tx.Bucket(bucketBlobs)
	if bucket == nil {
		return nil, nil
	}

	if err := bucket.ForEach(func(k, v []byte) error {
		ref, err := refs.NewBlob(string(k))
		if err != nil {
			return errors.Wrap(err, "could not create a ref")
		}
		result = append(result, ref)
		return nil
	}); err != nil {
		return nil, errors.Wrap(err, "for each failed")
	}

	return result, nil
}

func (r BlobRepository) createBucket(blob refs.Blob, feed refs.Feed) (*bbolt.Bucket, error) {
	return createBucket(r.tx, r.bucketPath(blob, feed))
}

func (r BlobRepository) bucketPath(blob refs.Blob, feed refs.Feed) []bucketName {
	return []bucketName{
		bucketBlobs,
		bucketName(blob.String()),
		bucketName("feeds"),
		bucketName(feed.String()),
	}
}
