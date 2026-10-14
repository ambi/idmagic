---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# アプリケーション管理画面で一覧と詳細の情報量を揃える

アプリケーション管理 UI の一覧（右ペイン）と詳細画面で表示する情報の情報密度を向上。
- SCL spec (spec/contexts/application.yaml) に category_names、binding_summaries、assigned_subject_count、sign_in_policy_summary を追加。
- Go バックエンド (admin_application_handler.go) にて、一覧 API では DB 一括ロードを用いて N+1 クエリを回避し、詳細 API および各更新 API では共通の集約ビルダーを用いて追加情報を収集するよう実装。
- UI (AdminApplicationsPage.tsx) にて、プレビューペインおよび詳細画面の表示を拡充。カテゴリ名、プロトコル binding 要約、割当数、適用中のログインポリシー概要、登録/更新日時を整理して表示。未設定項目には代替テキストを表示。

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。
