// Package authorization は Authorization bounded context を組み立てる
// ([[wi-53-rebac-fine-grained-authorization]])。
package authorization

import (
	"github.com/ambi/idmagic/backend/authorization/ports"
)

// Module は Authorization の管理 API と判定経路が必要とする依存を持つ。
// bootstrap が永続層 (memory / postgres) ごとに組み立てて渡す。
type Module struct {
	TupleRepo ports.RelationTupleRepository
	ModelRepo ports.AuthorizationModelRepository
}
