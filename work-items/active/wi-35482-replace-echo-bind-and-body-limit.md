---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-11-08
priority: p3
depends_on: []
change_kind: refactor
spec_impact: { kind: none, reason: "REQ-SYSTEM-027 の 413 と、動的クライアント登録の JSON の受理とエラーの形を保つ。Echo の Bind が JSON 以外の本文も受理していた差を解消する場合は、仕様の段階へ戻る。" }
---

# Echo の Bind と BodyLimit を標準の処理へ置き換える

## 動機

wi-34240 で Echo の維持を決めた。
Echo 固有の入力処理のうち、本番の経路に残るのは動的クライアント登録（RFC 7591）の `c.Bind` と、全要求にかかる `middleware.BodyLimit` の二つである。
どちらも利用者の入力が最初に届く処理なので、型付きの JSON デコードと `http.MaxBytesReader` に置き換え、入力処理を自前の検査した経路へ寄せる。

## 対象範囲

- `oauth2/internal/handlers_http/register_handler.go` の `c.Bind` を、`registerClientRequest` への JSON のデコードに置き換える。
- `cmd/idmagic/server.go` の `middleware.BodyLimit` を、`support_http` の本文上限のミドルウェアに置き換える。
- `bootstrap/hardening_test.go` の REQ-SYSTEM-027 のテストを、製品が配線するミドルウェアに向ける。

## 対象外

- 汎用 API の 64 KiB の上限（REQ-PLATFORM-005）の実装。
- Echo からの移行（wi-34240 で中止）。

## 設計

### 本文の上限

Echo の `BodyLimit` は、`Content-Length` が上限を超える要求をハンドラーへ渡す前に 413 で拒否し、長さの分からない要求には読み取りの上限を付ける。
REQ-SYSTEM-027 は「ハンドラーへ渡さずに 413 で拒否」を求めるので、同じ二段で実装する。

1. `Content-Length` が上限を超えれば、ハンドラーを呼ばず 413 を返す。
2. それ以外は `http.MaxBytesReader` で本文を包む。
   上限を超えた読み取りが返す `*http.MaxBytesError` は、中央のエラー処理で 413 へ写す。

413 の本文は、現在の `echo.ErrStatusRequestEntityTooLarge` が中央のエラー処理を通って作る形と同じにする。
着手時に現在の 413 の本文とヘッダーを観測し、テストで固定してから置き換える。

### 動的クライアント登録の本文

Echo の `Bind` は、POST では `Content-Type` に応じて JSON、XML、フォームのどれかとして本文を読み、本文が空なら何もしない。
置き換え後は JSON だけを読み、空の本文はこれまでどおり空のメタデータとして検証へ進める。
JSON 以外の `Content-Type` をどう扱うかは、利用者が観測する結果を変えるため、着手時に利用者へ確認する。
推奨は、RFC 7591 §3.1 が登録要求を `application/json` と定めることに合わせ、`invalid_request` で拒否することである。
`error_description` の文言は Echo のエラー文から `encoding/json` のエラー文に変わり得る。
文言は規範ではないが、テストが文言を表明していれば着手時に確かめる。

## 計画

1. 現在の 413 の応答と、登録の JSON、空の本文、不正な JSON の応答を観測してテストで固定する。
2. 本文上限のミドルウェアを `support_http` に作り、`server.go` の配線を置き換える。
3. 登録のデコードを置き換える。

## タスク

- [ ] T001 [Characterize] 413 の応答と登録の本文の扱いを固定する。
- [ ] T002 [Adapters] 本文上限のミドルウェアを作って配線を置き換える。
- [ ] T003 [Adapters] 登録の `Bind` を JSON のデコードに置き換える。
- [ ] T004 [Verify] 変更したパッケージと逆依存の検査を通す。

## 検証

- REQ-SYSTEM-027 は `bootstrap/hardening_test.go` と、`Content-Length` を持たない要求の場合の `support_http` のテストで確かめる。
- 登録は `oauth2/internal/handlers_http` の既存の RFC 7591 のテストで確かめる。
- 全体は `mise run verify-changed` で確かめる。

## リスク

本文上限の読み取りエラーを、本文を読むハンドラーが 400 として返すと、長さの分からない要求の超過が 413 にならない。
現在の Echo の `BodyLimit` でも同じ経路を通るので、置き換えの前後で同じ応答になることをテストで確かめる。
