---
depends_on: [wi-96-bulk-user-import-csv]
status: completed
authors: ["tn"]
risk: medium
created_at: 2026-07-12
---

# 管理画面に CSV ユーザーインポートウィザードを追加する

## Motivation
WI-96 は CSV インポートのジョブ API と worker を提供したが、管理者がブラウザから安全に
アップロード、検証結果の確認、適用を行う導線はまだない。API を直接呼ばずとも、行エラーを
理解して修正・再試行できる画面が必要である。

## Scope
- `spec/contexts/identity-management.yaml` の `user_experience.screens.AdminUsers` に、
  CSV アップロード、dry-run プレビュー、apply 確認、ジョブ進捗・結果表示を追加する。
- `frontend/src/features/admin-users/` にウィザードを実装し、`ImportAdminUsers` と
  `GetAdminUserImport` API を呼ぶ。
- CSV テンプレートのダウンロード、入力制限・password 列拒否の利用者向け表示、行番号・列・
  stable error code の表示を追加する。
- ja/en 翻訳とコンポーネントテストを追加する。

## Out of Scope
- CSV API、worker handler、永続化方式の変更（WI-96 の範囲）。
- インポートのキャンセル、結果 CSV のダウンロード、属性マッピング UI。

## Plan
- Admin Users のアクションからモーダルまたは専用パネルを開く。
- ファイルをクライアントで UTF-8 text として読み、dry-run ジョブを投入して完了まで polling する。
- 検証結果を行単位で表示し、明示的な確認操作後に同じ CSV で apply ジョブを投入する。
- 成功・失敗・タイムアウトを翻訳済みメッセージで提示し、完了後はユーザー一覧を再読込する。

## Tasks
- [x] T001 [SCL] AdminUsers の画面契約と受け入れシナリオを更新して再生成する。
- [x] T002 [UI] CSV 選択・テンプレート・dry-run・結果プレビュー・apply 確認のウィザードを実装する。
- [x] T003 [UI] ジョブ polling、失敗表示、一覧再読込、ja/en 文言を実装する。
- [x] T004 [Test] 正常系、行エラー、API 失敗、apply 確認をコンポーネントテストする。
- [x] T005 [Verify] `just verify` を通す。

## Verification
- `just yaml-check`
- `just test-ui-unit`
- `just verify-ui`
- `just verify`

## Risk Notes
apply はユーザー作成という副作用を持つ。dry-run 結果を表示するだけで適用せず、apply 操作には
確認を必須にする。CSV 内容やエラー表示にパスワード等の秘密情報を含めない。

## Completion

- **Completed At**: 2026-07-12
- **Summary**:
  `spec/contexts/identity-management.yaml` の AdminUsers 画面契約に import wizard の状態を追加し、
  既存の CSV インポート受け入れシナリオの error code を実装 (`invalid_header` / `csv_too_large` /
  `too_many_rows` / `field_too_large`) に合わせて訂正した。`frontend/src/features/admin-users/` に
  `AdminUserImportPage` (CSV 選択 → テンプレート DL → dry-run → 行エラープレビュー →
  明示確認付き apply → 結果表示) を実装し、`/admin/users/import` ルートと一覧ページからの導線を追加した。
  `importAdminUsers` / `getAdminUserImport` の API クライアントと型、stable error code → ja/en
  翻訳文言、job polling とタイムアウト処理、コンポーネントテストを追加した。
- **Affected Guarantees State**:
  CSV アップロードは常にクライアント側で UTF-8 text として読み、apply は dry-run と同じ CSV を
  明示確認後にのみ送信する。password 列を含む CSV はヘッダー不一致として拒否され、行エラーには
  行番号・列・stable error code のみを表示し値は表示しない。バックエンドの API・worker・永続化は
  変更していない (WI-96 の範囲を維持)。
- **Verification Results**:
  - `just yaml-check` — passed
  - `just scl-render` — passed (`spec/idmagic.html` のみ更新、`interfaces`/`models` は無変更のため
    OpenAPI/JSON Schema 差分なし)
  - `just test-ui-unit` — passed (58 files, 339 tests)
  - `just verify-ui` — passed
  - `just verify` — passed
- **Evidence**:
  - 実行日: 2026-07-12
  - 実行環境: ローカル開発環境 (macOS)
  - 実行主体: Claude Code (Sonnet 5)
  - 対象ソース版: main (コミット前作業ツリー)
  - 保存先: 外部成果物なし。`frontend/src/features/admin-users/AdminUsersPage.test.tsx` の
    `AdminUserImportPage` describe ブロックで dry-run 行エラー表示、apply 確認ゲート、
    投入時エラーの翻訳表示を確認。
