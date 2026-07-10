// Package eventsink: ドメインイベントの出力先アダプタ。
// TS adapters/event-sink/console.ts に対応 (本実装は JSON 出力)。
package eventsink

import (
	"context"
	"io"
	"os"
	"sync"

	"github.com/ambi/idmagic/backend/shared/logging"
	"github.com/ambi/idmagic/backend/shared/spec"
)

type Console struct {
	mu  sync.Mutex
	out io.Writer
}

func NewConsole() *Console {
	return &Console{out: os.Stdout}
}

func (c *Console) Emit(ctx context.Context, e spec.DomainEvent) {
	c.mu.Lock()
	defer c.mu.Unlock()
	b, err := spec.MarshalDomainEvent(e)
	if err != nil {
		logging.Error(ctx, "event encode failed", "error", err)
		return
	}
	_, _ = c.out.Write(append(b, '\n'))
}
