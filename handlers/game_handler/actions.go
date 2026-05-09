package game_handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"crypto/rand"
	"math/big"

	gamemodels "Ludo/models/game"
	gamerepo "Ludo/repository/game"
	"Ludo/utils"
)

// RollDice allows a player to roll the dice
// @Summary Roll the dice
// @Description Roll the dice for your turn (returns 1-6)
// @Tags Game Actions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Game ID"
// @Success 200 {object} utils.Response{data=gamemodels.RollDiceResponse} "Dice rolled successfully"
// @Failure 400 {object} utils.Response "Invalid request or already rolled"
// @Failure 401 {object} utils.Response "Unauthorized"
// @Failure 403 {object} utils.Response "Not your turn"
// @Failure 404 {object} utils.Response "Game not found"
// @Failure 500 {object} utils.Response "Internal server error"
// @Router /api/game/{id}/roll-dice [post]
func (h *GameHandler) RollDice(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by auth middleware)
	userID, ok := r.Context().Value("userID").(int64)
	if !ok {
		utils.SendError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Extract game ID from URL path
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

	// Get game details
	game, err := h.gameRepo.GetGameByID(gameID)
	if err != nil {
		if errors.Is(err, gamerepo.ErrGameNotFound) {
			utils.SendError(w, http.StatusNotFound, "Game not found")
			return
		}
		utils.SendError(w, http.StatusInternalServerError, "Error fetching game details")
		return
	}

	// Check if game is in progress
	if game.Status != gamemodels.GameStatusInProgress {
		utils.SendError(w, http.StatusBadRequest, "Game is not in progress")
		return
	}

	// Check if it's this player's turn
	if game.TurnPlayerID == nil || *game.TurnPlayerID != userID {
		utils.SendError(w, http.StatusForbidden, "Not your turn")
		return
	}

	// Get player details
	currentPlayer, err := h.gameRepo.GetCurrentPlayerTurn(gameID)
	if err != nil {
		utils.SendError(w, http.StatusInternalServerError, "Error getting player details")
		return
	}

	// Get token positions for this player
	positions, err := h.gameRepo.GetTokenPositions(gameID, userID)
	if err != nil {
		utils.SendError(w, http.StatusInternalServerError, "Error getting token positions")
		return
	}

	// If no positions exist, initialize them
	if len(positions) == 0 || (len(positions) == 4 && positions[0].Position == -1 && positions[1].Position == -1 && positions[2].Position == -1 && positions[3].Position == -1) {
		if err := h.gameRepo.InitializeTokenPositions(gameID, userID); err != nil {
			utils.SendError(w, http.StatusInternalServerError, "Error initializing token positions")
			return
		}
		// Get positions again after initialization
		positions, err = h.gameRepo.GetTokenPositions(gameID, userID)
		if err != nil {
			utils.SendError(w, http.StatusInternalServerError, "Error getting token positions")
			return
		}
	}

	// Roll the dice (generate random number 1-6)
	diceValue, err := rollDice()
	if err != nil {
		utils.SendError(w, http.StatusInternalServerError, "Error rolling dice")
		return
	}

	utils.DiceRollsTotal.WithLabelValues(strconv.Itoa(diceValue)).Inc()

	// Record the dice roll
	if err := h.gameRepo.RecordDiceRoll(gameID, userID, diceValue); err != nil {
		utils.SendError(w, http.StatusInternalServerError, "Error recording dice roll")
		return
	}

	// Record Prometheus metrics
	utils.DiceRollsTotal.WithLabelValues(fmt.Sprintf("%d", diceValue)).Inc()

	// Determine which tokens can move
	validTokens := h.gameRepo.GetValidTokensForDiceValue(positions, diceValue)
	canMove := len(validTokens) > 0

	// Check if rolled a 6 (gets extra turn)
	extraTurn := diceValue == 6

	// Create response
	response := gamemodels.RollDiceResponse{
		GameID:      gameID,
		DiceValue:   diceValue,
		PlayerID:    userID,
		PlayerColor: currentPlayer.Color,
		CanMove:     canMove,
		ValidTokens: validTokens,
		ExtraTurn:   extraTurn,
	}

	utils.SendSuccess(w, "Dice rolled successfully", response)
}

// rollDice generates a random number between 1 and 6
func rollDice() (int, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(6))
	if err != nil {
		return 0, err
	}
	return int(n.Int64()) + 1, nil
}

// MoveToken allows a player to move a token
// @Summary Move a token
// @Description Move a token based on dice roll
// @Tags Game Actions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Game ID"
// @Param request body gamemodels.MoveTokenRequest true "Move Token Request"
// @Success 200 {object} utils.Response{data=gamemodels.MoveTokenResponse} "Token moved successfully"
// @Failure 400 {object} utils.Response "Invalid request"
// @Failure 401 {object} utils.Response "Unauthorized"
// @Failure 403 {object} utils.Response "Not your turn"
// @Failure 404 {object} utils.Response "Game not found"
// @Failure 500 {object} utils.Response "Internal server error"
// @Router /api/game/{id}/move-token [post]
func (h *GameHandler) MoveToken(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value("userID").(int64)
	if !ok {
		utils.SendError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Extract game ID from URL
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

	// Parse request body
	var req gamemodels.MoveTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate token index
	if req.TokenIndex < 0 || req.TokenIndex > 3 {
		utils.SendError(w, http.StatusBadRequest, "Token index must be between 0-3")
		return
	}

	// Validate steps
	if req.Steps < 1 || req.Steps > 6 {
		utils.SendError(w, http.StatusBadRequest, "Steps must be between 1-6")
		return
	}

	// Get game details
	game, err := h.gameRepo.GetGameByID(gameID)
	if err != nil {
		if errors.Is(err, gamerepo.ErrGameNotFound) {
			utils.SendError(w, http.StatusNotFound, "Game not found")
			return
		}
		utils.SendError(w, http.StatusInternalServerError, "Error fetching game details")
		return
	}

	// Check if game is in progress
	if game.Status != gamemodels.GameStatusInProgress {
		utils.SendError(w, http.StatusBadRequest, "Game is not in progress")
		return
	}

	// Check if it's this player's turn
	if game.TurnPlayerID == nil || *game.TurnPlayerID != userID {
		utils.SendError(w, http.StatusForbidden, "Not your turn")
		return
	}

	// Get last dice roll
	lastRoll, err := h.gameRepo.GetLastDiceRoll(gameID, userID)
	if err != nil {
		utils.SendError(w, http.StatusInternalServerError, "Error checking last dice roll")
		return
	}

	if lastRoll == nil {
		utils.SendError(w, http.StatusBadRequest, "Must roll dice before moving")
		return
	}

	// Verify steps match dice roll
	if *lastRoll != req.Steps {
		utils.SendError(w, http.StatusBadRequest, fmt.Sprintf("Steps must match dice value (%d)", *lastRoll))
		return
	}

	// Get player details
	currentPlayer, err := h.gameRepo.GetCurrentPlayerTurn(gameID)
	if err != nil {
		utils.SendError(w, http.StatusInternalServerError, "Error getting player details")
		return
	}

	// Get current token positions
	positions, err := h.gameRepo.GetTokenPositions(gameID, userID)
	if err != nil {
		utils.SendError(w, http.StatusInternalServerError, "Error getting token positions")
		return
	}

	// Find the token to move
	var currentPosition int = -999
	for _, pos := range positions {
		if pos.TokenIndex == req.TokenIndex {
			currentPosition = pos.Position
			break
		}
	}

	if currentPosition == -999 {
		utils.SendError(w, http.StatusBadRequest, "Token not found")
		return
	}

	// Calculate new position
	var newPosition int
	if currentPosition == -1 {
		// Token at home - can only move with 6
		if req.Steps != 6 {
			utils.SendError(w, http.StatusBadRequest, "Need to roll 6 to move token from home")
			return
		}
		newPosition = 0 // Start position
	} else if currentPosition >= 0 && currentPosition < 57 {
		newPosition = currentPosition + req.Steps

		// Check if move exceeds finish
		if newPosition > 57 {
			utils.SendError(w, http.StatusBadRequest, "Move exceeds finish line")
			return
		}
	} else {
		utils.SendError(w, http.StatusBadRequest, "Token already finished")
		return
	}

	// Check if killing opponent
	killedOpponentID, killedTokenIndex, killedOpponentColor, err := h.gameRepo.GetOpponentTokensAtPosition(gameID, userID, newPosition)
	if err != nil {
		utils.SendError(w, http.StatusInternalServerError, "Error checking for opponent tokens")
		return
	}

	isKill := killedOpponentID != nil

	// If killing opponent, send their token back home
	if isKill {
		if err := h.gameRepo.UpdateTokenPosition(gameID, *killedOpponentID, *killedTokenIndex, -1); err != nil {
			utils.SendError(w, http.StatusInternalServerError, "Error resetting opponent token")
			return
		}
	}

	// Move the token
	if err := h.gameRepo.UpdateTokenPosition(gameID, userID, req.TokenIndex, newPosition); err != nil {
		utils.SendError(w, http.StatusInternalServerError, "Error moving token")
		return
	}

	// Record the move
	if err := h.gameRepo.RecordTokenMove(gameID, userID, req.Steps, req.TokenIndex, currentPosition, newPosition, isKill); err != nil {
		utils.SendError(w, http.StatusInternalServerError, "Error recording move")
		return
	}

	// Clear the dice roll
	h.gameRepo.ClearLastDiceRoll(gameID, userID)

	// Record metrics
	utils.TokenMovesTotal.Inc()

	// Check if token reached home
	tokenReachedHome := newPosition == 57

	// Check if player won
	hasWon, err := h.gameRepo.CheckPlayerWin(gameID, userID)
	if err != nil {
		utils.SendError(w, http.StatusInternalServerError, "Error checking win condition")
		return
	}

	var winnerID *int64
	var coinsWon *int
	var coinsLostPerPlayer *int

	if hasWon {
		// Set game winner
		if err := h.gameRepo.SetGameWinner(gameID, userID, game.BetAmount); err != nil {
			utils.SendError(w, http.StatusInternalServerError, "Error setting game winner")
			return
		}

		// Update user coins (add winnings)
		playerCount, _ := h.gameRepo.GetPlayerCount(gameID)
		totalWinnings := game.BetAmount * (playerCount - 1)
		//h.userRepo.UpdateCoins(userID, totalWinnings)

		// Record metrics
		utils.GameCompletionsTotal.Inc()
		utils.ActiveGamesGauge.Dec()

		winnerID = &userID
		coinsWon = &totalWinnings
		coinsLostPerPlayer = &game.BetAmount
	}

	// Determine next turn
	var nextPlayerID *int64
	var nextColor *string
	extraTurn := false

	if !hasWon {
		// Extra turn if rolled 6 or killed opponent
		if req.Steps == 6 || isKill {
			extraTurn = true
			nextPlayerID = &userID
			nextColor = &currentPlayer.Color
		} else {
			// Next player's turn
			nextPlayerID, nextColor, err = h.gameRepo.GetNextPlayer(gameID, userID)
			if err != nil {
				utils.SendError(w, http.StatusInternalServerError, "Error getting next player")
				return
			}
		}

		// Update game turn
		if nextPlayerID != nil {
			if err := h.gameRepo.UpdateGameTurn(gameID, *nextPlayerID); err != nil {
				utils.SendError(w, http.StatusInternalServerError, "Error updating game turn")
				return
			}
		}
	}

	// Create response
	response := gamemodels.MoveTokenResponse{
		GameID:              gameID,
		PlayerID:            userID,
		PlayerColor:         currentPlayer.Color,
		TokenIndex:          req.TokenIndex,
		FromPosition:        currentPosition,
		ToPosition:          newPosition,
		KilledOpponent:      isKill,
		KilledOpponentColor: killedOpponentColor,
		TokenReachedHome:    tokenReachedHome,
		NextTurnPlayerID:    nextPlayerID,
		NextTurnColor:       nextColor,
		ExtraTurn:           extraTurn,
		GameCompleted:       hasWon,
		WinnerID:            winnerID,
		CoinsWon:            coinsWon,
		CoinsLostPerPlayer:  coinsLostPerPlayer,
	}

	message := "Token moved successfully"
	if hasWon {
		message = "Token moved successfully. You won!"
	}

	utils.SendSuccess(w, message, response)
}

// SkipTurn allows a player to skip their turn when no valid moves
// @Summary Skip turn
// @Description Skip turn when no valid moves are available
// @Tags Game Actions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Game ID"
// @Success 200 {object} utils.Response{data=gamemodels.SkipTurnResponse} "Turn skipped successfully"
// @Failure 400 {object} utils.Response "Invalid request or has valid moves"
// @Failure 401 {object} utils.Response "Unauthorized"
// @Failure 403 {object} utils.Response "Not your turn"
// @Failure 404 {object} utils.Response "Game not found"
// @Failure 500 {object} utils.Response "Internal server error"
// @Router /api/game/{id}/skip-turn [post]
func (h *GameHandler) SkipTurn(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value("userID").(int64)
	if !ok {
		utils.SendError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Extract game ID from URL
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

	// Get game details
	game, err := h.gameRepo.GetGameByID(gameID)
	if err != nil {
		if errors.Is(err, gamerepo.ErrGameNotFound) {
			utils.SendError(w, http.StatusNotFound, "Game not found")
			return
		}
		utils.SendError(w, http.StatusInternalServerError, "Error fetching game details")
		return
	}

	// Check if game is in progress
	if game.Status != gamemodels.GameStatusInProgress {
		utils.SendError(w, http.StatusBadRequest, "Game is not in progress")
		return
	}

	// Check if it's this player's turn
	if game.TurnPlayerID == nil || *game.TurnPlayerID != userID {
		utils.SendError(w, http.StatusForbidden, "Not your turn")
		return
	}

	// Get last dice roll
	lastRoll, err := h.gameRepo.GetLastDiceRoll(gameID, userID)
	if err != nil {
		utils.SendError(w, http.StatusInternalServerError, "Error checking last dice roll")
		return
	}

	if lastRoll == nil {
		utils.SendError(w, http.StatusBadRequest, "Must roll dice before skipping turn")
		return
	}

	// Get player details
	currentPlayer, err := h.gameRepo.GetCurrentPlayerTurn(gameID)
	if err != nil {
		utils.SendError(w, http.StatusInternalServerError, "Error getting player details")
		return
	}

	// Get token positions
	positions, err := h.gameRepo.GetTokenPositions(gameID, userID)
	if err != nil {
		utils.SendError(w, http.StatusInternalServerError, "Error getting token positions")
		return
	}

	// Check if player has valid moves
	validTokens := h.gameRepo.GetValidTokensForDiceValue(positions, *lastRoll)
	if len(validTokens) > 0 {
		utils.SendError(w, http.StatusBadRequest, "Cannot skip - you have valid moves available")
		return
	}

	// Clear the dice roll
	h.gameRepo.ClearLastDiceRoll(gameID, userID)

	// Get next player
	nextPlayerID, nextColor, err := h.gameRepo.GetNextPlayer(gameID, userID)
	if err != nil {
		utils.SendError(w, http.StatusInternalServerError, "Error getting next player")
		return
	}

	// Update game turn
	if nextPlayerID != nil {
		if err := h.gameRepo.UpdateGameTurn(gameID, *nextPlayerID); err != nil {
			utils.SendError(w, http.StatusInternalServerError, "Error updating game turn")
			return
		}
	}

	// Create response
	response := gamemodels.SkipTurnResponse{
		GameID:             gameID,
		SkippedPlayerID:    userID,
		SkippedPlayerColor: currentPlayer.Color,
		DiceValue:          *lastRoll,
		NextTurnPlayerID:   *nextPlayerID,
		NextTurnColor:      *nextColor,
	}

	utils.SendSuccess(w, "Turn skipped", response)
}
