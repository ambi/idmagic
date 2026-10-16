package tokens_jose

import (
	"context"
	"sync"
	"time"
)

// replayStoreFake は jti を記録の期間だけ覚えるリプレイ記録である。DPoP と client assertion の
// 検証が使うポートを満たす。
type replayStoreFake struct {
	mu      sync.Mutex
	expires map[string]time.Time
}

func newReplayStoreFake() *replayStoreFake {
	return &replayStoreFake{expires: map[string]time.Time{}}
}

func (s *replayStoreFake) RecordIfNew(_ context.Context, jti string, windowSeconds int, now time.Time) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if expiresAt, ok := s.expires[jti]; ok && now.Before(expiresAt) {
		return false, nil
	}
	s.expires[jti] = now.Add(time.Duration(windowSeconds) * time.Second)
	return true, nil
}
