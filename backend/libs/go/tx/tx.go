package tx

import (
	"context"
)

type Isolation int

const (
	IsolationReadUncommitted Isolation = iota
	IsolationReadCommitted
	IsolationRepeatableRead
	IsolationSerializable
)

type Provider interface {
	RunWithTx(
		ctx context.Context,
		isolation Isolation,
		op func(context.Context) error,
	) error
}
