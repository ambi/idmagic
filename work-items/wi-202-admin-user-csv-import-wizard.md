---
depends_on: [wi-96-bulk-user-import-csv]
status: pending
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
- [ ] T001 [SCL] AdminUsers の画面契約と受け入れシナリオを更新して再生成する。
- [ ] T002 [UI] CSV 選択・テンプレート・dry-run・結果プレビュー・apply 確認のウィザードを実装する。
- [ ] T003 [UI] ジョブ polling、失敗表示、一覧再読込、ja/en 文言を実装する。
- [ ] T004 [Test] 正常系、行エラー、API 失敗、apply 確認をコンポーネントテストする。
- [ ] T005 [Verify] `just verify` を通す。

## Verification
- `just yaml-check`
- `just test-ui-unit`
- `just verify-ui`
- `just verify`

## Risk Notes
apply はユーザー作成という副作用を持つ。dry-run 結果を表示するだけで適用せず、apply 操作には
確認を必須にする。CSV 内容やエラー表示にパスワード等の秘密情報を含めない。
