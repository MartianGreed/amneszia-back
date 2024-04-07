package game

import (
	"math/rand"

	"github.com/MartianGreed/memo-backend/pkg/data"
)

type Tile struct {
	Attr     *data.Attributes `json:"attr"`
	Revealed bool             `json:"revealed"`
}
type Board struct {
	grid     [][]data.Attributes `json:"grid"`
	Revealed [][]Tile            `json:"revealed"`
}

func CreateBoard(collection *data.Collection) *Board {
	pairs := collection.GetPairs()
	pairs = append(pairs, pairs...)

	rand.Shuffle(len(pairs), func(i, j int) { pairs[i], pairs[j] = pairs[j], pairs[i] })

	// create 6x10 grid
	// place randomly 30 pairs of cards
	var grid [][]data.Attributes
	for i := 0; i < 6; i++ {
		var row []data.Attributes
		for j := 0; j < 10; j++ {
			row = append(row, pairs[i+1*j+1])
		}
		grid = append(grid, row)
	}
	row := make([]Tile, 10)

	return &Board{
		grid: grid,
		Revealed: [][]Tile{
			row,
			row,
			row,
			row,
			row,
			row,
		},
	}
}
