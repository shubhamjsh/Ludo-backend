package game

import (
	"time"
)

// Game status constants
const (
	GameStatusWaiting    = "waiting"
	GameStatusInProgress = "in_progress"
	GameStatusCompleted  = "completed"
	GameStatusCancelled  = "cancelled"
)

// Player colors
const (
	ColorRed    = "red"
	ColorBlue   = "blue"
	ColorGreen  = "green"
	ColorYellow = "yellow"
)

// Game represents a Ludo game room
type Game struct {
	ID           string     `json:"id" db:"id"`
	RoomName     string     `json:"room_name" db:"room_name"`
	RoomCode     *string    `json:"room_code,omitempty" db:"room_code"`
	CreatedBy    int64      `json:"created_by" db:"created_by"`
	MaxPlayers   int        `json:"max_players" db:"max_players"`
	BetAmount    int        `json:"bet_amount" db:"bet_amount"`
	IsPrivate    bool       `json:"is_private" db:"is_private"`
	Status       string     `json:"status" db:"status"`
	WinnerID     *int64     `json:"winner_id,omitempty" db:"winner_id"`
	CurrentTurn  int        `json:"current_turn" db:"current_turn"`
	TurnPlayerID *int64     `json:"turn_player_id,omitempty" db:"turn_player_id"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	StartedAt    *time.Time `json:"started_at,omitempty" db:"started_at"`
	EndedAt      *time.Time `json:"ended_at,omitempty" db:"ended_at"`
}

// GamePlayer represents a player in a game
type GamePlayer struct {
	ID         int64     `json:"id" db:"id"`
	GameID     string    `json:"game_id" db:"game_id"`
	UserID     int64     `json:"user_id" db:"user_id"`
	Color      string    `json:"color" db:"color"`
	Position   int       `json:"position" db:"position"`
	TokensHome int       `json:"tokens_home" db:"tokens_home"`
	IsWinner   bool      `json:"is_winner" db:"is_winner"`
	CoinsWon   int       `json:"coins_won" db:"coins_won"`
	JoinedAt   time.Time `json:"joined_at" db:"joined_at"`
}

// PlayerInfo represents basic player information for responses
type PlayerInfo struct {
	UserID   int64  `json:"user_id"`
	Name     string `json:"name"`
	Color    string `json:"color"`
	Position int    `json:"position"`
}

// ==================== REQUEST MODELS ====================

// CreateGameRequest represents the request to create a new game
type CreateGameRequest struct {
	RoomName   string  `json:"room_name" validate:"required,min=3,max=100"`
	MaxPlayers int     `json:"max_players" validate:"required,min=2,max=4"`
	BetAmount  int     `json:"bet_amount" validate:"min=0"`
	IsPrivate  bool    `json:"is_private"`
	RoomCode   *string `json:"room_code,omitempty" validate:"omitempty,len=6"`
}

// JoinGameRequest represents the request to join a game
type JoinGameRequest struct {
	GameID   string  `json:"game_id" validate:"required"`
	RoomCode *string `json:"room_code,omitempty"`
}

// ==================== RESPONSE MODELS ====================

// CreateGameResponse represents the response after creating a game
type CreateGameResponse struct {
	GameID    string  `json:"game_id"`
	RoomCode  *string `json:"room_code,omitempty"`
	CreatedBy int64   `json:"created_by"`
	Status    string  `json:"status"`
}

// JoinGameResponse represents the response after joining a game
type JoinGameResponse struct {
	GameID       string `json:"game_id"`
	PlayerColor  string `json:"player_color"`
	Position     int    `json:"position"`
	CurrentCount int    `json:"current_players"`
	MaxPlayers   int    `json:"max_players"`
}

// StartGameResponse represents the response after starting a game
type StartGameResponse struct {
	GameID    string       `json:"game_id"`
	Status    string       `json:"status"`
	StartedAt time.Time    `json:"started_at"`
	Players   []PlayerInfo `json:"players"`
}

// LeaveGameResponse represents the response after leaving a game
type LeaveGameResponse struct {
	GameID string `json:"game_id"`
	Status string `json:"status"`
}

// CreateLocalGameRequest represents the request to create a local multiplayer game
type CreateLocalGameRequest struct {
	RoomName    string   `json:"room_name" validate:"required,min=3,max=100"`
	PlayerNames []string `json:"player_names" validate:"required,min=2,max=4"`
}

// LocalPlayerInfo represents player info in local multiplayer
type LocalPlayerInfo struct {
	Name   string `json:"name"`
	Token  string `json:"token"`
	Color  string `json:"color"`
	UserID int64  `json:"user_id"`
}

// CreateLocalGameResponse represents the response after creating a local game
type CreateLocalGameResponse struct {
	GameID  string            `json:"game_id"`
	Status  string            `json:"status"`
	Players []LocalPlayerInfo `json:"players"`
}
