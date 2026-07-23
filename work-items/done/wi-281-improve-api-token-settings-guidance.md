---
status: completed
authors: [codex]
risk: low
created_at: 2026-07-24
depends_on: []
---

# API アクセストークン設定で接続先と scope の意味を理解できるようにする

## Motivation
設定画面では「API アクセストークン」が重複表示され、SCIM 以外の API 接続先も分からない。
さらに scope の正準名が大量に平置きされているため、管理者が用途と権限を判断しにくい。

## Scope
- `spec/contexts/api-tokens.yaml` の `scenarios`
- `frontend/src/features/admin-settings/ApiTokensTab.tsx` の接続情報と scope 選択 UI
- `frontend/src/features/admin-settings/AdminSettingsPage.i18n.ts` の日本語・英語表示
- `frontend/src/features/admin-settings/ApiTokensTab.test.tsx` の受け入れテスト

## Out of Scope
- API アクセストークンの scope 語彙、認可規則、発行・失効 API 契約の変更
- 新しい API endpoint の追加

## Tasks
- [x] T001 [SCL] 管理者が接続先と scope の意味を理解できるシナリオを追加する。
- [x] T002 [UI] 主見出しを維持しつつ重複した一覧見出しを改称し、通常 API と SCIM API の Base URL を提示する。
- [x] T003 [UI] scope を用途別・リソース別に整理し、権限の説明を付ける。
- [x] T004 [Verify] SCL、UI テスト、UI 検証を通す。

## Verification
- `just yaml-check-scl`
- `just scl-render`
- `just test-ui-unit`
- `just verify-ui`
- `just yaml-check-work-items`
- `just check-ids`

## Risk Notes
表示と選択方法だけを変更し、送信する正準 scope 値は維持する。既存の全 scope を選択できることを
テストで保証し、権限欠落を防ぐ。

## Completion
- **Completed At**: 2026-07-24
- **Summary**:
  API アクセストークンタブの主見出しを他の設定タブと同じ構造で維持し、
  重複していた一覧側を「発行済みトークン」と明確に命名した。
  管理 API、SCIM 2.0 API、発行者本人のアカウント API の Base URL を
  用途説明・コピー操作付きで提示した。
  scope 選択は管理 API、SCIM 2.0 API、アカウント API の折りたたみグループへ分割し、
  リソース名、参照/変更の意味、正準 scope 値、選択数を表示するようにした。
  2 列表示では左を参照、右を変更に固定し、write-only scope も右列へ揃えた。

### Affected Guarantees State
- `管理者は接続先とscopeの意味を理解してAPIアクセストークンを構成できる`: fulfilled。
  大量の scope を常時平置きせず、API surface と resource の意味を確認しながら必要権限だけを選べる。
- `管理者はAPIアクセストークンを発行・失効できる`: maintained。
  発行 API へ送る正準 scope 値と発行・一覧・失効フローは変更していない。

### Evidence
- 実行環境: ローカル workspace、2026-07-24、Codex。
- 対象ソース版: 本 WI の未コミット作業ツリー。
- `ApiTokensTab.test.tsx` で3種類の Base URL、scope group、説明、折りたたみ初期状態、
  全既存用途の scope 選択と送信値、write-only scope の右列配置を検証した。
- `AdminSettingsPage.test.tsx` で主見出しと「発行済みトークン」一覧見出しが
  適切な見出しレベルで共存することを検証した。
- SCL 派生物を `just scl-render` で再生成し、`spec/idmagic.html` を同期した。

- **Verification Results**:
  - `just verify-ui` — passed（77 files / 427 tests、format、lint、typecheck、build を含む）
  - `just yaml-check` — passed
  - `just scl-render` — passed
  - `git diff --check` — passed
