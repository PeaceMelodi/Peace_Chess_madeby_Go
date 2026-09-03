package main

import (
	"log"
	"net/http"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"

	"peacechess/internal/handler"
	"peacechess/internal/repository"
	"peacechess/internal/service"
	"peacechess/internal/ws"
)

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, relying on system env vars")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	db, err := sqlx.Connect("pgx", dbURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}

	log.Println("connected to Neon Postgres successfully")

	gameRepo := repository.NewGameRepository(db)
	gameService := service.NewGameService(gameRepo)
	gameHandler := handler.NewGameHandler(gameService)

	hub := ws.NewHub()
	wsHandler := handler.NewWebSocketHandler(gameService, hub)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /games", gameHandler.CreateGame)
	mux.HandleFunc("POST /games/{id}/join", gameHandler.JoinGame)
	mux.HandleFunc("GET /ws", wsHandler.HandleConnection)

	log.Println("server starting on :8080")
	if err := http.ListenAndServe(":8080", withCORS(mux)); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}