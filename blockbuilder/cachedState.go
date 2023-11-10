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

// GetStateDiff returns the cached state diff
func (cs *cachedState) GetStateDiff() *core.StateDiff {
	return cs.stateDiff
}

// GetStorageDiffValue returns the value of the storage diff for the given address and key
// If the value is not present in the cache, it is read from the underlying state
func (cs *cachedState) GetStorageDiffValue(address *felt.Felt, key *felt.Felt) (*felt.Felt, error) {
	if storageDiffs, ok := cs.stateDiff.StorageDiffs[*address]; ok {
		for _, storageDiff := range storageDiffs {
			if storageDiff.Key.Cmp(key) == 0 {
				return storageDiff.Value, nil
			}
		}
	}
	return (*cs.stateReader).ContractStorage(address, key)
}

// SetStorageDiffValue sets the value of the storage diff for the given address and key
func (cs *cachedState) SetStorageDiffValue(address *felt.Felt, key *felt.Felt, value *felt.Felt) {
	cs.stateDiff.StorageDiffs[*address] = append(cs.stateDiff.StorageDiffs[*address], core.StorageDiff{
		Key:   key,
		Value: value,
	})
}

// GetNonce returns the nonce for the given address
// If the nonce is not present in the cache, it is read from the underlying state
func (cs *cachedState) GetNonce(address *felt.Felt) (*felt.Felt, error) {
	if nonce, ok := cs.stateDiff.Nonces[*address]; ok {
		return nonce, nil
	}
	return (*cs.stateReader).ContractNonce(address)
}

// SetNonce sets the nonce for the given address
func (cs *cachedState) SetNonce(address *felt.Felt, nonce *felt.Felt) {
	cs.stateDiff.Nonces[*address] = nonce
}

// GetDeployedContracts returns the deployed contracts
func (cs *cachedState) GetDeployedContracts() []core.AddressClassHashPair {
	return cs.stateDiff.DeployedContracts
}

// SetDeployedContract sets the deployed contract
func (cs *cachedState) SetDeployedContract(address *felt.Felt, classHash *felt.Felt) {
	cs.stateDiff.DeployedContracts = append(cs.stateDiff.DeployedContracts, core.AddressClassHashPair{
		Address:   address,
		ClassHash: classHash,
	})
}

// GetDeclaredV0Classes returns the declared V0 classes
func (cs *cachedState) GetDeclaredV0Classes() []*felt.Felt {
	return cs.stateDiff.DeclaredV0Classes
}

// SetDeclaredV0Class sets the declared V0 class
func (cs *cachedState) SetDeclaredV0Class(classHash *felt.Felt) {
	cs.stateDiff.DeclaredV0Classes = append(cs.stateDiff.DeclaredV0Classes, classHash)
}

// GetDeclaredV1Classes returns the declared V1 classes
func (cs *cachedState) GetDeclaredV1Classes() []core.DeclaredV1Class {
	return cs.stateDiff.DeclaredV1Classes
}

// SetDeclaredV1Class sets the declared V1 class
func (cs *cachedState) SetDeclaredV1Class(classHash *felt.Felt, compiledClassHash *felt.Felt) {
	cs.stateDiff.DeclaredV1Classes = append(cs.stateDiff.DeclaredV1Classes, core.DeclaredV1Class{
		ClassHash:         classHash,
		CompiledClassHash: compiledClassHash,
	})
}

// GetReplacedClasses returns the replaced classes
func (cs *cachedState) GetReplacedClasses() []core.AddressClassHashPair {
	return cs.stateDiff.ReplacedClasses
}

// SetReplacedClass sets the replaced class
func (cs *cachedState) SetReplacedClass(address *felt.Felt, classHash *felt.Felt) {
	cs.stateDiff.ReplacedClasses = append(cs.stateDiff.ReplacedClasses, core.AddressClassHashPair{
		Address:   address,
		ClassHash: classHash,
	})
}
