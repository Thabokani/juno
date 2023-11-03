package blockbuilder

import (
	"context"
	"time"

	"github.com/NethermindEth/juno/blockchain"
	"github.com/NethermindEth/juno/core"
	"github.com/NethermindEth/juno/core/felt"
	"github.com/NethermindEth/juno/mempool"
	"github.com/NethermindEth/juno/vm"
)

// Todo:
// - Implement block building into blockbuilder.Build()
// - Modify sync service to use blockbuilder blocks instead of feeder-gateway blocks

const (
	numTxnsPerBlock int = 100
	blockTimeSec time.Duration = 2*time.Second
)

type Builder struct {
	chain  *blockchain.Blockchain
	starknetVM     vm.VM
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
		newBlock,err := b.buildNewBlock()
		if err!=nil{
			panic(err) // Todo: Should we handle this gracefully?
		}
		newCommitments,err := b.newCommitments()
		if err!=nil{
			panic(err) // Todo: Should we handle this gracefully?
		}
		newStateUpdate,err := b.newStateUpdate()
		if err!=nil{
			panic(err) // Todo: Should we handle this gracefully?
		}
		newClasses,err := b.newClasses()
		if err!=nil{
			panic(err) // Todo: Should we handle this gracefully?
		}
		err=b.chain.Store(newBlock,newCommitments,newStateUpdate,newClasses)
		if err!=nil{
			panic(err) // Todo: Should we handle this gracefully?
		}		
	 }
	 return nil // Todo: Should not happen
}

// ProcessTxn adds the transaction to the Builders mempool if it passes all filters
// otherwise it is dropped (note: for production we would need to relay 
// the transaction in full / by hash, etc)
func (b *Builder) ProcessTxn(tx *core.Transaction) (error) {
	return b.mempool.Process(tx)
}

// buildNewBlock() takes the first n-transactions, executes them in fifo order,
// builds a new block, and returns the resulting block.
func (b *Builder) buildNewBlock() (*core.Block,error) {	

	header, err := b.chain.HeadsHeader()
	if err != nil {
		return nil,err
	}

	pendingState, stateCloser, err := b.chain.HeadState()
	if err != nil {
		return nil,err
	}
	defer stateCloser()

	for i := 0; i < numTxnsPerBlock; i++ {
		// Todo:
		// 1. Include failing transactions (they should be charged fees)
		// 2. Make sure we reapply the new state before calling Execute() again
		// 3. Should we deduct fees?
		txn,err:=b.mempool.Dequeue()
		if err != nil {
			if err == mempool.ErrMempoolEmpty{
				break
			}
			return nil,err
		}
		_, _, err = b.starknetVM.Execute(
			[]core.Transaction{*txn},
			nil,
			header.Number, 
			header.Timestamp, 
			header.SequencerAddress, 
			pendingState,
			b.chain.Network(), 
			nil, 
			false, 
			nil, 
			false)
		if err!=nil {		// Todo: Make sure we don't return if txn just failed.
			return nil,err
		}
		// Todo: finish logic
		
	}

	return &core.Block{},nil // Todo: update when this function is completed
}

func (b *Builder) newCommitments() (*core.BlockCommitments,error){
	// todo: implement
	panic("newCommitments() not implemented")
}
func (b *Builder) newStateUpdate() (*core.StateUpdate,error){
	// todo: implement
	panic("newStateUpdate() not implemented")
}
func (b *Builder) newClasses() (map[felt.Felt]core.Class,error){
	// todo: implement
	panic("newClasses() not implemented")
}



