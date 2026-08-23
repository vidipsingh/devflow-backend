package service

import (
	"context"
	"errors"

	"devflow-backend/internal/models"
	"devflow-backend/internal/repository"
)

type PairService struct {
	repo *repository.PairRepo
}

func NewPairService(repo *repository.PairRepo) *PairService {
	return &PairService{repo: repo}
}

var participantColors = []string{
	"#FF6B6B", "#4ECDC4", "#45B7D1", "#96CEB4",
	"#FFEAA7", "#DDA0DD", "#98D8C8", "#F7DC6F",
}

func (s *PairService) CreateSession(ctx context.Context, ownerID, repoID, fileID, filePath, initialDoc string) (*models.PairSession, error) {
	session := &models.PairSession{
		OwnerID:  ownerID,
		RepoID:   repoID,
		FileID:   fileID,
		FilePath: filePath,
		Document: initialDoc,
		Version:  0,
		Status:   models.SessionStatusWaiting,
		Participants: []models.ParticipantInfo{},
	}
	if err := s.repo.Create(ctx, session); err != nil {
		return nil, err
	}
	s.repo.CacheSession(ctx, session)
	return session, nil
}

func (s *PairService) GetSession(ctx context.Context, id string) (*models.PairSession, error) {
	if cached, ok := s.repo.GetCached(ctx, id); ok {
		return cached, nil
	}
	return s.repo.GetByID(ctx, id)
}

func (s *PairService) JoinSession(ctx context.Context, sessionID, userID, username string) (*models.PairSession, error) {
	sess, err := s.repo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if sess == nil {
		return nil, errors.New("session not found")
	}
	if sess.Status == models.SessionStatusEnded {
		return nil, errors.New("session has ended")
	}
	
	// Check if already a participant
	for _, p := range sess.Participants {
		if p.UserID == userID {
			return sess, nil // already joined
		}
	}

	color := participantColors[len(sess.Participants)%len(participantColors)]
	p := models.ParticipantInfo{
		UserID:   userID,
		Username: username,
		Color:    color,
	}

	if err := s.repo.AddParticipants(ctx, sessionID, p); err != nil {
		return nil, err
	}
	s.repo.InvalidateCache(ctx, sessionID)

	sess.Participants = append(sess.Participants, p)
	sess.Status = models.SessionStatusActive
	return sess, nil
}

func (s *PairService) EndSession(ctx context.Context, sessionID, requesterID string) error {
	sess, err := s.repo.GetByID(ctx, sessionID)
	if err != nil {
		return err
	}
	if sess == nil {
		return errors.New("session not found")
	}
	if sess.OwnerID != requesterID {
		return errors.New("only the session owner can end the session")
	}
	if err := s.repo.EndSession(ctx, sessionID); err != nil {
		return err
	}
	s.repo.InvalidateCache(ctx, sessionID)
	return nil
}

func (s *PairService) ListSession(ctx context.Context, ownerID string) ([]models.PairSession, error) {
	return s.repo.ListByOwner(ctx, ownerID)
}
