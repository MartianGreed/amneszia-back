package game

import (
	"math/rand"

	"github.com/NethermindEth/juno/core/felt"

	"github.com/MartianGreed/memo-backend/pkg/data"
	"github.com/MartianGreed/memo-backend/pkg/starknet"
	"github.com/NethermindEth/juno/core/crypto"
)

type Tile struct {
	Attr     *data.Attributes `json:"attr"`
	Revealed bool             `json:"revealed"`
}
type Board struct {
	grid     [][]data.Attributes `json:"grid"`
	secrets   []FeltPair
	pubkeys   []*felt.Felt
	Revealed [][]Tile            `json:"revealed"`
	priv_g1 felt.Felt
	priv_g2 felt.Felt
}

type FeltPair struct {
	key   felt.Felt
	other bool
}

type DistinguishedPair struct {
	data  data.Attributes
	other bool
}

func CreateBoard(collection *data.Collection) *Board {
	server_seed := starknet.FeltFromInt(rand.Intn(9999999))
	priv_g1 := starknet.FeltFromInt(rand.Intn(9999999) + 1)
	priv_g2 := starknet.FeltFromInt(rand.Intn(9999999) + 1)

	pairs := collection.GetPairs()
	var originals []DistinguishedPair
	var copies []DistinguishedPair
	var tiles []DistinguishedPair

	originals = Map(pairs, func(p data.Attributes) DistinguishedPair { return DistinguishedPair{data: p, other: false} })
	copies = Map(pairs, func(p data.Attributes) DistinguishedPair { return DistinguishedPair{data: p, other: true} })
	tiles = append(originals, copies...)

	rand.Shuffle(len(tiles), func(i, j int) { tiles[i], tiles[j] = tiles[j], tiles[i] })

	// create 6x10 grid
	// place randomly 30 pairs of cards
	var grid [][]data.Attributes
	var secrets []FeltPair

	count := 0
	for i := 0; i < 6; i++ {
		var row []data.Attributes
		for j := 0; j < 10; j++ {
			row = append(row, tiles[count].data)

			tokenId := starknet.FeltFromInt(tiles[count].data.TokenId)
			key := crypto.PoseidonArray(server_seed, tokenId)
			secrets = append(secrets, FeltPair{key: *key, other: tiles[count].other})
			count++
		}
		grid = append(grid, row)
	}

	pubkeys := GenPublicKeys(secrets, *priv_g1, *priv_g2)

	return &Board{
		grid: grid,
		Revealed: [][]Tile{
			make([]Tile, 10),
			make([]Tile, 10),
			make([]Tile, 10),
			make([]Tile, 10),
			make([]Tile, 10),
			make([]Tile, 10),
		},
		secrets: secrets,
		pubkeys: pubkeys,
		priv_g1: *priv_g1,
		priv_g2: *priv_g2,
	}
}

func Map[T, U any](ts []T, f func(T) U) []U {
	us := make([]U, len(ts))
	for i := range ts {
		us[i] = f(ts[i])
	}
	return us
}

