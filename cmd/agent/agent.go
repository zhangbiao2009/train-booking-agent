package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
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

// Intent response structure
type IntentResponse struct {
	Intent            string            `json:"intent"`
	Parameters        map[string]string `json:"parameters"`
	MissingParameters []string          `json:"missing_parameters"`
	ClarifyQuestion   string            `json:"clarify_question"`
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
func (a *BookingAgent) callDeepSeek(userInput string) (*IntentResponse, error) {
	systemPrompt := `You are a train booking assistant. Analyze user requests and respond with structured JSON.

CRITICAL: Your response must be valid JSON only. Do not use markdown code blocks, do not wrap JSON in backticks, do not add any explanatory text. Return only the raw JSON object without any formatting or wrapper text.

AVAILABLE APIs:
/query - Get train information
Parameters: id (required, train ID like G100)

/book - Book a train ticket  
Parameters: id (required, train ID), user_id (required, user identifier)

/cancel - Cancel a train ticket
Parameters: id (required, train ID), user_id (required, user identifier)

/list - Show all available trains
Parameters: None

/tickets - Search trains by criteria
Parameters: from (optional, departure city), to (optional, destination city), date (optional, YYYY-MM-DD)

/user/tickets - Get user's booked tickets
Parameters: user_id (required, user identifier)

INTENT CLASSIFICATION:
- query_ticket: User wants information about a specific train
- book_ticket: User wants to book a ticket (specific train or search criteria)
- cancel_ticket: User wants to cancel a booked ticket
- list_trains: User wants to see all available trains
- search_trains: User wants to search for trains by criteria
- my_tickets: User wants to see their booked tickets
- unknown: Cannot determine intent

CONTEXT PARSING:
- Parse numbered results from previous responses like "1. G100: Beijing → Shanghai..."
- When user says "first", "second", extract train ID from numbered position
- For vague references with multiple options, ask for clarification

If the user's message is unclear or lacks required parameters, ask a clarifying question. Try to confirm the missing fields in natural, polite English.

RESPONSE FORMAT: Return ONLY valid JSON in this exact structure (no markdown, no backticks, no explanations):
{
  "intent": "query_ticket | book_ticket | cancel_ticket | list_trains | search_trains | my_tickets | unknown",
  "parameters": {
    "train_id": "",
    "from": "",
    "to": "",
    "date": "",
    "user_id": ""
  },
  "missing_parameters": [],
  "clarify_question": ""
}

EXAMPLES:
User: "Check train G100" → {"intent": "query_ticket", "parameters": {"train_id": "G100"}, "missing_parameters": [], "clarify_question": ""}
User: "Book ticket for D200" → {"intent": "book_ticket", "parameters": {"train_id": "D200"}, "missing_parameters": ["user_id"], "clarify_question": "Please provide your user ID to book the ticket."}
User: "Book G102 for me. my user id is 4343" → {"intent": "book_ticket", "parameters": {"train_id": "G102", "user_id": "4343"}, "missing_parameters": [], "clarify_question": ""}
User: "Find trains to Shanghai" → {"intent": "search_trains", "parameters": {"to": "Shanghai"}, "missing_parameters": [], "clarify_question": ""}
User: "Book a ticket" → {"intent": "book_ticket", "parameters": {}, "missing_parameters": ["train_id"], "clarify_question": "Which train would you like to book? Please provide the train ID or tell me your travel details."}
User: "Show my bookings" → {"intent": "my_tickets", "parameters": {}, "missing_parameters": ["user_id"], "clarify_question": "Please provide your user ID to view your tickets."}

If you cannot understand the user's intent at all, set intent to "unknown" and leave other fields empty.

IMPORTANT: Your entire response must be parseable JSON. No markdown formatting, no code blocks, no extra text.`

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
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", "https://api.deepseek.com/v1/chat/completions", bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+a.apiKey)

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return nil, err
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("no response from DeepSeek")
	}

	response := strings.TrimSpace(chatResp.Choices[0].Message.Content)

	// Debug logging - remove this in production
	fmt.Printf("\r🔍 Debug - DeepSeek response: %q\n", response)

	// Parse JSON response
	var intentResp IntentResponse
	if err := json.Unmarshal([]byte(response), &intentResp); err != nil {
		// If JSON parsing fails, treat as unknown intent
		return &IntentResponse{
			Intent:          "unknown",
			Parameters:      map[string]string{},
			ClarifyQuestion: "I didn't understand your request. Could you please rephrase it?",
		}, nil
	}

	return &intentResp, nil
}

// Execute the action determined by DeepSeek
func (a *BookingAgent) executeAction(intentResp *IntentResponse) string {
	// If there's a clarify question, return it directly
	if intentResp.ClarifyQuestion != "" {
		return "🤔 " + intentResp.ClarifyQuestion
	}

	switch intentResp.Intent {
	case "query_ticket":
		trainID := intentResp.Parameters["train_id"]
		return a.queryTrain(trainID)
	case "book_ticket":
		trainID := intentResp.Parameters["train_id"]
		userID := intentResp.Parameters["user_id"]
		return a.bookTicket(trainID, userID)
	case "cancel_ticket":
		trainID := intentResp.Parameters["train_id"]
		userID := intentResp.Parameters["user_id"]
		return a.cancelTicket(trainID, userID)
	case "list_trains":
		return a.listTrains()
	case "search_trains":
		from := intentResp.Parameters["from"]
		to := intentResp.Parameters["to"]
		date := intentResp.Parameters["date"]
		return a.searchTrains(from, to, date)
	case "my_tickets":
		userID := intentResp.Parameters["user_id"]
		return a.getUserTickets(userID)
	case "unknown":
		return "❌ I didn't understand your request. Please try asking to query, book, cancel, search for trains, or list all trains."
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

func (a *BookingAgent) bookTicket(trainID string, userID string) string {
	if trainID == "" {
		return "❌ Please specify a train ID to book (e.g., G100, D200, K300)"
	}

	// Use provided userID, fallback to agent's default if empty
	effectiveUserID := userID
	if effectiveUserID == "" {
		effectiveUserID = a.userID
	}

	// Debug logging - remove this in production
	fmt.Printf("🔍 Debug - Booking train ID: %q (length: %d)\n", trainID, len(trainID))

	url := fmt.Sprintf("%s/book?id=%s&user_id=%s", a.serverURL, trainID, effectiveUserID)
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

	return fmt.Sprintf("✅ Successfully booked ticket for train %s for user %s!", trainID, effectiveUserID)
}

func (a *BookingAgent) cancelTicket(trainID string, userID string) string {
	if trainID == "" {
		return "❌ Please specify a train ID to cancel (e.g., G100, D200, K300)"
	}

	// Use provided userID, fallback to agent's default if empty
	effectiveUserID := userID
	if effectiveUserID == "" {
		effectiveUserID = a.userID
	}

	resp, err := http.Get(fmt.Sprintf("%s/cancel?id=%s&user_id=%s", a.serverURL, trainID, effectiveUserID))
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

func (a *BookingAgent) searchTrains(from, to, date string) string {
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

func (a *BookingAgent) getUserTickets(userID string) string {
	// Use provided userID, fallback to agent's default if empty
	effectiveUserID := userID
	if effectiveUserID == "" {
		effectiveUserID = a.userID
	}

	resp, err := http.Get(fmt.Sprintf("%s/user/tickets?user_id=%s", a.serverURL, effectiveUserID))
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
		intentResp, err := a.callDeepSeek(userInput)
		if err != nil {
			fmt.Printf("\r❌ Error calling DeepSeek API: %v\n", err)
			continue
		}

		// Execute the action
		result := a.executeAction(intentResp)
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
