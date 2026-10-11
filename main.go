package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
)

func main() {
	client, err := NewInfraiClient()
	if err != nil {
		log.Fatal(err)
	}
	capUSD, err := strconv.ParseFloat(os.Getenv("HARD_CAP_USD"), 64)
	if err != nil || capUSD <= 0 {
		log.Fatal("HARD_CAP_USD must be a positive number")
	}
	if err := client.SetBudget(capUSD, "monthly"); err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/work-orders", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var order WorkOrder
		if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
			http.Error(w, "invalid work order", http.StatusBadRequest)
			return
		}
		if order.ID == "" || order.DispatchStatus == "" {
			http.Error(w, "id and dispatch_status are required", http.StatusBadRequest)
			return
		}
		answer, err := client.Chat(summarizeWorkOrder(order))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"request_id": requestID(order), "next_action": answer})
	})
	mux.HandleFunc("/usage", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		data, err := client.UsageTimeseries()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	})
	log.Println("field-service spend guard listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
