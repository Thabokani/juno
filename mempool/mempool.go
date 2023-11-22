package mempool

import (
	"sync"
	"time"

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

func (m *Mempool) WaitForTwoTransactions() []*broadcasted.BroadcastedTransaction {
	m.mu.Lock()
	defer m.mu.Unlock()

	for len(m.txs) < 2 {
		m.mu.Unlock()
		time.Sleep(time.Second)
		m.mu.Lock()
	}
	txn1 := m.Dequeue()
	txn2 := m.Dequeue()

	return []*broadcasted.BroadcastedTransaction{txn1, txn2}
}
