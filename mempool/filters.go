package mempool

import (
	"github.com/NethermindEth/juno/core"
	"github.com/NethermindEth/juno/core/felt"
)

func FilterZeroHash(tx *core.Transaction) bool{
	if (*tx).Hash().Equal(&felt.Zero){
		return false
	}
	return true
}

func FilterOneHash(tx *core.Transaction) bool{
	if (*tx).Hash().Equal(new(felt.Felt).SetUint64(1)){
		return false
	}
	return true
}