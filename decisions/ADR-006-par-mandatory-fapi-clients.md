---
status: accepted
authors: [tn]
created_at: 2026-07-04
---

# ADR-006: PAR を FAPI クライアントに必須とし、一般クライアントにはオプションで提供する

## コンテキスト

`/authorize` への直接アクセスは、認可リクエストパラメータがブラウザのアドレスバーを通る。
これは以下のリスクを生む:

- パラメータの改ざん（オープンリダイレクト・CSRF）
- 長大化したクエリ文字列（JAR / リッチ認可リクエスト等）の URL 上限
- リクエスト整合性の事前検証ができない（クライアント認証なしで /authorize に到達する）

Pushed Authorization Requests (RFC 9126) は、これらに対処するため
**先にバックチャネルでクライアント認証付きで認可パラメータを送る** という設計を導入する。
`/par` が `request_uri` を返し、`/authorize` ではこれを参照するだけになる。

FAPI 2.0 §5.2 は PAR を MUST とする。

## 決定

本アプリ IdP はすべてのクライアントに `/par` エンドポイントを提供し、クライアントメタデータの
`require_pushed_authorization_requests = true` を宣言したクライアント（FAPI プロファイル含む）
には PAR を必須とする。それ以外のクライアントは PAR / 通常 `/authorize` の両方を選択できる。
`request_uri` は TTL 600 秒以下・単一使用とし、`/authorize?request_uri=...` に追加のクエリ
パラメータが付随しても保存されたパラメータを優先して追加分は無視する（RFC 9126 §4）。

現在の設計は [`backend/oauth2/ARCHITECTURE.md`](../backend/oauth2/ARCHITECTURE.md) にある。

## 却下した代替案

- **JAR (RFC 9101) のみ**: JAR は同等の防御を提供するが、認可リクエストを JWT に
  封入するためクライアント実装の負荷が高い。PAR のほうがエコシステム採用が進んでいる
- **PAR を全クライアントに必須**: 既存クライアントの移行コストが高い。FAPI のみ必須化
- **request_uri を再利用可能に**: 攻撃時のウィンドウが広がる

## 影響

- `adapters/persistence/in-memory-par-store.ts` を実装する
- `policy/client-authorization.json` の `authorize:initiate.rules` に
  `par_required_if_fapi` を含める
- `spec/slo.yaml` の `par_request_uri_ttl_seconds` で TTL を制御
- Discovery `pushed_authorization_request_endpoint` と `require_pushed_authorization_requests`
  を反映する
