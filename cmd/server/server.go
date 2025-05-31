package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
)

// Train structure
type Train struct {
	ID            string `json:"id"`
	From          string `json:"from"`
	To            string `json:"to"`
	DepartureTime string `json:"departure_time"`
	ArrivalTime   string `json:"arrival_time"`
	TotalTickets  int    `json:"total_tickets"`
	Available     int    `json:"available"`
}

// Train ticket information stored in map
var (
	trains = map[string]*Train{}
	mu     sync.Mutex // Concurrency protection
)

func main() {
	// Initialize some train routes
	trains["G100"] = &Train{"G100", "Beijing", "Shanghai", "08:00", "13:30", 100, 100}
	trains["D200"] = &Train{"D200", "Guangzhou", "Shenzhen", "09:15", "10:45", 80, 80}
	trains["K300"] = &Train{"K300", "Chengdu", "Xi'an", "18:20", "07:40", 50, 3}
	http.HandleFunc("/query", handleQuery)
	http.HandleFunc("/book", handleBook)
	http.HandleFunc("/cancel", handleCancel)
	fmt.Println(":bullettrain_side: Ticket server is running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
func handleQuery(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	mu.Lock()
	defer mu.Unlock()
	if train, ok := trains[id]; ok {
		json.NewEncoder(w).Encode(train)
	} else {
		http.Error(w, "train not found", http.StatusNotFound)
	}
}
func handleBook(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	mu.Lock()
	defer mu.Unlock()
	if train, ok := trains[id]; ok {
		if train.Available > 0 {
			train.Available--
			json.NewEncoder(w).Encode(map[string]string{
				"message": "booked successfully",
			})
		} else {
			http.Error(w, "no tickets available", http.StatusConflict)
		}
	} else {
		http.Error(w, "train not found", http.StatusNotFound)
	}
}
func handleCancel(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	mu.Lock()
	defer mu.Unlock()
	if train, ok := trains[id]; ok {
		if train.Available < train.TotalTickets {
			train.Available++
			json.NewEncoder(w).Encode(map[string]string{
				"message": "cancellation successful",
			})
		} else {
			http.Error(w, "no tickets to cancel", http.StatusConflict)
		}
	} else {
		http.Error(w, "train not found", http.StatusNotFound)
	}
}
