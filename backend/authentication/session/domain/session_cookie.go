package domain

import "errors"

// SessionCookie はログインセッションを運ぶ cookie の名前である。ログアウトでセッションの cookie を
// 消すプロトコルのモジュールも、同じ名前を使う。
const SessionCookie = "idmagic_session"

// ErrSessionNotFound は対象セッションが存在しないか、本人のものでない場合。
var ErrSessionNotFound = errors.New("session not found")
