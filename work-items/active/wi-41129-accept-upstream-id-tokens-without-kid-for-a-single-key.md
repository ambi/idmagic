---
status: pending
authors: [tn]
risk: medium
created_at: 2026-11-05
priority: p2
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/modules/authentication/standards.md, requirement: OIDC-CORE-CODE-FLOW }
  - { path: docs/modules/authentication/features/federation.md, requirement: REQ-AUTHENTICATION-001 }
---

# JWK Set の鍵が一つの上流の ID Token を、`kid` なしでも検証する

## 動機

OpenID Connect Core §10.1 は、参照する JWK Set に鍵が複数あるときだけ `kid` を必須とする。
鍵が一つなら、OP は ID Token の JOSE ヘッダーの `kid` を省いてよい。
`verifyUpstreamIDToken` は `kid` で鍵を引き、`kid` のない ID Token を「ID token key not found」で拒否する。
[[wi-77372-inventory-standards-acceptance-rules]] の棚卸しで見つけた。

## 対象範囲

- `kid` がなく、JWK Set の署名用の鍵がちょうど一つのときは、その鍵で検証する。
- `kid` がなく鍵が複数あるときは、現在どおり拒否する。
- `standards.md` に受け入れの行を加え、`kid` を省いた ID Token を返す上流で外部 OIDC のログインを通す試験を加える。

## 対象外

- RS256 以外の署名アルゴリズム。宣言の扱いは [[wi-82993-declare-upstream-oidc-acceptance-and-algorithm-limits]] が決める。

## タスク

- [ ] T001 [Spec] 受け入れの行と REQ-AUTHENTICATION-001 の例を加える。
- [ ] T002 [Acceptance] `kid` のない ID Token で RED を確かめる。
- [ ] T003 [App] 鍵の選択を直し、GREEN にする。
- [ ] T004 [Verify] 変更を検証する。

## 検証

- `verifyUpstreamIDToken` の単体試験：鍵が一つと複数の両側を確かめるため。
- 外部 OIDC の callback の受け入れ試験：ログインの成立を観測するため。

## リスク

`use` が `enc` の鍵を数えると、鍵が一つという判定を誤る。
署名に使える RSA の鍵だけを数える。
