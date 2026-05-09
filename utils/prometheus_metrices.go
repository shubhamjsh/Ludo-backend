package utils

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTP request counter
	HttpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ludo_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	// HTTP request duration histogram
	HttpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ludo_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	// Active games gauge
	ActiveGamesGauge = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "ludo_active_games_total",
			Help: "Total number of active games",
		},
	)

	// Total players gauge
	TotalPlayersGauge = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "ludo_total_players",
			Help: "Total number of players online",
		},
	)

	// Dice rolls counter
	DiceRollsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ludo_dice_rolls_total",
			Help: "Total number of dice rolls",
		},
		[]string{"value"},
	)

	// Token moves counter
	TokenMovesTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "ludo_token_moves_total",
			Help: "Total number of token moves",
		},
	)

	// Game completions counter
	GameCompletionsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "ludo_game_completions_total",
			Help: "Total number of completed games",
		},
	)

	// API errors counter
	ApiErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ludo_api_errors_total",
			Help: "Total number of API errors",
		},
		[]string{"endpoint", "error_type"},
	)
)
