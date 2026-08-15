// Package usecases は Authorization Context のアプリケーションロジックを持つ。
// 判定の合成は AuthZEN の Authorizer が行い、ここは関係の事実を組み立てて渡す。
package usecases

import "errors"

// ErrAuthorizerUnavailable は AuthZEN 評価器へ到達できなかったことを表す。
// 呼び出し側は許可へ退避してはならない。
var ErrAuthorizerUnavailable = errors.New("authorization evaluator is unavailable")
