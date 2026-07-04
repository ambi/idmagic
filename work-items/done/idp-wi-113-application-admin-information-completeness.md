---
id: idp-wi-113-application-admin-information-completeness
title: "アプリケーション管理画面で一覧と詳細の情報量を揃える"
created_at: 2026-07-04
authors: [tn]
status: completed
risk: medium
---

# Motivation
アプリケーション管理 UI は、一覧画面の右側詳細が起動 URL だけに近く、詳細画面もカテゴリやポリシーなど Application が持つ情報を十分に表示していない。
管理者は Application の状態、種別、カテゴリ、binding、割当、ポリシー、更新日時を見比べながら設定確認や障害調査を行うため、既存の表示では画面の余白に対して判断材料が不足している。
一覧と詳細の情報密度を上げ、詳細画面では「詳細」と呼べるだけの情報を漏れなく確認できるようにする。

# Scope
- `spec/contexts/application.yaml`
  - `AdminApplicationResponse` / `AdminApplicationDetailResponse` にカテゴリ名、binding 要約、割当数、ログインポリシー要約など UI 表示に必要な情報が足りるか確認し、不足があれば SCL-first で追加する。
  - `ListAdminApplications` / `GetAdminApplication` の返却契約を、一覧ペインと詳細画面の表示要件に合わせる。
- `internal/application` と管理 API
  - 一覧と詳細に必要なカテゴリ、binding、割当、ポリシー情報を tenant 境界内で集約して返す。
- `ui/src/features/admin-applications`
  - 一覧画面の右側ペインに、種別、状態、カテゴリ、binding、割当状況、ログインポリシー概要、作成/更新日時を表示する。
  - アプリケーション詳細画面に、Application 本体、カテゴリ、binding 実設定、割当、ログインポリシー、起動 URL、監査上必要な日時を整理して表示する。
  - 情報が未設定の場合は空欄ではなく、未設定であることが分かる表示にする。

# Out of Scope
- ログインポリシーの概念名や条件モデルの再設計。
- 全アプリケーションにまたがる既定ポリシーの導入。
- Application の新しい業務機能や protocol binding 種別の追加。

# Verification
- `just yaml-check-scl`
- `just verify-go`
- `just verify-ui`
- 手動: 複数カテゴリ、複数 binding、割当、ログインポリシーを持つアプリを作成し、一覧右ペインと詳細画面で同じ判断材料が確認できること。
- 手動: 未設定項目を持つアプリで、未設定表示が破綻せず、既存の編集導線に遷移できること。

# Risk Notes
一覧 API に集約情報を増やす場合、N+1 クエリや tenant 境界漏れが起きやすい。
表示専用の要約と編集用の詳細契約を混同すると UI が古い状態を編集してしまうため、返却契約と更新契約を分けて検証する。

# Completion
- **Completed at**: 2026-07-05
- **Summary**: アプリケーション管理 UI の一覧（右ペイン）と詳細画面で表示する情報の情報密度を向上。
  - SCL spec (spec/contexts/application.yaml) に category_names、binding_summaries、assigned_subject_count、sign_in_policy_summary を追加。
  - Go バックエンド (admin_application_handler.go) にて、一覧 API では DB 一括ロードを用いて N+1 クエリを回避し、詳細 API および各更新 API では共通の集約ビルダーを用いて追加情報を収集するよう実装。
  - UI (AdminApplicationsPage.tsx) にて、プレビューペインおよび詳細画面の表示を拡充。カテゴリ名、プロトコル binding 要約、割当数、適用中のログインポリシー概要、登録/更新日時を整理して表示。未設定項目には代替テキストを表示。
