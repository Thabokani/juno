package blockbuilder

import (
	"context"
	_ "embed"
	"encoding/json"
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
	"github.com/NethermindEth/juno/utils"
	"github.com/NethermindEth/juno/vm"
)

var (
	//go:embed account.json
	accountClassString string
	//go:embed erc20.json
	erc20ClassString string
	//go:embed udc.json
	udcClassString string

	sequencerAddress = new(felt.Felt)
	udcAddress       *felt.Felt
	feeTokenAddress  *felt.Felt

	accountClassHash *felt.Felt
	erc20ClassHash   *felt.Felt
	udcClassHash     *felt.Felt

	accountClass core.Class
	erc20Class   core.Class
	udcClass     core.Class
)

func init() {
	var err error

	udcAddress, err = new(felt.Felt).SetString("0x041a78e741e5af2fec34b695679bc6891742439f7afb8484ecd7766661ad02bf")
	if err != nil {
		panic(fmt.Errorf("set udc address: %v", err))
	}
	feeTokenAddress, err = new(felt.Felt).SetString("0x049d36570d4e46f48e99674bd3fcc84644ddd6b96f7c741b1562b82f9e004dc7")
	if err != nil {
		panic(fmt.Errorf("set fee token address: %v", err))
	}

	accountClassHash, err = new(felt.Felt).SetString("0x04d07e40e93398ed3c76981e72dd1fd22557a78ce36c0515f679e27f0bb5bc5f")
	if err != nil {
		panic(fmt.Errorf("set account contract class hash: %v", err))
	}
	erc20ClassHash, err = new(felt.Felt).SetString("0x02a8846878b6ad1f54f6ba46f5f40e11cee755c677f130b2c4b60566c9003f1f")
	if err != nil {
		panic(fmt.Errorf("set erc20 class hash: %v", err))
	}
	udcClassHash, err = new(felt.Felt).SetString("0x07b3e05f48f0c69e4a65ce5e076a66271a527aff2c34ce1083ec6e1526997a69")
	if err != nil {
		panic(fmt.Errorf("set udc class hash: %v", err))
	}

	classJSON := json.RawMessage{}

	if err := json.Unmarshal([]byte(accountClassString), &classJSON); err != nil {
		panic(fmt.Errorf("unmarshal account class: %v", err))
	}
	accountClass, err = broadcasted.AdaptDeclaredClass(classJSON, false)
	if err != nil {
		panic(fmt.Errorf("adapt account class: %v", err))
	}

	if err := json.Unmarshal([]byte(erc20ClassString), &classJSON); err != nil {
		panic(fmt.Errorf("unmarshal erc20 class: %v", err))
	}
	erc20Class, err = broadcasted.AdaptDeclaredClass(classJSON, false)
	if err != nil {
		panic(fmt.Errorf("adapt erc20 class: %v", err))
	}

	if err := json.Unmarshal([]byte(udcClassString), &classJSON); err != nil {
		panic(fmt.Errorf("unmarshal udc class: %v", err))
	}
	udcClass, err = broadcasted.AdaptDeclaredClass(classJSON, false)
	if err != nil {
		panic(fmt.Errorf("adapt udc class: %v", err))
	}
}

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

func (b *Builder) storeGenesisBlockAndState() error {
	emptyReceipts := []*core.TransactionReceipt{}
	// Empty storage and empty classes trie (classes trie only stores Cairo 1 classes).
	// When classes trie is empty, use the root of storage trie as state commitment.
	emptyStateRoot := new(felt.Felt)
	block := &core.Block{
		Header: &core.Header{
			Hash:             nil,
			GlobalStateRoot:  emptyStateRoot,
			ParentHash:       new(felt.Felt),
			SequencerAddress: new(felt.Felt),
			TransactionCount: 0,
			EventCount:       0,
			ProtocolVersion:  "v0.12.3",
			GasPrice:         new(felt.Felt),
			Timestamp:        uint64(time.Now().Unix()),
			EventsBloom:      core.EventsBloom(emptyReceipts),
		},
		Transactions: []core.Transaction{},
		Receipts:     emptyReceipts,
	}
	blockHash, commitments, err := core.BlockHash(block, utils.GOERLI2, sequencerAddress)
	if err != nil {
		return fmt.Errorf("genesis block hash: %v", err)
	}
	block.Hash = blockHash
	// This is equivalent to three Declare v1 transactions.
	if err := b.chain.Store(block, commitments, &core.StateUpdate{
		BlockHash: blockHash,
		NewRoot:   emptyStateRoot,
		OldRoot:   new(felt.Felt),
		StateDiff: &core.StateDiff{
			StorageDiffs:      map[felt.Felt][]core.StorageDiff{},
			Nonces:            map[felt.Felt]*felt.Felt{},
			DeployedContracts: []core.AddressClassHashPair{},
			DeclaredV0Classes: []*felt.Felt{accountClassHash, erc20ClassHash, udcClassHash}, // Doesn't do anything.
			DeclaredV1Classes: []core.DeclaredV1Class{},
			ReplacedClasses:   []core.AddressClassHashPair{},
		},
	}, map[felt.Felt]core.Class{
		*accountClassHash: accountClass,
		*erc20ClassHash:   erc20Class,
		*udcClassHash:     udcClass,
	}); err != nil {
		return fmt.Errorf("store: %v", err)
	}

	return nil
}

// Run(ctx context.Context) defines blockbuilder as a Service.Service
func (b *Builder) Run(ctx context.Context) error {
	if _, err := b.chain.Height(); err != nil {
		if !errors.Is(err, db.ErrKeyNotFound) {
			return fmt.Errorf("chain height: %v", err)
		}

		if err := b.storeGenesisBlockAndState(); err != nil {
			return fmt.Errorf("store genesis block and state: %v", err)
		}
	}

	for ctx.Err() == nil {
		curHeader, err := b.chain.HeadsHeader()
		if err != nil {
			return fmt.Errorf("heads header: %v", err)
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
