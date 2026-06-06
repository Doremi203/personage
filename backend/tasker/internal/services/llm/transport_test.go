package llm

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubRoundTripper struct {
	resp *http.Response
	err  error
}

func (s *stubRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return s.resp, s.err
}

type spyLogger struct {
	warnings []error
}

func (s *spyLogger) Infof(string, ...any) {}
func (s *spyLogger) Warn(err error)       { s.warnings = append(s.warnings, err) }
func (s *spyLogger) Error(error)          {}

func makeResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func TestLoggingTransport_RoundTrip(t *testing.T) {
	const goodBody = `{"choices":[{"message":{"content":"hi"}}]}`
	const errBody = `{"error":{"message":"upstream provider error","code":429}}`

	tests := []struct {
		name     string
		resp     *http.Response
		baseErr  error
		wantWarn bool
		wantBody string
		wantErr  require.ErrorAssertionFunc
	}{
		{
			name:     "2xx with choices does not log",
			resp:     makeResponse(http.StatusOK, goodBody),
			wantWarn: false,
			wantBody: goodBody,
			wantErr:  require.NoError,
		},
		{
			name:     "2xx with error and no choices logs",
			resp:     makeResponse(http.StatusOK, errBody),
			wantWarn: true,
			wantBody: errBody,
			wantErr:  require.NoError,
		},
		{
			name:     "2xx with empty choices logs",
			resp:     makeResponse(http.StatusOK, `{"choices":[]}`),
			wantWarn: true,
			wantBody: `{"choices":[]}`,
			wantErr:  require.NoError,
		},
		{
			name:     "2xx with empty body logs",
			resp:     makeResponse(http.StatusOK, ``),
			wantWarn: true,
			wantBody: ``,
			wantErr:  require.NoError,
		},
		{
			name:     "non-2xx does not log",
			resp:     makeResponse(http.StatusTooManyRequests, errBody),
			wantWarn: false,
			wantBody: errBody,
			wantErr:  require.NoError,
		},
		{
			name:    "base transport error is propagated",
			baseErr: errors.Error("boom"),
			wantErr: require.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := &spyLogger{}
			transport := &loggingTransport{
				base:   &stubRoundTripper{resp: tt.resp, err: tt.baseErr},
				logger: logger,
			}

			req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "http://example/api", nil)
			require.NoError(t, err)

			resp, err := transport.RoundTrip(req)
			tt.wantErr(t, err)
			if err != nil {
				return
			}

			body, readErr := io.ReadAll(resp.Body)
			require.NoError(t, readErr)
			assert.Equal(t, tt.wantBody, string(body))
			assert.Len(t, logger.warnings, map[bool]int{true: 1, false: 0}[tt.wantWarn])
		})
	}
}
