package embedding_test

import (
	"context"
	"testing"

	"github.com/Doremi203/personage/backend/tasker/internal/services/embedding"
	einoembedding "github.com/cloudwego/eino/components/embedding"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubEmbedder struct {
	out      [][]float64
	err      error
	gotTexts []string
}

func (s *stubEmbedder) EmbedStrings(_ context.Context, texts []string, _ ...einoembedding.Option) ([][]float64, error) {
	s.gotTexts = texts
	return s.out, s.err
}

func TestEinoService_GenerateEmbeddings(t *testing.T) {
	type mocks struct {
		embedder *stubEmbedder
	}
	type args struct {
		strings []string
	}
	tests := []struct {
		name    string
		args    args
		setup   func(m mocks, a args)
		want    [][]float32
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "success converts float64 to float32 element-by-element",
			args: args{strings: []string{"hello"}},
			setup: func(m mocks, _ args) {
				m.embedder.out = [][]float64{{0.5, -1.25, 3.0}}
			},
			want:    [][]float32{{0.5, -1.25, 3.0}},
			wantErr: require.NoError,
		},
		{
			name: "empty result returns empty slice without error",
			args: args{strings: []string{}},
			setup: func(m mocks, _ args) {
				m.embedder.out = [][]float64{}
			},
			want:    [][]float32{},
			wantErr: require.NoError,
		},
		{
			name: "embedder error returned as-is",
			args: args{strings: []string{"x"}},
			setup: func(m mocks, _ args) {
				m.embedder.err = assert.AnError
			},
			want: nil,
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.ErrorIs(t, err, assert.AnError)
			},
		},
		{
			name: "multiple texts preserve order",
			args: args{strings: []string{"a", "b", "c"}},
			setup: func(m mocks, _ args) {
				m.embedder.out = [][]float64{
					{1.0, 2.0},
					{3.0, 4.0},
					{5.0, 6.0},
				}
			},
			want: [][]float32{
				{1.0, 2.0},
				{3.0, 4.0},
				{5.0, 6.0},
			},
			wantErr: require.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := mocks{embedder: &stubEmbedder{}}
			tt.setup(m, tt.args)

			svc := embedding.NewEinoService(m.embedder)
			got, err := svc.GenerateEmbeddings(t.Context(), tt.args.strings)

			tt.wantErr(t, err)
			assert.Equal(t, tt.args.strings, m.embedder.gotTexts)
			require.Len(t, got, len(tt.want))
			for i := range tt.want {
				assert.Equal(t, tt.want[i], got[i])
			}
		})
	}
}
