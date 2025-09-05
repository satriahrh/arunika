package stt

import (
	"context"
	"fmt"
	"io"

	speech "cloud.google.com/go/speech/apiv1"
	"cloud.google.com/go/speech/apiv1/speechpb"
	"github.com/satriahrh/arunika/server/domain/repositories"
)

// GoogleSpeechToText implements SpeechToText for Google Cloud
type GoogleSpeechToText struct{}

func (g *GoogleSpeechToText) InitTranscribeStreaming(ctx context.Context, config repositories.AudioConfig) (repositories.SpeechToTextStreaming, error) {
	// Create Google Cloud Speech client
	client, err := speech.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create speech client: %w", err)
	}

	// Create streaming recognize request
	stream, err := client.StreamingRecognize(ctx)
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to create streaming recognize: %w", err)
	}

	// Convert encoding string to Google Speech API enum
	encoding, err := getAudioEncoding(config.Encoding)
	if err != nil {
		stream.CloseSend()
		client.Close()
		return nil, fmt.Errorf("unsupported audio encoding: %s", config.Encoding)
	}

	// Configure recognition settings
	recognitionConfig := &speechpb.RecognitionConfig{
		Encoding:        encoding,
		SampleRateHertz: int32(config.SampleRate),
		LanguageCode:    config.Language,
	}

	// Send initial configuration
	if err := stream.Send(&speechpb.StreamingRecognizeRequest{
		StreamingRequest: &speechpb.StreamingRecognizeRequest_StreamingConfig{
			StreamingConfig: &speechpb.StreamingRecognitionConfig{
				Config:          recognitionConfig,
				InterimResults:  false, // We only want final results
				SingleUtterance: true,  // Treat as single utterance
			},
		},
	}); err != nil {
		stream.CloseSend()
		client.Close()
		return nil, fmt.Errorf("failed to send streaming config: %w", err)
	}

	// Create the streaming instance
	streamInstance := &GoogleSpeechToTextStream{
		client:         client,
		stream:         stream,
		ctx:            ctx,
		audioReceived:  false,
		resultChan:     make(chan string, 1),
		errorChan:      make(chan error, 1),
		receiverActive: false,
	}

	return streamInstance, nil
}

type GoogleSpeechToTextStream struct {
	client         *speech.Client
	stream         speechpb.Speech_StreamingRecognizeClient
	ctx            context.Context
	audioReceived  bool
	resultChan     chan string
	errorChan      chan error
	receiverActive bool
}

func (g *GoogleSpeechToTextStream) Stream(data []byte) error {
	// Start the result receiver goroutine only once
	if !g.receiverActive {
		g.receiverActive = true
		go g.receiveResults()
	}

	if len(data) > 0 {
		g.audioReceived = true

		// Send audio data to Google
		if err := g.stream.Send(&speechpb.StreamingRecognizeRequest{
			StreamingRequest: &speechpb.StreamingRecognizeRequest_AudioContent{
				AudioContent: data,
			},
		}); err != nil {
			return fmt.Errorf("failed to send audio data: %w", err)
		}
	}

	return nil
}

func (g *GoogleSpeechToTextStream) End() (string, error) {
	defer g.cleanup()

	if !g.audioReceived {
		return "", fmt.Errorf("no audio data received")
	}

	// Close the send stream to signal end of audio
	if err := g.stream.CloseSend(); err != nil {
		return "", fmt.Errorf("failed to close send stream: %w", err)
	}

	// Wait for final result or error
	select {
	case <-g.ctx.Done():
		return "", fmt.Errorf("context cancelled while waiting for result: %w", g.ctx.Err())
	case err := <-g.errorChan:
		if err != nil {
			return "", err
		}
	case result := <-g.resultChan:
		if result == "" {
			return "", fmt.Errorf("no speech detected in audio")
		}
		return result, nil
	}

	return "", fmt.Errorf("unexpected end of transcription")
}

func (g *GoogleSpeechToTextStream) receiveResults() {
	defer close(g.resultChan)
	defer close(g.errorChan)

	var finalTranscription string

	for {
		resp, err := g.stream.Recv()
		if err == io.EOF {
			// Stream ended normally
			g.resultChan <- finalTranscription
			return
		}
		if err != nil {
			g.errorChan <- fmt.Errorf("failed to receive response: %w", err)
			return
		}

		// Process results - only consider final ones
		if resp.Results != nil {
			for _, result := range resp.Results {
				if result.IsFinal && len(result.Alternatives) > 0 {
					// Take the best alternative
					finalTranscription = result.Alternatives[0].Transcript
				}
			}
		}
	}
}

func (g *GoogleSpeechToTextStream) cleanup() {
	if g.client != nil {
		g.client.Close()
	}
}

// TranscribeAudio converts audio data to text using Google Cloud Speech-to-Text (non-streaming)
func (g *GoogleSpeechToText) TranscribeAudio(ctx context.Context, audioData []byte, config repositories.AudioConfig) (string, error) {
	// Initialize streaming transcription
	stream, err := g.InitTranscribeStreaming(ctx, config)
	if err != nil {
		return "", fmt.Errorf("failed to initialize streaming: %w", err)
	}

	// Send the audio data in one go
	if err := stream.Stream(audioData); err != nil {
		return "", fmt.Errorf("failed to stream audio data: %w", err)
	}

	// End the stream and get the result
	return stream.End()
}

// getAudioEncoding converts string encoding to Google Speech API enum
func getAudioEncoding(encoding string) (speechpb.RecognitionConfig_AudioEncoding, error) {
	switch encoding {
	case "WAV", "LINEAR16":
		return speechpb.RecognitionConfig_LINEAR16, nil
	case "FLAC":
		return speechpb.RecognitionConfig_FLAC, nil
	case "MULAW":
		return speechpb.RecognitionConfig_MULAW, nil
	case "AMR":
		return speechpb.RecognitionConfig_AMR, nil
	case "AMR_WB":
		return speechpb.RecognitionConfig_AMR_WB, nil
	case "OGG_OPUS":
		return speechpb.RecognitionConfig_OGG_OPUS, nil
	case "SPEEX_WITH_HEADER_BYTE":
		return speechpb.RecognitionConfig_SPEEX_WITH_HEADER_BYTE, nil
	case "WEBM_OPUS":
		return speechpb.RecognitionConfig_WEBM_OPUS, nil
	default:
		return speechpb.RecognitionConfig_ENCODING_UNSPECIFIED, fmt.Errorf("unsupported encoding: %s", encoding)
	}
}
