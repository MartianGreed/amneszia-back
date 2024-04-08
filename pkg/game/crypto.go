package game

import (
	"math/big"

	"github.com/MartianGreed/memo-backend/pkg/starknet"
	"github.com/NethermindEth/juno/core/crypto"
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

func GenMatchProof(board Board, index1 int, index2 int, secret_key felt.Felt) (felt.Felt, felt.Felt) {
	_, g := starkcurve.Generators()
	g1 := g.ScalarMultiplication(&g, board.priv_g1.BigInt(new(big.Int)))
	g2 := g.ScalarMultiplication(&g, board.priv_g2.BigInt(new(big.Int)))
	k := starknet.FeltFromInt(99999999999999999)
	A := g1.ScalarMultiplication(g1, k.BigInt(new(big.Int)))
	B := g2.ScalarMultiplication(g2, k.BigInt(new(big.Int)))


	// Generate hash from data:
	// [g1_x, g2_x, y_x, z_x, A_x, B_x];
	var c felt.Felt
	if board.secrets[index1].other {
		c = *crypto.PoseidonArray(felt.NewFelt(&g1.X), felt.NewFelt(&g2.X), board.pubkeys[index1], board.pubkeys[index2], felt.NewFelt(&A.X), felt.NewFelt(&B.X))
	} else {
		c=*crypto.PoseidonArray(felt.NewFelt(&g1.X), felt.NewFelt(&g2.X), board.pubkeys[index2], board.pubkeys[index1], felt.NewFelt(&A.X), felt.NewFelt(&B.X))
	}
	s := k.Add(k, c.Mul(&c, &secret_key))
	return c,*s
}