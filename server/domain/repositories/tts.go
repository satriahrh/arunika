package repositories

import "context"

type TextToSpeech interface {
	ConvertTextToSpeech(ctx context.Context, text string) (<-chan []byte, error)
}
