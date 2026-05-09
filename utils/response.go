package utils

import (
	"encoding/json"
	"net/http"
)

type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func SendJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

func SendSuccess(w http.ResponseWriter, message string, data interface{}) {
	SendJSON(w, http.StatusOK, Response{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func SendError(w http.ResponseWriter, statusCode int, message string) {
	SendJSON(w, statusCode, Response{
		Success: false,
		Error:   message,
	})
}

func SendCreated(w http.ResponseWriter, message string, data interface{}) {
	SendJSON(w, http.StatusCreated, Response{
		Success: true,
		Message: message,
		Data:    data,
	})
}

