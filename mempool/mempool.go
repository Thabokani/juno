package mempool

import (
	"sync"

	"github.com/NethermindEth/juno/core"
)



type Mempool struct {
	mu sync.Mutex
	txs []*core.Transaction
}

func New() *Mempool {
	return &Mempool{
		mu: sync.Mutex{},
		txs: []*core.Transaction{},
	}
}

// Process adds the transaction to the mempool if it passes all
// filters, otherwise it is dropped.
func (m *Mempool) Process(tx *core.Transaction) error {
	if m.Valid(tx){
		m.Enqueue(tx)
		return nil
	}
	return ErrFailedValidation
}

func (m *Mempool) Valid(tx *core.Transaction) bool{
	return FilterZeroHash(tx) && FilterOneHash(tx)
}

func (m *Mempool) Enqueue(tx *core.Transaction){
	m.mu.Lock()
	defer m.mu.Unlock()
	m.txs = append(m.txs, tx)
}

func (m *Mempool) Dequeue() (*core.Transaction,error){
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if len(m.txs)==0{
		return nil,ErrMempoolEmpty
	}
	txn := m.txs[0]
	m.txs=m.txs[1:]
	return txn,nil
}


