package ports

import "context"

// ClientNames は、カタログ名へのフォールバックに必要な名前だけを要求する。
type ClientNames interface {
	FindClientName(context.Context, string, string) (*string, error)
}
