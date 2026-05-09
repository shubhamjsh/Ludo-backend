package game

import "time"

// ==================== GAME ACTION MODELS ====================

// RollDiceResponse represents the response after rolling dice
type RollDiceResponse struct {
	GameID      string `json:"game_id"`
	DiceValue   int    `json:"dice_value"`
	PlayerID    int64  `json:"player_id"`
	PlayerColor string `json:"player_color"`
	CanMove     bool   `json:"can_move"`
	ValidTokens []int  `json:"valid_tokens"`
	ExtraTurn   bool   `json:"extra_turn"`
}

// GameMove represents a dice roll or move in the game
type GameMove struct {
	ID           int64     `json:"id" db:"id"`
	GameID       string    `json:"game_id" db:"game_id"`
	PlayerID     int64     `json:"player_id" db:"player_id"`
	DiceValue    int       `json:"dice_value" db:"dice_value"`
	TokenIndex   *int      `json:"token_index,omitempty" db:"token_index"`
	FromPosition *int      `json:"from_position,omitempty" db:"from_position"`
	ToPosition   *int      `json:"to_position,omitempty" db:"to_position"`
	IsKill       bool      `json:"is_kill" db:"is_kill"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// TokenPosition represents the current position of a player's tokens
type TokenPosition struct {
	TokenIndex int `json:"token_index"`
	Position   int `json:"position"` // -1 = home, 0-56 = board, 57 = finished
}

// MoveTokenRequest represents the request to move a token
type MoveTokenRequest struct {
	TokenIndex int `json:"token_index" validate:"required,min=0,max=3"`
	Steps      int `json:"steps" validate:"required,min=1,max=6"`
}

// MoveTokenResponse represents the response after moving a token
type MoveTokenResponse struct {
	GameID              string  `json:"game_id"`
	PlayerID            int64   `json:"player_id"`
	PlayerColor         string  `json:"player_color"`
	TokenIndex          int     `json:"token_index"`
	FromPosition        int     `json:"from_position"`
	ToPosition          int     `json:"to_position"`
	KilledOpponent      bool    `json:"killed_opponent"`
	KilledOpponentColor *string `json:"killed_opponent_color,omitempty"`
	TokenReachedHome    bool    `json:"token_reached_home"`
	NextTurnPlayerID    *int64  `json:"next_turn_player_id"`
	NextTurnColor       *string `json:"next_turn_color"`
	ExtraTurn           bool    `json:"extra_turn"`
	GameCompleted       bool    `json:"game_completed"`
	WinnerID            *int64  `json:"winner_id,omitempty"`
	CoinsWon            *int    `json:"coins_won,omitempty"`
	CoinsLostPerPlayer  *int    `json:"coins_lost_per_player,omitempty"`
}

// SkipTurnResponse represents the response after skipping turn
type SkipTurnResponse struct {
	GameID             string `json:"game_id"`
	SkippedPlayerID    int64  `json:"skipped_player_id"`
	SkippedPlayerColor string `json:"skipped_player_color"`
	DiceValue          int    `json:"dice_value"`
	NextTurnPlayerID   int64  `json:"next_turn_player_id"`
	NextTurnColor      string `json:"next_turn_color"`
}
