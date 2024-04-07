package data

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"slices"
	"strings"
	"sync"

	"github.com/MartianGreed/memo-backend/pkg/starknet"
)

const MaxTokenId = 50

type Collection struct {
	inner map[int]Attributes
	sync.Mutex
}

func (c *Collection) GetPairs() []Attributes {
	v := getTokenIds()
	pairs := make([]int, 30)
	for i := 1; i < 31; i++ {
		r := rand.Intn(31-1) + 1
		if slices.Contains(pairs, v[r]) {
			i--
			continue
		}
		pairs[i-1] = v[r]
	}
	var attributes []Attributes
	for _, i := range pairs {
		attributes = append(attributes, c.inner[i])
	}
	return attributes
}

type Attributes struct {
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Image       string              `json:"image"`
	Attributes  []map[string]string `json:"attributes"`
}

func LoadCollection() *Collection {
	slog.Info("load collection")
	collection := &Collection{inner: make(map[int]Attributes)}
	rpc := starknet.MainnetJsonRpcStarknetClient()
	uriCh := make(chan string)
	for i := 1; i <= MaxTokenId; i++ {
		go fetchBloblertTokenUri(rpc, i, uriCh)
		uri := <-uriCh
		go appendToCollection(collection, i, uri)
	}

	slog.Info("collection loaded")
	return collection
}

func fetchBloblertTokenUri(rpc starknet.StarknetRpcClient, i int, uriCh chan string) {
	uri, err := starknet.GetTokenUri(rpc, "0x00539f522b29ae9251dbf7443c7a950cf260372e69efab3710a11bf17a9599f1", i)
	if err != nil {
		slog.Error(fmt.Sprintf("failed to fetch blobert id : %d", i), "error", err)
	}

	uriCh <- uri
}

func appendToCollection(c *Collection, i int, uri string) {
	uri = strings.Replace(uri, "data:application/json;base64,", "", 1)
	uri = decodeBase64(uri)
	attr := parseAttributesFromJsonStr(uri)

	c.inner[i] = attr
}

func decodeBase64(s string) string {
	decoded, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		slog.Error("failed to decode base64", "error", err)
	}
	return string(decoded)
}

func parseAttributesFromJsonStr(s string) Attributes {
	var attr Attributes
	err := json.Unmarshal([]byte(s), &attr)
	if err != nil {
		slog.Error("failed to parse json string", "error", err)
	}
	return attr
}

func getTokenIds() []int {
	var ids []int
	for i := 1; i <= MaxTokenId; i++ {
		ids = append(ids, i)
	}
	rand.Shuffle(len(ids), func(i, j int) { ids[i], ids[j] = ids[j], ids[i] })
	return ids
}
