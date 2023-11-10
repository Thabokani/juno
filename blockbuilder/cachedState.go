package blockbuilder

import (
	"github.com/NethermindEth/juno/core"
	"github.com/NethermindEth/juno/core/felt"
)

type cachedState struct {
	stateDiff   *core.StateDiff
	stateReader *core.StateReader
}

func NewCachedState(stateReader *core.StateReader) *cachedState {
	stateDiff := &core.StateDiff{
		StorageDiffs:      make(map[felt.Felt][]core.StorageDiff),
		Nonces:            make(map[felt.Felt]*felt.Felt),
		DeployedContracts: make([]core.AddressClassHashPair, 0),
		DeclaredV0Classes: make([]*felt.Felt, 0),
		DeclaredV1Classes: make([]core.DeclaredV1Class, 0),
		ReplacedClasses:   make([]core.AddressClassHashPair, 0),
	}
	return &cachedState{
		stateDiff:   stateDiff,
		stateReader: stateReader,
	}
}

func (cs *cachedState) GetStorageDiffValue(address *felt.Felt, key *felt.Felt) (*felt.Felt, error) {
	// Check if value is present in the cache, if not read from underlying state
	if storageDiffs, ok := cs.stateDiff.StorageDiffs[*address]; ok {
		for _, storageDiff := range storageDiffs {
			if storageDiff.Key.Cmp(key) == 0 {
				return storageDiff.Value, nil
			}
		}
	}
	return (*cs.stateReader).ContractStorage(address, key)
}

func (cs *cachedState) SetStorageDiffValue(address *felt.Felt, key *felt.Felt, value *felt.Felt) {
	cs.stateDiff.StorageDiffs[*address] = append(cs.stateDiff.StorageDiffs[*address], core.StorageDiff{
		Key:   key,
		Value: value,
	})
}

func (cs *cachedState) SetNonce(address *felt.Felt, nonce *felt.Felt) {
	cs.stateDiff.Nonces[*address] = nonce
}

func (cs *cachedState) GetNonce(address *felt.Felt) (*felt.Felt, error) {
	// Check if nonce is present in the cache, if not read from underlying state
	if nonce, ok := cs.stateDiff.Nonces[*address]; ok {
		return nonce, nil
	}
	return (*cs.stateReader).ContractNonce(address)
}
