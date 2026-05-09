package game

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	gamemodels "Ludo/models/game"
)

// GetCurrentPlayerTurn gets the player whose turn it is
func (r *GameRepository) GetCurrentPlayerTurn(gameID string) (*gamemodels.GamePlayer, error) {
	game, err := r.GetGameByID(gameID)
	if err != nil {
		return nil, err
	}

	if game.TurnPlayerID == nil {
		return nil, errors.New("no current turn player")
	}

	query := `
		SELECT id, game_id, user_id, color, position, tokens_home, is_winner, coins_won, joined_at
		FROM game_players
		WHERE game_id = $1 AND user_id = $2
	`

	var player gamemodels.GamePlayer
	err = r.db.QueryRow(query, gameID, *game.TurnPlayerID).Scan(
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
		return nil, fmt.Errorf("error getting current player: %w", err)
	}

	return &player, nil
}

// RecordDiceRoll saves a dice roll to the database
func (r *GameRepository) RecordDiceRoll(gameID string, playerID int64, diceValue int) error {
	query := `
		INSERT INTO game_moves (game_id, player_id, dice_value, is_kill, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := r.db.Exec(query, gameID, playerID, diceValue, false, time.Now())
	if err != nil {
		return fmt.Errorf("error recording dice roll: %w", err)
	}

	return nil
}

// GetLastDiceRoll gets the last dice roll for a player in current turn
func (r *GameRepository) GetLastDiceRoll(gameID string, playerID int64) (*int, error) {
	query := `
		SELECT dice_value 
		FROM game_moves 
		WHERE game_id = $1 AND player_id = $2 AND token_index IS NULL
		ORDER BY created_at DESC 
		LIMIT 1
	`

	var diceValue int
	err := r.db.QueryRow(query, gameID, playerID).Scan(&diceValue)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error getting last dice roll: %w", err)
	}

	return &diceValue, nil
}

// GetTokenPositions gets all token positions for a player
func (r *GameRepository) GetTokenPositions(gameID string, playerID int64) ([]gamemodels.TokenPosition, error) {
	query := `
		SELECT token_index, position 
		FROM token_positions 
		WHERE game_id = $1 AND user_id = $2
		ORDER BY token_index
	`

	rows, err := r.db.Query(query, gameID, playerID)
	if err != nil {
		return nil, fmt.Errorf("error getting token positions: %w", err)
	}
	defer rows.Close()

	var positions []gamemodels.TokenPosition
	for rows.Next() {
		var pos gamemodels.TokenPosition
		if err := rows.Scan(&pos.TokenIndex, &pos.Position); err != nil {
			return nil, err
		}
		positions = append(positions, pos)
	}

	// If no positions found, initialize all tokens at home (-1)
	if len(positions) == 0 {
		for i := 0; i < 4; i++ {
			positions = append(positions, gamemodels.TokenPosition{
				TokenIndex: i,
				Position:   -1, // -1 means at home
			})
		}
	}

	return positions, nil
}

// InitializeTokenPositions creates initial token positions for a player
func (r *GameRepository) InitializeTokenPositions(gameID string, playerID int64) error {
	query := `
		INSERT INTO token_positions (game_id, user_id, token_index, position)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (game_id, user_id, token_index) DO NOTHING
	`

	for i := 0; i < 4; i++ {
		_, err := r.db.Exec(query, gameID, playerID, i, -1) // All tokens start at home
		if err != nil {
			return fmt.Errorf("error initializing token positions: %w", err)
		}
	}

	return nil
}

// GetValidTokensForDiceValue returns which tokens can move with given dice value
func (r *GameRepository) GetValidTokensForDiceValue(positions []gamemodels.TokenPosition, diceValue int) []int {
	var validTokens []int

	for _, pos := range positions {
		// Token at home
		if pos.Position == -1 {
			// Can only move out with a 6
			if diceValue == 6 {
				validTokens = append(validTokens, pos.TokenIndex)
			}
		} else if pos.Position >= 0 && pos.Position < 57 {
			// Token on board - can move if won't exceed finish
			if pos.Position+diceValue <= 57 {
				validTokens = append(validTokens, pos.TokenIndex)
			}
		}
		// Token at position 57 (finished) cannot move
	}

	return validTokens
}

// UpdateTokenPosition updates a token's position
func (r *GameRepository) UpdateTokenPosition(gameID string, playerID int64, tokenIndex int, newPosition int) error {
	query := `
		UPDATE token_positions
		SET position = $1
		WHERE game_id = $2 AND user_id = $3 AND token_index = $4
	`

	result, err := r.db.Exec(query, newPosition, gameID, playerID, tokenIndex)
	if err != nil {
		return fmt.Errorf("error updating token position: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return errors.New("token position not found")
	}

	return nil
}

// RecordTokenMove saves a token move to the database
func (r *GameRepository) RecordTokenMove(gameID string, playerID int64, diceValue, tokenIndex, fromPosition, toPosition int, isKill bool) error {
	query := `
		INSERT INTO game_moves (game_id, player_id, dice_value, token_index, from_position, to_position, is_kill, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := r.db.Exec(query, gameID, playerID, diceValue, tokenIndex, fromPosition, toPosition, isKill, time.Now())
	if err != nil {
		return fmt.Errorf("error recording token move: %w", err)
	}

	return nil
}

// GetOpponentTokensAtPosition checks if any opponent tokens are at the given position
func (r *GameRepository) GetOpponentTokensAtPosition(gameID string, currentPlayerID int64, position int) (*int64, *int, *string, error) {
	// Get all players in the game
	players, err := r.GetPlayersByGameID(gameID)
	if err != nil {
		return nil, nil, nil, err
	}

	// Check each opponent's tokens
	for _, player := range players {
		if player.UserID == currentPlayerID {
			continue // Skip current player
		}

		// Get this opponent's token positions
		positions, err := r.GetTokenPositions(gameID, player.UserID)
		if err != nil {
			continue
		}

		// Check if any token is at the target position
		for _, tokenPos := range positions {
			if tokenPos.Position == position && position > 0 { // Can't kill at home (-1) or start (0)
				return &player.UserID, &tokenPos.TokenIndex, &player.Color, nil
			}
		}
	}

	return nil, nil, nil, nil // No opponent found
}

// GetNextPlayer returns the next player in turn order
func (r *GameRepository) GetNextPlayer(gameID string, currentPlayerID int64) (*int64, *string, error) {
	query := `
		SELECT user_id, color
		FROM game_players
		WHERE game_id = $1 AND position > (
			SELECT position FROM game_players WHERE game_id = $1 AND user_id = $2
		)
		ORDER BY position ASC
		LIMIT 1
	`

	var nextPlayerID int64
	var nextColor string
	err := r.db.QueryRow(query, gameID, currentPlayerID).Scan(&nextPlayerID, &nextColor)

	if err == sql.ErrNoRows {
		// Wrap around to first player
		query = `
			SELECT user_id, color
			FROM game_players
			WHERE game_id = $1
			ORDER BY position ASC
			LIMIT 1
		`
		err = r.db.QueryRow(query, gameID).Scan(&nextPlayerID, &nextColor)
		if err != nil {
			return nil, nil, fmt.Errorf("error getting next player: %w", err)
		}
	} else if err != nil {
		return nil, nil, fmt.Errorf("error getting next player: %w", err)
	}

	return &nextPlayerID, &nextColor, nil
}

// UpdateGameTurn updates whose turn it is
func (r *GameRepository) UpdateGameTurn(gameID string, nextPlayerID int64) error {
	query := `
		UPDATE games
		SET turn_player_id = $1, current_turn = current_turn + 1
		WHERE id = $2
	`

	_, err := r.db.Exec(query, nextPlayerID, gameID)
	if err != nil {
		return fmt.Errorf("error updating game turn: %w", err)
	}

	return nil
}

// CheckPlayerWin checks if a player has won (all 4 tokens at position 57)
func (r *GameRepository) CheckPlayerWin(gameID string, playerID int64) (bool, error) {
	positions, err := r.GetTokenPositions(gameID, playerID)
	if err != nil {
		return false, err
	}

	finishedCount := 0
	for _, pos := range positions {
		if pos.Position == 57 {
			finishedCount++
		}
	}

	return finishedCount == 4, nil
}

// SetGameWinner marks a player as winner and completes the game
func (r *GameRepository) SetGameWinner(gameID string, winnerID int64, betAmount int) error {
	now := time.Now()

	// Update game status
	query := `
		UPDATE games
		SET status = $1, winner_id = $2, ended_at = $3
		WHERE id = $4
	`

	_, err := r.db.Exec(query, gamemodels.GameStatusCompleted, winnerID, now, gameID)
	if err != nil {
		return fmt.Errorf("error setting game winner: %w", err)
	}

	// Get player count
	playerCount, err := r.GetPlayerCount(gameID)
	if err != nil {
		return err
	}

	// Calculate winnings (winner gets bet amount from all other players)
	coinsWon := betAmount * (playerCount - 1)

	// Update winner's game_player record
	updatePlayerQuery := `
		UPDATE game_players
		SET is_winner = true, coins_won = $1
		WHERE game_id = $2 AND user_id = $3
	`

	_, err = r.db.Exec(updatePlayerQuery, coinsWon, gameID, winnerID)
	if err != nil {
		return fmt.Errorf("error updating winner player record: %w", err)
	}

	return nil
}

// ClearLastDiceRoll clears the last dice roll after a move is made
func (r *GameRepository) ClearLastDiceRoll(gameID string, playerID int64) error {
	query := `
		DELETE FROM game_moves
		WHERE id IN (
			SELECT id FROM game_moves
			WHERE game_id = $1 AND player_id = $2 AND token_index IS NULL
			ORDER BY created_at DESC
			LIMIT 1
		)
	`

	_, err := r.db.Exec(query, gameID, playerID)
	return err
}
