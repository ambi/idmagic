---
status: completed
authors: [tn]
risk: low
created_at: 2026-07-20
depends_on: []
---

# 管理画面のUI改善と表記統一

## Motivation
管理画面の様々な画面において、UIの不統一（ボタン表記、レイアウト）、未翻訳の英語メッセージ（Password, active user, grantedなど）、不要なクイックリンク、一覧画面の右ペインの情報過多など、ユーザー体験を損ねる箇所が多数見受けられる。これらを一貫性のある直感的なUIへと改善する。

## Scope
- `frontend/` 内の各画面（ダッシュボード、ユーザー一覧・詳細・編集、グループ一覧、ロール一覧、アプリケーション一覧・編集、プロビジョニング一覧、サインインポリシー、同意一覧）のコンポーネント

## Out of Scope
- 管理画面の全体的なデザインシステムの変更
- 対象として挙げられていない画面のUI改善

## Plan
- 指摘されたUIの問題点を各画面（コンポーネント）ごとに修正する。
- ボタン表記を「追加」等に統一する。
- 未翻訳の文言のi18n対応を行う。

## Tasks
- [x] T001 [App] ダッシュボードの「管理者クイックリンク」の削除・見直し。サイドバーと重複するため右カラムごと削除。
- [x] T002 [App] ユーザー一覧右ペインの「グループ追加」UI見直し。参照画面（一覧右ペイン）は編集させない方針とし、`UserGroupsSection` に `allowEditing` を追加してグループ追加 UI を撤去（追加は専用詳細画面のみ）。
- [x] T003 [App] ユーザー・グループ一覧右ペインの「無効化・削除」ボタンのサイズと「アカウントを」「グループを」表記の見直し。`AdminPaneActions` を、ボタン 3 個以下は 1 行均等幅・ユーザーのみ 4 個で 2×2 グリッドに切り替える方式へ。ラベルの「アカウントを」「グループを」接頭辞を除去し「無効化」「削除」等へ短縮。
- [x] T004 [App] ユーザー一覧右ペインの「Subject ID」を「ユーザーID」に変更。詳細画面と同じ `userId` キーへ統一し、未使用になった `subjectId` キーを削除。
- [x] T005 [App] ユーザー詳細情報の「Password」の日本語化。`passwordBadge`/新設 `passwordAndMfaBadge` を追加し、ハードコードされていた `Password + MFA` も置換。
- [x] T006 [App] ユーザー編集画面のフォームレイアウト改善。中央寄せの狭い単一列を、左寄せ・幅拡大 (`max-w-4xl`)・縦積みセクション＋プロフィールの氏名フィールドのみ 2 列に再構成。
- [x] T007 [App] 各リソース追加ボタンの表記統一。users/groups/applications/agents/workflows/mcp-resource-servers/authz-detail-types の追加ボタンを「Xを追加」に統一（エージェント「登録」→「エージェントを追加」、ワークフロー「新規作成」→「ワークフローを追加」など）。グループ追加アイコンを `IconUsersPlus`、ワークフロー追加ボタンにアイコンを追加。
- [x] T008 [App] ロール一覧右ペインのエンドポイント情報非表示化。`RoleDetails` に `showEndpoints` を追加し、一覧右ペインでは HTTP method/path を非表示（ロール専用詳細画面では表示を維持）。加えて内部名 `permission.name` の表示を廃し `permission.action` (`admin:xxx`) を表示、右ペイン幅を固定化。
- [x] T009 [App] アプリケーション一覧画面への編集ボタン追加。`AdminPaneActions` に `editHref` を追加。
- [x] T010 [App] クライアントシークレットローテーションUIの改善と未翻訳文言の対応。i18n未対応だった `ClientSecretRotationPanel` を辞書化し、OIDC 設定から独立した警告ブロックへ分離。事故防止のため「猶予期間選択 → 確認 → 実行」の 2 段階にし、1 クリック適用を廃止。
- [x] T011 [App] プロビジョニング一覧画面への案内文追加。個別設定は各アプリケーションの「プロビジョニング」タブで行う旨の案内文を追加。
- [x] T012 [App] サインインポリシー画面の「active user」日本語化。「有効なユーザー」に変更。
- [x] T013 [App] 同意一覧画面の「granted」日本語化。`ConsentStateBadge` が生の state 文字列 (`granted`/`revoked`/`expired`) をそのまま出していたのを辞書経由の表示ラベルに変更。
- [x] T014 [Verify] UIの変更をローカル環境で確認する。`just verify-ui`（format/lint/typecheck/unit test/build）と `just test-ui-e2e`（20件）がグリーン。ブラウザ拡張が未接続のため実ブラウザでの目視確認はユーザー側で実施。

## Verification
- `just dev-ui` 等で起動し、各画面の表示が改善されていることを目視確認する。
- `just verify-ui` でテストやLintが通ることを確認。

## Risk Notes
軽微なUI変更であるためリスクは低い。

## Post-review refinements (2026-07-22)
ユーザーによる実画面レビューを受けた追補。原則「参照画面では in-place 編集させない」「ボタンの大きさ・表記・アイコンを一貫させる」「用語を適切に」に沿って手直しした。

- **ダッシュボード**: 直近の監査イベントと推奨セキュリティ構成を横 2 列、Quota を下段全幅に再構成し、縦の間延び・スクロールを解消。ヒーローのセキュリティスコアと重複していた「推奨タスク 残り 2 件」（ハードコード）を削除。
- **参照画面での in-place 編集撤去**: ユーザー一覧右ペインのグループ追加、グループ一覧右ペインのメンバー追加・除外を非表示化（`allowEditing`）。編集は各専用詳細画面で行う。
- **`AdminPaneActions`**: 二次アクションを別サイズ・右寄せにする案を撤回し、ボタン総数でレイアウトを分岐（3 個以下＝1 行均等、4 個＝2×2 グリッド）。
- **ユーザー編集画面**: 入れ子 2 列で崩れていたレイアウトを、縦積みセクション＋氏名のみ 2 列・左寄せ・幅拡大へ再構成。
- **ロール権限表示**: 内部名 (`AdminAgentsManage` 等) を廃し RFC/ポリシー上のアクション (`admin:agents_manage` 等) を表示。
- **追加ボタン統一とアイコン整備**: 上記 T007。専用の追加アイコンがある users/groups は `IconUserPlus`/`IconUsersPlus`、他は `IconPlus` に統一。
- **ライフサイクルワークフロー一覧の i18n 化**: ハードコード日本語だった一覧ページを `AdminLifecycleWorkflowsPage.i18n.ts` による日英対応へ移行（状態/トリガー/アクション/実行結果ラベル含む）。**編集・作成フォーム画面 (`AdminLifecycleWorkflowEditorPage` / `WorkflowDefinitionForm`) は依然ハードコード日本語で、別作業として残置**（下記 Follow-up）。
- **認可詳細タイプ画面の用語整理**: JA を「OAuth2 認可詳細」、`type` フィールドを「タイプ」、追加ボタンを「認可詳細タイプを追加」に。EN は RFC 用語「Authorization Details」を維持。サイドバーが 2 行折り返す問題からナビラベルは短縮。
- **ユーザー属性追加ボタン**: 「属性を追加」→「ユーザー属性を追加」。
- **クライアントシークレットローテーション**: 確認ステップ導入（上記 T010）。

## Follow-up (別作業として残置)
- ライフサイクルワークフローの**編集・作成フォーム画面全体**が依然ハードコード日本語（列挙ラベルヘルパー含む ~120 文字列）。本 work item のスコープ外のため、専用の work item で i18n 化することを推奨。

## Completion
- **Completed At**: 2026-07-22
- **Summary**:
  Scope に列挙した各画面のUI表記統一とレイアウト改善を行い (T001–T013)、実画面レビューに基づく追補（上記 Post-review refinements）まで反映した。SCL の振る舞い変更を伴わない presentation 層のみの変更のため `spec/scl.yaml` は更新していない。
- **Verification Results**:
  - `just verify-ui` (format-check / lint / typecheck / unit test / build) - passed
  - `just test-ui-e2e` (4 spec, 20 tests) - passed（一度 stray な vite プロセスがポート 5173 を占有して e2e が不安定化したが、掃除後にクリーンな全 20 件グリーンを確認）
- **Out of Scope に残したもの**: ライフサイクルワークフローの編集/作成フォーム画面の i18n（上記 Follow-up）。
