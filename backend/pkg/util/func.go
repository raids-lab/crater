package util

import "context"

type AnyFunc func(ctx context.Context) (any, error)
