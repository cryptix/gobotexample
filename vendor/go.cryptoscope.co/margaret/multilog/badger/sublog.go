package badger

import (
	"encoding/binary"

	"github.com/dgraph-io/badger"
	"github.com/pkg/errors"
	"go.cryptoscope.co/luigi"
	"go.cryptoscope.co/margaret"
)

type sublog struct {
	mlog   *mlog
	prefix []byte
	seq    luigi.Observable
}

func (log *sublog) Seq() luigi.Observable {
	return log.seq
}

func (log *sublog) Get(seq margaret.Seq) (interface{}, error) {
	var v interface{}

	npref := len(log.prefix)
	key := make([]byte, npref+8)
	binary.BigEndian.PutUint64(key[npref:], uint64(seq.Seq()))
	copy(key[:npref], log.prefix)

	err := log.mlog.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			return errors.Wrap(err, "error getting item")
		}

		err = item.Value(func(data []byte) error {
			v, err = log.mlog.codec.Unmarshal(data)
			return err
		})
		if err != nil {
			return errors.Wrap(err, "error getting value")
		}

		return nil
	})

	if errors.Cause(err) == badger.ErrKeyNotFound {
		return nil, luigi.EOS{}
	} else if err != nil {
		return nil, errors.Wrap(err, "error in badger transaction (view)")
	}

	return v, nil
}

func (log *sublog) Query(specs ...margaret.QuerySpec) (luigi.Source, error) {
	qry := &query{
		log: log,

		lt:      margaret.SeqEmpty,
		nextSeq: margaret.SeqEmpty,

		limit: -1, //i.e. no limit
	}

	for _, spec := range specs {
		err := spec(qry)
		if err != nil {
			return nil, err
		}
	}

	return qry, nil
}

func (log *sublog) Append(v interface{}) (margaret.Seq, error) {
	var seq margaret.BaseSeq

	data, err := log.mlog.codec.Marshal(v)
	if err != nil {
		return margaret.BaseSeq(-2), errors.Wrap(err, "error marshaling value")
	}

	err = log.mlog.db.Update(func(txn *badger.Txn) error {
		seqIface, err := log.seq.Value()
		if err != nil {
			return errors.Wrap(err, "error getting value from seq observable")
		}

		seq = margaret.BaseSeq(seqIface.(margaret.Seq).Seq() + 1)
		seqBs := make([]byte, 8)
		binary.BigEndian.PutUint64(seqBs, uint64(seq))
		key := append(log.prefix, seqBs...)

		err = txn.Set(key, data)
		if err != nil {
			return errors.Wrap(err, "errors setting value")
		}

		log.seq.Set(seq)
		return nil
	})
	if err != nil {
		return margaret.BaseSeq(-2), errors.Wrap(err, "error in write transaction")
	}

	return seq, nil
}
