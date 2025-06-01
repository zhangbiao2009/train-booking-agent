package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
)

// DeepSeek API structures
type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatResponse struct {
	Choices []Choice `json:"choices"`
}

type Choice struct {
	Message Message `json:"message"`
}

// Train booking structures
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

type BookingAgent struct {
	apiKey              string
	serverURL           string
	conversationHistory []Message
	userID              string // Add user ID support
}

func NewBookingAgent(apiKey, serverURL string) *BookingAgent {
	return &BookingAgent{
		apiKey:              apiKey,
		serverURL:           serverURL,
		conversationHistory: []Message{},
		userID:              "user_001", // Default user ID
	}
}

// Fetch available trains from server
func (a *BookingAgent) fetchAvailableTrains() ([]Train, error) {
	resp, err := http.Get(fmt.Sprintf("%s/list", a.serverURL))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server error: %s", resp.Status)
	}

	var trains []Train
	if err := json.NewDecoder(resp.Body).Decode(&trains); err != nil {
		return nil, err
	}

	return trains, nil
}

// Call DeepSeek API to understand user intent
func (a *BookingAgent) callDeepSeek(userInput string) (string, error) {
	systemPrompt := `You are a train booking assistant with conversation memory. 

ACTIONS (respond with ONLY these, no additional text):
1. QUERY:train_id - Get train information
2. BOOK:train_id - Book a ticket  
3. CANCEL:train_id - Cancel a ticket
4. LIST - Show all available trains
5. SEARCH:from:to:date - Search trains (any field can be empty)
6. MY_TICKETS - Show user's booked tickets
7. CLARIFY:question - Ask for clarification


CONTEXT PARSING RULES:
1. Look for numbered results like "1. G100: Beijing → Shanghai..." in [RESULT] sections
2. When user says "first", "second", extract train ID from the numbered position
3. Count actual results to validate references ("third" when only 2 shown = clarify)
4. For vague references with multiple options, always clarify

EXAMPLES:
User: "Check G100" → QUERY:G100
User: "Book D200" → BOOK:D200
User: "Find trains to Shanghai" → SEARCH::Shanghai:
User: "Show my tickets" → MY_TICKETS
User: "My bookings" → MY_TICKETS
User: "What tickets do I have" → MY_TICKETS
After "1. G100... 2. G101..." → User: "book it" → CLARIFY:Which train would you like to book - G100 or G101?
After "1. G100... 2. G101..." → User: "book the first one" → BOOK:G100
After "1. G100... 2. G101..." → User: "book the second" → BOOK:G101
After showing only G100 → User: "book it" → BOOK:G100
After "1. G100... 2. G101..." → User: "book the third" → CLARIFY:I only showed 2 trains. Which one did you mean - G100 or G101?

IMPORTANT: Extract train IDs from the numbered search results when users reference by position.`

	// Add user input to conversation history
	a.conversationHistory = append(a.conversationHistory, Message{
		Role:    "user",
		Content: userInput,
	})

	// Build messages with conversation history
	messages := []Message{
		{Role: "system", Content: systemPrompt},
	}

	// Add recent conversation history (last 10 messages to avoid token limits)
	historyStart := 0
	if len(a.conversationHistory) > 10 {
		historyStart = len(a.conversationHistory) - 10
	}
	messages = append(messages, a.conversationHistory[historyStart:]...)

	req := ChatRequest{
		Model:    "deepseek-chat",
		Messages: messages,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	httpReq, err := http.NewRequest("POST", "https://api.deepseek.com/v1/chat/completions", bytes.NewBuffer(data))
	if err != nil {
		return "", err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+a.apiKey)

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return "", err
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no response from DeepSeek")
	}

	response := strings.TrimSpace(chatResp.Choices[0].Message.Content)

	// Debug logging - remove this in production
	fmt.Printf("\r🔍 Debug - DeepSeek response: %q\n", response)

	return response, nil
}

// Execute the action determined by DeepSeek
func (a *BookingAgent) executeAction(action string) string {
	parts := strings.Split(action, ":")
	if len(parts) < 1 {
		return "❌ Invalid action format"
	}

	command := strings.TrimSpace(parts[0])
	// TODO: shouldn't parse train ID here, all the arguments shoulb be parsed in each command method.
	var trainID string
	if len(parts) > 1 {
		// Clean the train ID - remove whitespace, newlines, and any extra text
		trainID = strings.TrimSpace(parts[1])
		// Remove newlines, tabs, and other whitespace
		trainID = strings.ReplaceAll(trainID, "\n", "")
		trainID = strings.ReplaceAll(trainID, "\r", "")
		trainID = strings.ReplaceAll(trainID, "\t", "")
		// Only keep alphanumeric characters and common train ID patterns
		fields := strings.Fields(trainID)
		if len(fields) > 0 {
			trainID = fields[0] // Take only the first "word"
			// Further clean to only allow alphanumeric and common train ID characters
			trainID = regexp.MustCompile(`[^a-zA-Z0-9]`).ReplaceAllString(trainID, "")
		} else {
			trainID = ""
		}
	}

	switch command {
	case "QUERY":
		return a.queryTrain(trainID)
	case "BOOK":
		return a.bookTicket(trainID)
	case "CANCEL":
		return a.cancelTicket(trainID)
	case "LIST":
		return a.listTrains()
	case "SEARCH":
		return a.searchTickets(parts[1:]) // Pass remaining parts for from:to:date
	case "MY_TICKETS":
		return a.getUserTickets()
	case "CLARIFY":
		if len(parts) > 1 {
			return "🤔 " + parts[1]
		}
		return "🤔 Could you please clarify your request?"
	default:
		return "❌ I don't understand that action. Please try asking to query, book, cancel, search for trains, or list all trains."
	}
}

func (a *BookingAgent) queryTrain(trainID string) string {
	if trainID == "" {
		return "❌ Please specify a train ID (e.g., G100, D200, K300)"
	}

	resp, err := http.Get(fmt.Sprintf("%s/query?id=%s", a.serverURL, trainID))
	if err != nil {
		return fmt.Sprintf("❌ Error querying train: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Sprintf("❌ Train %s not found", trainID)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Sprintf("❌ Error: %s", resp.Status)
	}

	var train Train
	if err := json.NewDecoder(resp.Body).Decode(&train); err != nil {
		return fmt.Sprintf("❌ Error decoding response: %v", err)
	}

	return fmt.Sprintf("🚄 Train %s\n📍 Route: %s → %s\n📅 Date: %s\n🕐 Departure: %s | Arrival: %s\n🎫 Available: %d/%d tickets",
		train.ID, train.From, train.To, train.Date, train.DepartureTime, train.ArrivalTime, train.Available, train.TotalTickets)
}

func (a *BookingAgent) bookTicket(trainID string) string {
	if trainID == "" {
		return "❌ Please specify a train ID to book (e.g., G100, D200, K300)"
	}

	// Debug logging - remove this in production
	fmt.Printf("🔍 Debug - Booking train ID: %q (length: %d)\n", trainID, len(trainID))

	url := fmt.Sprintf("%s/book?id=%s&user_id=%s", a.serverURL, trainID, a.userID)
	fmt.Printf("🔍 Debug - Request URL: %q\n", url)

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Sprintf("❌ Error booking ticket: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Sprintf("❌ Train %s not found", trainID)
	}

	if resp.StatusCode == http.StatusConflict {
		return fmt.Sprintf("❌ No tickets available for train %s", trainID)
	}

	if resp.StatusCode == http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Sprintf("❌ Invalid request: %s", string(body))
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Sprintf("❌ Error: %s", resp.Status)
	}

	return fmt.Sprintf("✅ Successfully booked ticket for train %s!", trainID)
}

func (a *BookingAgent) cancelTicket(trainID string) string {
	if trainID == "" {
		return "❌ Please specify a train ID to cancel (e.g., G100, D200, K300)"
	}

	resp, err := http.Get(fmt.Sprintf("%s/cancel?id=%s&user_id=%s", a.serverURL, trainID, a.userID))
	if err != nil {
		return fmt.Sprintf("❌ Error canceling ticket: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Sprintf("❌ Train %s not found", trainID)
	}

	if resp.StatusCode == http.StatusConflict {
		return fmt.Sprintf("❌ No tickets to cancel for train %s", trainID)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Sprintf("❌ Error: %s", resp.Status)
	}

	return fmt.Sprintf("✅ Successfully canceled ticket for train %s!", trainID)
}

func (a *BookingAgent) listTrains() string {
	trains, err := a.fetchAvailableTrains()
	if err != nil {
		return fmt.Sprintf("❌ Error fetching train list: %v", err)
	}

	if len(trains) == 0 {
		return "❌ No trains available"
	}

	result := "🚄 Available Trains:\n"
	for _, train := range trains {
		result += fmt.Sprintf("• %s: %s → %s | %s | %s-%s (%d/%d available)\n",
			train.ID, train.From, train.To, train.Date, train.DepartureTime, train.ArrivalTime, train.Available, train.TotalTickets)
	}

	return result
}

func (a *BookingAgent) searchTickets(params []string) string {
	// Parse search parameters: from, to, date
	var from, to, date string
	if len(params) > 0 {
		from = params[0]
	}
	if len(params) > 1 {
		to = params[1]
	}
	if len(params) > 2 {
		date = params[2]
	}

	// Build query string
	var queryParams []string
	if from != "" {
		queryParams = append(queryParams, fmt.Sprintf("from=%s", from))
	}
	if to != "" {
		queryParams = append(queryParams, fmt.Sprintf("to=%s", to))
	}
	if date != "" {
		queryParams = append(queryParams, fmt.Sprintf("date=%s", date))
	}

	queryString := ""
	if len(queryParams) > 0 {
		queryString = "?" + strings.Join(queryParams, "&")
	}

	resp, err := http.Get(fmt.Sprintf("%s/tickets%s", a.serverURL, queryString))
	if err != nil {
		return fmt.Sprintf("❌ Error searching tickets: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Sprintf("❌ Error: %s", resp.Status)
	}

	var trains []Train
	if err := json.NewDecoder(resp.Body).Decode(&trains); err != nil {
		return fmt.Sprintf("❌ Error decoding response: %v", err)
	}

	if len(trains) == 0 {
		searchCriteria := []string{}
		if from != "" {
			searchCriteria = append(searchCriteria, fmt.Sprintf("from %s", from))
		}
		if to != "" {
			searchCriteria = append(searchCriteria, fmt.Sprintf("to %s", to))
		}
		if date != "" {
			searchCriteria = append(searchCriteria, fmt.Sprintf("on %s", date))
		}
		criteriaText := strings.Join(searchCriteria, " ")
		if criteriaText == "" {
			criteriaText = "matching your criteria"
		}
		return fmt.Sprintf("❌ No trains found %s", criteriaText)
	}

	result := "🔍 Search Results:\n"
	for i, train := range trains {
		result += fmt.Sprintf("%d. %s: %s → %s | %s | %s-%s (%d/%d available)\n",
			i+1, train.ID, train.From, train.To, train.Date, train.DepartureTime, train.ArrivalTime, train.Available, train.TotalTickets)
	}

	return result
}

// UserBooking represents a user's booking information
type UserBooking struct {
	TrainID string `json:"train_id"`
	Count   int    `json:"count"`
}

func (a *BookingAgent) getUserTickets() string {
	resp, err := http.Get(fmt.Sprintf("%s/user/tickets?user_id=%s", a.serverURL, a.userID))
	if err != nil {
		return fmt.Sprintf("❌ Error fetching your tickets: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Sprintf("❌ Error: %s", resp.Status)
	}

	var userBookings []UserBooking
	if err := json.NewDecoder(resp.Body).Decode(&userBookings); err != nil {
		return fmt.Sprintf("❌ Error decoding response: %v", err)
	}

	if len(userBookings) == 0 {
		return "📋 You don't have any booked tickets yet."
	}

	result := "🎫 Your Booked Tickets:\n"
	for _, booking := range userBookings {
		// Get train details for each booking
		train := a.getTrainDetails(booking.TrainID)
		if train != nil {
			result += fmt.Sprintf("• %s: %s → %s | %s | %s-%s (x%d tickets)\n",
				booking.TrainID, train.From, train.To, train.Date,
				train.DepartureTime, train.ArrivalTime, booking.Count)
		} else {
			result += fmt.Sprintf("• %s (x%d tickets)\n", booking.TrainID, booking.Count)
		}
	}

	return result
}

// Helper method to get train details
func (a *BookingAgent) getTrainDetails(trainID string) *Train {
	resp, err := http.Get(fmt.Sprintf("%s/query?id=%s", a.serverURL, trainID))
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	var train Train
	if err := json.NewDecoder(resp.Body).Decode(&train); err != nil {
		return nil
	}

	return &train
}

func (a *BookingAgent) chat() {
	fmt.Println("🤖 Train Booking Agent")
	fmt.Println("💬 I can help you query, book, and cancel train tickets!")
	fmt.Println("📝 Type 'quit' to exit")

	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("You: ")
		if !scanner.Scan() {
			break
		}

		userInput := strings.TrimSpace(scanner.Text())
		if userInput == "" {
			continue
		}

		if strings.ToLower(userInput) == "quit" {
			fmt.Println("👋 Goodbye!")
			break
		}

		fmt.Print("🤖 Agent: Thinking...")

		// Get intent from DeepSeek
		action, err := a.callDeepSeek(userInput)
		if err != nil {
			fmt.Printf("\r❌ Error calling DeepSeek API: %v\n", err)
			continue
		}

		// Execute the action
		result := a.executeAction(action)
		fmt.Printf("\r🤖 Agent: %s\n\n", result)

		a.conversationHistory = append(a.conversationHistory, Message{
			Role:    "assistant",
			Content: result,
		})
	}
}

func main() {
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if apiKey == "" {
		fmt.Println("❌ Please set DEEPSEEK_API_KEY environment variable")
		fmt.Println("💡 Example: export DEEPSEEK_API_KEY=your_api_key_here")
		os.Exit(1)
	}

	serverURL := "http://localhost:8080"
	agent := NewBookingAgent(apiKey, serverURL)

	// Test if server is running
	resp, err := http.Get(serverURL + "/query?id=G100")
	if err != nil {
		fmt.Printf("❌ Cannot connect to booking server at %s\n", serverURL)
		fmt.Println("💡 Make sure to start the server with: go run server.go")
		os.Exit(1)
	}
	resp.Body.Close()

	agent.chat()
}
