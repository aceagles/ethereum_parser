package eth_observer

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_QueryEthClient(t *testing.T) {
	tests := []struct {
		name     string
		req      EthRequestStruct
		response EthResponseStruct
		wantErr  bool
	}{
		{
			name: "Test QueryEthClient",
			req: EthRequestStruct{
				Jsonrpc: "2.0",
				Method:  "eth_blockNumber",
				Id:      0,
			},
			response: EthResponseStruct{
				Jsonrpc: "2.0",
				Result:  []byte(`"0x1b4"`),
				Id:      0,
			},
			wantErr: false,
		},
		{
			name: "Test return error",
			req: EthRequestStruct{
				Jsonrpc: "2.0",
				Method:  "eth_blockNumber",
				Id:      0,
			},
			response: EthResponseStruct{
				Jsonrpc: "2.0",
				Error:   &EthErrorStruct{Code: -32601, Message: "Method not found"},
				Id:      0,
			},
			wantErr: true,
		},
		{
			name: "Test id mismatch",
			req: EthRequestStruct{
				Jsonrpc: "2.0",
				Method:  "eth_blockNumber",
				Id:      0,
			},
			response: EthResponseStruct{
				Jsonrpc: "2.0",
				Result:  []byte(`"0x1b4"`),
				Id:      1,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(tt.response)
			}))
			defer ts.Close()
			e := NewEthereumObserver(ts.URL, nil)
			_, err := e.QueryEthClient(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("QueryEthClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})

	}
}

func TestEthereumObserver_GetBlockNumber(t *testing.T) {
	tests := []struct {
		name     string
		response EthResponseStruct
		want     string
		wantErr  bool
	}{
		{
			name: "Test GetBlockNumber",
			response: EthResponseStruct{
				Jsonrpc: "2.0",
				Result:  []byte(`"0x1b4"`),
				Id:      0,
			},
			want:    "0x1b4",
			wantErr: false,
		},
		{
			name: "Test return error",
			response: EthResponseStruct{
				Jsonrpc: "2.0",
				Error:   &EthErrorStruct{Code: -32601, Message: "Method not found"},
				Id:      0,
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "Test malformed response",
			response: EthResponseStruct{
				Jsonrpc: "2.0",
				Result:  []byte(`"qwe"`),
				Id:      0,
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(tt.response)
			}))
			defer ts.Close()
			e := NewEthereumObserver(ts.URL, nil)
			got, err := e.GetBlockNumber()
			if (err != nil) != tt.wantErr {
				t.Errorf("QueryEthClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})

	}
}

func Test_GetBlockByNumber(t *testing.T) {
	tests := []struct {
		name     string
		blockNum string
		response EthResponseStruct
		want     []Transaction
		wantErr  bool
	}{
		{
			name:     "Test GetBlockByNumber",
			blockNum: "0x1b4",
			response: EthResponseStruct{
				Jsonrpc: "2.0",
				Result:  []byte(`{"number":"0x1b4","transactions":[{"hash":"0x1","from":"0x2","to":"0x3","value":"0x4"}]}`),
				Id:      0,
			},
			want: []Transaction{
				{
					Hash:  "0x1",
					From:  "0x2",
					To:    "0x3",
					Value: "0x4",
				},
			},
			wantErr: false,
		},
		{
			name:     "Test malformed response",
			blockNum: "0x1b4",
			response: EthResponseStruct{
				Jsonrpc: "2.0",
				Result:  []byte(`{"number":"0x1b4","transactions":[{"hash":"0x1","from":"0x2","to":"0x3","value":"0x4"}]`),
				Id:      0,
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(tt.response)
			}))
			defer ts.Close()
			e := NewEthereumObserver(ts.URL, nil)
			got, err := e.GetBlockByNumber(tt.blockNum)
			if (err != nil) != tt.wantErr {
				t.Errorf("QueryEthClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})

	}
}

func TestEthereumObserver_collectSubscribedAddresses(t *testing.T) {
	type args struct {
		transactions []Transaction
	}
	tests := []struct {
		name string
		e    *EthereumObserver
		args args
		want map[string][]Transaction
	}{
		{
			name: "Test collectSubscribedAddresses",
			e:    &EthereumObserver{subscribedAddress: map[string]struct{}{"0x2": struct{}{}}},
			args: args{
				transactions: []Transaction{
					{
						Hash: "0x1",
						From: "0x2",
						To:   "0x3",
					},
				},
			},
			want: map[string][]Transaction{"0x2": {
				{
					Hash: "0x1",
					From: "0x2",
					To:   "0x3",
				},
			}},
		},
		{
			name: "Test collectSubscribedAddresses no match",
			e:    &EthereumObserver{subscribedAddress: map[string]struct{}{"0x4": struct{}{}}},
			args: args{
				transactions: []Transaction{
					{
						Hash: "0x1",
						From: "0x2",
						To:   "0x3",
					},
				},
			},
			want: map[string][]Transaction{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.e.collectSubscribedAddresses(tt.args.transactions); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("EthereumObserver.collectSubscribedAddresses() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEthereumObserver_addBlockToRead(t *testing.T) {
	tests := []struct {
		name     string
		e        *EthereumObserver
		blockNum int
		want     map[int]struct{}
	}{
		// TODO: Add test cases.
		{
			name:     "Test addBlockToRead",
			e:        &EthereumObserver{blocksToRead: map[int]struct{}{0: {}, 1: {}}},
			blockNum: 3,
			want:     map[int]struct{}{0: {}, 1: {}, 3: {}},
		},
		{
			name:     "Test addBlockToRead duplicate",
			e:        &EthereumObserver{blocksToRead: map[int]struct{}{1: {}, 2: {}}},
			blockNum: 2,
			want:     map[int]struct{}{1: {}, 2: {}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			tt.e.addBlockToRead(tt.blockNum)
			assert.Equal(t, tt.want, tt.e.blocksToRead)
		})
	}
}

func TestEthereumObserver_UpdateTransactions(t *testing.T) {
	tests := []struct {
		name             string
		e                *EthereumObserver
		blockNum         int
		wantLatestBlock  int
		wantBlocksToRead map[int]struct{}
		responseOK       bool
		wantErr          bool
	}{
		{
			name:             "Test UpdateTransactions",
			e:                &EthereumObserver{latestBlock: 0, blocksToRead: map[int]struct{}{0: {}}},
			blockNum:         1,
			wantLatestBlock:  1,
			wantBlocksToRead: map[int]struct{}{0: {}},
			responseOK:       true,
			wantErr:          false,
		},
		{
			name:             "Test UpdateTransactions error",
			e:                &EthereumObserver{latestBlock: 0, blocksToRead: map[int]struct{}{0: {}}},
			blockNum:         1,
			wantLatestBlock:  0,
			wantBlocksToRead: map[int]struct{}{0: {}, 1: {}},
			responseOK:       false,
			wantErr:          true,
		},
	}
	for _, tt := range tests {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if tt.responseOK {
				w.WriteHeader(http.StatusOK)
				sampleResponse := EthResponseStruct{
					Jsonrpc: "2.0",
					Result:  []byte(`{"number":"0x1b4","transactions":[{"hash":"0x1","from":"0x2","to":"0x3","value":"0x4"}]}`),
					Id:      0,
				}
				json.NewEncoder(w).Encode(sampleResponse)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
			}
		}))
		tt.e.endpoint = ts.URL
		t.Run(tt.name, func(t *testing.T) {
			tt.e.UpdateTransactions(tt.blockNum)
			assert.Equal(t, tt.wantLatestBlock, tt.e.latestBlock)
			assert.Equal(t, tt.wantBlocksToRead, tt.e.blocksToRead)
		})
	}
}

func TestEthereumObserver_updateLatestBlock(t *testing.T) {
	type args struct {
		blockNum int
	}
	tests := []struct {
		name string
		e    *EthereumObserver
		args args
		want bool
	}{
		{
			name: "Test updateLatestBlock",
			e:    &EthereumObserver{latestBlock: 0},
			args: args{blockNum: 1},
			want: true,
		},
		{
			name: "Test updateLatestBlock duplicate",
			e:    &EthereumObserver{latestBlock: 1},
			args: args{blockNum: 1},
			want: false,
		},
		{
			name: "Test updateLatestBlock less than",
			e:    &EthereumObserver{latestBlock: 1},
			args: args{blockNum: 0},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.e.updateLatestBlock(tt.args.blockNum); got != tt.want {
				t.Errorf("EthereumObserver.updateLatestBlock() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEthereumObserver_Subscribe(t *testing.T) {
	type args struct {
		address string
	}
	tests := []struct {
		name              string
		e                 *EthereumObserver
		args              args
		want              bool
		wantSubscriptions map[string]struct{}
	}{
		{
			name:              "Test Subscribe",
			e:                 &EthereumObserver{subscribedAddress: map[string]struct{}{}},
			args:              args{address: "0x1"},
			want:              true,
			wantSubscriptions: map[string]struct{}{"0x1": {}},
		},
		{
			name:              "Test Subscribe duplicate",
			e:                 &EthereumObserver{subscribedAddress: map[string]struct{}{"0x1": {}}},
			args:              args{address: "0x1"},
			want:              false,
			wantSubscriptions: map[string]struct{}{"0x1": {}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.e.Subscribe(tt.args.address); got != tt.want {
				t.Errorf("EthereumObserver.Subscribe() = %v, want %v", got, tt.want)
				assert.Equal(t, tt.wantSubscriptions, tt.e.subscribedAddress)
			}
		})
	}
}

func TestEthereumObserver_removeBlockToRead(t *testing.T) {
	type args struct {
		blockNum int
	}
	tests := []struct {
		name    string
		e       *EthereumObserver
		args    args
		wantMap map[int]struct{}
	}{
		// TODO: Add test cases.
		{
			name:    "Test removeBlockToRead",
			e:       &EthereumObserver{blocksToRead: map[int]struct{}{0: {}, 1: {}}},
			args:    args{blockNum: 1},
			wantMap: map[int]struct{}{0: {}},
		},
		{
			name:    "Test removeBlockToRead no match",
			e:       &EthereumObserver{blocksToRead: map[int]struct{}{0: {}, 1: {}}},
			args:    args{blockNum: 2},
			wantMap: map[int]struct{}{0: {}, 1: {}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.e.removeBlockToRead(tt.args.blockNum)
			assert.Equal(t, tt.wantMap, tt.e.blocksToRead)
		})
	}
}
