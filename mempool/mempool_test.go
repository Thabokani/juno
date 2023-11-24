package mempool_test

import (
	"testing"
	"time"

	"github.com/NethermindEth/juno/mempool"
	"github.com/NethermindEth/juno/rpc/broadcasted"
	"github.com/stretchr/testify/assert" // You might need to import testify for assertions
)

func TestWaitForTwoTransactions(t *testing.T) {
	m := mempool.New()

	// Simulate adding transactions in a separate goroutine

	m.Enqueue(&broadcasted.BroadcastedTransaction{})
	time.Sleep(50 * time.Millisecond) // slight delay to simulate async behavior
	m.Enqueue(&broadcasted.BroadcastedTransaction{})

	// Call WaitForTwoTransactions and check the result
	txns := m.WaitForTwoTransactions()

	assert.Len(t, txns, 2, "Expected two transactions")

}
