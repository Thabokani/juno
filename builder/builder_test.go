package builder_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/NethermindEth/juno/blockchain"
	"github.com/NethermindEth/juno/builder"
	"github.com/NethermindEth/juno/clients/feeder"
	"github.com/NethermindEth/juno/core"
	"github.com/NethermindEth/juno/core/felt"
	"github.com/NethermindEth/juno/db/pebble"
	"github.com/NethermindEth/juno/mempool"
	"github.com/NethermindEth/juno/mocks"
	adaptfeeder "github.com/NethermindEth/juno/starknetdata/feeder"
	"github.com/NethermindEth/juno/utils"
	"github.com/NethermindEth/juno/vm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestValidateAgainstPendingState(t *testing.T) {
	testDB := pebble.NewMemTest(t)
	mockCtrl := gomock.NewController(t)
	mockVM := mocks.NewMockVM(mockCtrl)
	bc := blockchain.New(testDB, utils.Integration, utils.NewNopZapLogger())
	seqAddr := utils.HexToFelt(t, "0xDEADBEEF")
	testBuilder := builder.New(seqAddr, bc, mockVM, utils.NewNopZapLogger())

	client := feeder.NewTestClient(t, utils.Integration)
	gw := adaptfeeder.New(client)

	su, b, err := gw.StateUpdateWithBlock(context.Background(), 0)
	require.NoError(t, err)

	require.NoError(t, bc.StorePending(&blockchain.Pending{
		Block:       b,
		StateUpdate: su,
	}))

	userTxn := mempool.BroadcastedTransaction{
		Transaction: &core.InvokeTransaction{
			TransactionHash: utils.HexToFelt(t, "0x1337"),
		},
		DeclaredClass: &core.Cairo0Class{
			Program: "best program",
		},
	}

	mockVM.EXPECT().Execute([]core.Transaction{userTxn.Transaction},
		[]core.Class{userTxn.DeclaredClass}, uint64(0), b.Timestamp, seqAddr,
		gomock.Any(), utils.Integration, []*felt.Felt{}, false, false,
		false, b.GasPrice, b.GasPriceSTRK, false).Return(nil, nil, nil)
	assert.NoError(t, testBuilder.ValidateAgainstPendingState(&userTxn))

	require.NoError(t, bc.Store(b, &core.BlockCommitments{}, su, nil))
	mockVM.EXPECT().Execute([]core.Transaction{userTxn.Transaction},
		[]core.Class{userTxn.DeclaredClass}, uint64(1), b.Timestamp+1, seqAddr,
		gomock.Any(), utils.Integration, []*felt.Felt{}, false, false,
		false, b.GasPrice, b.GasPriceSTRK, false).Return(nil, nil, errors.New("oops"))
	assert.EqualError(t, testBuilder.ValidateAgainstPendingState(&userTxn), "oops")
}

func TestGenesisStateDiff(t *testing.T) {
	log := utils.NewNopZapLogger()
	chain := blockchain.New(pebble.NewMemTest(t), utils.Goerli, log)
	b := builder.New(new(felt.Felt).SetUint64(1), chain, vm.New(log), utils.NewNopZapLogger())
	t.Run("empty genesis config", func(t *testing.T) {
		genesisConfig := builder.GenesisConfig{}
		_, err := b.GenesisStateDiff(genesisConfig)
		require.NoError(t, err)
	})

	t.Run("valid non-empty genesis config", func(t *testing.T) {

		accountClassHash, err := new(felt.Felt).SetString("0x04d07e40e93398ed3c76981e72dd1fd22557a78ce36c0515f679e27f0bb5bc5f")
		require.NoError(t, err)
		erc20ClassHash, err := new(felt.Felt).SetString("0x02a8846878b6ad1f54f6ba46f5f40e11cee755c677f130b2c4b60566c9003f1f")
		require.NoError(t, err)

		genesisConfig := builder.GenesisConfig{
			ChainID: "SN_GOERLI",
			Classes: []string{
				"./contracts/account.json",
				"./contracts/erc20.json",
				"./contracts/udc.json",
			},
			Contracts: map[string]builder.GenesisContractData{
				"0x111": {
					ClassHash:       *accountClassHash,
					ConstructorArgs: []felt.Felt{*new(felt.Felt).SetUint64(222)}, // todo: what pub-key to use ??
				},
				"0x049d36570d4e46f48e99674bd3fcc84644ddd6b96f7c741b1562b82f9e004dc7": { // fee Token
					ClassHash:       *erc20ClassHash,
					ConstructorArgs: []felt.Felt{*new(felt.Felt).SetUint64(333)}, // todo: what pub-key to use ??
				},
				"0x041a78e741e5af2fec34b695679bc6891742439f7afb8484ecd7766661ad02bf": { // udc - lets us deploy using invoke-txns
					ClassHash:       *erc20ClassHash,
					ConstructorArgs: []felt.Felt{*new(felt.Felt).SetUint64(444)}, // todo: what pub-key to use ??
				},
			},
			// FunctionCalls: []builder.FunctionCall{
			// 	{
			// 		ContractAddress:    felt.Zero,
			// 		EntryPointSelector: felt.Zero,
			// 		Calldata:           []felt.Felt{},
			// 	},
			// },
		}

		resp, err := b.GenesisStateDiff(genesisConfig)
		fmt.Println(resp, err)
		require.NoError(t, err)

	})
}
