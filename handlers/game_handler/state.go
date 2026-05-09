package game_handler

import (
	"net/http"
	"strings"

	gamemodels "Ludo/models/game"
	gamerepo "Ludo/repository/game"

	"Ludo/utils"
)

// GetGameState retrieves the complete state of a game
// @Summary Get game state
// @Description Get complete game state including players and token positions
// @Tags Game State
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Game ID"
// @Success 200 {object} utils.Response{data=game.GameStateResponse} "Game state retrieved successfully"
// @Failure 401 {object} utils.Response "Unauthorized"
// @Failure 404 {object} utils.Response "Game not found"
// @Failure 500 {object} utils.Response "Internal server error"
// @Router /api/game/{id}/state [get]
func (h *GameHandler) GetGameState(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (for authentication)
	_, ok := r.Context().Value("userID").(int64)
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
		if err == gamerepo.ErrGameNotFound {
			utils.SendError(w, http.StatusNotFound, "Game not found")
			return
		}
		utils.SendError(w, http.StatusInternalServerError, "Error fetching game details")
		return
	}

	// Get all players with user details
	players, err := h.gameRepo.GetPlayersWithUserDetails(gameID)
	if err != nil {
		utils.SendError(w, http.StatusInternalServerError, "Error fetching players")
		return
	}

	// Get all token positions
	tokenPositionsMap, err := h.gameRepo.GetAllTokenPositionsByGame(gameID)
	if err != nil {
		utils.SendError(w, http.StatusInternalServerError, "Error fetching token positions")
		return
	}

	// Convert token positions map to array format
	var tokenPositions []gamemodels.PlayerTokenPositions
	for _, player := range players {
		positions, exists := tokenPositionsMap[player.UserID]
		if !exists {
			// Initialize with home positions if not found
			positions = make([]gamemodels.TokenPosition, 4)
			for i := 0; i < 4; i++ {
				positions[i] = gamemodels.TokenPosition{
					TokenIndex: i,
					Position:   -1,
				}
			}
		}

		tokenPositions = append(tokenPositions, gamemodels.PlayerTokenPositions{
			UserID: player.UserID,
			Color:  player.Color,
			Tokens: positions,
		})
	}

	// Get current turn player details
	currentTurnPlayer, err := h.gameRepo.GetCurrentTurnPlayerDetails(gameID)
	if err != nil {
		utils.SendError(w, http.StatusInternalServerError, "Error fetching current turn player")
		return
	}

	// Get last dice roll (if any)
	var lastDiceRoll *int
	if game.TurnPlayerID != nil {
		lastRoll, err := h.gameRepo.GetLastDiceRoll(gameID, *game.TurnPlayerID)
		if err == nil {
			lastDiceRoll = lastRoll
		}
	}

	// Convert game to response format
	gameResponse := gamemodels.Game{
		ID:           game.ID,
		RoomName:     game.RoomName,
		RoomCode:     game.RoomCode,
		CreatedBy:    game.CreatedBy,
		MaxPlayers:   game.MaxPlayers,
		BetAmount:    game.BetAmount,
		IsPrivate:    game.IsPrivate,
		Status:       game.Status,
		WinnerID:     game.WinnerID,
		CurrentTurn:  game.CurrentTurn,
		TurnPlayerID: game.TurnPlayerID,
		CreatedAt:    game.CreatedAt,
		StartedAt:    game.StartedAt,
		EndedAt:      game.EndedAt,
	}

	// Convert players to response format
	var playersResponse []gamemodels.PlayerWithUser
	for _, p := range players {
		playersResponse = append(playersResponse, gamemodels.PlayerWithUser{
			ID:         p.ID,
			GameID:     p.GameID,
			UserID:     p.UserID,
			Color:      p.Color,
			Position:   p.Position,
			TokensHome: p.TokensHome,
			IsWinner:   p.IsWinner,
			CoinsWon:   p.CoinsWon,
			JoinedAt:   p.JoinedAt,
			Name:       p.Name,
			AvatarURL:  p.AvatarURL,
		})
	}

	// Create response
	response := gamemodels.StateResponse{
		Game:              gameResponse,
		Players:           playersResponse,
		TokenPositions:    tokenPositions,
		CurrentTurnPlayer: currentTurnPlayer,
		LastDiceRoll:      lastDiceRoll,
	}

	utils.SendSuccess(w, "Game state retrieved successfully", response)
}
