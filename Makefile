.PHONY: proto build test dev backend adapter stack seed install-frontend frontend clean

# Generate Go + Python + TypeScript stubs from proto/
proto:
	./scripts/gen-proto.sh

# Build the backend binary
build:
	cd backend && go build -o ../bin/usip-server ./cmd/server

# Run all Go tests (backend + custom-engine adapter)
test:
	cd backend && go test ./...
	cd adapters/custom-engine && go test ./...

# Start backend + adapter + frontend together (one terminal)
dev:
	./scripts/run-dev.sh

# Start only the backend
backend:
	cd backend && go run ./cmd/server

# Start only the custom-engine adapter (expects the backend to be up)
adapter:
	cd adapters/custom-engine && go run . -addr :50051 -host localhost -port 50051 -backend http://localhost:8080

# Start backend + adapter (no frontend) — handy for API/seed testing
stack:
	@echo "Starting backend + adapter (Ctrl-C to stop both)..."
	@trap 'kill 0' EXIT INT TERM; \
	(cd backend && go run ./cmd/server) & \
	sleep 2; \
	(cd adapters/custom-engine && go run . -addr :50051 -host localhost -port 50051 -backend http://localhost:8080) & \
	wait

# Seed the running backend with the sample scenario
seed:
	./scripts/seed.sh

# Install frontend dependencies
install-frontend:
	cd frontend && npm install

# Start only the frontend dev server (Vite on :5173, proxies /api -> :8080)
frontend:
	cd frontend && npm run dev

clean:
	rm -rf bin/
	rm -rf backend/schema/*.go
	rm -rf adapters/panopticon/proto/
	rm -rf frontend/src/proto/
