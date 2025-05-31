package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
)

// Train structure
type Train struct {
	ID            string `json:"id"`
	From          string `json:"from"`
	To            string `json:"to"`
	Date          string `json:"date"`
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
	trains["G100"] = &Train{"G100", "Beijing", "Shanghai", "2025-06-01", "08:00", "13:30", 100, 100}
	trains["D200"] = &Train{"D200", "Guangzhou", "Shenzhen", "2025-06-01", "09:15", "10:45", 80, 80}
	trains["K300"] = &Train{"K300", "Chengdu", "Xi'an", "2025-06-01", "18:20", "07:40", 50, 3}
	// Add more dates for testing
	trains["G101"] = &Train{"G101", "Beijing", "Shanghai", "2025-06-02", "08:00", "13:30", 100, 95}
	trains["D201"] = &Train{"D201", "Guangzhou", "Shenzhen", "2025-06-02", "09:15", "10:45", 80, 75}
	trains["G102"] = &Train{"G102", "Shanghai", "Beijing", "2025-06-01", "14:00", "19:30", 100, 88}

	http.HandleFunc("/query", handleQuery)
	http.HandleFunc("/book", handleBook)
	http.HandleFunc("/cancel", handleCancel)
	http.HandleFunc("/list", handleList)
	http.HandleFunc("/tickets", handleTickets)
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

func handleList(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	var trainList []*Train
	for _, train := range trains {
		if train.Available > 0 {
			trainList = append(trainList, train)
		}
	}

	json.NewEncoder(w).Encode(trainList)
}

func handleTickets(w http.ResponseWriter, r *http.Request) {
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	date := r.URL.Query().Get("date")

	mu.Lock()
	defer mu.Unlock()

	var matchingTrains []*Train
	for _, train := range trains {
		matches := true

		// Check from parameter (case insensitive)
		if from != "" && !strings.EqualFold(train.From, from) {
			matches = false
		}

		// Check to parameter (case insensitive)
		if to != "" && !strings.EqualFold(train.To, to) {
			matches = false
		}

		// Check date parameter
		if date != "" && train.Date != date {
			matches = false
		}

		// Only include trains with available tickets
		if matches && train.Available > 0 {
			matchingTrains = append(matchingTrains, train)
		}
	}

	json.NewEncoder(w).Encode(matchingTrains)
}
