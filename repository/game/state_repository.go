package game

import (
	gamemodels "Ludo/models/game"
	"database/sql"
	"fmt"
)

// GetAllTokenPositionsByGame gets token positions for all players in a game
func (r *GameRepository) GetAllTokenPositionsByGame(gameID string) (map[int64][]gamemodels.TokenPosition, error) {
	query := `
		SELECT user_id, token_index, position
		FROM token_positions
		WHERE game_id = $1
		ORDER BY user_id, token_index
	`

	rows, err := r.db.Query(query, gameID)
	if err != nil {
		return nil, fmt.Errorf("error getting all token positions: %w", err)
	}
	defer rows.Close()

	positionsMap := make(map[int64][]gamemodels.TokenPosition)

	for rows.Next() {
		var userID int64
		var pos gamemodels.TokenPosition

		if err := rows.Scan(&userID, &pos.TokenIndex, &pos.Position); err != nil {
			return nil, err
		}

		positionsMap[userID] = append(positionsMap[userID], pos)
	}

	// If no positions found for a player, initialize with home positions
	players, err := r.GetPlayersByGameID(gameID)
	if err != nil {
		return nil, err
	}

	for _, player := range players {
		if _, exists := positionsMap[player.UserID]; !exists {
			// Initialize all tokens at home
			for i := 0; i < 4; i++ {
				positionsMap[player.UserID] = append(positionsMap[player.UserID], gamemodels.TokenPosition{
					TokenIndex: i,
					Position:   -1,
				})
			}
		}
	}

	return positionsMap, nil
}

// GetCurrentTurnPlayerDetails gets detailed info about current turn player
func (r *GameRepository) GetCurrentTurnPlayerDetails(gameID string) (*gamemodels.CurrentTurnPlayer, error) {
	game, err := r.GetGameByID(gameID)
	if err != nil {
		return nil, err
	}

	if game.TurnPlayerID == nil {
		return nil, nil
	}

	query := `
		SELECT gp.user_id, u.name, gp.color
		FROM game_players gp
		JOIN users u ON gp.user_id = u.id
		WHERE gp.game_id = $1 AND gp.user_id = $2
	`

	var turnPlayer gamemodels.CurrentTurnPlayer
	err = r.db.QueryRow(query, gameID, *game.TurnPlayerID).Scan(
		&turnPlayer.UserID,
		&turnPlayer.Name,
		&turnPlayer.Color,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error getting current turn player: %w", err)
	}

	return &turnPlayer, nil
}
