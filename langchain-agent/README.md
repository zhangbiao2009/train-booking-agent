# Enhanced Train Booking Agent (LangChain + DeepSeek)

This is an enhanced version of the train booking agent built with LangChain and DeepSeek API that provides:

## Features

- **🧠 Conversation Memory**: ConversationBufferWindowMemory maintains chat history and context
- **🎯 Intent Recognition**: Structured output parsing with Pydantic models for reliable intent extraction
- **🛠️ Server API Integration**: Direct calls to train booking server (no hallucination)
- **👤 User Context**: Remembers user ID across conversations
- **📝 Parameter Extraction**: Extracts train IDs, cities, dates, and user IDs from natural language
- **🔄 Context Awareness**: Uses conversation history for better understanding
- **⚡ DeepSeek Integration**: Uses DeepSeek API through LangChain's ChatOpenAI interface

## Setup

1. **Install Dependencies**:
   ```bash
   pip install -r requirements.txt
   ```

2. **Set Environment Variables**:
   ```bash
   export DEEPSEEK_API_KEY=your_deepseek_api_key
   ```
   Or create a `.env` file:
   ```
   DEEPSEEK_API_KEY=your_deepseek_api_key
   ```

3. **Start the Train Booking Server**:
   ```bash
   # In the main project directory
   go run cmd/server/server.go
   ```

4. **Run the Enhanced Agent**:
   ```bash
   python agent.py
   ```

5. **Test the Agent**:
   ```bash
   python test_enhanced_agent.py
   ```

## Usage Examples

```
You: my user id is 4343
🤖 Agent: 👤 Got it! I've noted your user ID as 4343. What would you like to do?

You: list all trains
🤖 Agent: 🚄 Available Trains:
1. G100: Beijing → Shanghai | 2025-06-01 | 08:00-13:30 (100/100 available)
2. D200: Guangzhou → Shenzhen | 2025-06-01 | 09:15-10:45 (80/80 available)
...

You: book train G100
🤖 Agent: ✅ Successfully booked ticket for train G100 for user 4343!

You: show my tickets
🤖 Agent: 🎫 Booked Tickets for user 4343:
• G100: Beijing → Shanghai | 2025-06-01 | 08:00-13:30 (x1 tickets)

You: search trains from Beijing to Shanghai
🤖 Agent: 🔍 Search Results:
1. G100: Beijing → Shanghai | 2025-06-01 | 08:00-13:30 (99/100 available)
2. G101: Beijing → Shanghai | 2025-06-02 | 08:00-13:30 (95/100 available)
```

## Key Improvements Over Go Version

1. **🧠 Structured Intent Recognition**: Uses Pydantic models for reliable parameter extraction
2. **💭 Conversation Memory**: Automatically maintains conversation context and user information
3. **🎯 Context Awareness**: Remembers user preferences and previous interactions
4. **🛡️ No Hallucination**: All data comes from real server API calls
5. **🔄 Better User ID Handling**: Extracts and remembers user IDs from natural language
6. **⚡ Enhanced Error Handling**: Robust error recovery and user feedback

## Architecture

- **agent.py**: Main enhanced LangChain agent with structured output parsing
- **test_enhanced_agent.py**: Comprehensive test suite
- **requirements.txt**: Python dependencies
- **.env**: Environment configuration

## Technical Features

- **Structured Output**: Pydantic models ensure reliable intent and parameter extraction
- **Conversation Memory**: ConversationBufferWindowMemory with 10-message history
- **Intent Classification**: 7 different intents (query, book, cancel, list, search, tickets, unknown)
- **Parameter Extraction**: train_id, user_id, from_city, to_city, date
- **Server Integration**: Direct HTTP calls to localhost:8080 booking server
- **.env**: Environment configuration

The agent uses LangChain's `initialize_agent` with `OPENAI_FUNCTIONS` type to automatically:
- Route user requests to appropriate tools
- Maintain conversation context
- Handle multi-turn conversations
- Extract parameters from natural language
