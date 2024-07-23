package main

import (
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"strings"

	"github.com/aceagles/etherum_parser/pkg/eth_observer"
	memorystore "github.com/aceagles/etherum_parser/pkg/memory_store"
)

func main() {
	slog.SetLogLoggerLevel(slog.LevelDebug)

	// Create a memory store to hold transactions
	memoryStore := memorystore.NewMemStore()

	// Create an observer to watch the ethereum chain
	ethObserver := eth_observer.NewEthereumObserver("https://cloudflare-eth.com", memoryStore)
	go ethObserver.ObserveChain() // Start observing the chain

	// Define rest api for interfacing with the observer
	// in practice the observer would be passed to a notification handler using the Parser interface
	http.HandleFunc("/getLatestBlock", func(w http.ResponseWriter, r *http.Request) {
		latestBlock := struct {
			LatestBlock int `json:"latestBlock"`
		}{
			LatestBlock: ethObserver.GetCurrentBlock(),
		}
		err := json.NewEncoder(w).Encode(latestBlock)
		if err != nil {
			http.Error(w, "Error encoding response", http.StatusInternalServerError)
		}
	})

	http.HandleFunc("/getTransactions", func(w http.ResponseWriter, r *http.Request) {
		transactionsResponse := struct {
			Transactions []eth_observer.Transaction `json:"transactions"`
		}{
			Transactions: ethObserver.GetTransactions(r.URL.Query().Get("address")),
		}
		err := json.NewEncoder(w).Encode(transactionsResponse)
		if err != nil {
			http.Error(w, "Error encoding response", http.StatusInternalServerError)
		}
	})

	http.HandleFunc("/subscribe", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
			return
		}
		decoder := json.NewDecoder(r.Body)
		var t struct {
			Address string `json:"address"`
		}
		err := decoder.Decode(&t)
		if err != nil {
			fmt.Fprintf(w, "Error decoding request: %v", err)
			return
		}
		ethObserver.Subscribe(strings.ToLower(t.Address))
		fmt.Fprintf(w, "Subscribed to address: %s", t.Address)
	})

	log.Fatal(http.ListenAndServe(":8081", nil))

}
