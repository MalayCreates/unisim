package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/usip/backend/schema"
	"google.golang.org/grpc"
)

func main() {
	listenAddr := flag.String("addr", ":50051", "gRPC listen address")
	advertiseHost := flag.String("host", "localhost", "host the backend should dial to reach this adapter")
	advertisePort := flag.Int("port", 50051, "port the backend should dial (advertised)")
	backendURL := flag.String("backend", "http://localhost:8080", "backend base URL for registration")
	flag.Parse()

	lis, err := net.Listen("tcp", *listenAddr)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	schema.RegisterSimAdapterServer(grpcServer, newAdapterServer())

	// Register with the backend orchestrator once we're listening.
	go registerWithBackend(*backendURL, *advertiseHost, *advertisePort)

	// Graceful shutdown on SIGINT/SIGTERM.
	go func() {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		<-sigs
		log.Println("shutting down gRPC server...")
		grpcServer.GracefulStop()
	}()

	log.Printf("custom-engine adapter (%s) listening on %s, advertising %s:%d",
		adapterVersion, *listenAddr, *advertiseHost, *advertisePort)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("serve: %v", err)
	}
}

// registerWithBackend POSTs this adapter's coordinates to the backend registry,
// retrying until the backend is reachable.
func registerWithBackend(backendURL, host string, port int) {
	payload, _ := json.Marshal(map[string]any{
		"engine_id": engineID,
		"host":      host,
		"port":      port,
		"version":   adapterVersion,
	})
	url := backendURL + "/api/v1/adapters"

	for attempt := 1; attempt <= 30; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		cancel()
		if err == nil && resp.StatusCode < 300 {
			resp.Body.Close()
			log.Printf("registered with backend as %q (%s:%d)", engineID, host, port)
			return
		}
		if resp != nil {
			resp.Body.Close()
		}
		log.Printf("registration attempt %d failed (%v); retrying...", attempt, errOrStatus(err, resp))
		time.Sleep(2 * time.Second)
	}
	log.Printf("WARNING: could not register with backend at %s after retries; adapter still serving", url)
}

func errOrStatus(err error, resp *http.Response) string {
	if err != nil {
		return err.Error()
	}
	if resp != nil {
		return fmt.Sprintf("HTTP %d", resp.StatusCode)
	}
	return "unknown"
}
