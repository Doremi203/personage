package push_test

import (
	"testing"

	"github.com/Doremi203/personage/backend/notificator/internal/domain/push"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSubscription(t *testing.T) {
	recipientID := push.RecipientID(uuid.MustParse("11111111-1111-1111-1111-111111111111"))

	type args struct {
		recipientID push.RecipientID
		endpoint    string
		p256dh      string
		authKey     string
	}
	tests := []struct {
		name    string
		args    args
		want    push.Subscription
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "success",
			args: args{
				recipientID: recipientID,
				endpoint:    "https://push.example/abc",
				p256dh:      "public",
				authKey:     "secret",
			},
			want: push.Subscription{
				RecipientID: recipientID,
				Endpoint:    push.Endpoint("https://push.example/abc"),
				Credentials: push.Credentials{P256dh: "public", AuthKey: "secret"},
			},
			wantErr: require.NoError,
		},
		{
			name: "empty endpoint",
			args: args{
				recipientID: recipientID,
				endpoint:    "",
				p256dh:      "public",
				authKey:     "secret",
			},
			wantErr: require.Error,
		},
		{
			name: "empty public key",
			args: args{
				recipientID: recipientID,
				endpoint:    "https://push.example/abc",
				p256dh:      "",
				authKey:     "secret",
			},
			wantErr: require.Error,
		},
		{
			name: "empty auth key",
			args: args{
				recipientID: recipientID,
				endpoint:    "https://push.example/abc",
				p256dh:      "public",
				authKey:     "",
			},
			wantErr: require.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := push.NewSubscription(tt.args.recipientID, tt.args.endpoint, tt.args.p256dh, tt.args.authKey)
			tt.wantErr(t, err)
			if err == nil {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
