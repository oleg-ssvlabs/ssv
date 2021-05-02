package kv

import (
	"bytes"
	"github.com/bloxapp/ssv/storage"
	"github.com/dgraph-io/badger/v3"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type BadgerDb struct {
	db     *badger.DB
	logger zap.Logger
}

func New(logger zap.Logger) (storage.Db, error) {
	// Open the Badger database located in the /tmp/badger directory.
	// It will be created if it doesn't exist.
	opt := badger.DefaultOptions("/Users/nivmuroch/Documents/backend_projects/Infra/eth2-ssv/tmp/badger")
	db, err := badger.Open(opt)
	if err != nil {
		return &BadgerDb{}, errors.Wrap(err, "failed to open badger")
	}
	_db := BadgerDb{
		db:     db,
		logger: logger,
	}

	logger.Info("Badger db initialized")
	return &_db, nil
}

func (b *BadgerDb) Set(prefix []byte, key []byte, value []byte) error {
	return b.db.Update(func(txn *badger.Txn) error {
		err := txn.Set(append(prefix, key...), value)
		return err
	})
}

func (b *BadgerDb) Get(prefix []byte, key []byte) (storage.Obj, error) {
	var resValue []byte
	err := b.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(append(prefix, key...))
		if err != nil {
			return err
		}
		resValue, err = item.ValueCopy(nil)
		return err
	})
	return storage.Obj{
		Key: key,
		Value: resValue,
	}, err
}

func (b *BadgerDb) GetAllByBucket(prefix []byte) ([]storage.Obj, error) {
	var res []storage.Obj
	var err error
	err = b.db.View(func(txn *badger.Txn) error {
		opt := badger.DefaultIteratorOptions
		it := txn.NewIterator(opt)
		defer it.Close()
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			resKey := item.Key()
			trimmedResKey := bytes.TrimPrefix(resKey, prefix)
			val, err := item.ValueCopy(nil)
			obj := storage.Obj{
				Key: trimmedResKey,
				Value: val,
			}
			res = append(res, obj)
			return err
		}
		return err
	})
	return res, err
}
