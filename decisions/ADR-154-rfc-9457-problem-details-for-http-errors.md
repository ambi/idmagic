---
status: accepted
authors: ["tn"]
created_at: 2026-08-08
---

# ADR-154: HTTP エラーレスポンスの既定 envelope として RFC 9457 Problem Details を採用する

## コンテキスト

`WriteBrowserError` (`{"error": code, "message": text}`) が全体で約 390 箇所から
使われているが、これは RFC 9457 (旧 RFC 7807) が定める相互運用可能な
`application/problem+json` 形式ではない。この不整合は `wi-111`
(`work-items/done/wi-111-panic-recovery-and-request-id-middleware.md`) で
「全エンドポイントの RFC 7807 problem+json 統一」として既に認識されながら
明示的にスコープ外とされ、先送りされていた。

一方、ステータスコードも RFC 9110 の意味論から外れており、
「構文は正しいが内容が処理できない」系のビジネスルール違反 (`invalid_role`,
`password_reuse` など) が軒並み 400 に丸められ、422 は全体で 1 箇所
(`quota_exceeded`) しか使われていない。両者は独立した issue に見えるが、
どちらも同じ「エラーレスポンスの意味論を仕様に合わせる」作業であり、
分離すると status だけ合わせて envelope が食い違う中間状態が生まれるため、
一つの決定としてまとめて扱う。

## 決定

- 汎用 API のエラーレスポンスは RFC 9457 Problem Details
  (`type`/`title`/`status`/`detail`/`instance`、`application/problem+json`)
  を既定 envelope とする。`instance` には既存の `request_id`
  (`ARCHITECTURE.md` Request correlation、wi-111 で導入済み) をそのまま使う。
  `type` は当面 idmagic 内部限定の識別子 (`urn:idmagic:error:<code>` 形式) とし、
  公開ドキュメント URL の発行は将来の課題として持ち越す。
- ステータスコードは RFC 9110 に従い、「リクエストを解釈できない」
  (JSON parse 失敗など構文レベル) は 400、「解釈はできたが内容で処理できない」
  (ビジネスルール・参照整合性違反) は 422 を使う。
- 以下は既存仕様がエラー形式を規定しているため対象外とし、現状の形式を維持する:
  OAuth2 (RFC 6749 §5.2 の `{error, error_description}`)、
  SCIM (RFC 7644 の `scimType`)、
  Dynamic Client Registration (RFC 7591 の `invalid_client_metadata` 等)。
- SharedSignals の inbound SET receiver エンドポイント
  (`POST /ssf/streams/:id/events`) は RFC 8935 (Push-Based SET Delivery) が
  エラー応答形式を規定している可能性があり未検証。**この ADR の時点では
  結論を出さず、対象外候補として残し、実装 work item の中で仕様を確認してから
  Problem Details 化するか除外に加えるかを決める**。同じ SharedSignals でも
  stream 管理 (admin CRUD) API は idmagic 独自 API であり Problem Details の
  対象とする。
- 設計の詳細 (規約の全文、SCL への表現方法) は `ARCHITECTURE.md` の
  HTTP error responses 節に置く。

## 却下した代替案

- 既存の `{error, message}` 形式を維持し続ける: 相互運用性がなく、
  クライアント側でエラー種別 (構文/意味論/認可等) を型的に区別できない。
- RFC 9457 を全エンドポイントに強制し OAuth2/SCIM/DCR も置き換える:
  これらは自身が別の相互運用フォーマットを規定する標準プロトコルであり、
  Problem Details に寄せるとクライアント側 (標準ライブラリ) の期待する形式を
  壊す。標準準拠を優先し、対象から除外する。

## 影響

- SCL: `models` の `kind: error` に HTTP status を表す field、
  `bindings` (`kind: http`) にエラー envelope 形式を表す field を追加する
  言語拡張が必要 (別途 work item で実施)。
- `spec/contexts/*.yaml` の 75+ 個の `kind: error` model への status 付与、
  および `backend/shared/http/support_http` の実装・約 390 箇所の
  ハンドラ移行 (別途 work item で実施)。
