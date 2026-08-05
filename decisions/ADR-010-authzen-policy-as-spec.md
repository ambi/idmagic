---
status: accepted
authors: [tn]
created_at: 2026-07-04
---

# ADR-010: 認可ポリシーを仕様核に置き、AuthZEN スタイルのインターフェースで評価する

## コンテキスト

OAuth2 / OIDC IdP には複数種類の認可判断が散在する:

- 認可エンドポイントでクライアントとリダイレクト URI を検証
- トークンエンドポイントでクライアントが宣言したグラントタイプを保持しているか確認
- リフレッシュトークンが失効していないか、絶対 TTL を超えていないか
- センダー制約 (DPoP/mTLS) と所有証明の整合
- UserInfo エンドポイントの scope チェック
- /introspect 呼び出し元がリソースサーバーとして認証されているか

これらをアダプター層に `if` で散在させると、再生成時に脱落のリスクがある。
Regenerative Architecture が「セキュリティポリシーは仕様核に置く」と主張する所以である。

## 決定

`spec/policy/client-authorization.json` に、全アクションと判定ルールを宣言的に記述する。
評価は AuthZEN（OpenID Foundation Authorization API 仕様）スタイルの
`{ subject, action, resource, context }` インターフェースで行う。仕様核がアクションと判定
ルールの唯一の権威を持ち、アダプター層 (`local-authzen-adapter.ts`) が `authorize()` 関数を
提供する。現在は仕様核の `evaluate()` を直接呼ぶが、将来は外部の AuthZEN サービス・OPA・Cedar
への差し替えをこのアダプターだけで吸収できる。JSON 側の各 `rules[].id` は TypeScript 側の
`ruleEvaluators` に同名キーで実装され、未実装ルールが残っていないことを
`spec/invariants.test.ts` の網羅性テストが検知する。ポリシー言語として本アプリでは JSON +
TypeScript 純粋関数を採用した（参照実装としての明快さ・外部ランタイム依存なし・即時テスト
可能性のため）。

現在の設計は [`backend/oauth2/ARCHITECTURE.md`](../backend/oauth2/ARCHITECTURE.md) にある。

## 却下した代替案

- **HTTP ルーターに `if` で散在**: 再生成リスク・監査困難
- **OPA Rego を仕様核に直接**: language-agnostic ではあるが可読性が低く、
  ビジネスステークホルダーが読めない
- **Cedar を仕様核に直接**: 良い候補だが、ランタイム依存（AWS）が発生する。
  今回は中立な JSON + 純粋関数を選び、Cedar 移行は ADR を分岐させる前提とする

## 影響

- 認可ルールの変更は `spec/policy/client-authorization.json` のみで完結する
- HTTP / トークン / UserInfo / Introspection の各ルーターは `authorize()` を呼ぶだけ
- ポリシー変更後は `authorization.test.ts` の全テストが通ることを必ず確認する
- `spec/invariants.test.ts` の網羅性テストで、ルールの実装漏れを防ぐ
