---
status: pending
authors: [tn]
risk: medium
created_at: 2026-11-05
priority: p1
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/modules/oauth2/standards.md, requirement: RFC7662-INTROSPECT }
  - { path: docs/modules/api-tokens/standards.md, requirement: RFC7662-API-TOKEN-INTROSPECT }
---

# イントロスペクションのヒントが外れても、すべての種類のトークンを探す

## 動機

RFC 7662 §2.1 は、`token_type_hint` で示された種類にトークンが見つからなければ、対応するすべての種類を探すことを MUST とする。
`IntrospectToken` は、ヒントが空か `refresh_token` のときだけリフレッシュトークンを探す。
`access_token` か未知の値を付けてリフレッシュトークンを送ると、有効なトークンでも `active=false` が返る。
[[wi-77372-inventory-standards-acceptance-rules]] の棚卸しで見つけた。

## 対象範囲

- ヒントを探す順序の指定としてだけ使い、見つからなければ残りの種類を探す。
- 未知のヒントの値を、ヒントのない要求と同じに扱う。
- `standards.md` に、ヒントが外れた要求と未知のヒントを受け入れる行を加える。
- ヒントと実体が食い違う要求を `/introspect` へ送る受け入れ試験を加える。

## 対象外

- 失効エンドポイントのヒント。`RevokeToken` はヒントを読まずにすべての種類を探しており、宣言の行は [[wi-41643-declare-oauth2-acceptance-rows-for-optional-elements]] が加える。

## タスク

- [ ] T001 [Spec] 受け入れの行と要件の例を加える。
- [ ] T002 [Acceptance] `token_type_hint=access_token` のリフレッシュトークンと、未知のヒントの要求で RED を確かめる。
- [ ] T003 [App] 探索の順序を直し、GREEN にする。
- [ ] T004 [Verify] 変更を検証する。

## 検証

- `/introspect` の HTTP の受け入れ試験：ヒントを外した要求が `active=true` を返すことを製品の入口で確かめるため。

## リスク

探す順序を変えると、アクセストークンとリフレッシュトークンの値が衝突したときの結果が変わり得る。
リフレッシュトークンはハッシュで引くので衝突は実用上ないが、テナントの照合を両方の経路で保つ。
