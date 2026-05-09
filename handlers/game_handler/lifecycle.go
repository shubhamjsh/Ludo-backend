package game_handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"Ludo/models"
	"Ludo/models/game"
	_ "Ludo/models/game"
	gamemodels "Ludo/models/game"
	"Ludo/repository"
	gamerepo "Ludo/repository/game"
	"Ludo/utils"
)

type GameHandler struct {
	gameRepo *gamerepo.GameRepository
	userRepo *repository.UserRepository
}

func NewGameHandler() *GameHandler {
	return &GameHandler{
		gameRepo: gamerepo.NewGameRepository(),
		userRepo: repository.NewUserRepository(),
	}
}

// CreateGame creates a new game room
// @Summary Create a new game
// @Description Create a new game room with specified settings
// @Tags Game
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body gamemodels.CreateGameRequest true "Create Game Request"
// @Success 201 {object} utils.Response{data=gamemodels.CreateGameResponse} "Game created successfully"
// @Failure 400 {object} utils.Response "Invalid request"
// @Failure 401 {object} utils.Response "Unauthorized"
// @Failure 402 {object} utils.Response "Insufficient coins"
// @Failure 500 {object} utils.Response "Internal server error"
// @Router /api/game/create [post]
func (h *GameHandler) CreateGame(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by auth middleware)
	userID, ok := r.Context().Value("userID").(int64)
	if !ok {
		utils.SendError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Parse request body
	var req game.CreateGameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	if req.MaxPlayers < 2 || req.MaxPlayers > 4 {
		utils.SendError(w, http.StatusBadRequest, "Max players must be between 2 and 4")
		return
	}

	if req.BetAmount < 0 {
		utils.SendError(w, http.StatusBadRequest, "Bet amount cannot be negative")
		return
	}

	// Validate room code if provided
	if req.IsPrivate && req.RoomCode != nil && len(*req.RoomCode) != 6 {
		utils.SendError(w, http.StatusBadRequest, "Room code must be exactly 6 characters")
		return
	}

	// Check if user has enough coins
	user, err := h.userRepo.GetUserByID(userID)
	if err != nil {
		utils.SendError(w, http.StatusInternalServerError, "Error fetching user details")
		return
	}

	if user.Coins < req.BetAmount {
		utils.SendError(w, http.StatusPaymentRequired, "Insufficient coins")
		return
	}

	// Create game
	gameRepo, err := h.gameRepo.CreateGame(userID, &req)
	if err != nil {
		utils.SendError(w, http.StatusInternalServerError, "Error creating game")
		return
	}

	// Deduct bet amount from user's coins
	//if req.BetAmount > 0 {
	//	if err := h.userRepo.UpdateCoins(userID, -req.BetAmount); err != nil {
	//		// Rollback game creation if coin deduction fails
	//		h.gameRepo.DeleteGame(game.ID)
	//		utils.SendError(w, http.StatusInternalServerError, "Error processing bet amount")
	//		return
	//	}
	//}

	// Create response
	response := game.CreateGameResponse{
		GameID:    gameRepo.ID,
		RoomCode:  gameRepo.RoomCode,
		Status:    gameRepo.Status,
		CreatedBy: gameRepo.CreatedBy,
	}

	utils.ActiveGamesGauge.Inc()

	utils.SendCreated(w, "Game created successfully", response)
}

// JoinGame allows a user to join an existing game
// @Summary Join a game
// @Description Join an existing game room
// @Tags Game
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body gamemodels.JoinGameRequest true "Join Game Request"
// @Success 200 {object} utils.Response{data=gamemodels.JoinGameResponse} "Joined game successfully"
// @Failure 400 {object} utils.Response "Invalid request or game full"
// @Failure 401 {object} utils.Response "Unauthorized"
// @Failure 402 {object} utils.Response "Insufficient coins"
// @Failure 403 {object} utils.Response "Invalid room code"
// @Failure 404 {object} utils.Response "Game not found"
// @Failure 409 {object} utils.Response "Already in game"
// @Failure 500 {object} utils.Response "Internal server error"
// @Router /api/game/join [post]
func (h *GameHandler) JoinGame(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by auth middleware)
	userID, ok := r.Context().Value("userID").(int64)
	if !ok {
		utils.SendError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Parse request body
	var req game.JoinGameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate game ID
	if req.GameID == "" {
		utils.SendError(w, http.StatusBadRequest, "Game ID is required")
		return
	}

	// Get game details to check if it's private and validate room code
	gameRepo, err := h.gameRepo.GetGameByID(req.GameID)
	if err != nil {
		if errors.Is(err, gamerepo.ErrGameNotFound) {
			utils.SendError(w, http.StatusNotFound, "Game not found")
			return
		}
		utils.SendError(w, http.StatusInternalServerError, "Error fetching game details")
		return
	}

	// Validate room code for private games
	if gameRepo.IsPrivate {
		if req.RoomCode == nil || gameRepo.RoomCode == nil || *req.RoomCode != *gameRepo.RoomCode {
			utils.SendError(w, http.StatusForbidden, "Invalid room code")
			return
		}
	}

	// Check if user has enough coins for bet amount
	user, err := h.userRepo.GetUserByID(userID)
	if err != nil {
		utils.SendError(w, http.StatusInternalServerError, "Error fetching user details")
		return
	}

	if user.Coins < gameRepo.BetAmount {
		utils.SendError(w, http.StatusPaymentRequired, "Insufficient coins")
		return
	}

	// Join the game
	response, err := h.gameRepo.JoinGame(req.GameID, userID)
	if err != nil {
		if errors.Is(err, gamerepo.ErrGameNotFound) {
			utils.SendError(w, http.StatusNotFound, "Game not found")
			return
		}
		if errors.Is(err, gamerepo.ErrGameAlreadyStarted) {
			utils.SendError(w, http.StatusBadRequest, "Game already started")
			return
		}
		if errors.Is(err, gamerepo.ErrGameFull) {
			utils.SendError(w, http.StatusBadRequest, "Game is full")
			return
		}
		if errors.Is(err, gamerepo.ErrAlreadyInGame) {
			utils.SendError(w, http.StatusConflict, "Already in this game")
			return
		}
		utils.SendError(w, http.StatusInternalServerError, "Error joining game")
		return
	}

	// Deduct bet amount from user's coins
	//if game.BetAmount > 0 {
	//	if err := h.userRepo.UpdateCoins(userID, -game.BetAmount); err != nil {
	//		// This is a critical error - player was added but coins not deducted
	//		// In production, you might want to use database transactions
	//		utils.SendError(w, http.StatusInternalServerError, "Error processing bet amount")
	//		return
	//	}
	//}

	utils.TotalPlayersGauge.Inc()

	utils.SendSuccess(w, "Joined game successfully", response)
}

// StartGame starts a game (only creator can start)
// @Summary Start a game
// @Description Start a game (only the creator can start)
// @Tags Game
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Game ID"
// @Success 200 {object} utils.Response{data=gamemodels.StartGameResponse} "Game started successfully"
// @Failure 400 {object} utils.Response "Not enough players or game already started"
// @Failure 401 {object} utils.Response "Unauthorized"
// @Failure 403 {object} utils.Response "Only creator can start the game"
// @Failure 404 {object} utils.Response "Game not found"
// @Failure 500 {object} utils.Response "Internal server error"
// @Router /api/game/{id}/start [post]
func (h *GameHandler) StartGame(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by auth middleware)
	userID, ok := r.Context().Value("userID").(int64)
	if !ok {
		utils.SendError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Extract game ID from URL path
	// Expected path: /api/game/{id}/start
	path := r.URL.Path
	parts := strings.Split(strings.Trim(path, "/"), "/")

	var gameID string
	for i, part := range parts {
		if part == "game" && i+1 < len(parts) {
			gameID = parts[i+1]
			break
		}
	}

	if gameID == "" {
		utils.SendError(w, http.StatusBadRequest, "Game ID is required")
		return
	}

	// Start the game
	gameRepo, err := h.gameRepo.StartGame(gameID, userID)
	if err != nil {
		if errors.Is(err, gamerepo.ErrGameNotFound) {
			utils.SendError(w, http.StatusNotFound, "Game not found")
			return
		}
		if errors.Is(err, gamerepo.ErrNotGameCreator) {
			utils.SendError(w, http.StatusForbidden, "Only the game creator can start the game")
			return
		}
		if errors.Is(err, gamerepo.ErrGameAlreadyStarted) {
			utils.SendError(w, http.StatusBadRequest, "Game already started")
			return
		}
		if errors.Is(err, gamerepo.ErrNotEnoughPlayers) {
			utils.SendError(w, http.StatusBadRequest, "Need at least 2 players to start the game")
			return
		}
		utils.SendError(w, http.StatusInternalServerError, "Error starting game")
		return
	}

	// Get all players with user details
	playersWithDetails, err := h.gameRepo.GetPlayersWithUserDetails(gameID)
	if err != nil {
		utils.SendError(w, http.StatusInternalServerError, "Error fetching player details")
		return
	}

	// Convert to PlayerInfo format
	players := make([]game.PlayerInfo, len(playersWithDetails))
	for i, p := range playersWithDetails {
		players[i] = game.PlayerInfo{
			UserID:   p.UserID,
			Name:     p.Name,
			Color:    p.Color,
			Position: p.Position,
		}
	}

	// Create response
	response := game.StartGameResponse{
		GameID:    gameRepo.ID,
		Status:    gameRepo.Status,
		StartedAt: *gameRepo.StartedAt,
		Players:   players,
	}

	utils.SendSuccess(w, "Game started successfully", response)
}

// LeaveGame allows a user to leave a game
// @Summary Leave a game
// @Description Leave a game room (creator leaving will cancel the game)
// @Tags Game
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Game ID"
// @Success 200 {object} utils.Response{data=gamemodels.LeaveGameResponse} "Left game successfully"
// @Failure 400 {object} utils.Response "Not in this game"
// @Failure 401 {object} utils.Response "Unauthorized"
// @Failure 404 {object} utils.Response "Game not found"
// @Failure 500 {object} utils.Response "Internal server error"
// @Router /api/game/{id}/leave [post]
func (h *GameHandler) LeaveGame(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by auth middleware)
	userID, ok := r.Context().Value("userID").(int64)
	if !ok {
		utils.SendError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Extract game ID from URL path
	// Expected path: /api/game/{id}/leave
	path := r.URL.Path
	parts := strings.Split(strings.Trim(path, "/"), "/")

	var gameID string
	for i, part := range parts {
		if part == "game" && i+1 < len(parts) {
			gameID = parts[i+1]
			break
		}
	}

	if gameID == "" {
		utils.SendError(w, http.StatusBadRequest, "Game ID is required")
		return
	}

	// Get game details before leaving (to check bet amount for refund)
	//game, err := h.gameRepo.GetGameByID(gameID)
	//if err != nil {
	//	if err == repository.ErrGameNotFound {
	//		utils.SendError(w, http.StatusNotFound, "Game not found")
	//		return
	//	}
	//	utils.SendError(w, http.StatusInternalServerError, "Error fetching game details")
	//	return
	//}

	// Leave the game
	updatedGame, err := h.gameRepo.LeaveGame(gameID, userID)
	if err != nil {
		if errors.Is(err, gamerepo.ErrGameNotFound) {
			utils.SendError(w, http.StatusNotFound, "Game not found")
			return
		}
		if errors.Is(err, gamerepo.ErrNotInGame) {
			utils.SendError(w, http.StatusBadRequest, "You are not in this game")
			return
		}
		utils.SendError(w, http.StatusInternalServerError, "Error leaving game")
		return
	}

	// Refund bet amount if game hasn't started
	//if game.Status == models.GameStatusWaiting && game.BetAmount > 0 {
	//	if err := h.userRepo.UpdateCoins(userID, game.BetAmount); err != nil {
	//		// Log this error but don't fail the request - player has already left
	//		// In production, you'd want proper logging here
	//	}
	//}

	// Record metrics
	utils.TotalPlayersGauge.Dec()
	if updatedGame.Status == gamemodels.GameStatusCancelled {
		utils.ActiveGamesGauge.Dec()
	}

	// Create response
	response := game.LeaveGameResponse{
		GameID: updatedGame.ID,
		Status: updatedGame.Status,
	}

	utils.SendSuccess(w, "Left game successfully", response)
}

// CreateLocalGame creates a local multiplayer game with multiple guest accounts
// @Summary Create local multiplayer game
// @Description Create a game for local multiplayer on a single device (no auth required)
// @Tags Game
// @Accept json
// @Produce json
// @Param request body models.CreateLocalGameRequest true "Create Local Game Request"
// @Success 201 {object} utils.Response{data=models.CreateLocalGameResponse} "Local game created successfully"
// @Failure 400 {object} utils.Response "Invalid request"
// @Failure 500 {object} utils.Response "Internal server error"
// @Router /api/game/create-local [post]
func (h *GameHandler) CreateLocalGame(w http.ResponseWriter, r *http.Request) {
	// Parse request body
	var req gamemodels.CreateLocalGameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate number of players
	if len(req.PlayerNames) < 2 || len(req.PlayerNames) > 4 {
		utils.SendError(w, http.StatusBadRequest, "Need 2-4 players for local multiplayer")
		return
	}

	// Validate player names
	for i, name := range req.PlayerNames {
		if len(name) < 2 || len(name) > 50 {
			utils.SendError(w, http.StatusBadRequest, fmt.Sprintf("Player %d name must be between 2-50 characters", i+1))
			return
		}
	}

	// Step 1: Create guest accounts for all players
	var userIDs []int64
	var tokens []string

	for _, playerName := range req.PlayerNames {
		// Generate unique device ID for each player
		//deviceID := fmt.Sprintf("local-game-%d-%d", time.Now().Unix(), i)

		// Generate avatar
		avatarURL := fmt.Sprintf("https://ui-avatars.com/api/?name=%s&background=random", playerName)

		// Create guest user
		user := &models.User{
			Name:      playerName,
			AvatarURL: avatarURL,
			Coins:     0, // Local players don't need coins
			IsGuest:   true,
			CreatedAt: time.Now(),
		}

		if err := h.userRepo.CreateUser(user); err != nil {
			utils.SendError(w, http.StatusInternalServerError, fmt.Sprintf("Error creating guest user for %s", playerName))
			return
		}

		// Generate JWT token
		token, err := utils.GenerateToken(user.ID, true)
		if err != nil {
			utils.SendError(w, http.StatusInternalServerError, fmt.Sprintf("Error generating token for %s", playerName))
			return
		}

		userIDs = append(userIDs, user.ID)
		tokens = append(tokens, token)
	}

	// Step 2: Create game with first player
	createGameReq := gamemodels.CreateGameRequest{
		RoomName:   req.RoomName,
		MaxPlayers: len(req.PlayerNames),
		BetAmount:  0, // No betting for local games
		IsPrivate:  false,
	}

	game, err := h.gameRepo.CreateGame(userIDs[0], &createGameReq)
	if err != nil {
		utils.SendError(w, http.StatusInternalServerError, "Error creating game")
		return
	}

	// Step 3: Join ALL players to the game (including the creator)
	var players []gamemodels.LocalPlayerInfo

	for i := 0; i < len(req.PlayerNames); i++ {
		joinResponse, err := h.gameRepo.JoinGame(game.ID, userIDs[i])
		if err != nil {
			// Rollback: delete game if join fails
			h.gameRepo.DeleteGame(game.ID)
			utils.SendError(w, http.StatusInternalServerError, fmt.Sprintf("Error adding player %s to game", req.PlayerNames[i]))
			return
		}

		players = append(players, gamemodels.LocalPlayerInfo{
			Name:   req.PlayerNames[i],
			Token:  tokens[i],
			Color:  joinResponse.PlayerColor,
			UserID: userIDs[i],
		})
	}

	// Step 4: Auto-start the game
	startedGame, err := h.gameRepo.StartGame(game.ID, userIDs[0])
	if err != nil {
		utils.ErrorLogger.Printf("Failed to auto-start local game: %v", err)
		// Don't fail - game is still created, just not started
	} else {
		game = startedGame // Update with started game state
	}

	// Step 5: Record metrics
	utils.ActiveGamesGauge.Inc()
	utils.TotalPlayersGauge.Add(float64(len(req.PlayerNames)))

	// Step 6: Create response
	response := gamemodels.CreateLocalGameResponse{
		GameID:  game.ID,
		Status:  game.Status,
		Players: players,
	}

	utils.SendCreated(w, "Local multiplayer game created successfully", response)
}
