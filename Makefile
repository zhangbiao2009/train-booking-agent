# Train Booking System Makefile

# Variables
BINARY_DIR := bin
SERVER_BINARY := $(BINARY_DIR)/server
AGENT_BINARY := $(BINARY_DIR)/agent
SERVER_SRC := cmd/server/server.go
AGENT_SRC := cmd/agent/agent.go

# Go variables
GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod

# Build flags
LDFLAGS := -ldflags "-s -w"
BUILD_FLAGS := $(LDFLAGS)

# Default target
.PHONY: all
all: clean build

# Create binary directory
$(BINARY_DIR):
	mkdir -p $(BINARY_DIR)

# Build server
.PHONY: server
server: $(BINARY_DIR)
	@echo "Building server..."
	$(GOBUILD) $(BUILD_FLAGS) -o $(SERVER_BINARY) $(SERVER_SRC)
	@echo "Server built: $(SERVER_BINARY)"

# Build agent
.PHONY: agent
agent: $(BINARY_DIR)
	@echo "Building agent..."
	$(GOBUILD) $(BUILD_FLAGS) -o $(AGENT_BINARY) $(AGENT_SRC)
	@echo "Agent built: $(AGENT_BINARY)"

# Build both binaries
.PHONY: build
build: server agent
	@echo "All binaries built successfully!"

# Run server
.PHONY: run-server
run-server: server
	@echo "Starting train booking server..."
	./$(SERVER_BINARY)

# Run agent (requires DEEPSEEK_API_KEY environment variable)
.PHONY: run-agent
run-agent: agent
	@echo "Starting train booking agent..."
	@if [ -z "$$DEEPSEEK_API_KEY" ]; then \
		echo "❌ Error: DEEPSEEK_API_KEY environment variable not set"; \
		echo "💡 Set it with: export DEEPSEEK_API_KEY=your_api_key_here"; \
		exit 1; \
	fi
	./$(AGENT_BINARY)

# Run server in background
.PHONY: start-server
start-server: server
	@echo "Starting server in background..."
	@if pgrep -f "$(SERVER_BINARY)" > /dev/null; then \
		echo "⚠️  Server is already running"; \
	else \
		nohup ./$(SERVER_BINARY) > server.log 2>&1 & \
		echo "✅ Server started in background (PID: $$!)"; \
		echo "📝 Logs: server.log"; \
	fi

# Stop background server
.PHONY: stop-server
stop-server:
	@echo "Stopping server..."
	@if pgrep -f "$(SERVER_BINARY)" > /dev/null; then \
		pkill -f "$(SERVER_BINARY)" && echo "✅ Server stopped"; \
	else \
		echo "ℹ️  Server is not running"; \
	fi

# Development workflow: start server and run agent
.PHONY: dev
dev: start-server
	@echo "Waiting for server to start..."
	@sleep 2
	@$(MAKE) run-agent

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	$(GOCLEAN)
	rm -rf $(BINARY_DIR)
	rm -f server.log
	@echo "Clean complete!"

# Test all packages
.PHONY: test
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

# Download dependencies
.PHONY: deps
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# Show running processes
.PHONY: status
status:
	@echo "🔍 Process Status:"
	@if pgrep -f "$(SERVER_BINARY)" > /dev/null; then \
		echo "✅ Server: Running (PID: $$(pgrep -f "$(SERVER_BINARY)"))"; \
	else \
		echo "❌ Server: Not running"; \
	fi
	@if pgrep -f "$(AGENT_BINARY)" > /dev/null; then \
		echo "✅ Agent: Running (PID: $$(pgrep -f "$(AGENT_BINARY)"))"; \
	else \
		echo "❌ Agent: Not running"; \
	fi

# Quick test with curl commands
.PHONY: test-api
test-api:
	@echo "🧪 Testing API endpoints..."
	@echo "📋 Querying train G100:"
	@curl -s "http://localhost:8080/query?id=G100" | jq . || echo "Server not responding"
	@echo "\n🎫 Booking ticket for G100:"
	@curl -s "http://localhost:8080/book?id=G100" | jq . || echo "Server not responding"
	@echo "\n📋 Querying train G100 again:"
	@curl -s "http://localhost:8080/query?id=G100" | jq . || echo "Server not responding"

# Build for multiple platforms
.PHONY: build-all
build-all: $(BINARY_DIR)
	@echo "Building for multiple platforms..."
	@echo "Building Linux amd64..."
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) -o $(BINARY_DIR)/server-linux-amd64 $(SERVER_SRC)
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) -o $(BINARY_DIR)/agent-linux-amd64 $(AGENT_SRC)
	@echo "Building Windows amd64..."
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) -o $(BINARY_DIR)/server-windows-amd64.exe $(SERVER_SRC)
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) -o $(BINARY_DIR)/agent-windows-amd64.exe $(AGENT_SRC)
	@echo "Building macOS amd64..."
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) -o $(BINARY_DIR)/server-darwin-amd64 $(SERVER_SRC)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) -o $(BINARY_DIR)/agent-darwin-amd64 $(AGENT_SRC)
	@echo "Building macOS arm64..."
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(BUILD_FLAGS) -o $(BINARY_DIR)/server-darwin-arm64 $(SERVER_SRC)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(BUILD_FLAGS) -o $(BINARY_DIR)/agent-darwin-arm64 $(AGENT_SRC)
	@echo "Cross-platform builds complete!"

# Help target
.PHONY: help
help:
	@echo "🚄 Train Booking System - Available Commands:"
	@echo ""
	@echo "📦 Building:"
	@echo "  make build        - Build both server and agent"
	@echo "  make server       - Build server only"
	@echo "  make agent        - Build agent only"
	@echo "  make build-all    - Build for multiple platforms"
	@echo ""
	@echo "🚀 Running:"
	@echo "  make run-server   - Run server in foreground"
	@echo "  make run-agent    - Run agent (requires DEEPSEEK_API_KEY)"
	@echo "  make start-server - Start server in background"
	@echo "  make stop-server  - Stop background server"
	@echo "  make dev          - Start server + run agent"
	@echo ""
	@echo "🧪 Testing:"
	@echo "  make test         - Run Go tests"
	@echo "  make test-api     - Test API with curl"
	@echo "  make status       - Show process status"
	@echo ""
	@echo "🛠️  Maintenance:"
	@echo "  make clean        - Clean build artifacts"
	@echo "  make deps         - Download dependencies"
	@echo ""
	@echo "📖 Usage Examples:"
	@echo "  export DEEPSEEK_API_KEY=your_key_here"
	@echo "  make dev          # Start everything"
	@echo "  make test-api     # Test the server"
	@echo "  make stop-server  # Stop when done"
