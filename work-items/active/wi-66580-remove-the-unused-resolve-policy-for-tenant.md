---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-10-09
priority: p3
depends_on: []
change_kind: refactor
spec_impact:
  kind: none
  reason: "本番からもテストからも呼ばれない関数を削除するだけで、パスワードポリシーの解決結果、要件、TypeSpec の契約、永続状態、発行するイベント、外向きの呼び出しは変えない。"
---

# 使われていない ResolvePolicyForTenant を削除する

## 動機

`backend/authentication/password/usecases/password_expiry.go` の `ResolvePolicyForTenant` は、コメントで「文脈にテナントを持たない経路（バッチなど）向け」とされている。
しかし、本番コードにもテストにも呼び出し元がない。
パスワードポリシーを解決する経路は `ResolveTenantPolicy` だけであり、その呼び出し元はすべて HTTP のハンドラーから届く。

使われていない入口が残っていると、読み手は「バッチではこちらを使う」という存在しない経路を前提にしてしまう。
テナントを解決していない文脈で default テナントへ落ちる処理を調べたときに見つけた。

## 対象範囲

- `ResolvePolicyForTenant` とそのコメントを削除する。

## 対象外

- `ResolveTenantPolicy` の振る舞いの変更。
- ジョブの文脈が ID だけのテナントを運ぶ問題。
  wi-73872 が扱う。

## 設計

関数を削除する。
同じ計算は `passworddomain.ResolvePasswordPolicy(tenant, DefaultPasswordPolicySnapshot())` で書けるので、将来テナントを手元に持つ経路が現れたら、その時点で呼び出し側から組み立てる。

## 計画

1. 呼び出し元がないことを `rg` で確かめてから削除する。

## タスク

- [ ] T001 [App] `ResolvePolicyForTenant` を削除する。
  N/A: 振る舞いを変えない削除である。代わりにビルドと `mise run lint-go` を検査に使う。
- [ ] T002 [Verify] 変更を検証する。

## 検証

- `mise run lint-go`
- `mise run verify`

## リスク

- 低い。呼び出し元がないので、削除してもビルドが通ればどの経路も変わらない。
