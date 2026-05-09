package utils

import (
	"io"
	"log"
	"os"
)

var (
	InfoLogger  *log.Logger
	ErrorLogger *log.Logger
	DebugLogger *log.Logger
)

// InitLogger initializes loggers to write to both file and stdout
func InitLogger(logFile string) error {
	// Create logs directory if it doesn't exist
	if err := os.MkdirAll("logs", 0755); err != nil {
		return err
	}

	// Open log file
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}

	// Create multi-writer to write to both file and stdout
	multiWriter := io.MultiWriter(os.Stdout, file)

	// Initialize loggers
	InfoLogger = log.New(multiWriter, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLogger = log.New(multiWriter, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	DebugLogger = log.New(multiWriter, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)

	return nil
}
