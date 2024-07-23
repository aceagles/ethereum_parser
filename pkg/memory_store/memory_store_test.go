package memorystore

import (
	"testing"

	"github.com/aceagles/etherum_parser/pkg/eth_observer"
	"github.com/stretchr/testify/assert"
)

func Test_memStores(t *testing.T) {
	type args struct {
		address      string
		transactions []eth_observer.Transaction
	}
	tests := []struct {
		name             string
		m                *memStore
		args             args
		wantTransactions []eth_observer.Transaction
	}{
		{
			name: "Add new address",
			m:    NewMemStore(),
			args: args{
				address: "0x123",
				transactions: []eth_observer.Transaction{
					{
						Hash: "0x123",
					},
				},
			},
			wantTransactions: []eth_observer.Transaction{
				{
					Hash: "0x123",
				},
			},
		},
		{
			name: "Add transactions to existing address",
			m:    &memStore{transactions: map[string][]eth_observer.Transaction{"0x123": {}}},
			args: args{
				address: "0x123",
				transactions: []eth_observer.Transaction{
					{
						Hash: "0x456",
					},
				},
			},
			wantTransactions: []eth_observer.Transaction{
				{
					Hash: "0x456",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.m.AddTransactions(tt.args.address, tt.args.transactions)
			assert.Equal(t, tt.wantTransactions, tt.m.GetTransactions(tt.args.address))
		})
	}
}
