package starknet

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"

	"github.com/NethermindEth/juno/core/felt"
)

type (
	BlockId         string
	StarknetNetwork string
)

const (
	BlockLatest  BlockId = "latest"
	BlockPending BlockId = "pending"

	Mainnet StarknetNetwork = "mainnet"
	Goerli  StarknetNetwork = "goerli"
	Sepolia StarknetNetwork = "sepolia"
)

type StarknetRpcClient interface {
	Call(address string, method string, params []felt.Felt) ([]felt.Felt, error)
}

type JsonRpcStarknetClient struct {
	Client   *http.Client
	Endpoint string
}

func (c *JsonRpcStarknetClient) Call(address string, method string, params []felt.Felt) ([]felt.Felt, error) {
	req := newRpcRequest("starknet_call", newCallRequestParams(address, method, params, BlockLatest))
	jsonBody, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequest("POST", c.Endpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}

	request.Header.Add("x-apikey", os.Getenv("RPC_API_KEY"))

	resp, err := c.Client.Do(request)
	if err != nil {
		slog.Error(err.Error())
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error(err.Error())
		return nil, err
	}

	if resp.StatusCode != 200 {
		slog.Error(fmt.Sprintf("http status code : %d", resp.StatusCode))
		slog.Error(fmt.Sprintf("response body : %s", body))
		return nil, fmt.Errorf("%s", resp.Status)
	}

	var response rpcResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		slog.Error(err.Error())
		return nil, err
	}

	return response.Result, nil
}

func GetTokenUri(rpc StarknetRpcClient, address string, tokenId int) (string, error) {
	res, err := rpc.Call(address, "token_uri", []felt.Felt{*FeltFromInt(tokenId), *Zero})
	if err != nil {
		return "", err
	}
	if len(res) == 0 {
		return "", fmt.Errorf("token %d not minted yet", tokenId)
	}

	uri := DecodeToString(res[1 : len(res)-1])
	if err != nil {
		return "", err
	}
	return uri, nil
}

type client struct {
	currentRequests chan interface{}
	baseUrl         string
}

func newCLient(baseUrl string, maxRpm int) *client {
	return &client{
		baseUrl:         baseUrl,
		currentRequests: make(chan interface{}, maxRpm),
	}
}

func (c *client) Get(path string) ([]byte, error) {
	donech := make(chan []byte)
	errch := make(chan error)

	go func() { c.currentRequests <- true }()
	go func() {
		resp, err := http.Get(fmt.Sprintf("%s/%s", c.baseUrl, path))
		if err != nil {
			errch <- err
			return
		}

		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			errch <- fmt.Errorf("invalid status code %d", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			errch <- err
		}
		donech <- body
	}()

	select {
	case body := <-donech:
		<-c.currentRequests
		return body, nil
	case err := <-errch:
		<-c.currentRequests
		return []byte(""), err
	}
}

func NewJsonRpcStarknetClient(endpoint string) *JsonRpcStarknetClient {
	return &JsonRpcStarknetClient{
		Endpoint: endpoint,
		Client:   &http.Client{},
	}
}

// Create mainnet json rpc client
func MainnetJsonRpcStarknetClient() *JsonRpcStarknetClient {
	return NewJsonRpcStarknetClient("https://rpc.nethermind.io/mainnet-juno")
}

// Create goerli json rpc client
func GoerliJsonRpcStarknetClient() *JsonRpcStarknetClient {
	return NewJsonRpcStarknetClient("https://rpc.nethermind.io/goerli-juno")
}

// Create sepolia  json rpc client
func SepoliaJsonRpcStarknetClient() *JsonRpcStarknetClient {
	return NewJsonRpcStarknetClient("https://rpc.nethermind.io/sepolia-juno")
}

type rpcRequest[T any] struct {
	Params  T      `json:"params"`
	JsonRpc string `json:"jsonrpc"`
	Method  string `json:"method"`
	Id      int8   `json:"id"`
}

func newRpcRequest[T any](method string, params T) *rpcRequest[T] {
	return &rpcRequest[T]{
		Params:  params,
		JsonRpc: "2.0",
		Method:  method,
		Id:      1,
	}
}

type rpcResponse struct {
	JsonRpc string
	Result  []felt.Felt
	Id      int8
}

func newRpcResponse(result []felt.Felt) *rpcResponse {
	return &rpcResponse{
		JsonRpc: "2.0",
		Result:  result,
		Id:      1,
	}
}

type callRequest struct {
	ContractAddress    string   `json:"contract_address"`
	EntryPointSelector string   `json:"entry_point_selector"`
	Calldata           []string `json:"calldata"`
}

type callRequestParams struct {
	BlockId BlockId     `json:"block_id"`
	Request callRequest `json:"request"`
}

func newCallRequestParams(address string, method string, params []felt.Felt, blockId BlockId) *callRequestParams {
	var callData []string
	for _, param := range params {
		callData = append(callData, param.String())
	}
	entryPoint, err := StarknetKeccak([]byte(method))
	if err != nil {
		slog.Error(err.Error())
	}

	return &callRequestParams{
		Request: callRequest{
			ContractAddress:    address,
			EntryPointSelector: entryPoint.String(),
			Calldata:           callData,
		},
		BlockId: blockId,
	}
}

func FeltArrToBytesArr(feltArr []felt.Felt) []byte {
	var bArr []byte
	for _, f := range feltArr {
		b := f.Marshal()
		bArr = append(bArr, bytes.Trim(b[0:], "\x00")...)
	}
	return bArr
}

func DecodeToString(feltArr []felt.Felt) string {
	var s string
	for _, f := range feltArr {
		b := f.Marshal()
		s += string(bytes.Trim(b[0:], "\x00"))
	}
	return s
}
