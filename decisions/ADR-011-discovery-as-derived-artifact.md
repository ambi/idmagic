---
status: accepted
authors: [tn]
created_at: 2026-07-04
---

# ADR-011: Discovery 文書はルーティングと grant matrix から導出され、独立に編集しない

## コンテキスト

OAuth 2.0 Authorization Server Metadata (RFC 8414) と OIDC Discovery (Discovery 1.0) は、
クライアントが IdP の能力を自動発見するための重要なメタデータである。
これが現実と乖離すると、クライアントの自動構成が壊れる。

実装上の罠:

- Discovery 文書を「文書として」手書きで保守する → grant_types_supported と
  実装が乖離するバグが頻発
- 各エンドポイントの実装を追加・削除しても Discovery に反映され忘れる
- 鍵アルゴリズムを変更しても `id_token_signing_alg_values_supported` が古いまま

## 決定

Discovery 文書を「導出された成果物」として扱う。仕様核に Discovery テンプレートを置き
(`spec/discovery.json`、issuer は `{{ISSUER}}` プレースホルダー)、その内容を他の仕様核ファイル
（grant matrix、トークンスキーマの署名アルゴリズム、ADR-002 の PKCE メソッド等）と
`spec/invariants.test.ts` で機械的に整合させる。アダプター層はテンプレートを読み `{{ISSUER}}`
を置換して返すだけで、個別フィールドを書き換えない。ビルド時生成ではなくランタイムでの
テンプレート読み込みを選んだのは、生成済みファイルのコミットがテンプレートとの二重管理・
ビルド忘れによる乖離を招くため。

現在の設計は [`backend/oauth2/ARCHITECTURE.md`](../backend/oauth2/ARCHITECTURE.md) にある。

## 却下した代替案

- **Discovery を完全自動生成**: 上述のとおり二重管理になる
- **手書きで保守**: 乖離リスクが高い
- **コードジェネレーターで型を生成**: 「JSON が権威」原則に反する。型生成は補助であり、
  権威ではない

## 影響

- 新規エンドポイント追加時は `spec/discovery.json` を必ず更新する
- 新規アルゴリズムサポート時は `spec/discovery.json` と `spec/tokens/*.schema.json` の両方を更新
- `spec/invariants.test.ts` の Discovery 整合性テストを CI で常時実行する
