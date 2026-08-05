---
status: accepted
authors: [tn]
created_at: 2026-07-04
---

# ADR-007: コンセントは「subject × client × scope 集合」単位で永続化する

## コンテキスト

コンセント（同意）の粒度設計には複数の選択肢がある:

- **クライアントごと（最大粒度）**: ユーザーが特定のクライアントに対して 1 度同意したら、
  以降そのクライアントが要求するスコープは UI なしで通過
- **スコープごと**: 各スコープに対して独立に同意状態を持つ
- **subject × client × scope 集合**: クライアント × 「過去に同意したスコープ集合」を 1 レコード
- **対話ごと**: 認可ごとに必ず再表示

業界の主流は「subject × client × scope 集合」である（Google, Microsoft, Auth0 など）。

## 決定

`spec/consent.schema.json` に従い、コンセントは `sub × client_id → { scopes[], granted_at,
expires_at }` の単位で永続化する（「クライアントごと」では後から強権限スコープが追加されたとき
過去の同意で自動許可されてしまい、GDPR の「特定の目的に対する同意」原則に反する。「対話ごと」
では同意疲れを招く）。同意は `granted_at + 365 日` で期限切れとし、`prompt=consent` は既存同意
の有無に関わらず UI を再表示する。取り消しは以降の認可にのみ影響し、既発行トークンの失効は
明示的な操作とする（ADR-004 のファミリー失効ロジックを再利用できる）。

現在の設計は [`backend/oauth2/ARCHITECTURE.md`](../backend/oauth2/ARCHITECTURE.md) にある。

## 却下した代替案

- **コンセントを JWT として発行（持ち回り型）**: 取り消しが困難。サーバーサイドの
  失効リストが必要になり、結局ストアが要る
- **scope ごとに独立**: scope 数が増えるとレコード爆発・UI 複雑化
- **対話ごとに必ず再表示**: UX が悪く、ユーザーが「同意疲れ」を起こす

## 影響

- `consent.schema.json` は scope を配列として持つ（uniqueItems 制約あり）
- `requirements.md §2` がこの挙動を EARS で記述している
- `scenarios.feature` に「prompt=consent で再表示」と「既存同意でスキップ」の両方が含まれる
