//go:build !pubsub

package publishers_pubsub

import (
	"context"
	"errors"

	eventports "github.com/ambi/idmagic/backend/shared/events/ports"
)

// ErrPubSubUnsupported は pubsub build タグ無しでビルドされたバイナリで
// RELAY_SINK=pubsub が要求されたときに返る。既定ビルドは GCP SDK を含めないため
// ([[ADR-120]])、Pub/Sub を使うには `-tags pubsub` で再ビルドする。
var ErrPubSubUnsupported = errors.New("eventsink: built without pubsub support; rebuild with -tags pubsub")

// NewPubSubPublisher は既定ビルドでは常に ErrPubSubUnsupported を返す。
func NewPubSubPublisher(_ context.Context, _ string) (eventports.Publisher, error) {
	return nil, ErrPubSubUnsupported
}
