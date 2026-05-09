package game

import "time"

// StateResponse GameStateResponse represents the complete game state
type StateResponse struct {
	Game              Game                   `json:"game"`
	Players           []PlayerWithUser       `json:"players"`
	TokenPositions    []PlayerTokenPositions `json:"token_positions"`
	CurrentTurnPlayer *CurrentTurnPlayer     `json:"current_turn_player"`
	LastDiceRoll      *int                   `json:"last_dice_roll"`
}

// PlayerTokenPositions represents all token positions for a player
type PlayerTokenPositions struct {
	UserID int64           `json:"user_id"`
	Color  string          `json:"color"`
	Tokens []TokenPosition `json:"tokens"`
}

// CurrentTurnPlayer represents whose turn it is
type CurrentTurnPlayer struct {
	UserID int64  `json:"user_id"`
	Name   string `json:"name"`
	Color  string `json:"color"`
}

// PlayerWithUser GamePlayerWithUser represents a game player with user details
type PlayerWithUser struct {
	ID         int64     `json:"id"`
	GameID     string    `json:"game_id"`
	UserID     int64     `json:"user_id"`
	Color      string    `json:"color"`
	Position   int       `json:"position"`
	TokensHome int       `json:"tokens_home"`
	IsWinner   bool      `json:"is_winner"`
	CoinsWon   int       `json:"coins_won"`
	JoinedAt   time.Time `json:"joined_at"`
	Name       string    `json:"name"`
	AvatarURL  string    `json:"avatar_url"`
}
