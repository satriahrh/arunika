package stt_test

import (
	"github.com/satriahrh/arunika/server/adapters/stt"
	"github.com/satriahrh/arunika/server/domain/repositories"
)

var _ repositories.SpeechToText = &stt.GoogleSpeechToText{}
