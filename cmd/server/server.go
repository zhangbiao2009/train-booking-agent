package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ResponseWriter wrapper to capture response data
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	body       *bytes.Buffer
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
		body:           &bytes.Buffer{},
	}
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	rw.body.Write(b)
	return rw.ResponseWriter.Write(b)
}

// Logging middleware
func loggingMiddleware(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Log incoming request
		log.Printf("📥 [REQUEST] %s %s from %s", r.Method, r.URL.String(), r.RemoteAddr)
		if r.URL.RawQuery != "" {
			log.Printf("📋 [PARAMS] %s", r.URL.RawQuery)
		}

		// Wrap response writer to capture response
		rw := newResponseWriter(w)

		// Call the handler
		handler(rw, r)

		// Log response
		duration := time.Since(start)
		status := rw.statusCode
		responseBody := rw.body.String()

		if status >= 200 && status < 300 {
			log.Printf("✅ [RESPONSE] %d - %v - Body: %s", status, duration, responseBody)
		} else {
			log.Printf("❌ [RESPONSE] %d - %v - Error: %s", status, duration, responseBody)
		}
	}
}

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

// User booking information
type UserBooking struct {
	TrainID string `json:"train_id"`
	Count   int    `json:"count"`
}

// Train ticket information stored in map
var (
	trains      = map[string]*Train{}
	userTickets = map[string]map[string]int{} // userID -> trainID -> count
	mu          sync.Mutex                    // Concurrency protection
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

	http.HandleFunc("/query", loggingMiddleware(handleQuery))
	http.HandleFunc("/book", loggingMiddleware(handleBook))
	http.HandleFunc("/cancel", loggingMiddleware(handleCancel))
	http.HandleFunc("/list", loggingMiddleware(handleList))
	http.HandleFunc("/tickets", loggingMiddleware(handleTickets))
	http.HandleFunc("/user/tickets", loggingMiddleware(handleUserTickets))
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
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		userID = "default" // Default user ID if not provided
	}

	mu.Lock()
	defer mu.Unlock()

	if train, ok := trains[id]; ok {
		if train.Available > 0 {
			train.Available--

			// Initialize user tickets map if not exists
			if userTickets[userID] == nil {
				userTickets[userID] = make(map[string]int)
			}

			// Increment user's booking count for this train
			userTickets[userID][id]++

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
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		userID = "default" // Default user ID if not provided
	}

	mu.Lock()
	defer mu.Unlock()

	if train, ok := trains[id]; ok {
		// Check if user has bookings for this train
		if userTickets[userID] != nil && userTickets[userID][id] > 0 {
			train.Available++
			userTickets[userID][id]--

			// Remove train from user's bookings if count reaches 0
			if userTickets[userID][id] == 0 {
				delete(userTickets[userID], id)
				// Clean up empty user map
				if len(userTickets[userID]) == 0 {
					delete(userTickets, userID)
				}
			}

			json.NewEncoder(w).Encode(map[string]string{
				"message": "cancellation successful",
			})
		} else {
			http.Error(w, "no tickets to cancel for this user", http.StatusConflict)
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

func handleUserTickets(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		userID = "default" // Default user ID if not provided
	}

	mu.Lock()
	defer mu.Unlock()

	var userBookings []UserBooking

	if userTickets[userID] != nil {
		for trainID, count := range userTickets[userID] {
			userBookings = append(userBookings, UserBooking{
				TrainID: trainID,
				Count:   count,
			})
		}
	}

	json.NewEncoder(w).Encode(userBookings)
}
