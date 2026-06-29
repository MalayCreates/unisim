.PHONY: proto build test run clean

# Generate Go + Python + TypeScript stubs from proto/
proto:
	./scripts/gen-proto.sh

# Build the backend binary
build:
	cd backend && go build -o ../bin/usip-server ./cmd/server

# Run all backend tests
test:
	cd backend && go test ./...

# Start backend + frontend in dev mode
dev:
	./scripts/run-dev.sh

# Start only the backend
backend:
	cd backend && go run ./cmd/server

# Start only the frontend
frontend:
	cd frontend && npm run dev

# Install frontend dependencies
install-frontend:
	cd frontend && npm install

clean:
	rm -rf bin/
	rm -rf backend/internal/schema/*.go
	rm -rf adapters/panopticon/proto/
	rm -rf frontend/src/proto/
