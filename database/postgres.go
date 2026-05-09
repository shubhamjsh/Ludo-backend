package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
)

var DB *sql.DB

type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

func Connect(cfg Config) error {
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
	)

	var err error
	DB, err = sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("error opening database: %w", err)
	}

	// Set connection pool settings
	DB.SetMaxOpenConns(25)
	DB.SetMaxIdleConns(5)
	DB.SetConnMaxLifetime(5 * time.Minute)

	// Test the connection
	if err = DB.Ping(); err != nil {
		return fmt.Errorf("error connecting to database: %w", err)
	}

	log.Println("Successfully connected to PostgreSQL database")
	return nil
}

func Close() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}

// InitSchema creates the users table if it doesn't exist
func InitSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		name VARCHAR(100) NOT NULL,
		phone VARCHAR(20) UNIQUE,
		email VARCHAR(255) UNIQUE,
		password VARCHAR(255),
		avatar_url TEXT DEFAULT '',
		coins INTEGER DEFAULT 1000,
		is_guest BOOLEAN DEFAULT FALSE,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		last_login_at TIMESTAMP,
		CONSTRAINT check_email_or_phone CHECK (
			(email IS NOT NULL AND email != '') OR 
			(phone IS NOT NULL AND phone != '') OR 
			is_guest = TRUE
		)
	);

	CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
	CREATE INDEX IF NOT EXISTS idx_users_phone ON users(phone);
	CREATE INDEX IF NOT EXISTS idx_users_is_guest ON users(is_guest);
	`

	_, err := DB.Exec(schema)
	if err != nil {
		return fmt.Errorf("error creating schema: %w", err)
	}

	log.Println("Database schema initialized successfully")
	return nil
}

// InitGameSchema creates the game-related tables if they don't exist
func InitGameSchema() error {
	schema := `
	-- Games table
	CREATE TABLE IF NOT EXISTS games (
		id UUID PRIMARY KEY,
		room_name VARCHAR(100) NOT NULL,
		room_code VARCHAR(10) UNIQUE,
		created_by BIGINT NOT NULL REFERENCES users(id),
		max_players INTEGER DEFAULT 4 CHECK (max_players >= 2 AND max_players <= 4),
		bet_amount INTEGER DEFAULT 0 CHECK (bet_amount >= 0),
		is_private BOOLEAN DEFAULT FALSE,
		status VARCHAR(20) DEFAULT 'waiting' CHECK (status IN ('waiting', 'in_progress', 'completed', 'cancelled')),
		winner_id BIGINT REFERENCES users(id),
		current_turn INTEGER DEFAULT 0,
		turn_player_id BIGINT REFERENCES users(id),
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		started_at TIMESTAMP,
		ended_at TIMESTAMP
	);

	-- Game players table
	CREATE TABLE IF NOT EXISTS game_players (
		id SERIAL PRIMARY KEY,
		game_id UUID NOT NULL REFERENCES games(id) ON DELETE CASCADE,
		user_id BIGINT NOT NULL REFERENCES users(id),
		color VARCHAR(10) NOT NULL CHECK (color IN ('red', 'blue', 'green', 'yellow')),
		position INTEGER NOT NULL CHECK (position >= 1 AND position <= 4),
		tokens_home INTEGER DEFAULT 0 CHECK (tokens_home >= 0 AND tokens_home <= 4),
		is_winner BOOLEAN DEFAULT FALSE,
		coins_won INTEGER DEFAULT 0,
		joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(game_id, user_id),
		UNIQUE(game_id, color),
		UNIQUE(game_id, position)
	);

	-- Game tokens table
	CREATE TABLE IF NOT EXISTS game_tokens (
		id SERIAL PRIMARY KEY,
		game_id UUID NOT NULL REFERENCES games(id) ON DELETE CASCADE,
		player_id BIGINT NOT NULL REFERENCES users(id),
		token_index INTEGER NOT NULL CHECK (token_index >= 0 AND token_index <= 3),
		position INTEGER DEFAULT -1,
		is_home BOOLEAN DEFAULT FALSE,
		is_safe BOOLEAN DEFAULT FALSE,
		UNIQUE(game_id, player_id, token_index)
	);

	CREATE TABLE IF NOT EXISTS game_moves (
		id SERIAL PRIMARY KEY,
		game_id UUID NOT NULL,
		player_id BIGINT NOT NULL,
		dice_value INTEGER NOT NULL,
		token_index INTEGER,
		from_position INTEGER,
		to_position INTEGER,
		is_kill BOOLEAN DEFAULT FALSE,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (game_id) REFERENCES games(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS token_positions (
		id SERIAL PRIMARY KEY,
		game_id UUID NOT NULL,
		user_id BIGINT NOT NULL,
		token_index INTEGER NOT NULL,
		position INTEGER NOT NULL DEFAULT -1,
		UNIQUE(game_id, user_id, token_index),
		FOREIGN KEY (game_id) REFERENCES games(id) ON DELETE CASCADE
	);

	-- Indexes for better query performance
	CREATE INDEX IF NOT EXISTS idx_games_status ON games(status);
	CREATE INDEX IF NOT EXISTS idx_games_created_by ON games(created_by);
	CREATE INDEX IF NOT EXISTS idx_games_created_at ON games(created_at);
	CREATE INDEX IF NOT EXISTS idx_games_room_code ON games(room_code) WHERE room_code IS NOT NULL;
	CREATE INDEX IF NOT EXISTS idx_game_players_game_id ON game_players(game_id);
	CREATE INDEX IF NOT EXISTS idx_game_players_user_id ON game_players(user_id);
	CREATE INDEX IF NOT EXISTS idx_game_tokens_game_id ON game_tokens(game_id);
	CREATE INDEX IF NOT EXISTS idx_game_tokens_player_id ON game_tokens(player_id);
	CREATE INDEX IF NOT EXISTS idx_token_positions_game_id ON token_positions(game_id);
	CREATE INDEX IF NOT EXISTS idx_token_positions_user_id ON token_positions(user_id);
    CREATE INDEX IF NOT EXISTS idx_game_moves_game_player ON game_moves(game_id, player_id);
    CREATE INDEX IF NOT EXISTS idx_token_positions_game_user ON token_positions(game_id, user_id);
	`

	_, err := DB.Exec(schema)
	if err != nil {
		return fmt.Errorf("error creating game schema: %w", err)
	}

	log.Println("Game schema initialized successfully")
	return nil
}
