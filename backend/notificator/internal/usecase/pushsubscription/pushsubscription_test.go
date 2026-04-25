package pushsubscription_test

import (
	"testing"

	"github.com/Doremi203/personage/backend/notificator/internal/domain/push"
	mock_push "github.com/Doremi203/personage/backend/notificator/internal/domain/push/mock"
	"github.com/Doremi203/personage/backend/notificator/internal/usecase/pushsubscription"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

var (
	subRecipientID = push.RecipientID(uuid.MustParse("11111111-1111-1111-1111-111111111111"))
	testSub        = push.Subscription{
		Endpoint:    "endpoint",
		Credentials: push.Credentials{P256dh: "public_key", AuthKey: "auth_key"},
		RecipientID: subRecipientID,
	}
)

func TestService_Subscribe(t *testing.T) {
	type mocks struct {
		repo *mock_push.MockRepo
	}
	type args struct {
		subscription push.Subscription
	}
	tests := []struct {
		name    string
		args    args
		setup   func(m mocks, a args)
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "success",
			args: args{subscription: testSub},
			setup: func(m mocks, a args) {
				m.repo.EXPECT().UpsertSubscription(gomock.Any(), a.subscription).Return(nil)
			},
			wantErr: require.NoError,
		},
		{
			name: "repo error wraps",
			args: args{subscription: testSub},
			setup: func(m mocks, a args) {
				m.repo.EXPECT().UpsertSubscription(gomock.Any(), a.subscription).Return(assert.AnError)
			},
			wantErr: require.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			m := mocks{repo: mock_push.NewMockRepo(ctrl)}
			tt.setup(m, tt.args)

			s := pushsubscription.New(m.repo)
			err := s.Subscribe(t.Context(), tt.args.subscription)

			tt.wantErr(t, err)
		})
	}
}

func TestService_Unsubscribe(t *testing.T) {
	type mocks struct {
		repo *mock_push.MockRepo
	}
	type args struct {
		subscription push.Subscription
	}
	tests := []struct {
		name    string
		args    args
		setup   func(m mocks, a args)
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "success",
			args: args{subscription: testSub},
			setup: func(m mocks, a args) {
				m.repo.EXPECT().DeleteSubscription(gomock.Any(), a.subscription).Return(nil)
			},
			wantErr: require.NoError,
		},
		{
			name: "repo error wraps",
			args: args{subscription: testSub},
			setup: func(m mocks, a args) {
				m.repo.EXPECT().DeleteSubscription(gomock.Any(), a.subscription).Return(assert.AnError)
			},
			wantErr: require.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			m := mocks{repo: mock_push.NewMockRepo(ctrl)}
			tt.setup(m, tt.args)

			s := pushsubscription.New(m.repo)
			err := s.Unsubscribe(t.Context(), tt.args.subscription)

			tt.wantErr(t, err)
		})
	}
}

func TestService_GetRecipient(t *testing.T) {
	type mocks struct {
		repo *mock_push.MockRepo
	}
	type args struct {
		recipientID push.RecipientID
	}
	tests := []struct {
		name    string
		args    args
		setup   func(m mocks, a args)
		want    push.Recipient
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "success returns subscriptions",
			args: args{recipientID: subRecipientID},
			setup: func(m mocks, a args) {
				m.repo.EXPECT().
					GetSubscriptionsByRecipientID(gomock.Any(), a.recipientID).
					Return([]push.Subscription{testSub}, nil)
			},
			want: push.Recipient{
				ID:            subRecipientID,
				Subscriptions: []push.Subscription{testSub},
			},
			wantErr: require.NoError,
		},
		{
			name: "repo error wraps",
			args: args{recipientID: subRecipientID},
			setup: func(m mocks, a args) {
				m.repo.EXPECT().
					GetSubscriptionsByRecipientID(gomock.Any(), a.recipientID).
					Return(nil, assert.AnError)
			},
			wantErr: require.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			m := mocks{repo: mock_push.NewMockRepo(ctrl)}
			tt.setup(m, tt.args)

			s := pushsubscription.New(m.repo)
			got, err := s.GetRecipient(t.Context(), tt.args.recipientID)

			tt.wantErr(t, err)
			if err == nil {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
