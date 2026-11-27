package app

import (
	"context"

	"github.com/obalunenko/fallout/internal/domain"
)

type sourceKey struct{}

// WithSource returns a context that marks the operations run with it as
// performed by src. The assistant adapter marks every call it makes.
func WithSource(ctx context.Context, src domain.OperationSource) context.Context {
	return context.WithValue(ctx, sourceKey{}, src)
}

// SourceFromContext returns the operation source carried by ctx, or
// domain.SourceDesktop when none was set.
func SourceFromContext(ctx context.Context) domain.OperationSource {
	if ctx == nil {
		return domain.SourceDesktop
	}
	src, _ := ctx.Value(sourceKey{}).(domain.OperationSource)
	return src.Normalize()
}
