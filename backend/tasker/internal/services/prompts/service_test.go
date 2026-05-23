package prompts_test

import (
	"testing"
	"time"

	"github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/Doremi203/personage/backend/tasker/internal/domain"
	mock_domain "github.com/Doremi203/personage/backend/tasker/internal/domain/mock"
	"github.com/Doremi203/personage/backend/tasker/internal/services/prompts"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestService_Get(t *testing.T) {
	const ttl = 30 * time.Second

	type mocks struct {
		repo *mock_domain.MockPromptRepo
	}

	tests := []struct {
		name    string
		setup   func(m mocks, now *time.Time)
		actions func(t *testing.T, s *prompts.Service, now *time.Time)
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "first call loads, second hits cache",
			setup: func(m mocks, now *time.Time) {
				m.repo.EXPECT().
					GetPrompt(gomock.Any(), domain.PromptIDClassifier).
					Return(domain.Prompt{ID: domain.PromptIDClassifier, SystemTemplate: "v1"}, nil).
					Times(1)
			},
			actions: func(t *testing.T, s *prompts.Service, now *time.Time) {
				p, err := s.Get(t.Context(), domain.PromptIDClassifier)
				require.NoError(t, err)
				require.Equal(t, "v1", p.SystemTemplate)

				p, err = s.Get(t.Context(), domain.PromptIDClassifier)
				require.NoError(t, err)
				require.Equal(t, "v1", p.SystemTemplate)
			},
			wantErr: require.NoError,
		},
		{
			name: "expired entry reloads",
			setup: func(m mocks, now *time.Time) {
				gomock.InOrder(
					m.repo.EXPECT().
						GetPrompt(gomock.Any(), domain.PromptIDClassifier).
						Return(domain.Prompt{ID: domain.PromptIDClassifier, SystemTemplate: "v1"}, nil),
					m.repo.EXPECT().
						GetPrompt(gomock.Any(), domain.PromptIDClassifier).
						Return(domain.Prompt{ID: domain.PromptIDClassifier, SystemTemplate: "v2"}, nil),
				)
			},
			actions: func(t *testing.T, s *prompts.Service, now *time.Time) {
				p, err := s.Get(t.Context(), domain.PromptIDClassifier)
				require.NoError(t, err)
				require.Equal(t, "v1", p.SystemTemplate)

				*now = now.Add(ttl + time.Second)

				p, err = s.Get(t.Context(), domain.PromptIDClassifier)
				require.NoError(t, err)
				require.Equal(t, "v2", p.SystemTemplate)
			},
			wantErr: require.NoError,
		},
		{
			name: "invalidate forces reload before TTL",
			setup: func(m mocks, now *time.Time) {
				gomock.InOrder(
					m.repo.EXPECT().
						GetPrompt(gomock.Any(), domain.PromptIDClassifier).
						Return(domain.Prompt{ID: domain.PromptIDClassifier, SystemTemplate: "v1"}, nil),
					m.repo.EXPECT().
						GetPrompt(gomock.Any(), domain.PromptIDClassifier).
						Return(domain.Prompt{ID: domain.PromptIDClassifier, SystemTemplate: "v2"}, nil),
				)
			},
			actions: func(t *testing.T, s *prompts.Service, now *time.Time) {
				_, err := s.Get(t.Context(), domain.PromptIDClassifier)
				require.NoError(t, err)

				s.Invalidate(domain.PromptIDClassifier)

				p, err := s.Get(t.Context(), domain.PromptIDClassifier)
				require.NoError(t, err)
				require.Equal(t, "v2", p.SystemTemplate)
			},
			wantErr: require.NoError,
		},
		{
			name: "repo error propagates",
			setup: func(m mocks, now *time.Time) {
				m.repo.EXPECT().
					GetPrompt(gomock.Any(), domain.PromptIDClassifier).
					Return(domain.Prompt{}, errors.Error("boom"))
			},
			actions: func(t *testing.T, s *prompts.Service, now *time.Time) {
				_, err := s.Get(t.Context(), domain.PromptIDClassifier)
				require.Error(t, err)
			},
			wantErr: require.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			m := mocks{repo: mock_domain.NewMockPromptRepo(ctrl)}

			now := time.Date(2026, 5, 23, 12, 0, 0, 0, time.UTC)
			clock := func() time.Time { return now }

			tt.setup(m, &now)

			s := prompts.NewService(m.repo, ttl, clock)
			tt.actions(t, s, &now)
		})
	}
}
