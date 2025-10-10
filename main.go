package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"api-gateway/config"
	"api-gateway/proto/auth"
	"api-gateway/proto/stakeholder"

	"api-gateway/middleware"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type serviceDef struct {
	name     string
	address  string
	register func(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error
}

func registerAuth(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	client := auth.NewAuthServiceClient(conn)
	return auth.RegisterAuthServiceHandlerClient(ctx, mux, client)
}

func registerStakeholder(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	client := stakeholder.NewStakeholderClient(conn)
	return stakeholder.RegisterStakeholderHandlerClient(ctx, mux, client)
}

func withCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allowedOrigin := "http://localhost:5173"

		if r.Header.Get("Origin") == allowedOrigin {
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
			w.Header().Set("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		h.ServeHTTP(w, r)
	})
}

func dialWithRetry(ctx context.Context, target string, attempts int, delay time.Duration) (*grpc.ClientConn, error) {
	var lastErr error
	for i := 0; i < attempts; i++ {
		dctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		conn, err := grpc.DialContext(dctx, target, grpc.WithBlock(), grpc.WithTransportCredentials(insecure.NewCredentials()))
		cancel()
		if err == nil {
			return conn, nil
		}
		lastErr = err
		log.Printf("dial %s failed (attempt %d/%d): %v", target, i+1, attempts, err)
		time.Sleep(delay)
	}
	return nil, lastErr
}

func main() {
	cfg := config.GetConfig()
	svcs := []serviceDef{
		{name: "auth", address: cfg.AuthServiceAddress, register: registerAuth},
		{name: "stakeholder", address: cfg.StakeHolderServiceAddress, register: registerStakeholder},
	}

	gwmux := runtime.NewServeMux()

	var conns []*grpc.ClientConn
	ctx := context.Background()

	for _, s := range svcs {
		log.Printf("Dialing %s -> %s", s.name, s.address)
		conn, err := dialWithRetry(ctx, s.address, 5, 2*time.Second)
		if err != nil {
			for _, c := range conns {
				_ = c.Close()
			}
			log.Fatalf("failed to dial %s: %v", s.name, err)
		}
		conns = append(conns, conn)

		if err := s.register(ctx, gwmux, conn); err != nil {
			for _, c := range conns {
				_ = c.Close()
			}
			log.Fatalf("failed to register %s: %v", s.name, err)
		}
		log.Printf("Registered %s", s.name)
	}

	jwtMid := middleware.NewJWTMiddlewareFromEnv()

	jwtMid.AllowUnauthenticated = true
	jwtMid.RequireAuthForAll = false

	handler := jwtMid.Middleware(withCORS(gwmux))

	gwServer := &http.Server{
		Addr:    cfg.Address,
		Handler: handler,
	}

	go func() {
		log.Printf("Gateway listening on %s", cfg.Address)
		if err := gwServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("gateway ListenAndServe: %v", err)
		}
	}()

	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, syscall.SIGINT, syscall.SIGTERM)
	<-stopCh
	log.Println("Shutting down...")

	shCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := gwServer.Shutdown(shCtx); err != nil {
		log.Printf("gw shutdown error: %v", err)
	}

	for _, c := range conns {
		_ = c.Close()
	}
	log.Println("Stopped")
}
