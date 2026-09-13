---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-13
priority: p3
depends_on: []
change_kind: bugfix
spec_impact: { kind: none, reason: "EX-OAUTH2-001-01 は既に宣言済みである。実装をその宣言へ合わせるか、宣言の側を現在の設計へ寄せるかを決める作業であり、判断のあとで仕様先行に戻る。" }
affected_spec:
  - { path: docs/contexts/oauth2/scenarios.feature.md, requirement: REQ-OAUTH2-001 }
---

# account スコープのトークンが、宛先のリソースサーバーを名乗らない

## Motivation

[[wi-559-back-oauth2-remaining-examples-with-tests]] が `EX-OAUTH2-001-01` を消化する過程で、実装が具体例の `Then` を 1 つ満たしていないことを測った。

具体例は「アクセストークンの `sub` は同意した User、**audience はレルムの IdMagic API**、スコープは `account:read` になる」と言う。実測では `sub` と `scope` は宣言どおりだが、`aud` は `client_id` だった。

**レルムの IdMagic API を audience として名乗る経路が存在しない。** 理由は 3 つある。

1. `tokens_jose.JWTSigner.SignAccessToken` は、`Audiences` が空なら `aud` を `client_id` にする。
2. `Audiences` を埋めるのは resource indicator の解決だけで、`ResolveResourceIndicator` は登録済みの `McpResourceServer` しか受け付けない。レルム自身の API はそこに登録されていない。`/authorize` へ `resource=<レルムの発行者>` を付けると、認可リクエストごと拒否される。
3. account リソースサーバー側 (`support_http.Authenticator`) は audience を 1 度も読まない。判定に使うのはスコープだけである。

つまり「名乗っていない」だけでなく「名乗っても誰も読まない」状態で、直すには発行と検証の両方に判断が要る。

`sub`、`scope`、そして account リソースサーバーが参照だけを許すことは wi-559 が `TestAccountScopedGrantIssuesAUserBoundTokenTheAccountApiAccepts` で固定した。残るのは audience だけである。

## Scope

- レルムの IdMagic API を audience として名乗る経路を決める。resource indicator の解決先にレルム自身を加えるのか、account スコープの発行だけが特別に audience を差し替えるのかを Design に書く。
- account リソースサーバーが audience を検証するかどうかを決める。検証しないなら、具体例から audience の `Then` を落とす規範の変更として扱う。
- 決めた側に合わせて、`EX-OAUTH2-001-01` を名指すテストを `TestAccountScopedGrantIssuesAUserBoundTokenTheAccountApiAccepts` へ足し、`tools/check/example-coverage-debt.json` から当該行を外す。

## Out of Scope

- `McpResourceServer` に対する resource indicator の既存の振る舞い。RFC 8707 の行は別の記録が持つ。
- REQ-OAUTH2-001 のほかの具体例。`EX-OAUTH2-001-02` と `EX-OAUTH2-001-03` は消化済みである。
- ほかの Context のアクセストークンの audience。

## Verification

- `mise run test-go-package -- ./backend/oauth2/handlers_http`
- `mise run check-spec` が `EX-OAUTH2-001-01` を台帳から外した状態で通る。
- `mise run verify`

## Risk Notes

- **発行だけを直して、検証を足さない。** audience を名乗らせても誰も読まなければ、宣言が 1 つ増えるだけで境界は動かない。発行と提示の両方を観測する。
- **audience の検証を足して、既存のトークンを一斉に拒否する。** account リソースサーバーは現在 `aud=client_id` のトークンを受けている。検証を入れるなら、どの値を受けるかを先に決める。
- **resource indicator の解決先にレルムを足すと、既存の拒否が緩む。** `ResolveResourceIndicator` は登録済みかつ Active の資源だけを通す設計で、そこへ例外を作ると未登録の resource を通す経路になりうる。例外の形を Design に書く。
