package mempool

import (
	"sync"
	"github.com/NethermindEth/juno/rpc/broadcasted"
)

type Mempool struct {
	mu  sync.Mutex
	txs []*broadcasted.BroadcastedTransaction
}

func New() *Mempool {
	return &Mempool{
		txs: make([]*broadcasted.BroadcastedTransaction, 0),
	}
}

func (m *Mempool) Enqueue(tx *broadcasted.BroadcastedTransaction) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.txs = append(m.txs, tx)
}

func (m *Mempool) Dequeue() *broadcasted.BroadcastedTransaction {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.txs) == 0 {
		return nil
	}
	txn := m.txs[0]
	m.txs[0] = nil // avoid memory leak
	m.txs = m.txs[1:]
	return txn
}
