package starknetdata_test

import (
	"context"
	"strconv"
	"testing"

	client "github.com/NethermindEth/juno/clients"
	"github.com/NethermindEth/juno/core"
	"github.com/NethermindEth/juno/starknetdata"
	"github.com/NethermindEth/juno/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBlockByNumber(t *testing.T) {
	numbers := []uint64{147, 11817}

	cli := client.NewTestClient(t, core.MAINNET)
	feeder := client.NewFeeder(cli)
	adapter := starknetdata.NewStarknetData(cli)
	ctx := context.Background()

	for _, number := range numbers {
		t.Run("mainnet block number "+strconv.FormatUint(number, 10), func(t *testing.T) {
			response, err := feeder.Block(ctx, strconv.FormatUint(number, 10))
			require.NoError(t, err)
			block, err := adapter.BlockByNumber(ctx, number)
			require.NoError(t, err)
			adaptedResponse, err := utils.AdaptBlock(response)
			require.NoError(t, err)
			assert.Equal(t, adaptedResponse, block)
		})
	}
}

func TestBlockLatest(t *testing.T) {
	cli := client.NewTestClient(t, core.MAINNET)
	feeder := client.NewFeeder(cli)
	adapter := starknetdata.NewStarknetData(cli)
	ctx := context.Background()

	response, err := feeder.Block(ctx, "latest")
	require.NoError(t, err)
	block, err := adapter.BlockLatest(ctx)
	require.NoError(t, err)
	adaptedResponse, err := utils.AdaptBlock(response)
	require.NoError(t, err)
	assert.Equal(t, adaptedResponse, block)
}

func TestStateUpdate(t *testing.T) {
	numbers := []uint64{0, 1, 2, 21656}

	cli := client.NewTestClient(t, core.MAINNET)
	feeder := client.NewFeeder(cli)
	adapter := starknetdata.NewStarknetData(cli)
	ctx := context.Background()

	for _, number := range numbers {
		t.Run("number "+strconv.FormatUint(number, 10), func(t *testing.T) {
			response, err := feeder.StateUpdate(ctx, strconv.FormatUint(number, 10))
			require.NoError(t, err)
			feederUpdate, err := adapter.StateUpdate(ctx, number)
			require.NoError(t, err)

			adaptedResponse, err := utils.AdaptStateUpdate(response)
			require.NoError(t, err)
			assert.Equal(t, adaptedResponse, feederUpdate)
		})
	}
}

func TestClassV0(t *testing.T) {
	classHashes := []string{
		"0x79e2d211e70594e687f9f788f71302e6eecb61d98efce48fbe8514948c8118",
		"0x1924aa4b0bedfd884ea749c7231bafd91650725d44c91664467ffce9bf478d0",
		"0x10455c752b86932ce552f2b0fe81a880746649b9aee7e0d842bf3f52378f9f8",
		"0x56b96c1d1bbfa01af44b465763d1b71150fa00c6c9d54c3947f57e979ff68c3",
	}

	cli := client.NewTestClient(t, core.GOERLI)
	feeder := client.NewFeeder(cli)
	adapter := starknetdata.NewStarknetData(cli)
	ctx := context.Background()

	for _, hashString := range classHashes {
		t.Run("hash "+hashString, func(t *testing.T) {
			hash := utils.HexToFelt(t, hashString)
			response, err := feeder.ClassDefinition(ctx, hash)
			require.NoError(t, err)
			classGeneric, err := adapter.Class(ctx, hash)
			require.NoError(t, err)

			adaptedResponse, err := utils.AdaptCairo0Class(response.V0)
			require.NoError(t, err)
			require.Equal(t, adaptedResponse, classGeneric)
		})
	}
}

func TestTransaction(t *testing.T) {
	clientGoerli := client.NewTestClient(t, core.GOERLI)
	feederGoerli := client.NewFeeder(clientGoerli)
	adapterGoerli := starknetdata.NewStarknetData(clientGoerli)

	clientMainnet := client.NewTestClient(t, core.MAINNET)
	feederMainnet := client.NewFeeder(clientMainnet)
	adapterMainnet := starknetdata.NewStarknetData(clientMainnet)

	ctx := context.Background()

	t.Run("invoke transaction", func(t *testing.T) {
		hash := utils.HexToFelt(t, "0x7e3a229febf47c6edfd96582d9476dd91a58a5ba3df4553ae448a14a2f132d9")
		response, err := feederGoerli.Transaction(ctx, hash)
		require.NoError(t, err)
		responseTx := response.Transaction

		txn, err := adapterGoerli.Transaction(ctx, hash)
		require.NoError(t, err)
		invokeTx, ok := txn.(*core.InvokeTransaction)
		require.True(t, ok)
		assert.Equal(t, utils.AdaptInvokeTransaction(responseTx), invokeTx)
	})

	t.Run("deploy transaction", func(t *testing.T) {
		hash := utils.HexToFelt(t, "0x15b51c2f4880b1e7492d30ada7254fc59c09adde636f37eb08cdadbd9dabebb")
		response, err := feederGoerli.Transaction(ctx, hash)
		require.NoError(t, err)
		responseTx := response.Transaction

		txn, err := adapterGoerli.Transaction(ctx, hash)
		require.NoError(t, err)
		deployTx, ok := txn.(*core.DeployTransaction)
		require.True(t, ok)
		assert.Equal(t, utils.AdaptDeployTransaction(responseTx), deployTx)
	})

	t.Run("deploy account transaction", func(t *testing.T) {
		hash := utils.HexToFelt(t, "0xd61fc89f4d1dc4dc90a014957d655d38abffd47ecea8e3fa762e3160f155f2")
		response, err := feederMainnet.Transaction(ctx, hash)
		require.NoError(t, err)
		responseTx := response.Transaction

		txn, err := adapterMainnet.Transaction(ctx, hash)
		require.NoError(t, err)
		deployAccountTx, ok := txn.(*core.DeployAccountTransaction)
		require.True(t, ok)
		assert.Equal(t, utils.AdaptDeployAccountTransaction(responseTx), deployAccountTx)
	})

	t.Run("declare transaction", func(t *testing.T) {
		hash := utils.HexToFelt(t, "0x6eab8252abfc9bbfd72c8d592dde4018d07ce467c5ce922519d7142fcab203f")
		response, err := feederGoerli.Transaction(ctx, hash)
		require.NoError(t, err)
		responseTx := response.Transaction

		txn, err := adapterGoerli.Transaction(ctx, hash)
		require.NoError(t, err)
		declareTx, ok := txn.(*core.DeclareTransaction)
		require.True(t, ok)
		assert.Equal(t, utils.AdaptDeclareTransaction(responseTx), declareTx)
	})

	t.Run("l1handler transaction", func(t *testing.T) {
		hash := utils.HexToFelt(t, "0x537eacfd3c49166eec905daff61ff7feef9c133a049ea2135cb94eec840a4a8")
		response, err := feederMainnet.Transaction(ctx, hash)
		require.NoError(t, err)
		responseTx := response.Transaction

		txn, err := adapterMainnet.Transaction(ctx, hash)
		require.NoError(t, err)
		l1HandlerTx, ok := txn.(*core.L1HandlerTransaction)
		require.True(t, ok)
		assert.Equal(t, utils.AdaptL1HandlerTransaction(responseTx), l1HandlerTx)
	})
}

func TestClassV1(t *testing.T) {
	cli := client.NewTestClient(t, core.INTEGRATION)
	feeder := client.NewFeeder(cli)
	adapter := starknetdata.NewStarknetData(cli)

	classHash := utils.HexToFelt(t, "0x1cd2edfb485241c4403254d550de0a097fa76743cd30696f714a491a454bad5")
	class, err := adapter.Class(context.Background(), classHash)
	require.NoError(t, err)

	feederClass, err := feeder.ClassDefinition(context.Background(), classHash)
	require.NoError(t, err)
	compiled, err := feeder.CompiledClassDefinition(context.Background(), classHash)
	require.NoError(t, err)

	adaptedResponse, err := utils.AdaptCairo1Class(feederClass.V1, compiled)
	require.NoError(t, err)
	assert.Equal(t, adaptedResponse, class)
}