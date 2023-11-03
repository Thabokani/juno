package blockbuilder

import (
	"context"
	"time"

	"github.com/NethermindEth/juno/core"
	"github.com/NethermindEth/juno/mempool"
)

// Todo:
// - Implement block building into blockbuilder.Build()
// - Modify sync service to use blockbuilder blocks instead of feeder-gateway blocks

const (
	numTxnsPerBlock = 100
	blockTimeSec = 2*time.Second
)

type Builder struct {
	mempool *mempool.Mempool
}

func New(mempool *mempool.Mempool) *Builder {
	return &Builder{
		mempool: mempool,
	}
}

// Run(ctx context.Context) defines blockbuilder as a Service.Service
func (b *Builder) Run(ctx context.Context) (error){
	 ticker := time.NewTicker(blockTimeSec)
	 for range ticker.C {
		b.Build()
	 }
	 return nil
}

// ProcessTxn adds the transaction to the Builders mempool if it passes all filters
// otherwise it is dropped (note: for production we would need to relay 
// the transaction in full / by hash, etc)
func (b *Builder) ProcessTxn(tx *core.Transaction) (error) {
	return b.mempool.Process(tx)
}

// Build takes the first n-transactions, executes them in order,
// builds a block, and returns the block.
func (b *Builder) Build() *core.Block {

	newBlock :=core.Block{} // Should be latest block!!

	for i:=0; i<numTxnsPerBlock; i++ {
		_,err:=b.mempool.Dequeue() // todo: use dequeue'd txn
		switch err {
		case mempool.ErrMempoolEmpty:
			break
		default:
			panic("should never happen")
		}

		// TODO: Execute transactions, and apply new state.
		// 1. Get latest state
		// 2. Get the state diff from execution of txn
		// 3. Apply the state diff
		// 4. repeat
	}

	return &newBlock
}



