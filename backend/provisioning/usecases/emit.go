package usecases

import "github.com/ambi/idmagic/backend/shared/spec"

// emit は、状態の保存に成功した後の遷移イベントを発行ポートへ渡す。発行ポートが
// 配線されていない単体の組み立てでは何もしない。本番の組み立ては Module の引数で
// 発行ポートを必須にしている。
func emit(sink func(spec.DomainEvent), event spec.DomainEvent) {
	if sink != nil {
		sink(event)
	}
}
