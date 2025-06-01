# Train Booking Agent

**An experimental conversational AI agent** powered by DeepSeek API that helps users book, query, and cancel train tickets.

## Features

- 🤖 Natural language processing using DeepSeek API
- 🚄 Query train information with departure/arrival times and dates
- 🎫 Book train tickets
- ❌ Cancel bookings
- 📋 List available trains
- 🔍 Search trains by route, date, or combination of criteria
- 👤 User ticket state management with counters
- 📊 View your booked tickets and booking counts

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

### View Your Tickets
- "Show my tickets"
- "What tickets do I have?"
- "My bookings"
- "List my reservations"

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
- `GET /book?id={train_id}&user_id={user_id}` - Book a ticket for a train (user_id required)
- `GET /cancel?id={train_id}&user_id={user_id}` - Cancel a ticket booking (user_id required)
- `GET /list` - List all available trains (with tickets > 0)
- `GET /tickets?from={city}&to={city}&date={YYYY-MM-DD}` - Search trains by criteria
- `GET /user/tickets?user_id={user_id}` - Get user's booked tickets with counts (user_id required)

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

The agent and server handle various error scenarios:

### Server API Errors
- ❌ **400 Bad Request**: Missing required parameters (user_id, train_id)
- ❌ **404 Not Found**: Invalid train IDs
- ❌ **409 Conflict**: No tickets available or no tickets to cancel
- ❌ **500 Internal Server Error**: Server connection issues

### Agent Errors
- ❌ API key not set
- ❌ Server not running
- ❌ Network connection issues
- ❌ Invalid user input

### Parameter Validation
All user-specific endpoints now require proper parameter validation:
- `book` and `cancel`: Require both `id` and `user_id` parameters
- `user/tickets`: Requires `user_id` parameter
- Missing parameters return 400 status with clear error messages

## Development

To modify the agent behavior:

1. **Update System Prompt**: Edit the `systemPrompt` in `callDeepSeek()` function
2. **Add New Actions**: Extend the `executeAction()` switch statement
3. **Modify Responses**: Update the response formatting in individual action methods

## API Endpoints Used

- `GET /query?id={trainId}` - Query train information
- `GET /book?id={trainId}` - Book a ticket
- `GET /cancel?id={trainId}` - Cancel a booking

## User Ticket State Management

The system now maintains user ticket state with intelligent counters:

### Features
- **Multiple Bookings**: Users can book the same ticket multiple times
- **Smart Counters**: Each booking increments a counter for that train
- **Automatic Cleanup**: When a user cancels all tickets for a train, it's removed from their state
- **User Isolation**: Each user's bookings are tracked separately
- **Persistent State**: Ticket counts are maintained during the server session

### How It Works
1. **Booking**: `book?id=G100&user_id=user123` increments the user's G100 ticket count
2. **Cancellation**: `cancel?id=G100&user_id=user123` decrements the count
3. **View Tickets**: `user/tickets?user_id=user123` shows all user's tickets with counts
4. **Zero Count Cleanup**: When count reaches 0, the train is removed from user's bookings

### Example Workflow
```bash
# Book 2 tickets for G100
curl "localhost:8080/book?id=G100&user_id=user123"  # count: 1
curl "localhost:8080/book?id=G100&user_id=user123"  # count: 2

# Book 1 ticket for D200  
curl "localhost:8080/book?id=D200&user_id=user123"  # count: 1

# Check tickets: [{"train_id":"G100","count":2},{"train_id":"D200","count":1}]
curl "localhost:8080/user/tickets?user_id=user123"

# Cancel 1 G100 ticket
curl "localhost:8080/cancel?id=G100&user_id=user123"  # count: 1

# Cancel last G100 ticket (removes from state)
curl "localhost:8080/cancel?id=G100&user_id=user123"  # G100 removed

# Final state: [{"train_id":"D200","count":1}]
```

## Available Trains
