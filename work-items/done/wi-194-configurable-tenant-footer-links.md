---
status: completed
authors: ["tn"]
risk: medium
created_at: 2026-07-12
depends_on: []
---

# テナントのフッターリンクを任意のリンクテキスト付きで設定可能にする

## Motivation
現行の support URL と legal URL は URL だけを設定し、表示名は固定される。実際のフッター用途はサポート・法務に限定されず、リンクの目的を管理者が正確に利用者へ伝えられない。

## Scope
- `spec/contexts/tenancy.yaml` の `models`、`interfaces`、`invariants`、`scenarios`、`user_experience`。
- `support_url` / `legal_url` を、リンクテキストと URL を持つ順序固定の `footer_link_1` / `footer_link_2` へ置換する保存形式・API・管理 UI・hosted UI。
- HTTPS allowlist。

## Out of Scope
- 任意個数のリンク、HTML / Markdown を含むリンクテキスト、リンクごとのアイコン。

## Plan
- リンクの役割名をプロダクト契約から除き、各リンクを `{ label, url }` の安全な値として扱う。
- 未リリースのため既存値の後方互換 migration は行わず、旧 API フィールドを廃止する。

## Tasks
- [x] T001 [SCL] footer link model、更新・公開契約、入力制約を定義する。
- [x] T002 [Persistence/Domain] 保存形式と footer link の検証を実装する。
- [x] T003 [HTTP/UI] admin editor と hosted UI footer を任意ラベルの二つのリンクへ変更する。
- [x] T004 [Verify] XSS-safe label rendering、HTTPS-only URL、空リンクを回帰テストする。

## Verification
- `just scl-render`
- `just sqlc-generate`
- `just test-go`
- `just verify-ui`

## Risk Notes
公開 footer にテナント入力を描画するため、ラベルはプレーンテキストとして escape し、URL は HTTPS allowlist を維持する。

## Completion

- **Completed At**: 2026-07-12
- **Summary**:
  テナントの footer link を順序固定の `footer_link_1` / `footer_link_2`（ラベルと HTTPS URL の組）へ置換した。
  PostgreSQL、HTTP API、管理画面、hosted UI を同期し、未リリースであるため旧フィールドの移行は行わない。
- **Affected Guarantees State**:
  各リンクは完全なラベル・HTTPS URL の組でのみ保存でき、空の組で解除できる。hosted UI はラベルを React の標準エスケープでテキスト表示し、外部リンクは `noopener noreferrer` を維持する。
- **Verification Results**:
  - `just sqlc-generate` — passed
  - `just scl-render` — passed
  - `just test-go` — passed
  - `just verify-ui` — passed (291 tests)
  - `just verify` — passed
- **Evidence**:
  - 実行日: 2026-07-12
  - 実行環境: ローカル開発環境 (macOS)
  - 実行主体: Codex (GPT-5)
  - 対象ソース版: main (コミット前作業ツリー)
  - 保存先: 外部成果物なし。上記検証結果を本記録に要約。
