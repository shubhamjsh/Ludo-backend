package game

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"Ludo/database"
	"Ludo/models/game"
	gameModel "Ludo/models/game"
	//gamerepo "Ludo/repository/game"
)

var (
	ErrGameNotFound       = errors.New("game not found")
	ErrGameAlreadyStarted = errors.New("game already started")
	ErrGameFull           = errors.New("game is full")
	ErrInvalidRoomCode    = errors.New("invalid room code")
	ErrAlreadyInGame      = errors.New("already in this game")
	ErrNotInGame          = errors.New("not in this game")
	ErrNotGameCreator     = errors.New("only game creator can start the game")
	ErrInsufficientFunds  = errors.New("insufficient coins")
	ErrNotEnoughPlayers   = errors.New("need at least 2 players to start")
)

type GameRepository struct {
	db *sql.DB
}

func NewGameRepository() *GameRepository {
	return &GameRepository{db: database.DB}
}

// generateRoomCode generates a random 6-character uppercase alphanumeric room code
func generateRoomCode() string {
	bytes := make([]byte, 3)
	rand.Read(bytes)
	code := strings.ToUpper(hex.EncodeToString(bytes))
	return code
}

// CreateGame creates a new game room
func (r *GameRepository) CreateGame(userID int64, req *game.CreateGameRequest) (*game.Game, error) {
	gameID := uuid.New().String()
	now := time.Now()

	var roomCode *string
	if req.IsPrivate {
		if req.RoomCode != nil {
			roomCode = req.RoomCode
		} else {
			code := generateRoomCode()
			roomCode = &code
		}
	}

	query := `
		INSERT INTO games (id, room_name, room_code, created_by, max_players, bet_amount, is_private, status, current_turn, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, room_name, room_code, created_by, max_players, bet_amount, is_private, status, current_turn, created_at
	`

	gameLifecycle := &game.Game{}
	err := r.db.QueryRow(
		query,
		gameID,
		req.RoomName,
		roomCode,
		userID,
		req.MaxPlayers,
		req.BetAmount,
		req.IsPrivate,
		game.GameStatusWaiting,
		0,
		now,
	).Scan(
		&gameLifecycle.ID,
		&gameLifecycle.RoomName,
		&gameLifecycle.RoomCode,
		&gameLifecycle.CreatedBy,
		&gameLifecycle.MaxPlayers,
		&gameLifecycle.BetAmount,
		&gameLifecycle.IsPrivate,
		&gameLifecycle.Status,
		&gameLifecycle.CurrentTurn,
		&gameLifecycle.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("error creating game: %w", err)
	}

	return gameLifecycle, nil
}

// GetGameByID retrieves a game by its ID
func (r *GameRepository) GetGameByID(gameID string) (*game.Game, error) {
	query := `
		SELECT id, room_name, room_code, created_by, max_players, bet_amount, is_private, 
		       status, winner_id, current_turn, turn_player_id, created_at, started_at, ended_at
		FROM games
		WHERE id = $1
	`

	game := &game.Game{}
	err := r.db.QueryRow(query, gameID).Scan(
		&game.ID,
		&game.RoomName,
		&game.RoomCode,
		&game.CreatedBy,
		&game.MaxPlayers,
		&game.BetAmount,
		&game.IsPrivate,
		&game.Status,
		&game.WinnerID,
		&game.CurrentTurn,
		&game.TurnPlayerID,
		&game.CreatedAt,
		&game.StartedAt,
		&game.EndedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrGameNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("error getting game: %w", err)
	}

	return game, nil
}

// GetPlayersByGameID retrieves all players in a game
func (r *GameRepository) GetPlayersByGameID(gameID string) ([]game.GamePlayer, error) {
	query := `
		SELECT id, game_id, user_id, color, position, tokens_home, is_winner, coins_won, joined_at
		FROM game_players
		WHERE game_id = $1
		ORDER BY position ASC
	`

	rows, err := r.db.Query(query, gameID)
	if err != nil {
		return nil, fmt.Errorf("error getting players: %w", err)
	}
	defer rows.Close()

	var players []game.GamePlayer
	for rows.Next() {
		var player game.GamePlayer
		err := rows.Scan(
			&player.ID,
			&player.GameID,
			&player.UserID,
			&player.Color,
			&player.Position,
			&player.TokensHome,
			&player.IsWinner,
			&player.CoinsWon,
			&player.JoinedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning player: %w", err)
		}
		players = append(players, player)
	}

	return players, nil
}

// GetPlayersWithUserDetails retrieves all players with their user information
func (r *GameRepository) GetPlayersWithUserDetails(gameID string) ([]game.PlayerWithUser, error) {
	query := `
		SELECT gp.id, gp.game_id, gp.user_id, gp.color, gp.position, gp.tokens_home, 
		       gp.is_winner, gp.coins_won, gp.joined_at, u.name, u.avatar_url
		FROM game_players gp
		JOIN users u ON gp.user_id = u.id
		WHERE gp.game_id = $1
		ORDER BY gp.position ASC
	`

	rows, err := r.db.Query(query, gameID)
	if err != nil {
		return nil, fmt.Errorf("error getting players with user details: %w", err)
	}
	defer rows.Close()

	var players []game.PlayerWithUser
	for rows.Next() {
		var player game.PlayerWithUser
		err := rows.Scan(
			&player.ID,
			&player.GameID,
			&player.UserID,
			&player.Color,
			&player.Position,
			&player.TokensHome,
			&player.IsWinner,
			&player.CoinsWon,
			&player.JoinedAt,
			&player.Name,
			&player.AvatarURL,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning player with user: %w", err)
		}
		players = append(players, player)
	}

	return players, nil
}

// GetPlayerCount returns the number of players in a game
func (r *GameRepository) GetPlayerCount(gameID string) (int, error) {
	query := `SELECT COUNT(*) FROM game_players WHERE game_id = $1`

	var count int
	err := r.db.QueryRow(query, gameID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("error counting players: %w", err)
	}

	return count, nil
}

// IsPlayerInGame checks if a user is already in a game
func (r *GameRepository) IsPlayerInGame(gameID string, userID int64) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM game_players WHERE game_id = $1 AND user_id = $2)`

	var exists bool
	err := r.db.QueryRow(query, gameID, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("error checking player in game: %w", err)
	}

	return exists, nil
}

// getAvailableColor returns the next available color for a game
func (r *GameRepository) getAvailableColor(gameID string) (string, error) {
	usedColors := make(map[string]bool)

	players, err := r.GetPlayersByGameID(gameID)
	if err != nil {
		return "", err
	}

	for _, player := range players {
		usedColors[player.Color] = true
	}

	colors := []string{game.ColorRed, game.ColorBlue, game.ColorGreen, game.ColorYellow}
	for _, color := range colors {
		if !usedColors[color] {
			return color, nil
		}
	}

	return "", ErrGameFull
}

// JoinGame adds a player to a game
func (r *GameRepository) JoinGame(gameID string, userID int64) (*game.JoinGameResponse, error) {
	// Get game details
	gameLifecycle, err := r.GetGameByID(gameID)
	if err != nil {
		return nil, err
	}

	// Check if game already started
	if gameLifecycle.Status != game.GameStatusWaiting {
		return nil, ErrGameAlreadyStarted
	}

	// Check if user already in game
	inGame, err := r.IsPlayerInGame(gameID, userID)
	if err != nil {
		return nil, err
	}
	if inGame {
		return nil, ErrAlreadyInGame
	}

	// Check if game is full
	playerCount, err := r.GetPlayerCount(gameID)
	if err != nil {
		return nil, err
	}
	if playerCount >= gameLifecycle.MaxPlayers {
		return nil, ErrGameFull
	}

	// Get available color
	color, err := r.getAvailableColor(gameID)
	if err != nil {
		return nil, err
	}

	// Add player to game
	position := playerCount + 1
	query := `
		INSERT INTO game_players (game_id, user_id, color, position, tokens_home, is_winner, coins_won, joined_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`

	var playerID int64
	err = r.db.QueryRow(
		query,
		gameID,
		userID,
		color,
		position,
		4, // All 4 tokens start at home
		false,
		0,
		time.Now(),
	).Scan(&playerID)

	if err != nil {
		return nil, fmt.Errorf("error adding player to game: %w", err)
	}

	// Return response
	response := &game.JoinGameResponse{
		GameID:       gameID,
		PlayerColor:  color,
		Position:     position,
		CurrentCount: position,
		MaxPlayers:   gameLifecycle.MaxPlayers,
	}

	return response, nil
}

// StartGame starts a game (only creator can start)
func (r *GameRepository) StartGame(gameID string, userID int64) (*game.Game, error) {
	// Get game details
	gameLifecycle, err := r.GetGameByID(gameID)
	if err != nil {
		return nil, err
	}

	// Check if user is the creator
	if gameLifecycle.CreatedBy != userID {
		return nil, ErrNotGameCreator
	}

	// Check if game already started
	if gameLifecycle.Status != game.GameStatusWaiting {
		return nil, ErrGameAlreadyStarted
	}

	// Check if enough players
	playerCount, err := r.GetPlayerCount(gameID)
	if err != nil {
		return nil, err
	}
	if playerCount < 2 {
		return nil, ErrNotEnoughPlayers
	}

	// Get first player (position 1)
	players, err := r.GetPlayersByGameID(gameID)
	if err != nil {
		return nil, err
	}

	var firstPlayerID *int64
	if len(players) > 0 {
		firstPlayerID = &players[0].UserID
	}

	// Update game status
	now := time.Now()
	query := `
		UPDATE games 
		SET status = $1, started_at = $2, current_turn = 1, turn_player_id = $3
		WHERE id = $4
		RETURNING id, room_name, room_code, created_by, max_players, bet_amount, is_private, 
		          status, winner_id, current_turn, turn_player_id, created_at, started_at, ended_at
	`

	updatedGame := &game.Game{}
	err = r.db.QueryRow(
		query,
		game.GameStatusInProgress,
		now,
		firstPlayerID,
		gameID,
	).Scan(
		&updatedGame.ID,
		&updatedGame.RoomName,
		&updatedGame.RoomCode,
		&updatedGame.CreatedBy,
		&updatedGame.MaxPlayers,
		&updatedGame.BetAmount,
		&updatedGame.IsPrivate,
		&updatedGame.Status,
		&updatedGame.WinnerID,
		&updatedGame.CurrentTurn,
		&updatedGame.TurnPlayerID,
		&updatedGame.CreatedAt,
		&updatedGame.StartedAt,
		&updatedGame.EndedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("error starting game: %w", err)
	}

	return updatedGame, nil
}

// LeaveGame removes a player from a game
func (r *GameRepository) LeaveGame(gameID string, userID int64) (*game.Game, error) {
	// Get game details
	game, err := r.GetGameByID(gameID)
	if err != nil {
		return nil, err
	}

	// Check if user is in the game
	inGame, err := r.IsPlayerInGame(gameID, userID)
	if err != nil {
		return nil, err
	}
	if !inGame {
		return nil, ErrNotInGame
	}

	// Remove player from game
	deleteQuery := `DELETE FROM game_players WHERE game_id = $1 AND user_id = $2`
	_, err = r.db.Exec(deleteQuery, gameID, userID)
	if err != nil {
		return nil, fmt.Errorf("error removing player from game: %w", err)
	}

	// Check if creator left or no players remaining
	playerCount, err := r.GetPlayerCount(gameID)
	if err != nil {
		return nil, err
	}

	// If creator left OR no players left, cancel the game
	if game.CreatedBy == userID || playerCount == 0 {
		cancelQuery := `UPDATE games SET status = $1 WHERE id = $2`
		_, err = r.db.Exec(cancelQuery, gameModel.GameStatusCancelled, gameID)
		if err != nil {
			return nil, fmt.Errorf("error cancelling game: %w", err)
		}
		game.Status = gameModel.GameStatusCancelled
	}

	// Return updated game
	return r.GetGameByID(gameID)
}

// DeleteGame deletes a game (for rollback purposes)
func (r *GameRepository) DeleteGame(gameID string) error {
	// First delete all players
	_, err := r.db.Exec(`DELETE FROM game_players WHERE game_id = $1`, gameID)
	if err != nil {
		return fmt.Errorf("error deleting game players: %w", err)
	}

	// Then delete the game
	_, err = r.db.Exec(`DELETE FROM games WHERE id = $1`, gameID)
	if err != nil {
		return fmt.Errorf("error deleting game: %w", err)
	}

	return nil
}
