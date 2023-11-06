package blockbuilder

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/NethermindEth/juno/blockbuilder/vm2core"
	"github.com/NethermindEth/juno/blockchain"
	"github.com/NethermindEth/juno/core"
	"github.com/NethermindEth/juno/core/felt"
	"github.com/NethermindEth/juno/db"
	"github.com/NethermindEth/juno/mempool"
	"github.com/NethermindEth/juno/rpc/broadcasted"
	"github.com/NethermindEth/juno/vm"
)

// Todo:
// - Implement block building into blockbuilder.Build()
// - Modify sync service to use blockbuilder blocks instead of feeder-gateway blocks

const (
	numTxnsPerBlock int           = 100
	blockTime       time.Duration = 2 * time.Second
)

type Builder struct {
	chain       *blockchain.Blockchain
	blockNumber uint64
	starknetVM  vm.VM
	mempool     *mempool.Mempool
}

func New(chain *blockchain.Blockchain, starknetVM vm.VM, mempool *mempool.Mempool) *Builder {
	return &Builder{
		chain:      chain,
		starknetVM: starknetVM,
		mempool:    mempool,
	}
}

// Run(ctx context.Context) defines blockbuilder as a Service.Service
func (b *Builder) Run(ctx context.Context) error {
	for ctx.Err() == nil {
		curHeader, err := b.chain.HeadsHeader()
		if err != nil {
			if !errors.Is(err, db.ErrKeyNotFound) {
				return fmt.Errorf("heads header: %v", err)
			}
			// TODO need to set a fake account
			curHeader = &core.Header{
				ParentHash:       new(felt.Felt),
				SequencerAddress: new(felt.Felt),
				TransactionCount: 0,
				ProtocolVersion:  "v0.13.0",
				GasPrice:         new(felt.Felt),
			}
		}

		pendingHeader := &core.Header{
			ParentHash:       curHeader.Hash,
			Number:           curHeader.Number + 1,
			SequencerAddress: curHeader.SequencerAddress,
			TransactionCount: 1,
			Timestamp:        uint64(time.Now().Unix()),
			ProtocolVersion:  curHeader.ProtocolVersion,
			GasPrice:         new(felt.Felt),
		}

		txn := b.mempool.Dequeue()
		if txn == nil {
			time.Sleep(time.Second)
			continue
		}
		tx, class, paidFeeOnL1, err := broadcasted.AdaptBroadcastedTransaction(txn, b.chain.Network())
		if err != nil {
			return fmt.Errorf("adapt broadcasted transaction: %v", err)
		}

		stateReader, stateCloser, err := b.chain.HeadState()
		if err != nil {
			return fmt.Errorf("head state: %v", err)
		}
		_, traces, err := b.starknetVM.Execute([]core.Transaction{tx}, []core.Class{class}, pendingHeader.Number, pendingHeader.Timestamp, pendingHeader.SequencerAddress, stateReader, b.chain.Network(), []*felt.Felt{paidFeeOnL1}, false, new(felt.Felt), false)
		stateCloser()
		if err != nil {
			return fmt.Errorf("execute transaction: %v", err)
		}

		stateDiff, err := vm2core.TraceToStateDiff(traces[0])
		if err != nil {
			return fmt.Errorf("trace to state diff: %v", err)
		}
		fmt.Printf("state diff: %+v\n", stateDiff)
		// TODO: need to calculate transaction receipt
		// Fill in missing fields in block header (e.g., EventCount)
		// Calculate block hash and block commitments
		// err = b.chain.Store(newBlock, newCommitments, newStateUpdate, newClasses)
	}
	return nil
}
