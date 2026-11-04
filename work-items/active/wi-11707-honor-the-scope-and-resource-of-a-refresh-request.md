---
status: pending
authors: [tn]
risk: medium
created_at: 2026-11-05
priority: p2
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/modules/oauth2/features/token.md, requirement: REQ-OAUTH2-006 }
  - { path: docs/modules/oauth2/standards.md, requirement: RFC8707-MCP-RESOURCE-BINDING }
---

# リフレッシュの要求の `scope` と `resource` を読み、範囲を狭める

## 動機

RFC 6749 §6 は、リフレッシュの要求の `scope` を OPTIONAL とし、省けば元の範囲、付ければ元の範囲を超えない範囲で発行すると定める。
RFC 8707 §2.2 も、リフレッシュの要求の `resource` で対象を狭められるとする。
`/token` の `refresh_token` は `scope` も `resource` も読まず、狭めた要求にも元の範囲のトークンを返す。
元の範囲を超える `scope` も `invalid_scope` にならない。
`RFC8707-MCP-RESOURCE-BINDING` はリフレッシュトークンのローテーションを適用の経路に挙げているが、要求の `resource` を見ていない。
[[wi-77372-inventory-standards-acceptance-rules]] の棚卸しで見つけた。

## 対象範囲

- `scope` を付けたリフレッシュの要求では、元の範囲の部分集合だけを発行し、超える値を `invalid_scope` で拒否する。
- `resource` を付けたリフレッシュの要求では、元の付与が許した対象へ `audience` を狭め、許していない対象を `invalid_target` で拒否する。
- 省いた要求の結果（元の範囲と対象）を変えない。
- 狭めた要求が、新しいリフレッシュトークンの範囲を狭めるかどうかを決めて仕様に書く。RFC 6749 §6 は元の範囲を保つことを許す。

## 対象外

- リフレッシュトークンのローテーションと再利用の検知。REQ-OAUTH2-006 の既存の振る舞いを変えない。

## タスク

- [ ] T001 [Spec] REQ-OAUTH2-006 に `scope` と `resource` の例を加え、`standards.md` に受け入れの行を加える。
- [ ] T002 [Acceptance] 狭めた要求と超えた要求を `/token` へ送り、RED を確かめる。
- [ ] T003 [App] `RefreshInput` に範囲と対象を足し、GREEN にする。
- [ ] T004 [Verify] 変更を検証する。

## 検証

- `/token` の HTTP の受け入れ試験：発行したアクセストークンの `scope` と `aud` を観測するため。

## リスク

狭めたリフレッシュトークンを保存し直すと、後の要求が元の範囲へ戻せなくなる。
T001 で決める扱いを、ローテーションのファミリーの記録と一致させる。
