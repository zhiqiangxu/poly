package db

import (
	"bytes"
	"errors"
	"fmt"
	"sync"

	pool "github.com/valyala/bytebufferpool"

	"github.com/Ontology/common"
	storcomm"github.com/Ontology/core/store/common"
	leveldb "github.com/Ontology/core/store/leveldbstore"
	"github.com/Ontology/core/types"
	tx "github.com/Ontology/core/types"

	"github.com/Ontology/common/serialization"
)

var keyPool pool.Pool
var valuePool pool.Pool

type Store struct {
	db storcomm.IStore

	mutex           sync.RWMutex // guard the following var
	bestBlockHeader *types.Header
	genesisBlock    *types.Block
}

func NewStore(path string) (*Store, error) {
	ldb, err := leveldb.NewLevelDBStore(path)
	if err != nil {
		return nil, err
	}

	st := &Store{db: ldb}
	err = st.init()
	if err != nil {
		return nil, err
	}

	return st, nil
}

func (self *Store) init() error {
	prefix := []byte{byte(SYS_Version)}
	version, err := self.db.Get(prefix)
	if err != nil {
		version = []byte{0x00}
	}

	if version[0] == 0x01 {
		//test if genesis block in db
		genesis, err := self.db.Get([]byte{byte(SYS_GenesisBlock)})
		if err != nil {
			self.bestBlockHeader = nil
			self.genesisBlock = nil
			return nil
		}

		self.genesisBlock = new(types.Block)
		buf := bytes.NewBuffer(genesis)
		err = self.genesisBlock.Deserialize(buf)
		if err != nil {
			return errors.New(fmt.Sprint("inconsist db: genesis block deserialize failed. cause of:\n ", err.Error()))
		}

		best, err := self.db.Get([]byte{byte(SYS_BestBlockHeader)})
		if err != nil {
			return errors.New("inconsist db: best blockheader not in db")
		}

		self.bestBlockHeader = new(types.Header)
		buf = bytes.NewBuffer(best)
		err = self.bestBlockHeader.Deserialize(buf)
		if err != nil {
			return errors.New(fmt.Sprint("inconsist db: best blockheader deserialize failed. cause of:\n ", err.Error()))
		}

		return nil
	} else {
		self.bestBlockHeader = nil
		self.genesisBlock = nil
		// can not find version info
		iter := self.db.NewIterator(nil)
		if iter.Next() {
			iter.Release()
			return errors.New("not a fresh db")
		}
		iter.Release()

		// put version to db
		err := self.db.Put(prefix, []byte{0x01})

		return err
	}

}

func (self *Store) GetBestBlock() (BestBlock, error) {
	if self.bestBlockHeader == nil {
		return BestBlock{}, errors.New("fresh db")
	}
	return BestBlock{
		Height: self.bestBlockHeader.Height,
		Hash:   self.bestBlockHeader.Hash(),
	}, nil
}

func (self *Store) GetBestHeader() (*types.Header, error) {
	if self.bestBlockHeader == nil {
		return nil, errors.New("fresh db")
	}

	return self.bestBlockHeader, nil
}

// implement  TransactionProvider interface
func (self *Store) ContainTransaction(hash common.Uint256) bool {
	_, err := self.GetTransactionBytes(hash)
	return err == nil
}

func (self *Store) GetTransactionBytes(hash common.Uint256) ([]byte, error) {
	key := GenDataTransactionKey(hash)
	defer keyPool.Put(key)
	txn, err := self.db.Get(key.Bytes())

	return txn, err
}

func (self *Store) GetTransaction(hash common.Uint256) (*tx.Transaction, error) {
	buf, err := self.GetTransactionBytes(hash)
	if err != nil {
		return nil, err
	}
	txn := new(tx.Transaction)
	err = txn.Deserialize(bytes.NewBuffer(buf))
	if err != nil {
		return nil, err
	}
	return txn, nil
}

//implement  TransactionMetaProvider interface
func (self *Store) GetTransactionMeta(hash common.Uint256) (TransactionMeta, error) {
	key := GenTxMetaKey(hash)
	defer keyPool.Put(key)
	txbuf, err := self.db.Get(key.Bytes())
	if err != nil {
		return TransactionMeta{}, err
	}

	txMeta := TransactionMeta{}
	err = txMeta.Deserialize(bytes.NewBuffer(txbuf))
	if err != nil {
		return TransactionMeta{}, err
	}

	return txMeta, nil
}

func (self *Store) Close() error {
	err := self.db.Close()
	self.db = nil
	return err
}

func (self *Store) saveTransaction(tx *tx.Transaction, height uint32) error {
	// generate key with DATA_Transaction prefix
	key := GenDataTransactionKey(tx.Hash())
	defer keyPool.Put(key)
	value := valuePool.Get()
	defer valuePool.Put(value)

	serialization.WriteUint32(value, height)
	tx.Serialize(value)

	// put value
	err := self.db.BatchPut(key.Bytes(), value.Bytes())
	return err
}

func (self *Store) PersistBlock(block *types.Block) error {
	height := block.Header.Height
	if !((self.bestBlockHeader == nil && height == 0) || height == self.bestBlockHeader.Height+1) {
		return errors.New("can't persist discontinuous block")
	}

	self.mutex.Lock()
	defer self.mutex.Unlock()

	self.db.NewBatch()
	for _, txn := range block.Transactions {
		err := self.saveTransaction(txn, height)
		if err != nil {
			return err
		}
	}

	// is genesis block
	if self.bestBlockHeader == nil {
		key := GenGenesisBlockKey()
		defer keyPool.Put(key)
		value := valuePool.Get()
		defer valuePool.Put(value)

		block.Serialize(value)
		self.db.BatchPut(key.Bytes(), value.Bytes())
	}

	key := GenBestBlockHeaderKey()
	defer keyPool.Put(key)
	value := valuePool.Get()
	defer valuePool.Put(value)

	header := block.Header

	header.Serialize(value)
	self.db.BatchPut(key.Bytes(), value.Bytes())

	err := self.db.BatchCommit()

	if err != nil {
		return err
	}

	if self.bestBlockHeader == nil {
		self.genesisBlock = block
	}
	self.bestBlockHeader = block.Header

	return err
}
