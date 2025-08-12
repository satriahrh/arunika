package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/satriahrh/arunika/server/repository"
)

// ChatService handles conversation logic
type ChatService struct {
	llm repository.Llm
}

// NewChatService creates a new chat service
func NewChatService(llm repository.Llm) *ChatService {
	return &ChatService{llm: llm}
}

// Execute runs the chat service for a user session
func (s *ChatService) Execute(ctx context.Context, userID int, input chan string, output chan string) error {
	// Load conversation history from file
	filename := fmt.Sprintf("chat_history_%d.json", userID)
	var history []repository.ChatMessage

	file, err := os.Open(filename)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("error opening file: %w", err)
		}
		// File doesn't exist, start with empty history
	} else {
		defer file.Close()
		decoder := json.NewDecoder(file)
		err = decoder.Decode(&history)
		if err != nil {
			return fmt.Errorf("error decoding JSON: %w", err)
		}
	}

	// Create chat session with history
	chatSession, err := s.llm.GenerateChat(ctx, history)
	if err != nil {
		return err
	}

	// Process messages
	for {
		select {
		case msg := <-input:
			responseMessage, err := chatSession.SendMessage(ctx, repository.ChatMessage{
				Role:    repository.UserRole,
				Content: msg,
			})
			if err != nil {
				output <- "ERROR: " + err.Error()
				continue
			}
			output <- responseMessage.Content

		case <-ctx.Done():
			// Save conversation history when context is done
			history, err := chatSession.History()
			if err != nil {
				fmt.Println("Error getting history:", err)
			}

			// Store history to JSON file
			file, err := os.Create(filename)
			if err != nil {
				fmt.Println("Error creating file:", err)
			} else {
				encoder := json.NewEncoder(file)
				err = encoder.Encode(history)
				if err != nil {
					fmt.Println("Error encoding history to JSON:", err)
				}
				file.Close()
			}

			fmt.Println("History stored to", filename)
			return nil
		}
	}
}
