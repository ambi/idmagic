package domain

import "errors"

// モデル・タプル・整合トークンの検証で返す番兵エラー。ユースケースはこれらを
// HTTP のステータスへ写し、判定そのものは Decision の拒否理由で表す。
var (
	ErrModelInvalid            = errors.New("authorization model is invalid")
	ErrModelNotFound           = errors.New("authorization model not found")
	ErrTupleInvalid            = errors.New("relation tuple is invalid")
	ErrConsistencyNotSatisfied = errors.New("consistency token is not satisfied")
)
