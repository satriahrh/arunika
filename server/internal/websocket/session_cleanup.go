package websocket

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/satriahrh/arunika/server/domain/repositories"
)

// SessionCleanupService handles background tasks for session management
type SessionCleanupService struct {
	sessionRepo repositories.SessionRepository
	logger      *zap.Logger
	stopChan    chan struct{}
}

// NewSessionCleanupService creates a new session cleanup service
func NewSessionCleanupService(sessionRepo repositories.SessionRepository, logger *zap.Logger) *SessionCleanupService {
	return &SessionCleanupService{
		sessionRepo: sessionRepo,
		logger:      logger,
		stopChan:    make(chan struct{}),
	}
}

// Start begins the background cleanup process
func (s *SessionCleanupService) Start() {
	go s.cleanupLoop()
	s.logger.Info("Session cleanup service started")
}

// Stop gracefully stops the cleanup service
func (s *SessionCleanupService) Stop() {
	close(s.stopChan)
	s.logger.Info("Session cleanup service stopped")
}

// cleanupLoop runs the cleanup process periodically
func (s *SessionCleanupService) cleanupLoop() {
	// Run cleanup every 30 minutes
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

	// Run initial cleanup after 1 minute
	initialTimer := time.NewTimer(1 * time.Minute)
	defer initialTimer.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-initialTimer.C:
			s.runCleanup()
			// Initial timer only runs once
		case <-ticker.C:
			s.runCleanup()
		}
	}
}

// runCleanup performs the actual cleanup of expired sessions
func (s *SessionCleanupService) runCleanup() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	s.logger.Info("Starting session cleanup")

	err := s.sessionRepo.ExpireSessions(ctx)
	if err != nil {
		s.logger.Error("Failed to expire sessions", zap.Error(err))
		return
	}

	s.logger.Info("Session cleanup completed successfully")
}