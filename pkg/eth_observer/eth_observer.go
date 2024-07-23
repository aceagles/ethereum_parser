package eth_observer

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Parser interface for parsing ethereum transactions
type Parser interface {
	// last parsed block
	GetCurrentBlock() int
	// add address to observer
	Subscribe(address string) bool
	// list of inbound or outbound transactions for an address
	GetTransactions(address string) []Transaction
}

type Transaction struct {
	BlockHash            string        `json:"blockHash"`
	BlockNumber          string        `json:"blockNumber"`
	From                 string        `json:"from"`
	Gas                  string        `json:"gas"`
	GasPrice             string        `json:"gasPrice"`
	MaxFeePerGas         string        `json:"maxFeePerGas"`
	MaxPriorityFeePerGas string        `json:"maxPriorityFeePerGas"`
	Hash                 string        `json:"hash"`
	Input                string        `json:"input"`
	Nonce                string        `json:"nonce"`
	To                   string        `json:"to"`
	TransactionIndex     string        `json:"transactionIndex"`
	Value                string        `json:"value"`
	Type                 string        `json:"type"`
	AccessList           []interface{} `json:"accessList"`
	ChainId              string        `json:"chainId"`
	V                    string        `json:"v"`
	R                    string        `json:"r"`
	S                    string        `json:"s"`
	YParity              string        `json:"yParity"`
}

type block struct {
	Transactions []Transaction `json:"transactions"`
}
type EthRequestStruct struct {
	Jsonrpc string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	Id      int           `json:"id"`
}

type EthErrorStruct struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
type EthResponseStruct struct {
	Jsonrpc string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result"`
	Error   *EthErrorStruct `json:"error"`
	Id      int             `json:"id"`
}

type TransactionsStore interface {
	GetTransactions(address string) []Transaction
	AddTransactions(address string, transactions []Transaction)
}

type EthereumObserver struct {
	endpoint          string
	mux               sync.Mutex
	latestBlock       int
	blocksToRead      map[int]struct{}
	subscribedAddress map[string]struct{}
	transactionsStore TransactionsStore
}

func NewEthereumObserver(endpoint string, txStore TransactionsStore) *EthereumObserver {
	return &EthereumObserver{
		endpoint:          endpoint,
		latestBlock:       0,
		blocksToRead:      make(map[int]struct{}),
		subscribedAddress: make(map[string]struct{}),
		transactionsStore: txStore,
	}
}

// QueryEthClient sends a request to the ethereum client and returns the response
// it checks for errors in the response and returns an error if there is one
func (e *EthereumObserver) QueryEthClient(request EthRequestStruct) (EthResponseStruct, error) {

	b := new(bytes.Buffer)
	err := json.NewEncoder(b).Encode(request)
	if err != nil {
		return EthResponseStruct{}, err
	}

	resp, err := http.Post(e.endpoint, "application/json", b)
	if err != nil {
		return EthResponseStruct{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return EthResponseStruct{}, err
	}

	var response EthResponseStruct
	err = json.Unmarshal(body, &response)

	if err != nil {
		return EthResponseStruct{}, err
	}
	if response.Error != nil {
		return EthResponseStruct{}, fmt.Errorf("error code: %d, message: %s", response.Error.Code, response.Error.Message)
	}
	if response.Id != request.Id {
		return EthResponseStruct{}, errors.New("response ID does not match request ID")
	}

	return response, nil
}

// GetBlockNumber returns the current block number as a hex string
func (e *EthereumObserver) GetBlockNumber() (string, error) {
	blockNumReq := EthRequestStruct{
		Jsonrpc: "2.0",
		Method:  "eth_blockNumber",
		Id:      0,
	}

	response, err := e.QueryEthClient(blockNumReq)
	if err != nil {
		return "", err
	}

	var blockNum string
	err = json.Unmarshal(response.Result, &blockNum)
	if err != nil {
		return "", err
	}
	if blockNum[:2] != "0x" {
		return "", errors.New("invalid block number")
	}
	_, err = strconv.ParseInt(blockNum[2:], 16, 64)
	if err != nil {
		return "", err
	}

	return blockNum, nil
}

// GetBlockByNumber returns a list of transactions in a block given the block number
// transactions are returned as a list of Transaction structs. blockNum is a hex string
func (e *EthereumObserver) GetBlockByNumber(blockNum string) ([]Transaction, error) {
	blockNumReq := EthRequestStruct{
		Jsonrpc: "2.0",
		Method:  "eth_getBlockByNumber",
		Params:  []interface{}{blockNum, true},
		Id:      0,
	}

	response, err := e.QueryEthClient(blockNumReq)
	if err != nil {
		return nil, err
	}

	var blk block
	err = json.Unmarshal(response.Result, &blk)
	if err != nil {
		return nil, err
	}

	return blk.Transactions, nil
}

// collectSubscribedAddresses returns a map of transactions by address. it filters transactions
// by the subscribed addresses in the observer
func (e *EthereumObserver) collectSubscribedAddresses(transactions []Transaction) map[string][]Transaction {
	transactionsByAddress := make(map[string][]Transaction)
	for _, transaction := range transactions {
		for _, address := range []string{transaction.From, transaction.To} {
			if _, ok := e.subscribedAddress[address]; ok {
				transactionsByAddress[address] = append(transactionsByAddress[address], transaction)
				slog.Debug("Transaction added", "transaction", transaction)
			}
		}
	}
	return transactionsByAddress
}

// addBlockToRead adds a block to the list of blocks to read
func (e *EthereumObserver) addBlockToRead(blockNum int) {
	e.mux.Lock()
	defer e.mux.Unlock()
	e.blocksToRead[blockNum] = struct{}{}
}

// UpdateTransactions updates the transactions in the observer for a given block number
// it collects transactions by number, filters them by subscribed addresses and adds them to the transaction store
// if there are errors fetching the transactions, the block is added back to the list of blocks to read
// if the block number is greater than the latest block, the latest block is updated
func (e *EthereumObserver) UpdateTransactions(blockNum int) {
	slog.Debug("Updating transactions", "block", blockNum)

	// Format to hex string
	blockNumStr := fmt.Sprintf("0x%x", blockNum)
	transactions, err := e.GetBlockByNumber(blockNumStr)
	if err != nil {
		slog.Error(err.Error())
		// if error, add block back to read list
		e.addBlockToRead(blockNum)
		return
	}

	transactionsByAddress := e.collectSubscribedAddresses(transactions)
	// iterate over transactions by address and add them to the transaction store
	for address, transactions := range transactionsByAddress {
		e.transactionsStore.AddTransactions(address, transactions)
	}
	e.updateLatestBlock(blockNum)
}

// updateLatestBlock updates the latest block in the observer
// if the block number is greater than the current latest block
// it returns true if the block number was updated
func (e *EthereumObserver) updateLatestBlock(blockNum int) bool {
	e.mux.Lock()
	defer e.mux.Unlock()
	if blockNum > e.latestBlock {
		e.latestBlock = blockNum
		slog.Info("Updated latest block", "value", e.latestBlock)
		return true
	}
	return false
}

// Subscribe adds an address to the list of subscribed addresses
// it sets the address to lowercase as the input address may have EIP55 checksum encoding
// while the transactions are returned in lowercase
func (e *EthereumObserver) Subscribe(address string) bool {
	e.mux.Lock()
	defer e.mux.Unlock()
	if _, ok := e.subscribedAddress[strings.ToLower(address)]; ok {
		slog.Debug("Already subscribed to address", "address", address)
		return false
	}
	e.subscribedAddress[strings.ToLower(address)] = struct{}{}
	slog.Debug("Subscribed to address", "address", address)
	return true
}

// GetCurrentBlock returns the current block number in the observer
func (e *EthereumObserver) GetCurrentBlock() int {
	return e.latestBlock
}

// GetTransactions returns transactions for a given address
func (e *EthereumObserver) GetTransactions(address string) []Transaction {
	return e.transactionsStore.GetTransactions(strings.ToLower(address))
}

func (e *EthereumObserver) removeBlockToRead(blockNum int) {
	e.mux.Lock()
	defer e.mux.Unlock()
	delete(e.blocksToRead, blockNum)
}

// ObserveChain observes the ethereum chain and updates transactions in the observer
// it polls the ethereum client for the latest block number and checks for new blocks
// if a new block is found, it adds the block to the list of blocks to read
// it then reads the blocks and updates the transactions in the observer
// if there are no blocks to read, it waits for 10s before checking again
func (e *EthereumObserver) ObserveChain() {
	// Seed the observer with the latest block. This is to prevent parsing from the genesis block
	var blocknum int64
	for blocknum == 0 {
		blockNum, err := e.GetBlockNumber()
		if err != nil {
			slog.Error(err.Error())
			continue
		}

		blocknum, err := strconv.ParseInt(blockNum[2:], 16, 64)
		if err != nil {
			slog.Error(err.Error())
			continue
		}
		e.updateLatestBlock(int(blocknum))
	}

	// Start observing the chain
	for {
		blockNum, err := e.GetBlockNumber()
		if err != nil {
			slog.Error(err.Error())
			continue
		}
		// convert from hex string to int
		blockNumInt, err := strconv.ParseInt(blockNum[2:], 16, 64)
		if err != nil {
			slog.Error(err.Error())
			continue
		}

		// add blocks to read. Looping ensures no blocks are missed
		for i := e.latestBlock + 1; i < int(blockNumInt); i++ {
			e.addBlockToRead(i)
		}

		// update transactions for each block
		for blockNum := range e.blocksToRead {
			e.removeBlockToRead(blockNum)
			e.UpdateTransactions(blockNum)
		}

		// wait 10s if no blocks to read (they will have been added in the case of read failre in Update Transactions).
		// Avg time between blocks is 13s.
		if len(e.blocksToRead) == 0 {
			<-time.After(10 * time.Second)
		}
	}
}
