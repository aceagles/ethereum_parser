package memorystore

import (
	"github.com/aceagles/etherum_parser/pkg/eth_observer"
)

// memStore is an in-memory store for transactions
// it implements the TransactionStore interface
type memStore struct {
	transactions map[string][]eth_observer.Transaction
}

// NewMemStore creates a new memStore
func NewMemStore() *memStore {
	return &memStore{transactions: make(map[string][]eth_observer.Transaction)}
}

// AddTransactions adds transactions to the store for a given address
func (m *memStore) AddTransactions(address string, transactions []eth_observer.Transaction) {
	m.transactions[address] = append(m.transactions[address], transactions...)
}

// GetTransactions returns transactions for a given address
func (m *memStore) GetTransactions(address string) []eth_observer.Transaction {
	return m.transactions[address]
}
