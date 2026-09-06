// Package testing_passwords は、鍵導出関数そのものを検査しないテストのための
// Argon2id ハッシャーを提供する。
//
// 本番の OWASP パラメータ (m=19456 KiB, t=2) は 1 回のハッシュにおよそ 100 ミリ秒、
// -race の下ではおよそ 300 ミリ秒かかる。ログインを踏むだけのテストがこれを 1 件ずつ
// 払うと、費用はテストの本数だけ積み上がる。パラメータは PHC 文字列に埋め込まれるので、
// ここが返すハッシュは本番コストの検証器がそのまま受理する。
//
// 本番コストでしか出ない不具合 (メモリ確保、パラメータの符号化) の検出は
// passwords_argon2id package 自身のテストが持つ。そこは本番コストのままである。
package testing_passwords

import "github.com/ambi/idmagic/backend/shared/security/passwords_argon2id"

// NewHasher はテスト用のコスト設定を持つ Argon2id ハッシャーを返す。
func NewHasher() *passwords_argon2id.Argon2idPasswordHasher {
	return &passwords_argon2id.Argon2idPasswordHasher{MemoryCost: 64, TimeCost: 1, Parallelism: 1}
}
