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
	DepartureTime string `json:"departure_time"`
	ArrivalTime   string `json:"arrival_time"`
	TotalTickets  int    `json:"total_tickets"`
	Available     int    `json:"available"`
}

type BookingAgent struct {
	apiKey    string
	serverURL string
}

func NewBookingAgent(apiKey, serverURL string) *BookingAgent {
	return &BookingAgent{
		apiKey:    apiKey,
		serverURL: serverURL,
	}
}

// Call DeepSeek API to understand user intent
func (a *BookingAgent) callDeepSeek(userInput string) (string, error) {
	systemPrompt := `You are a train booking assistant. Analyze the user's request and respond with ONLY ONE of these actions:

1. For querying train info: "QUERY:train_id" (e.g., "QUERY:G100")
2. For booking tickets: "BOOK:train_id" (e.g., "BOOK:G100") 
3. For canceling tickets: "CANCEL:train_id" (e.g., "CANCEL:G100")
4. For listing available trains: "LIST"
5. If unclear: "CLARIFY:question to ask user"

Available trains with departure/arrival times:
- G100: Beijing-Shanghai (08:00-13:30)
- D200: Guangzhou-Shenzhen (09:15-10:45) 
- K300: Chengdu-Xi'an (18:20-07:40)

Examples:
- "Check G100 train" → "QUERY:G100"
- "Book a ticket for D200" → "BOOK:D200"
- "Cancel my K300 booking" → "CANCEL:K300"
- "What trains are available?" → "LIST"

Respond with ONLY the action, no explanation.`

	req := ChatRequest{
		Model: "deepseek-chat",
		Messages: []Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userInput},
		},
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

	return strings.TrimSpace(chatResp.Choices[0].Message.Content), nil
}

// Execute the action determined by DeepSeek
func (a *BookingAgent) executeAction(action string) string {
	parts := strings.Split(action, ":")
	if len(parts) < 1 {
		return "❌ Invalid action format"
	}

	command := parts[0]
	var trainID string
	if len(parts) > 1 {
		trainID = parts[1]
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
	case "CLARIFY":
		if len(parts) > 1 {
			return "🤔 " + parts[1]
		}
		return "🤔 Could you please clarify your request?"
	default:
		return "❌ I don't understand that action. Please try asking to query, book, or cancel a train ticket."
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

	return fmt.Sprintf("🚄 Train %s\n📍 Route: %s → %s\n🕐 Departure: %s | Arrival: %s\n🎫 Available: %d/%d tickets",
		train.ID, train.From, train.To, train.DepartureTime, train.ArrivalTime, train.Available, train.TotalTickets)
}

func (a *BookingAgent) bookTicket(trainID string) string {
	if trainID == "" {
		return "❌ Please specify a train ID to book (e.g., G100, D200, K300)"
	}

	resp, err := http.Get(fmt.Sprintf("%s/book?id=%s", a.serverURL, trainID))
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

	if resp.StatusCode != http.StatusOK {
		return fmt.Sprintf("❌ Error: %s", resp.Status)
	}

	return fmt.Sprintf("✅ Successfully booked ticket for train %s!", trainID)
}

func (a *BookingAgent) cancelTicket(trainID string) string {
	if trainID == "" {
		return "❌ Please specify a train ID to cancel (e.g., G100, D200, K300)"
	}

	resp, err := http.Get(fmt.Sprintf("%s/cancel?id=%s", a.serverURL, trainID))
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
	trains := []string{"G100", "D200", "K300"}
	result := "🚄 Available Trains:\n"

	for _, trainID := range trains {
		resp, err := http.Get(fmt.Sprintf("%s/query?id=%s", a.serverURL, trainID))
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			var train Train
			if json.NewDecoder(resp.Body).Decode(&train) == nil {
				result += fmt.Sprintf("• %s: %s → %s | %s-%s (%d/%d available)\n",
					train.ID, train.From, train.To, train.DepartureTime, train.ArrivalTime, train.Available, train.TotalTickets)
			}
		}
	}

	return result
}

func (a *BookingAgent) chat() {
	fmt.Println("🤖 Train Booking Agent")
	fmt.Println("💬 I can help you query, book, and cancel train tickets!")
	fmt.Println("📝 Type 'quit' to exit\n")

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
