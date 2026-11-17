package ports

import (
	"context"

	"github.com/ambi/idmagic/backend/shared/spec"
)

// AccessRevocationReactor は、User または Agent を止めたイベントを受け、その主体のアクセスを失効させる。
// 失敗は止める操作の誤りとして返る (fail-closed)。SharedSignals の反応器が構造的に満たし、組み立ての地点が渡す。
// IdManagement は SharedSignals に依存しないので、この型が要求側の宣言になる。
type AccessRevocationReactor interface {
	React(ctx context.Context, event spec.DomainEvent) error
}
