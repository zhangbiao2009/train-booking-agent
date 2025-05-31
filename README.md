# Train Booking Agent

A conversational AI agent powered by DeepSeek API that helps users book, query, and cancel train tickets.

## Features

- 🤖 Natural language processing using DeepSeek API
- 🚄 Query train information with departure/arrival times and dates
- 🎫 Book train tickets
- ❌ Cancel bookings
- 📋 List available trains
- 🔍 Search trains by route, date, or combination of criteria

## Setup

1. **Get DeepSeek API Key**
   - Sign up at [DeepSeek](https://platform.deepseek.com/)
   - Get your API key from the dashboard

2. **Set Environment Variable**
   ```bash
   export DEEPSEEK_API_KEY=your_api_key_here
   ```

3. **Start the Train Booking Server**
   ```bash
   go run server.go
   ```

4. **Run the Agent**
   ```bash
   go run cmd/agent/main.go
   ```

## Usage Examples

Once the agent is running, you can interact with it using natural language:

### Query Trains
- "Check G100 train"
- "What's the status of D200?"
- "Show me train K300 info"

### Book Tickets
- "Book a ticket for G100"
- "I want to book D200"
- "Reserve a seat on K300"

### Cancel Tickets
- "Cancel my G100 booking"
- "I need to cancel D200"
- "Remove my K300 reservation"

### List Trains
- "What trains are available?"
- "Show me all trains"
- "List available trains"

### Search Trains
- "Find trains from Beijing to Shanghai"
- "Show me trains from Beijing to Shanghai on June 1st"
- "Any trains to Shanghai?"
- "Trains on June 2nd"
- "Find trains from Guangzhou"

## Available Trains

Current trains with dates and times:

### June 1st, 2025 (2025-06-01)
- **G100**: Beijing → Shanghai | 08:00-13:30 (100 seats)
- **D200**: Guangzhou → Shenzhen | 09:15-10:45 (80 seats)  
- **K300**: Chengdu → Xi'an | 18:20-07:40+1 (50 seats)
- **G102**: Shanghai → Beijing | 14:00-19:30 (100 seats)

### June 2nd, 2025 (2025-06-02)
- **G101**: Beijing → Shanghai | 08:00-13:30 (100 seats)
- **D201**: Guangzhou → Shenzhen | 09:15-10:45 (80 seats)

## API Endpoints

### Server Endpoints
- `GET /query?id={train_id}` - Get specific train information
- `GET /book?id={train_id}` - Book a ticket for a train
- `GET /cancel?id={train_id}` - Cancel a ticket booking
- `GET /list` - List all available trains (with tickets > 0)
- `GET /tickets?from={city}&to={city}&date={YYYY-MM-DD}` - Search trains by criteria

## Architecture

```
User Input → DeepSeek API → Intent Recognition → Action Execution → HTTP API Calls → Train Server
```

1. **User Input**: Natural language request
2. **DeepSeek API**: Analyzes intent and extracts action
3. **Action Execution**: Performs the requested operation
4. **HTTP API**: Communicates with the train booking server
5. **Response**: Formatted result back to user

## Error Handling

The agent handles various error scenarios:
- ❌ Invalid train IDs
- ❌ No tickets available
- ❌ Server connection issues
- ❌ API key not set
- ❌ Server not running

## Development

To modify the agent behavior:

1. **Update System Prompt**: Edit the `systemPrompt` in `callDeepSeek()` function
2. **Add New Actions**: Extend the `executeAction()` switch statement
3. **Modify Responses**: Update the response formatting in individual action methods

## API Endpoints Used

- `GET /query?id={trainId}` - Query train information
- `GET /book?id={trainId}` - Book a ticket
- `GET /cancel?id={trainId}` - Cancel a booking
