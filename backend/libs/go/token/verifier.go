package token

import (
	"context"
	"net/http"

	"github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/go-resty/resty/v2"
)

type Verifier interface {
	VerifyToken(ctx context.Context, token string) (bool, error)
}

func NewVerifierStub() *verifierStub {
	return &verifierStub{}
}

type verifierStub struct {
}

func (v *verifierStub) VerifyToken(ctx context.Context, token string) (bool, error) {
	return true, nil
}

func NewVerifier(verifyURL string) *verifier {
	return &verifier{
		client:    resty.New(),
		verifyURL: verifyURL,
	}
}

type verifier struct {
	client    *resty.Client
	verifyURL string
}

func (v *verifier) VerifyToken(ctx context.Context, rawToken string) (bool, error) {
	resp, err := v.client.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetBody(verifyRequest{Token: rawToken}).
		Post(v.verifyURL)
	if err != nil {
		return false, errors.WrapFail(err, "call token verification service")
	}

	if resp.StatusCode() != http.StatusOK {
		return false, errors.Errorf(
			"token verification returned unexpected status: %s",
			errors.Token("status", resp.StatusCode()),
		)
	}

	return true, nil
}

type verifyRequest struct {
	Token string `json:"token"`
}
