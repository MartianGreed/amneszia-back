package game

import (
	"math/big"
	"github.com/NethermindEth/juno/core/felt"
	starkcurve "github.com/consensys/gnark-crypto/ecc/stark-curve"
)

func GenPublicKeys(secret []FeltPair, priv_g1 felt.Felt, priv_g2 felt.Felt) []*felt.Felt {
	var pubkeys []*felt.Felt
	_, g := starkcurve.Generators()
	g1 := g.ScalarMultiplication(&g, priv_g1.BigInt(new(big.Int)))
	g2 := g.ScalarMultiplication(&g, priv_g2.BigInt(new(big.Int)))
	for _, s := range secret {
		if s.other {
			pubkeys = append(pubkeys, felt.NewFelt(&g1.ScalarMultiplication(g1, s.key.BigInt(new(big.Int))).X))
		} else {
			pubkeys = append(pubkeys, felt.NewFelt(&g2.ScalarMultiplication(g2, s.key.BigInt(new(big.Int))).X))
		}
	}
	return pubkeys
}