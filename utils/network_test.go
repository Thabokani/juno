package utils_test

import (
	"math/big"
	"strings"
	"testing"

	"github.com/NethermindEth/juno/core/felt"
	"github.com/NethermindEth/juno/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var networkStrings = map[utils.Network]string{
	utils.MAINNET:     "mainnet",
	utils.GOERLI:      "goerli",
	utils.GOERLI2:     "goerli2",
	utils.INTEGRATION: "integration",
}

func TestNetwork(t *testing.T) {
	t.Run("string", func(t *testing.T) {
		for network, str := range networkStrings {
			assert.Equal(t, str, network.String())
		}
	})
	t.Run("url", func(t *testing.T) {
		for n := range networkStrings {
			switch n {
			case utils.GOERLI:
				assert.Equal(t, "https://alpha4.starknet.io/feeder_gateway/", n.FeederURL())
			case utils.MAINNET:
				assert.Equal(t, "https://alpha-mainnet.starknet.io/feeder_gateway/", n.FeederURL())
			case utils.GOERLI2:
				assert.Equal(t, "https://alpha4-2.starknet.io/feeder_gateway/", n.FeederURL())
			case utils.INTEGRATION:
				assert.Equal(t, "https://external.integration.starknet.io/feeder_gateway/", n.FeederURL())
			default:
				assert.Fail(t, "unexpected network")
			}
		}
	})
	t.Run("chainId", func(t *testing.T) {
		for n := range networkStrings {
			switch n {
			case utils.GOERLI, utils.INTEGRATION:
				assert.Equal(t, new(felt.Felt).SetBytes([]byte("SN_GOERLI")), n.ChainID)
			case utils.MAINNET:
				assert.Equal(t, new(felt.Felt).SetBytes([]byte("SN_MAIN")), n.ChainID)
			case utils.GOERLI2:
				assert.Equal(t, new(felt.Felt).SetBytes([]byte("SN_GOERLI2")), n.ChainID)
			default:
				assert.Fail(t, "unexpected network")
			}
		}
	})
	t.Run("default L1 chainId", func(t *testing.T) {
		for n := range networkStrings {
			got := n.DefaultL1ChainID()
			switch n {
			case utils.MAINNET:
				assert.Equal(t, big.NewInt(1), got)
			case utils.GOERLI, utils.GOERLI2:
				assert.Equal(t, big.NewInt(5), got)
			case utils.INTEGRATION:
				assert.Nil(t, got)
			default:
				assert.Fail(t, "unexpected network")
			}
		}
	})
}

//nolint:dupl // see comment in utils/log_test.go
func TestNetworkSet(t *testing.T) {
	for network, str := range networkStrings {
		t.Run("network "+str, func(t *testing.T) {
			n := new(utils.Network)
			require.NoError(t, n.Set(str))
			assert.Equal(t, network, *n)
		})
		uppercase := strings.ToUpper(str)
		t.Run("network "+uppercase, func(t *testing.T) {
			n := new(utils.Network)
			require.NoError(t, n.Set(uppercase))
			assert.Equal(t, network, *n)
		})
	}

	t.Run("custom network - success", func(t *testing.T) {
		n := new(utils.Network)
		networkJSON := `{
				"name": "custom",
				"base_url": "baseURL/",
				"chain_id": "SN_CUSTOM",
				"l1_chain_id": "123",
				"core_contract_address": "0x0",
				"block_hash_meta_info": {
					"first_07_block": 1,
					"unverifiable_range": [2, 3],
					"fallback_sequencer_address": "0x0"
				}
		}`

		require.NoError(t, n.Set(networkJSON))
		assert.Equal(t, "custom", n.String())
		assert.Equal(t, "baseURL/feeder_gateway/", n.FeederURL())
		assert.Equal(t, "SN_CUSTOM", n.ChainID)
		assert.Equal(t, "0x0", n.MetaInfo().FallBackSequencerAddress.String())
		assert.Equal(t, uint64(1), n.MetaInfo().First07Block)
		assert.Equal(t, []uint64{2, 3}, n.MetaInfo().UnverifiableRange)
	})

	t.Run("custom network - fail - name error", func(t *testing.T) {
		n := new(utils.Network)
		networkJSON := `{
				"name": "ccstom",
				"base_url": "baseURL/",
				"chain_id": "SN_CUSTOM",
				"l1_chain_id": "123",
				"core_contract_address": "0x0",
				"block_hash_meta_info": {
					"first_07_block": 1,
					"unverifiable_range": [2, 3],
					"fallback_sequencer_address": "0x0"
				}
		}`
		require.ErrorIs(t, n.Set(networkJSON), utils.ErrUnknownNetwork)
	})

	t.Run("custom network - fail - base_url not set", func(t *testing.T) {
		n := new(utils.Network)
		networkJSON := `{
				"name": "custom"
		}`
		require.Equal(t, n.Set(networkJSON).Error(), "failed to unmarhsal the json string to a custom Network struct - no base_url field")
	})
}

//nolint:dupl
func TestNetworkUnmarshalText(t *testing.T) {
	for network, str := range networkStrings {
		t.Run("network "+str, func(t *testing.T) {
			n := new(utils.Network)
			require.NoError(t, n.UnmarshalText([]byte(str)))
			assert.Equal(t, network, *n)
		})
		uppercase := strings.ToUpper(str)
		t.Run("network "+uppercase, func(t *testing.T) {
			n := new(utils.Network)
			require.NoError(t, n.UnmarshalText([]byte(uppercase)))
			assert.Equal(t, network, *n)
		})
	}

	t.Run("unknown network", func(t *testing.T) {
		l := new(utils.Network)
		require.ErrorIs(t, l.UnmarshalText([]byte("blah")), utils.ErrUnknownNetwork)
	})
}

func TestNetworkMarshalJSON(t *testing.T) {
	for network, str := range networkStrings {
		t.Run("network "+str, func(t *testing.T) {
			nb, err := network.MarshalJSON()
			require.NoError(t, err)

			expectedStr := `"` + str + `"`
			assert.Equal(t, expectedStr, string(nb))
		})
	}
}

func TestNetworkType(t *testing.T) {
	assert.Equal(t, "Network", new(utils.Network).Type())
}

func TestCoreContractAddress(t *testing.T) {
	addresses := map[utils.Network]common.Address{
		utils.MAINNET: common.HexToAddress("0xc662c410C0ECf747543f5bA90660f6ABeBD9C8c4"),
		utils.GOERLI:  common.HexToAddress("0xde29d060D45901Fb19ED6C6e959EB22d8626708e"),
		utils.GOERLI2: common.HexToAddress("0xa4eD3aD27c294565cB0DCc993BDdCC75432D498c"),
	}

	for n := range networkStrings {
		t.Run("core contract for "+n.String(), func(t *testing.T) {
			switch n {
			case utils.INTEGRATION:
				require.NotNil(t, n.CoreContractAddress)
			default:
				require.NotNil(t, n.CoreContractAddress)
				want := addresses[n]
				assert.Equal(t, want, n.CoreContractAddress)
			}
		})
	}
}
