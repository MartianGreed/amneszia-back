package data

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"os"
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
	TokenId     int                 `json:"token_id"`
}

func LoadCollection() *Collection {
	slog.Info("load collection")
	collection := &Collection{inner: make(map[int]Attributes)}
	rpc := starknet.MainnetJsonRpcStarknetClient()
	uriCh := make(chan string)
	for i := 1; i <= MaxTokenId; i++ {
		go fetchBloblertTokenUri(rpc, i, uriCh)
		uri := <-uriCh
		appendToCollection(collection, i, uri)
	}

	slog.Info("collection loaded")
	return collection
}

func fetchBloblertTokenUri(rpc starknet.StarknetRpcClient, i int, uriCh chan string) {
	// check if file i.json exists in filesystem
	fName := fmt.Sprintf("data/%d.json", i)
	if fileExists(fName) {
		uriCh <- readFromFile(fName)
		return
	}

	uri, err := starknet.GetTokenUri(rpc, "0x00539f522b29ae9251dbf7443c7a950cf260372e69efab3710a11bf17a9599f1", i)
	if err != nil {
		slog.Error(fmt.Sprintf("failed to fetch blobert id : %d", i), "error", err)
	}

	go writeToFile(fName, uri)

	uriCh <- uri
}

func fileExists(fName string) bool {
	_, err := os.Stat(fName)
	return !os.IsNotExist(err)
}

func readFromFile(fName string) string {
	f, err := os.Open(fName)
	if err != nil {
		slog.Error("failed to open file", "error", err)
	}
	defer f.Close()
	b, err := io.ReadAll(f)
	if err != nil {
		slog.Error("failed to read file", "error", err)
	}
	return string(b)
}

func writeToFile(fName, data string) {
	f, err := os.Create(fName)
	if err != nil {
		slog.Error("failed to create file", "error", err)
	}
	defer f.Close()
	_, err = f.WriteString(data)
	if err != nil {
		slog.Error("failed to write to file", "error", err)
	}
}

func appendToCollection(c *Collection, i int, uri string) {
	uri = strings.Replace(uri, "data:application/json;base64,", "", 1)
	uri = decodeBase64(uri)
	attr := parseAttributesFromJsonStr(uri)
	attr.TokenId = i

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
