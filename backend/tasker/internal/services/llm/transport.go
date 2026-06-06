package llm

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/Doremi203/personage/backend/libs/go/log"
)

const loggedResponseBodyLimit = 16 * 1024

func NewHTTPClient(logger log.Logger) *http.Client {
	return &http.Client{
		// Above the per-request context deadline so the context cancels first and
		// the retry loop sees a meaningful error rather than a bare client timeout.
		Timeout: llmRequestTimeout + 5*time.Second,
		Transport: &loggingTransport{
			base:   http.DefaultTransport,
			logger: logger,
		},
	}
}

// loggingTransport logs the raw OpenRouter response body whenever the API answers
// 2xx without usable choices. OpenRouter puts the real failure in the body's
// "error" field, but the OpenAI client treats any 2xx as success, so the caller
// otherwise sees only "received empty choices" with the actual cause discarded.
type loggingTransport struct {
	base   http.RoundTripper
	logger log.Logger
}

func (t *loggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.base.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	// Non-2xx already surfaces as a descriptive error with the body in the OpenAI
	// client, so leave those responses untouched instead of buffering them here.
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return resp, nil
	}

	body, readErr := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if readErr != nil {
		return nil, errors.WrapFail(readErr, "read openrouter response body")
	}
	resp.Body = io.NopCloser(bytes.NewReader(body))

	t.logIfNoUsableChoices(resp.StatusCode, body)

	return resp, nil
}

func (t *loggingTransport) logIfNoUsableChoices(status int, body []byte) {
	var parsed struct {
		Error   json.RawMessage   `json:"error"`
		Choices []json.RawMessage `json:"choices"`
	}
	if err := json.Unmarshal(body, &parsed); err == nil &&
		len(parsed.Error) == 0 && len(parsed.Choices) > 0 {
		return
	}

	logged := body
	if len(logged) > loggedResponseBodyLimit {
		logged = logged[:loggedResponseBodyLimit]
	}

	t.logger.Warn(errors.Errorf(
		"openrouter returned 2xx without usable choices %v %v %v",
		errors.Token("status", status),
		errors.Token("body_size", len(body)),
		errors.Token("body", string(logged)),
	))
}
