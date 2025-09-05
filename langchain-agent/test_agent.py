#!/usr/bin/env python3
"""
Test script for the enhanced LangChain agent
Tests all functionality including conversation memory and intent recognition
"""

import os
from agent import EnhancedTrainBookingAgent

def test_agent():
    print("🧪 Testing Enhanced LangChain Agent")
    print("=" * 50)
    
    # Initialize agent
    agent = EnhancedTrainBookingAgent()
    
    # Test connection first
    print("1. Testing server connection...")
    if agent.test_connection():
        print("✅ Server connection OK")
    else:
        print("❌ Server connection failed")
        return
    
    print()
    
    # Test cases
    test_cases = [
        # Basic intent recognition
        "list all trains",
        
        # Conversation with user_id
        "my user id is 4343",
        
        # Intent with parameters
        "query train G100",
        
        # Book with extracted user_id
        "book train G100",
        
        # Show my tickets
        "show my tickets",
        
        # Search by criteria
        "search trains from Beijing to Shanghai",
        
        # Cancel ticket
        "cancel train G100",
        
        # Complex conversation
        "I want to go from Shanghai to Beijing tomorrow"
    ]
    
    print("2. Testing conversation flow:")
    print("-" * 30)
    
    for i, test_input in enumerate(test_cases, 1):
        print(f"\n🧪 Test {i}: {test_input}")
        
        try:
            # Extract intent
            intent_response = agent._extract_intent(test_input)
            print(f"🔍 Intent: {intent_response.intent}")
            print(f"📝 Parameters: {intent_response.parameters}")
            
            # Execute action
            result = agent._execute_action(intent_response)
            print(f"🤖 Response: {result[:200]}{'...' if len(result) > 200 else ''}")
            
            # Add to memory
            agent.memory.chat_memory.add_user_message(test_input)
            agent.memory.chat_memory.add_ai_message(result)
            
        except Exception as e:
            print(f"❌ Error: {e}")
        
        print("-" * 30)
    
    print("\n3. Testing conversation memory:")
    print("-" * 30)
    chat_history = agent.memory.chat_memory.messages
    print(f"💭 Memory contains {len(chat_history)} messages")
    
    if len(chat_history) >= 2:
        print("📝 Last exchange:")
        print(f"   User: {chat_history[-2].content}")
        print(f"   Agent: {chat_history[-1].content[:100]}{'...' if len(chat_history[-1].content) > 100 else ''}")
    
    print("\n4. Testing user context:")
    print("-" * 30)
    print(f"👤 Current user ID: {agent.current_user_id}")
    
    print("\n✅ Enhanced agent test completed!")

if __name__ == "__main__":
    test_agent()
