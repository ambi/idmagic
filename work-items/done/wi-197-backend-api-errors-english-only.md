---
status: completed
authors: ["tn"]
risk: medium
created_at: 2026-07-12
depends_on: []
---

# バックエンド API エラーの利用者向けメッセージを英語に統一する

## Motivation

API の `message` と `error_description` に日本語と英語が混在すると、REST/OIDC/SAML などの外部利用者は言語を予測できず、UI の翻訳責務も曖昧になる。バックエンドは安定した error code と英語メッセージだけを返し、表示層だけが必要に応じて翻訳する境界を確立する。

## Scope

- `spec/contexts/system.yaml` の glossary / interfaces / invariants / scenarios / objectives / `user_experience.requirements.UX-API-ERROR-ENGLISH-ONLY` を、全 backend API error へ適用する契約として精密化する。
- `backend/` の HTTP、OAuth/OIDC redirect、validation、domain-to-HTTP error で返す日本語 message / `error_description` を英語へ置換する。
- error code を変えず、既存クライアントの機械可読な契約を維持する。
- 日本語メッセージを検出し、代表的な HTTP と protocol error が英語で返ることをテストする。

## Out of Scope

- UI 自身が表示するネットワークエラーや画面内 validation 文言の翻訳。
- 監査ログ、メール本文、管理者が設定した任意テキストの英語化。

## Plan

- SCL に「backend が送信する利用者向け error text は英語」を interface/invariant として明記する。
- backend の error response 作成箇所と protocol redirect を横断検索し、error code を変えずに英文へ統一する。
- テストの期待値を更新し、日本語の API error text を回帰検出する静的・契約テストを追加する。

## Tasks

- [x] T001 [SCL] API error 英語固定の契約・正常/拒否/未知 error scenario を更新する。
- [x] T002 [Backend] HTTP・protocol・validation error の日本語 message / error_description を英語に統一する。
- [x] T003 [Test] error code 維持と英語 message を API/redirect テストで保証する。
- [x] T004 [Verify] `just yaml-check`、`just verify-go`、`just verify` を通す。

## Verification

- `just yaml-check`
- `just verify-go`
- `just verify`

## Risk Notes

error text は既存利用者が表示している可能性があるため、error code・HTTP status・response schema は変更しない。プロトコル規定の error code は翻訳しない。

## Completion

- **Completed At**: 2026-07-12
- **Summary**:
  SCL に横断的な BackendErrorText 契約を追加し、共通 HTTP、OAuth/OIDC、SCIM、SAML、WS-Federation の API エラー本文を英語へ正規化した。
  日本語を含む本文は安定した英語フォールバックへ置換し、error code と HTTP status は維持する。
- **Affected Guarantees State**:
  バックエンドが外部へ返す message、error_description、detail、プレーンテキストのエラー本文は表示言語に依存せず英語となる。機械可読な error code、HTTP status、response schema は変更しない。
- **Verification Results**:
  - `just yaml-check` — passed
  - `just scl-render` — passed
  - `just test-go` — passed
  - `just verify-go` — passed
  - `just verify-ui` — passed
  - `just verify` — passed
- **Evidence**:
  - 実行日: 2026-07-12
  - 実行環境: ローカル開発環境 (macOS)
  - 実行主体: Codex (GPT-5)
  - 対象ソース版: main (コミット前作業ツリー)
  - 保存先: 外部成果物なし。`backend/shared/kernel/error_text_test.go` と OAuth client authentication 契約テストで英語フォールバックと error code 維持を確認。
