package starknet

import (
	"github.com/NethermindEth/juno/core/felt"
	"golang.org/x/crypto/sha3"
)

var Zero = FeltFromInt(0)

func FeltFromInt(i int) *felt.Felt {
	var felt felt.Felt
	felt.SetUint64(uint64(i))
	return &felt
}

func StarknetKeccak(b []byte) (*felt.Felt, error) {
	h := sha3.NewLegacyKeccak256()
	_, err := h.Write(b)
	if err != nil {
		return nil, err
	}
	d := h.Sum(nil)
	// Remove the first 6 bits from the first byte
	d[0] &= 3
	return new(felt.Felt).SetBytes(d), nil
}
