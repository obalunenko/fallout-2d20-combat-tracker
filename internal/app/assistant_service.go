package app

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"github.com/obalunenko/fallout/internal/domain"
)

const (
	accessSecretBytes        = 32
	maxAssistantActivityList = 200
)

// GetAssistantSettings returns the assistant connection settings. On first
// use it stores defaults: disabled, the default port and a new access secret.
func (s *Service) GetAssistantSettings(ctx context.Context) (domain.AssistantConnectionSettings, error) {
	ctx, op, err := s.beginWrite(ctx)
	if err != nil {
		return domain.AssistantConnectionSettings{}, err
	}
	defer op.end()
	settings, found, err := s.repo.GetAssistantConnectionSettings(ctx)
	if err != nil {
		return domain.AssistantConnectionSettings{}, err
	}
	if found {
		return settings, nil
	}
	secret, err := newAccessSecret()
	if err != nil {
		return domain.AssistantConnectionSettings{}, err
	}
	settings = domain.AssistantConnectionSettings{Port: domain.DefaultAssistantPort, AccessSecret: secret}
	if err := s.repo.SaveAssistantConnectionSettings(ctx, settings); err != nil {
		return domain.AssistantConnectionSettings{}, err
	}
	return settings, nil
}

// SaveAssistantSettings validates and stores settings.
func (s *Service) SaveAssistantSettings(ctx context.Context, settings domain.AssistantConnectionSettings) error {
	ctx, op, err := s.beginWrite(ctx)
	if err != nil {
		return err
	}
	defer op.end()
	if err := settings.Validate(); err != nil {
		return domain.InvalidInput(err)
	}
	return s.repo.SaveAssistantConnectionSettings(ctx, settings)
}

// RegenerateAccessSecret stores a new access secret and returns the updated
// settings. Clients using the old secret are refused once the connection
// applies the new settings.
func (s *Service) RegenerateAccessSecret(ctx context.Context) (domain.AssistantConnectionSettings, error) {
	ctx, op, err := s.beginWrite(ctx)
	if err != nil {
		return domain.AssistantConnectionSettings{}, err
	}
	defer op.end()
	settings, found, err := s.repo.GetAssistantConnectionSettings(ctx)
	if err != nil {
		return domain.AssistantConnectionSettings{}, err
	}
	if !found {
		settings.Port = domain.DefaultAssistantPort
	}
	if settings.AccessSecret, err = newAccessSecret(); err != nil {
		return domain.AssistantConnectionSettings{}, err
	}
	if err := s.repo.SaveAssistantConnectionSettings(ctx, settings); err != nil {
		return domain.AssistantConnectionSettings{}, err
	}
	return settings, nil
}

// AppendAssistantActivity records an assistant activity entry. A failure is a
// non-critical side effect: it is logged and counted, never returned
// (constitution Principle VII).
func (s *Service) AppendAssistantActivity(ctx context.Context, entry domain.AssistantActivityEntry) {
	ctx, op, err := s.beginWrite(ctx)
	if err != nil {
		s.runNonCriticalSideEffect(sideEffectCategoryAudit, sideEffectNameAppendAssistantActivity, func() error { return err })
		return
	}
	defer op.end()
	s.runNonCriticalSideEffect(sideEffectCategoryAudit, sideEffectNameAppendAssistantActivity, func() error {
		if err := s.repo.AppendAssistantActivity(ctx, entry); err != nil {
			return fmt.Errorf("operation=%s outcome=%s: %w", entry.Operation, entry.Outcome, err)
		}
		return nil
	})
}

// ListAssistantActivity returns up to limit entries, newest first. limit must
// be between 1 and 200.
func (s *Service) ListAssistantActivity(ctx context.Context, limit int) ([]domain.AssistantActivityEntry, error) {
	ctx, op, err := s.beginRead(ctx)
	if err != nil {
		return nil, err
	}
	defer op.end()
	if limit < 1 || limit > maxAssistantActivityList {
		return nil, invalidf("limit must be between 1 and %d", maxAssistantActivityList)
	}
	return s.repo.ListAssistantActivity(ctx, limit)
}

// ClearAssistantActivity deletes every assistant activity entry.
func (s *Service) ClearAssistantActivity(ctx context.Context) error {
	ctx, op, err := s.beginWrite(ctx)
	if err != nil {
		return err
	}
	defer op.end()
	return s.repo.ClearAssistantActivity(ctx)
}

func newAccessSecret() (domain.AccessSecret, error) {
	b := make([]byte, accessSecretBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate access secret: %w", err)
	}
	return domain.AccessSecret(base64.RawURLEncoding.EncodeToString(b)), nil
}
