"""
Enhanced LangChain Train Booking Agent
- Full LangChain conversation memory and intent recognition
- Structured output parsing for reliable intent extraction
- Direct server API calls (no hallucination)
- All features from the Go agent
"""

import os
import requests
import json
from typing import List, Dict, Any, Optional, Literal
from dataclasses import dataclass
from pydantic import BaseModel, Field

from langchain.memory import ConversationBufferWindowMemory
from langchain_openai import ChatOpenAI
from langchain.prompts import ChatPromptTemplate, MessagesPlaceholder
from langchain.schema import HumanMessage, AIMessage
from langchain.output_parsers import PydanticOutputParser
from dotenv import load_dotenv

# Load environment variables
load_dotenv()

# Pydantic models for structured output
class IntentResponse(BaseModel):
    intent: Literal["query_ticket", "book_ticket", "cancel_ticket", "list_trains", "search_trains", "my_tickets", "unknown"] = Field(
        description="The user's intent"
    )
    parameters: Dict[str, Optional[str]] = Field(
        default_factory=dict,
        description="Extracted parameters like train_id, user_id, from_city, to_city, date"
    )
    missing_parameters: List[str] = Field(
        default_factory=list,
        description="Required parameters that are missing"
    )
    clarify_question: Optional[str] = Field(
        default="",
        description="Question to ask user if parameters are missing"
    )

@dataclass
class Train:
    id: str
    from_city: str
    to_city: str
    date: str
    departure_time: str
    arrival_time: str
    total_tickets: int
    available: int
    
    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> 'Train':
        return cls(
            id=data["id"],
            from_city=data["from"],  # Map "from" to "from_city"
            to_city=data["to"],      # Map "to" to "to_city"
            date=data["date"],
            departure_time=data["departure_time"],
            arrival_time=data["arrival_time"],
            total_tickets=data["total_tickets"],
            available=data["available"]
        )

@dataclass
class UserBooking:
    train_id: str
    count: int

class EnhancedTrainBookingAgent:
    def __init__(self, server_url: str = "http://localhost:8080"):
        self.server_url = server_url
        self.default_user_id = "user_001"
        
        # Initialize DeepSeek LLM
        self.llm = ChatOpenAI(
            base_url="https://api.deepseek.com",
            api_key=os.getenv("DEEPSEEK_API_KEY"),
            model="deepseek-chat",
            temperature=0.1
        )
        
        # Initialize conversation memory
        self.memory = ConversationBufferWindowMemory(
            k=10,  # Keep last 10 exchanges
            memory_key="chat_history",
            return_messages=True
        )
        
        # Initialize structured output parser
        self.parser = PydanticOutputParser(pydantic_object=IntentResponse)
        
        # Create intent extraction prompt
        self.intent_prompt = ChatPromptTemplate.from_messages([
            ("system", self._get_intent_system_prompt()),
            MessagesPlaceholder(variable_name="chat_history"),
            ("human", "{input}"),
        ])
        
        # Create intent extraction chain
        self.intent_chain = self.intent_prompt | self.llm | self.parser
        
        # Current user context
        self.current_user_id = None
        
    def _get_intent_system_prompt(self) -> str:
        return """You are a train booking assistant that analyzes user requests to extract intent and parameters.

AVAILABLE OPERATIONS:
- query_ticket: Get information about a specific train by ID
- book_ticket: Book a train ticket (requires train_id and user_id)  
- cancel_ticket: Cancel a booked ticket (requires train_id and user_id)
- list_trains: Show all available trains
- search_trains: Search trains by criteria (from_city, to_city, date)
- my_tickets: Show user's booked tickets (requires user_id)
- unknown: Cannot determine intent

PARAMETER EXTRACTION RULES:
- train_id: Look for train IDs like G100, D200, K300, etc.
- user_id: Extract from phrases like "my user id is 4343", "user 1234", etc.
- from_city: Extract departure city names
- to_city: Extract destination city names  
- date: Extract dates in YYYY-MM-DD format

CONTEXT AWARENESS:
- Remember previous conversations and user information
- Parse numbered results from previous responses (e.g., "book the first one")
- Maintain user context across conversations

MISSING PARAMETERS:
- For booking/canceling: user_id is required
- For searching: at least one of from_city, to_city, or date
- Ask clarifying questions for missing required parameters

RESPONSE FORMAT:
Return a JSON object with this exact structure:
{{
  "intent": "one of: query_ticket, book_ticket, cancel_ticket, list_trains, search_trains, my_tickets, unknown",
  "parameters": {{
    "train_id": "extracted train ID or empty string",
    "user_id": "extracted user ID or empty string", 
    "from_city": "extracted departure city or empty string",
    "to_city": "extracted destination city or empty string",
    "date": "extracted date or empty string"
  }},
  "missing_parameters": ["list of required but missing parameters"],
  "clarify_question": "question to ask user if parameters are missing or empty string"
}}

IMPORTANT: Always use empty strings ("") instead of null values. Always return valid JSON matching the exact schema above."""

    def _extract_intent(self, user_input: str) -> IntentResponse:
        """Extract intent and parameters from user input using LangChain structured parsing"""
        try:
            # Get chat history for context
            chat_history = self.memory.chat_memory.messages
            
            # Use the chain to extract intent
            result = self.intent_chain.invoke({
                "input": user_input,
                "chat_history": chat_history
            })
            
            # Update current user_id if provided
            if result.parameters.get("user_id"):
                self.current_user_id = result.parameters["user_id"]
            
            return result
            
        except Exception as e:
            print(f"🔍 Debug - Intent extraction error: {e}")
            # Fallback to unknown intent
            return IntentResponse(
                intent="unknown",
                parameters={},
                missing_parameters=[],
                clarify_question="I didn't understand your request. Could you please rephrase it?"
            )

    # Server API methods (same as Go agent functionality)
    def _query_train(self, train_id: str) -> str:
        """Get train information from server"""
        try:
            response = requests.get(f"{self.server_url}/query?id={train_id}")
            
            if response.status_code == 404:
                return f"❌ Train {train_id} not found"
            elif response.status_code != 200:
                return f"❌ Error: {response.status_text}"
            
            train_data = response.json()
            train = Train.from_dict(train_data)
            
            return f"""🚄 Train {train.id}
📍 Route: {train.from_city} → {train.to_city}
📅 Date: {train.date}
🕐 Departure: {train.departure_time} | Arrival: {train.arrival_time}
🎫 Available: {train.available}/{train.total_tickets} tickets"""
            
        except Exception as e:
            return f"❌ Error querying train: {str(e)}"

    def _book_ticket(self, train_id: str, user_id: str = "") -> str:
        """Book a train ticket"""
        effective_user_id = user_id or self.current_user_id or self.default_user_id
        
        try:
            url = f"{self.server_url}/book?id={train_id}&user_id={effective_user_id}"
            print(f"🔍 Debug - Booking URL: {url}")
            
            response = requests.get(url)
            
            if response.status_code == 404:
                return f"❌ Train {train_id} not found"
            elif response.status_code == 409:
                return f"❌ No tickets available for train {train_id}"
            elif response.status_code == 400:
                return f"❌ Invalid request: {response.text}"
            elif response.status_code != 200:
                return f"❌ Error: {response.status_text}"
            
            return f"✅ Successfully booked ticket for train {train_id} for user {effective_user_id}!"
            
        except Exception as e:
            return f"❌ Error booking ticket: {str(e)}"

    def _cancel_ticket(self, train_id: str, user_id: str = "") -> str:
        """Cancel a train ticket"""
        effective_user_id = user_id or self.current_user_id or self.default_user_id
        
        try:
            response = requests.get(f"{self.server_url}/cancel?id={train_id}&user_id={effective_user_id}")
            
            if response.status_code == 404:
                return f"❌ Train {train_id} not found"
            elif response.status_code == 409:
                return f"❌ No tickets to cancel for train {train_id}"
            elif response.status_code != 200:
                return f"❌ Error: {response.status_text}"
            
            return f"✅ Successfully canceled ticket for train {train_id}!"
            
        except Exception as e:
            return f"❌ Error canceling ticket: {str(e)}"

    def _list_trains(self) -> str:
        """Get all available trains"""
        try:
            response = requests.get(f"{self.server_url}/list")
            
            if response.status_code != 200:
                return f"❌ Error: {response.status_text}"
            
            trains_data = response.json()
            if not trains_data:
                return "❌ No trains available"
            
            result = "🚄 Available Trains:\n"
            for i, train_data in enumerate(trains_data, 1):
                train = Train.from_dict(train_data)
                result += f"{i}. {train.id}: {train.from_city} → {train.to_city} | {train.date} | {train.departure_time}-{train.arrival_time} ({train.available}/{train.total_tickets} available)\n"
            
            return result
            
        except Exception as e:
            return f"❌ Error fetching train list: {str(e)}"

    def _search_trains(self, from_city: str = "", to_city: str = "", date: str = "") -> str:
        """Search trains by criteria"""
        try:
            params = {}
            if from_city:
                params["from"] = from_city
            if to_city:
                params["to"] = to_city
            if date:
                params["date"] = date
            
            response = requests.get(f"{self.server_url}/tickets", params=params)
            
            if response.status_code != 200:
                return f"❌ Error: {response.status_text}"
            
            trains_data = response.json()
            if not trains_data:
                criteria = []
                if from_city:
                    criteria.append(f"from {from_city}")
                if to_city:
                    criteria.append(f"to {to_city}")
                if date:
                    criteria.append(f"on {date}")
                criteria_text = " ".join(criteria) if criteria else "matching your criteria"
                return f"❌ No trains found {criteria_text}"
            
            result = "🔍 Search Results:\n"
            for i, train_data in enumerate(trains_data, 1):
                train = Train.from_dict(train_data)
                result += f"{i}. {train.id}: {train.from_city} → {train.to_city} | {train.date} | {train.departure_time}-{train.arrival_time} ({train.available}/{train.total_tickets} available)\n"
            
            return result
            
        except Exception as e:
            return f"❌ Error searching trains: {str(e)}"

    def _get_user_tickets(self, user_id: str = "") -> str:
        """Get user's booked tickets"""
        effective_user_id = user_id or self.current_user_id or self.default_user_id
        
        try:
            response = requests.get(f"{self.server_url}/user/tickets?user_id={effective_user_id}")
            
            if response.status_code != 200:
                return f"❌ Error: {response.status_text}"
            
            bookings_data = response.json()
            if not bookings_data:
                return f"📋 No booked tickets found for user {effective_user_id}."
            
            result = f"🎫 Booked Tickets for user {effective_user_id}:\n"
            for booking_data in bookings_data:
                booking = UserBooking(**booking_data)
                # Get train details
                train_details = self._get_train_details(booking.train_id)
                if train_details:
                    result += f"• {booking.train_id}: {train_details.from_city} → {train_details.to_city} | {train_details.date} | {train_details.departure_time}-{train_details.arrival_time} (x{booking.count} tickets)\n"
                else:
                    result += f"• {booking.train_id} (x{booking.count} tickets)\n"
            
            return result
            
        except Exception as e:
            return f"❌ Error fetching tickets: {str(e)}"

    def _get_train_details(self, train_id: str) -> Optional[Train]:
        """Helper to get train details"""
        try:
            response = requests.get(f"{self.server_url}/query?id={train_id}")
            if response.status_code == 200:
                return Train.from_dict(response.json())
            return None
        except:
            return None

    def _execute_action(self, intent_response: IntentResponse) -> str:
        """Execute the action based on extracted intent"""
        
        # If there's a clarify question, return it
        if intent_response.clarify_question:
            return f"🤔 {intent_response.clarify_question}"
        
        params = intent_response.parameters
        
        # Execute based on intent
        if intent_response.intent == "query_ticket":
            train_id = params.get("train_id", "")
            if not train_id:
                return "❌ Please specify a train ID (e.g., G100, D200, K300)"
            return self._query_train(train_id)
            
        elif intent_response.intent == "book_ticket":
            train_id = params.get("train_id", "")
            user_id = params.get("user_id", "")
            if not train_id:
                return "❌ Please specify a train ID to book"
            return self._book_ticket(train_id, user_id)
            
        elif intent_response.intent == "cancel_ticket":
            train_id = params.get("train_id", "")
            user_id = params.get("user_id", "")
            if not train_id:
                return "❌ Please specify a train ID to cancel"
            return self._cancel_ticket(train_id, user_id)
            
        elif intent_response.intent == "list_trains":
            return self._list_trains()
            
        elif intent_response.intent == "search_trains":
            from_city = params.get("from_city", "")
            to_city = params.get("to_city", "")
            date = params.get("date", "")
            return self._search_trains(from_city, to_city, date)
            
        elif intent_response.intent == "my_tickets":
            user_id = params.get("user_id", "")
            return self._get_user_tickets(user_id)
            
        else:
            return "❌ I don't understand that request. I can help you query trains, book tickets, cancel bookings, list all trains, search by criteria, or show your tickets."

    def chat(self):
        """Start interactive chat with full LangChain conversation memory"""
        print("🤖 Enhanced Train Booking Agent (Full LangChain + Server API)")
        print("💬 I can help you with train bookings using natural conversation!")
        print("📝 I remember our conversation and understand complex requests.")
        print("📝 Type 'quit' to exit")
        print()
        
        while True:
            try:
                user_input = input("You: ").strip()
                
                if not user_input:
                    continue
                    
                if user_input.lower() == 'quit':
                    print("👋 Goodbye!")
                    break
                
                print("🤖 Agent: Thinking...", end="", flush=True)
                
                # Extract intent using LangChain structured parsing
                intent_response = self._extract_intent(user_input)
                
                # Debug output
                print(f"\r🔍 Debug - Intent: {intent_response.intent}, Params: {intent_response.parameters}")
                
                # Execute the action
                result = self._execute_action(intent_response)
                
                print(f"🤖 Agent: {result}")
                print()
                
                # Save to conversation memory
                self.memory.chat_memory.add_user_message(user_input)
                self.memory.chat_memory.add_ai_message(result)
                
            except KeyboardInterrupt:
                print("\n👋 Goodbye!")
                break
            except Exception as e:
                print(f"\r❌ Error: {str(e)}")
                print()

    def test_connection(self) -> bool:
        """Test server connection"""
        try:
            response = requests.get(f"{self.server_url}/query?id=G100")
            return True
        except:
            return False

if __name__ == "__main__":
    # Check environment
    if not os.getenv("DEEPSEEK_API_KEY"):
        print("❌ Please set DEEPSEEK_API_KEY environment variable")
        exit(1)
    
    # Initialize agent
    agent = EnhancedTrainBookingAgent()
    
    # Test server connection
    if not agent.test_connection():
        print("❌ Cannot connect to booking server at http://localhost:8080")
        print("💡 Make sure to start the server with: go run cmd/server/server.go")
        exit(1)
    
    # Start chat
    agent.chat()
