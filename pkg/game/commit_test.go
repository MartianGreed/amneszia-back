package game

import (
	"fmt"
	"log/slog"
	"math/rand"
	// "math/big"
	"strconv"
	"strings"
	"testing"

	starkcurve "github.com/consensys/gnark-crypto/ecc/stark-curve"

	"github.com/MartianGreed/memo-backend/pkg/starknet"
	"github.com/NethermindEth/juno/core/crypto"
)

func TestStarkcurve(t *testing.T) {
	_,_ = starkcurve.Generators()

	server_seed := starknet.FeltFromInt(rand.Intn(9999999))

	token, err := strconv.Atoi(strings.Split("blobert #12", "#")[1])
	if err != nil {
		slog.Error("failed to parse tokenId", "error", err)
	}
	tokenId := starknet.FeltFromInt(token)
	key := crypto.PoseidonArray(server_seed, tokenId)
	fmt.Println(key)

	var secret []FeltPair
	secret = append(secret, FeltPair{key: *starknet.FeltFromInt(1), other: false})
	secret = append(secret, FeltPair{key: *starknet.FeltFromInt(1), other: true})

	keys:= GenPublicKeys(secret, *starknet.FeltFromInt(1), *starknet.FeltFromInt(1))
	fmt.Println(keys)
}