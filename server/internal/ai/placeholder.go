package ai

// Placeholder AI service for saga compatibility
type AIService struct{}

func (s *AIService) ProcessSpeechToText(audioData []byte) (string, error) {
	return "", nil
}

func (s *AIService) ValidateContent(text string) (bool, error) {
	return true, nil
}

func (s *AIService) GenerateResponse(text string, context map[string]interface{}) (string, error) {
	return "", nil
}

func (s *AIService) ProcessTextToSpeech(text, style string) ([]byte, error) {
	return nil, nil
}