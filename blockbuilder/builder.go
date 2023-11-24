package blockbuilder

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/NethermindEth/juno/blockbuilder/vm2core"
	"github.com/NethermindEth/juno/blockchain"
	"github.com/NethermindEth/juno/core"
	"github.com/NethermindEth/juno/core/crypto"
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
)

func hexToFelt(hex string) *felt.Felt {
	result, err := new(felt.Felt).SetString(hex)
	if err != nil {
		panic(fmt.Errorf("string %s to felt: %v", hex, err))
	}
	return result
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
	// Initialize values.

	udcAddress, err := new(felt.Felt).SetString("0x041a78e741e5af2fec34b695679bc6891742439f7afb8484ecd7766661ad02bf")
	if err != nil {
		return fmt.Errorf("set udc address: %v", err)
	}
	feeTokenAddress, err := new(felt.Felt).SetString("0x049d36570d4e46f48e99674bd3fcc84644ddd6b96f7c741b1562b82f9e004dc7")
	if err != nil {
		return fmt.Errorf("set fee token address: %v", err)
	}

	accountClassHash, err := new(felt.Felt).SetString("0x04d07e40e93398ed3c76981e72dd1fd22557a78ce36c0515f679e27f0bb5bc5f")
	if err != nil {
		return fmt.Errorf("set account contract class hash: %v", err)
	}
	erc20ClassHash, err := new(felt.Felt).SetString("0x02a8846878b6ad1f54f6ba46f5f40e11cee755c677f130b2c4b60566c9003f1f")
	if err != nil {
		return fmt.Errorf("set erc20 class hash: %v", err)
	}
	udcClassHash, err := new(felt.Felt).SetString("0x07b3e05f48f0c69e4a65ce5e076a66271a527aff2c34ce1083ec6e1526997a69")
	if err != nil {
		return fmt.Errorf("set udc class hash: %v", err)
	}

	classJSON := json.RawMessage{}

	if err := json.Unmarshal([]byte(accountClassString), &classJSON); err != nil {
		panic(fmt.Errorf("unmarshal account class: %v", err))
	}
	accountClass, err := broadcasted.AdaptDeclaredClass(classJSON, false)
	if err != nil {
		panic(fmt.Errorf("adapt account class: %v", err))
	}

	if err := json.Unmarshal([]byte(erc20ClassString), &classJSON); err != nil {
		panic(fmt.Errorf("unmarshal erc20 class: %v", err))
	}
	erc20Class, err := broadcasted.AdaptDeclaredClass(classJSON, false)
	if err != nil {
		panic(fmt.Errorf("adapt erc20 class: %v", err))
	}

	if err = json.Unmarshal([]byte(udcClassString), &classJSON); err != nil {
		panic(fmt.Errorf("unmarshal udc class: %v", err))
	}
	udcClass, err := broadcasted.AdaptDeclaredClass(classJSON, false)
	if err != nil {
		panic(fmt.Errorf("adapt udc class: %v", err))
	}

	testAddress := hexToFelt("0x2")
	testAddress2 := hexToFelt("0x8e1d0e590bff38a8f2bdee751a63d7d9d964bf034271ca369abfb9756ed253")

	defaultPrefundedAccountBalance, err := new(felt.Felt).SetString("0x3635c9adc5dea00000") // 10^21
	if err != nil {
		return fmt.Errorf("default prefunded account balance: %v", err)
	}

	erc20NameStorageSlot, err := new(felt.Felt).SetString("0x0341c1bdfd89f69748aa00b5742b03adbffd79b8e80cab5c50d91cd8c2a79be1")
	if err != nil {
		return fmt.Errorf("erc20 name storage slot: %v", err)
	}
	erc20SymbolStorageSlot, err := new(felt.Felt).SetString("0x00b6ce5410fca59d078ee9b2a4371a9d684c530d697c64fbef0ae6d5e8f0ac72")
	if err != nil {
		return fmt.Errorf("erc20 symbol storage slot: %v", err)
	}
	erc20DecimalsStorageSlot, err := new(felt.Felt).SetString("0x01f0d4aa99431d246bac9b8e48c33e888245b15e9678f64f9bdfc8823dc8f979")
	if err != nil {
		return fmt.Errorf("erc20 decimals storage slot: %v", err)
	}

	// Store genesis state.

	stateRoot := hexToFelt("0xd149d5719ffbce57fca2673f412b88565f1c2ebf37b570efc3034b18545c45")
	emptyReceipts := []*core.TransactionReceipt{}
	block := &core.Block{
		Header: &core.Header{
			Hash:             nil,
			GlobalStateRoot:  stateRoot,
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
	if err = b.chain.Store(block, commitments, &core.StateUpdate{
		BlockHash: blockHash,
		NewRoot:   stateRoot,
		OldRoot:   new(felt.Felt),
		StateDiff: &core.StateDiff{
			StorageDiffs: map[felt.Felt][]core.StorageDiff{
				*feeTokenAddress: {
					{Key: erc20NameStorageSlot, Value: hexToFelt("0x4574686572")},
					{Key: erc20SymbolStorageSlot, Value: hexToFelt("0x455448")},
					{Key: erc20DecimalsStorageSlot, Value: new(felt.Felt).SetUint64(18)},
					{Key: getStorageVarAddress("ERC20_balances", testAddress), Value: defaultPrefundedAccountBalance},
					{Key: getStorageVarAddress("ERC20_balances", testAddress2), Value: defaultPrefundedAccountBalance},
				},
				*testAddress: {
					{Key: hexToFelt("0x01379ac0624b939ceb9dede92211d7db5ee174fe28be72245b0a1a2abd81c98f"), Value: hexToFelt("0x043661740237e2be32500042dbd2afda8ab94ad11d6cea9da379ee5de3d376a2")},
				},
				*testAddress2: {
					{Key: hexToFelt("0x01379ac0624b939ceb9dede92211d7db5ee174fe28be72245b0a1a2abd81c98f"), Value: hexToFelt("0x043661740237e2be32500042dbd2afda8ab94ad11d6cea9da379ee5de3d376a2")},
				},
			},
			Nonces: map[felt.Felt]*felt.Felt{
				*feeTokenAddress: new(felt.Felt).SetUint64(1),
				*udcAddress:      new(felt.Felt).SetUint64(1),
				*testAddress:     new(felt.Felt).SetUint64(1),
				*testAddress2:    new(felt.Felt).SetUint64(1),
			},
			DeployedContracts: []core.AddressClassHashPair{
				{Address: feeTokenAddress, ClassHash: erc20ClassHash},
				{Address: udcAddress, ClassHash: udcClassHash},
				{Address: testAddress, ClassHash: accountClassHash},
				{Address: testAddress2, ClassHash: accountClassHash},
			},
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
	fmt.Println("======= RUN ")
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

		txns := b.mempool.WaitForOneTransactions()
		// txns := b.mempool.WaitForTwoTransactions()
		fmt.Println("--- Now I have two transactions. Time to execute and build.")
		// Adapt transactions to core type
		tx1, class, paidFeeOnL1, err := broadcasted.AdaptBroadcastedTransaction(txns[0], b.chain.Network())
		if err != nil {
			return fmt.Errorf("adapt broadcasted transaction: %v", err)
		}
		classes := []core.Class{}
		switch class.(type) {
		case *core.Cairo0Class, *core.Cairo1Class:
			classes = append(classes, class)
		}
		paidFeesOnL1 := []*felt.Felt{}
		declaredClasses := map[felt.Felt]core.Class{}
		switch snTx := tx1.(type) {
		case *core.L1HandlerTransaction:
			paidFeesOnL1 = append(paidFeesOnL1, paidFeeOnL1)
		case *core.DeclareTransaction:
			declaredClasses[*snTx.ClassHash] = class
		}

		// Adapt transactions to core type
		// tx2, class, paidFeeOnL1, err := broadcasted.AdaptBroadcastedTransaction(txns[1], b.chain.Network())
		// if err != nil {
		// 	return fmt.Errorf("adapt broadcasted transaction: %v", err)
		// }
		// switch class.(type) {
		// case *core.Cairo0Class, *core.Cairo1Class:
		// 	classes = append(classes, class)
		// }
		// switch snTx := tx2.(type) {
		// case *core.L1HandlerTransaction:
		// 	paidFeesOnL1 = append(paidFeesOnL1, paidFeeOnL1)
		// case *core.DeclareTransaction:
		// 	declaredClasses[*snTx.ClassHash] = class
		// }

		// Build up the header
		singleReceipt := []*core.TransactionReceipt{{}}
		pendingHeader := &core.Header{
			ParentHash:       curHeader.Hash,
			Number:           curHeader.Number + 1,
			GlobalStateRoot:  new(felt.Felt), // TODO
			SequencerAddress: curHeader.SequencerAddress,
			TransactionCount: 1,
			Timestamp:        uint64(time.Now().Unix()),
			ProtocolVersion:  curHeader.ProtocolVersion,
			GasPrice:         new(felt.Felt),
			EventCount:       0,                               // TODO
			EventsBloom:      core.EventsBloom(singleReceipt), // TODO
		}

		txs := []core.Transaction{tx1}
		stateReader, stateCloser, err := b.chain.HeadState()
		if err != nil {
			return fmt.Errorf("head state: %v", err)
		}

		// cState := NewCachedState(stateReader, pendingHeader.Number)

		// Execute all the transactions in sequnce
		_, traces, err := b.starknetVM.Execute(txs, classes, pendingHeader.Number, pendingHeader.Timestamp, pendingHeader.SequencerAddress, stateReader, b.chain.Network(), paidFeesOnL1, false, new(felt.Felt), false)
		stateCloser()
		if err != nil {
			fmt.Println("=== Execution Failed")
			return fmt.Errorf("execute transaction: %v", err)
		}
		fmt.Println("=== Execution Succeeded")
		tStateDiffs := make([]*core.StateDiff, len(traces))
		for i, trace := range traces {
			if tStateDiffs[i], err = vm2core.TraceToStateDiff(trace); err != nil {
				return err
			}
		}
		mergedStateDiffs, err := mergeStateDiffs(tStateDiffs)
		fmt.Println("---")
		fmt.Println(vm2core.TraceToStateDiff(traces[0]))
		fmt.Println(mergedStateDiffs)

		block := &core.Block{
			Header:       pendingHeader,
			Receipts:     singleReceipt, // TODO
			Transactions: txs,
		}
		blockHash, commitments, err := core.BlockHash(block, utils.GOERLI2, sequencerAddress)
		if err != nil {
			return fmt.Errorf("block hash: %v", err)
		}
		block.Hash = blockHash
		fmt.Println("=== About to call b.chain.Store()")
		if err = b.chain.Store(block, commitments, &core.StateUpdate{
			BlockHash: blockHash,
			// TODO. There isn't a good way to get this when we're sequencing. We need to refactor core/state.go and core/state_update.go.
			NewRoot:   new(felt.Felt),
			OldRoot:   curHeader.GlobalStateRoot,
			StateDiff: mergedStateDiffs,
		}, declaredClasses); err != nil {
			return fmt.Errorf("store: %v", err)
		}
		fmt.Printf("stored block %d\n", block.Number)
		fmt.Printf("  transaction 1 hash: %s\n", tx1.Hash())
	}
	return nil
}

var (
	patiricaUpperBound  = hexToFelt("0x800000000000000000000000000000000000000000000000000000000000000") // 2^251
	l2AddressUpperBound = new(felt.Felt).Sub(patiricaUpperBound, new(felt.Felt).SetUint64(256))
)

func getStorageVarAddress(name string, args ...*felt.Felt) *felt.Felt {
	nameKeccak, err := crypto.StarknetKeccak([]byte(name))
	if err != nil {
		panic(fmt.Errorf("starknet keccak: %v", err))
	}
	x := crypto.PedersenArray(slices.Insert(args, 0, nameKeccak)...)
	if x.Cmp(l2AddressUpperBound) == -1 {
		return x
	}
	return new(felt.Felt).Sub(x, l2AddressUpperBound)
}

/*
starkli deploy --account myaccount 0x04d07e40e93398ed3c76981e72dd1fd22557a78ce36c0515f679e27f0bb5bc5f --keystore keystore
*/

// TODO: This is not efficient, we should update *core.StateDiff
func mergeStateDiffs(tStateDiffs []*core.StateDiff) (*core.StateDiff, error) {
	mergedStateDiff := &core.StateDiff{
		StorageDiffs:      make(map[felt.Felt][]core.StorageDiff),
		Nonces:            make(map[felt.Felt]*felt.Felt),
		DeployedContracts: make([]core.AddressClassHashPair, 0),
		DeclaredV0Classes: make([]*felt.Felt, 0),
		DeclaredV1Classes: make([]core.DeclaredV1Class, 0),
		ReplacedClasses:   make([]core.AddressClassHashPair, 0),
	}

	for _, tStateDiff := range tStateDiffs {
		for newAddr, newStorageDiffs := range tStateDiff.StorageDiffs {
			if mergedStateDiff.StorageDiffs[newAddr] == nil {
				mergedStateDiff.StorageDiffs[newAddr] = append(mergedStateDiff.StorageDiffs[newAddr], newStorageDiffs...)
			} else {
				for _, newStorageDiff := range newStorageDiffs {
					keyExists := false
					for i := range mergedStateDiff.StorageDiffs[newAddr] {
						if mergedStateDiff.StorageDiffs[newAddr][i].Key.Cmp(newStorageDiff.Key) == 0 {
							mergedStateDiff.StorageDiffs[newAddr][i].Value = newStorageDiff.Value
							keyExists = true
							break
						}
					}
					if !keyExists {
						mergedStateDiff.StorageDiffs[newAddr] = append(mergedStateDiff.StorageDiffs[newAddr], newStorageDiff)
					}
				}
			}
		}

		// Nonces
		for tAddr, tNonce := range tStateDiff.Nonces {
			mergedStateDiff.Nonces[tAddr] = tNonce
		}

		// Deployed contracts
		for _, tAddressClashHashPair := range tStateDiff.DeployedContracts {
			if mergedStateDiff.DeployedContracts == nil {
				mergedStateDiff.DeployedContracts = append(mergedStateDiff.DeployedContracts, tAddressClashHashPair)
			} else {
				alreadyExists := false
				for _, mergedAddressClassHashPair := range mergedStateDiff.DeployedContracts {
					if mergedAddressClassHashPair.Address.Cmp(tAddressClashHashPair.Address) == 0 && mergedAddressClassHashPair.ClassHash.Cmp(tAddressClashHashPair.ClassHash) == 0 {
						alreadyExists = true
						break
					}
				}
				if !alreadyExists {
					mergedStateDiff.DeployedContracts = append(mergedStateDiff.DeployedContracts, tAddressClashHashPair)
				}
			}
		}

		// DeclaredV0 classes
		for _, tClassHash := range tStateDiff.DeclaredV0Classes {
			alreadyExists := false
			for _, mergedClassHash := range mergedStateDiff.DeclaredV0Classes {
				if mergedClassHash.Cmp(tClassHash) == 0 {
					alreadyExists = true
					break
				}
			}
			if !alreadyExists {
				mergedStateDiff.DeclaredV0Classes = append(mergedStateDiff.DeclaredV0Classes, tClassHash)
			}
		}

		// DeclaredV1 classes
		for _, tClass := range tStateDiff.DeclaredV1Classes {
			alreadyExists := false
			for _, mergedClass := range mergedStateDiff.DeclaredV1Classes {
				if mergedClass.ClassHash.Cmp(tClass.ClassHash) == 0 && mergedClass.CompiledClassHash.Cmp(tClass.CompiledClassHash) == 0 {
					alreadyExists = true
					break
				}
			}
			if !alreadyExists {
				mergedStateDiff.DeclaredV1Classes = append(mergedStateDiff.DeclaredV1Classes, tClass)
			}
		}

		// Replaced classes
		for _, tAddressClassHashPair := range tStateDiff.ReplacedClasses {
			alreadyExists := false
			for _, mergedAddressClassHashPair := range mergedStateDiff.ReplacedClasses {
				if mergedAddressClassHashPair.Address.Cmp(tAddressClassHashPair.Address) == 0 && mergedAddressClassHashPair.ClassHash.Cmp(tAddressClassHashPair.ClassHash) == 0 {
					alreadyExists = true
					break
				}
			}
			if !alreadyExists {
				mergedStateDiff.ReplacedClasses = append(mergedStateDiff.ReplacedClasses, tAddressClassHashPair)
			}
		}

	}

	return mergedStateDiff, nil
}
