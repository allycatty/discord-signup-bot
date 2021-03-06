package storage

import (
	"context"
	"strings"

	bolt "github.com/coreos/bbolt"
	"github.com/golang/protobuf/proto"
	"github.com/gsmcwhirter/go-util/v5/errors"
	census "github.com/gsmcwhirter/go-util/v5/stats"
)

// ErrTrialNotExist is the error returned if a trial does not exist
var ErrTrialNotExist = errors.New("trial does not exist")

type boltTrialAPI struct {
	db     *bolt.DB
	census *census.Census
}

// NewBoltTrialAPI constructs a boltDB-backed TrialAPI
func NewBoltTrialAPI(db *bolt.DB, c *census.Census) (TrialAPI, error) {
	b := boltTrialAPI{
		db:     db,
		census: c,
	}

	return &b, nil
}

func (b *boltTrialAPI) NewTransaction(ctx context.Context, guild string, writable bool) (TrialAPITx, error) {
	_, span := b.census.StartSpan(ctx, "boltTrialAPI.NewTransaction")
	defer span.End()

	bucketName := []byte(guild)

	err := b.db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(bucketName)
		if err != nil {
			return errors.Wrap(err, "could not create bucket")
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	tx, err := b.db.Begin(writable)
	if err != nil {
		return nil, err
	}
	return &boltTrialAPITx{
		bucketName: bucketName,
		tx:         tx,
		census:     b.census,
	}, nil
}

type boltTrialAPITx struct {
	bucketName []byte
	tx         *bolt.Tx
	census     *census.Census
}

func (b *boltTrialAPITx) Commit(ctx context.Context) error {
	_, span := b.census.StartSpan(ctx, "boltTrialAPITx.Commit")
	defer span.End()

	return b.tx.Commit()
}

func (b *boltTrialAPITx) Rollback(ctx context.Context) error {
	_, span := b.census.StartSpan(ctx, "boltTrialAPITx.Rollback")
	defer span.End()

	err := b.tx.Rollback()
	if err != nil && err != bolt.ErrTxClosed {
		return err
	}
	return nil
}

func (b *boltTrialAPITx) AddTrial(ctx context.Context, name string) (Trial, error) {
	ctx, span := b.census.StartSpan(ctx, "boltTrialAPITx.AddTrial")
	defer span.End()

	name = strings.ToLower(name)

	trial, err := b.GetTrial(ctx, name)
	if err == ErrTrialNotExist {
		trial = &boltTrial{
			protoTrial: &ProtoTrial{Name: name},
			census:     b.census,
		}
		err = nil
	}
	return trial, err
}

func (b *boltTrialAPITx) SaveTrial(ctx context.Context, t Trial) error {
	ctx, span := b.census.StartSpan(ctx, "boltTrialAPITx.SaveTrial")
	defer span.End()

	bucket := b.tx.Bucket(b.bucketName)

	serial, err := t.Serialize(ctx)
	if err != nil {
		return err
	}

	return bucket.Put([]byte(strings.ToLower(t.GetName(ctx))), serial)
}

func (b *boltTrialAPITx) GetTrial(ctx context.Context, name string) (Trial, error) {
	_, span := b.census.StartSpan(ctx, "boltTrialAPITx.GetTrial")
	defer span.End()

	bucket := b.tx.Bucket(b.bucketName)

	val := bucket.Get([]byte(strings.ToLower(name)))
	if val == nil {
		val = bucket.Get([]byte(name))
		if val == nil {
			return nil, ErrTrialNotExist
		}
	}

	protoTrial := ProtoTrial{}
	err := proto.Unmarshal(val, &protoTrial)
	if err != nil {
		return nil, errors.Wrap(err, "trial record is corrupt")
	}

	return &boltTrial{&protoTrial, b.census}, nil
}

func (b *boltTrialAPITx) DeleteTrial(ctx context.Context, name string) error {
	ctx, span := b.census.StartSpan(ctx, "boltTrialAPITx.DeleteTrial")
	defer span.End()

	_, err := b.GetTrial(ctx, name)
	if err != nil {
		return err
	}

	name = strings.ToLower(name)
	bucket := b.tx.Bucket(b.bucketName)

	return bucket.Delete([]byte(name))
}

func (b *boltTrialAPITx) GetTrials(ctx context.Context) []Trial {
	_, span := b.census.StartSpan(ctx, "boltTrialAPITx.GetTrials")
	defer span.End()

	bucket := b.tx.Bucket(b.bucketName)

	t := make([]Trial, 0, 10)
	_ = bucket.ForEach(func(k []byte, v []byte) error {
		protoTrial := ProtoTrial{}
		err := proto.Unmarshal(v, &protoTrial)
		if err == nil {
			t = append(t, &boltTrial{&protoTrial, b.census})
		}

		return nil
	})

	return t
}
