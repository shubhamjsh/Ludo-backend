package main

import (
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	httpSwagger "github.com/swaggo/http-swagger"

	"Ludo/config"
	"Ludo/database"
	"Ludo/handlers"
	"Ludo/handlers/game_handler"
	"Ludo/middleware"
	"Ludo/utils"

	_ "Ludo/docs"
)

// @title Ludo Game API
// @version 1.0
// @description REST API for Ludo Game with authentication and user management
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.email support@ludogame.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /
// @schemes http https

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.
func main() {
	// Load configuration from .env file
	cfg := config.Load()
	log.Println("Configuration loaded successfully")

	if err := utils.InitLogger("logs/ludo-api.log"); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	utils.InfoLogger.Println("Logger initialized successfully")

	// Initialize JWT with secret key
	utils.InitJWT(cfg.JWTSecret)
	log.Println("JWT initialized successfully")

	// Connect to PostgreSQL database
	dbConfig := database.Config{
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		DBName:   cfg.Database.DBName,
		SSLMode:  cfg.Database.SSLMode,
	}

	if err := database.Connect(dbConfig); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	// Initialize database schema
	if err := database.InitSchema(); err != nil {
		log.Fatalf("Failed to initialize database schema: %v", err)
	}

	// Initialize handlers
	authHandler := handlers.NewAuthHandler()
	userHandler := handlers.NewUserHandler()
	gameHandler := game_handler.NewGameHandler()

	// Create HTTP router
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		utils.SendSuccess(w, "Server is healthy", map[string]string{
			"status": "ok",
		})
	})

	// Swagger documentation
	mux.HandleFunc("/swagger/", httpSwagger.WrapHandler)

	// ==================== AUTH ROUTES (PUBLIC) ====================
	mux.HandleFunc("/api/auth/signup", authHandler.Signup)
	mux.HandleFunc("/api/auth/login", authHandler.Login)
	mux.HandleFunc("/api/auth/guest-login", authHandler.GuestLogin)

	// ==================== USER ROUTES (PROTECTED) ====================
	mux.Handle("/api/user/profile", middleware.AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			userHandler.GetProfile(w, r)
		case http.MethodPut:
			userHandler.UpdateProfile(w, r)
		default:
			utils.SendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		}
	})))

	// ==================== GAME ROUTES (PROTECTED) ====================

	// POST /api/game/create - Create a new game
	mux.Handle("/api/game/create", middleware.AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			utils.SendError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}
		gameHandler.CreateGame(w, r)
	})))

	// POST /api/game/join - Join an existing game
	mux.Handle("/api/game/join", middleware.AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			utils.SendError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}
		gameHandler.JoinGame(w, r)
	})))

	// POST /api/game/create-local - Create local multiplayer game (NO AUTH REQUIRED)
	mux.HandleFunc("/api/game/create-local", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			utils.SendError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}
		gameHandler.CreateLocalGame(w, r)
	})

	// Handle /api/game/{id}/* routes (with path parameters)
	mux.HandleFunc("/api/game/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// GET requests
		if r.Method == http.MethodGet {
			// GET /api/game/{id}/state
			if contains(path, "/state") {
				middleware.AuthMiddleware(http.HandlerFunc(gameHandler.GetGameState)).ServeHTTP(w, r)
				return
			}
		}

		// POST requests
		if len(path) > len("/api/game/") && r.Method == http.MethodPost {
			// POST /api/game/{id}/start
			if contains(path, "/start") {
				middleware.AuthMiddleware(http.HandlerFunc(gameHandler.StartGame)).ServeHTTP(w, r)
				return
			}

			// POST /api/game/{id}/leave
			if contains(path, "/leave") {
				middleware.AuthMiddleware(http.HandlerFunc(gameHandler.LeaveGame)).ServeHTTP(w, r)
				return
			}

			// POST /api/game/{id}/roll-dice
			if contains(path, "/roll-dice") {
				middleware.AuthMiddleware(http.HandlerFunc(gameHandler.RollDice)).ServeHTTP(w, r)
				return
			}

			// POST /api/game/{id}/move-token
			if contains(path, "/move-token") {
				middleware.AuthMiddleware(http.HandlerFunc(gameHandler.MoveToken)).ServeHTTP(w, r)
				return
			}

			// POST /api/game/{id}/skip-turn
			if contains(path, "/skip-turn") {
				middleware.AuthMiddleware(http.HandlerFunc(gameHandler.SkipTurn)).ServeHTTP(w, r)
				return
			}
		}

		utils.SendError(w, http.StatusNotFound, "Endpoint not found")
	})

	// Metrics dashboard (no auth required for easier access)
	mux.Handle("/metrics/prometheus", promhttp.Handler())

	// ==================== APPLY MIDDLEWARE ====================
	// Apply Logging and CORS middleware to all routes
	//handler := middleware.CORSMiddleware(mux)
	handler := middleware.LoggingMiddleware(middleware.CORSMiddleware(mux))

	// Start HTTP server
	serverAddr := ":" + cfg.ServerPort
	log.Println("  📊 Metrics & Monitoring")
	log.Println("  GET  /metrics                     - View metrics dashboard")
	log.Println("  GET  /metrics/data                - Get metrics JSON")
	log.Println("  GET  /metrics/prometheus          - Prometheus metrics endpoint")
	log.Println("  POST /metrics/reset               - Reset metrics")
	log.Println("═══════════════════════════════════════════════════════════")
	log.Printf("🚀 Starting HTTP server on port %s...", cfg.ServerPort)
	log.Printf("🌐 Server is running at http://localhost%s", serverAddr)
	log.Println("═══════════════════════════════════════════════════════════")
	log.Println("📋 Available endpoints:")
	log.Println("")
	log.Println("  🏥 Health & Docs")
	log.Println("  GET  /health                      - Health check")
	log.Println("  GET  /swagger/                    - API Documentation")
	log.Println("")
	log.Println("  🔐 Authentication (Public)")
	log.Println("  POST /api/auth/signup             - Register new user")
	log.Println("  POST /api/auth/login              - Login with email/phone + password")
	log.Println("  POST /api/auth/guest-login        - Create guest account")
	log.Println("")
	log.Println("  👤 User (Protected)")
	log.Println("  GET  /api/user/profile            - Get user profile")
	log.Println("  PUT  /api/user/profile            - Update user profile")
	log.Println("")
	log.Println("  🎮 Game Lifecycle (Protected)")
	log.Println("  POST /api/game/create             - Create a new game")
	log.Println("  POST /api/game/join               - Join an existing game")
	log.Println("  POST /api/game/create-local       - Create local multiplayer game (no auth)")
	log.Println("  POST /api/game/{id}/start         - Start a game (creator only)")
	log.Println("  POST /api/game/{id}/leave         - Leave a game")
	log.Println("")
	log.Println("  🎲 Game Actions (Protected)")
	log.Println("  POST /api/game/{id}/roll-dice     - Roll the dice")
	log.Println("  POST /api/game/{id}/move-token    - Move a token")
	log.Println("  POST /api/game/{id}/skip-turn     - Skip turn (when no valid moves)")
	log.Println("")
	log.Println("  📊 Game State (Protected)")
	log.Println("  GET  /api/game/{id}/state         - Get complete game state")
	log.Println("═══════════════════════════════════════════════════════════")
	log.Println("🎯 Logging enabled - All requests will be logged")
	log.Println("═══════════════════════════════════════════════════════════")

	if err := http.ListenAndServe(serverAddr, handler); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s[len(s)-len(substr):] == substr || indexOf(s, substr) >= 0)
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
