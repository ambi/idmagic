---
id: idp-wi-75-ui-scenario-e2e-coverage
title: "追加した UI シナリオ (管理 / マイアカウント / ログイン補助) の E2E テストを整備する"
created_at: 2026-06-27
authors: ["tn"]
status: completed
risk: low
---
# Motivation
spec/scl.yaml の `user_experience` と `scenarios` を最新 UI コードへ追従させ、
管理画面 13 / マイアカウント 10 / ログイン補助 2 画面と、対応する主要シナリオ
(アプリケーション CRUD + protocol binding、エージェント、監査ログ、署名鍵回転、
属性スキーマ、設定、ダッシュボード集計、プロフィール / パスワード変更 / メール変更 /
TOTP 登録・解除の step-up / セッション管理 / サインイン履歴 / 同意撤回 /
データエクスポート、パスワードリセット、TOTP ログイン) を追加した。

しかしこれらのシナリオは spec レベルの記述で、ブラウザ E2E はほぼ存在しない。
現状の Playwright は ui/tests/e2e/authorize-golden-path.spec.ts の 1 本のみで、
認可コードフローの黄金パスしか覆っていない。SCL coherence (internal/spec) は
参照整合の構造検査だけで、シナリオを実行する harness は無い。

バックエンド側は HTTP handler / usecase の単体・結合テストでフローの大半を
カバーしている (password_reset / change_password / totp / sessions /
signin_activity / account_mfa / account_step_up / account_consents /
account_profile / admin_agents / admin_groups / admin_user_attributes /
admin_consent / admin_key / rotate_signing_key / admin_settings /
application_handler 等)。本 WI は不足している「ブラウザ越しの画面シナリオ」を
Playwright E2E として整備し、追加シナリオに実行可能な裏付けを与える。

# Scope
- **decision**: E2E の編成方針 (画面群ごとの spec 分割、fixture でのテナント/ユーザ/ メール確認リンクの用意、TOTP コード生成の扱い、CSRF cookie の取り回し) を ADR か README で軽く確定する。新規 ADR が要るほどでなければ ui/tests/e2e/README にとどめる。
- **scl**: 必要なら assurance/evidence に「UI シナリオは Playwright E2E が裏付ける」 旨の obligation を追加し、evidence.covers で対象 scenario を束ねる (TestAssuranceEvidenceHasExecutableBindings の対象に載せる)。spec 本文の scenario 追加は wi 完了済みのため本 WI では変更しない想定。
- **ui**: ui/tests/e2e に画面群ごとの spec を追加: (1) ログイン補助 — forgot/reset password、TOTP ログイン成功・失敗、 (2) マイアカウント — profile 更新、password 変更、email 変更+確認、 TOTP 登録、TOTP 解除の step-up、session 一覧・失効、signin activity、 接続済みアプリ撤回、data export、 (3) 管理 — application 作成+binding+割当+削除、agent 登録+資格情報、 audit ログ絞り込み+export、署名鍵回転、属性スキーマ、設定更新、 ダッシュボード集計表示。, fixtures.ts を拡張し、テスト用テナント・admin / end-user・メール確認/ リセットリンク取得 (dev のメール sink 経由) ・TOTP secret からのコード 生成ヘルパを共通化する。

# Out of Scope
- バックエンド HTTP / usecase テストの追加 (大半は既存。不足が判明したら 各機能の WI で対応する)。
- advanced 面 (AdminClients = /admin/clients、AdminWsFedRelyingParties = /admin/wsfed/relying-parties) の E2E。サイドバー導線が無く URL 直叩きの 低レベル画面のため初期スコープ外。
- AdminRoles / WS-Fed RP CRUD の interface 未宣言 drift の是正 (spec interface 層の別課題。本 WI はテスト整備に限定)。
- 視覚回帰 (スクリーンショット差分) テスト。

# Verification
- [object Object]
- [object Object]
- [object Object]
- [object Object]
- [object Object]

# Risk Notes
既存挙動を変えずテストを足す WI のため本番リスクは低い。難所は環境依存:
メール確認/リセットリンクの取得 (dev のメール sink への依存)、TOTP コードの
時刻同期、CSRF cookie とセッションの取り回し。これらを fixtures.ts に集約し、
flaky を避けるため待機は明示的な状態 (DOM / network) に紐付ける。
E2E はネットワーク I/O を伴うため CI 実行時間の増加に注意する。

# Completion
- **Completed At**: 2026-06-30
- **Summary**:
  Bun.WebView ベースの UI E2E fixture を整備し、Go サーバ、Vite、
  callback listener、ローカル SMTP sink をテスト内で起動する構成にした。
  メール変更とパスワードリセットは外部送信せず、SMTP sink で確認リンクを
  捕捉して検証する。ログイン補助、マイアカウント、管理画面の主要ブラウザ
  シナリオを `ui-scenario-smoke.spec.ts` と `ui-scenario-actions.spec.ts`
  に追加した。
- **Verification Results**:
  - [object Object]
  - [object Object]
  - [object Object]
  - [object Object]
