---
status: pending
authors: ["tn"]
risk: medium
created_at: 2026-07-16
depends_on: [wi-218-lifecycle-workflow-action-execution-and-audit, wi-6-real-email-sender-adapter]
---

# lifecycle workflow の通知テンプレートをテナントでカスタマイズ可能にする

## Motivation
現在の `send_email` action は固定の `template_key` + locale fallback のみを使い
(`spec/contexts/identity-management.yaml` L1642-1645)、テナントごとの件名・本文・変数のカスタマイズ
ができない。Okta / Entra ID / midPoint はいずれもテナント (組織) ごとに通知文面を編集できる仕組みを
持ち、実運用では「退職時の通知文面を自社の言い回しに合わせたい」「入社時の歓迎メールに社内ポータルの
リンクを含めたい」といった要求が高頻度で発生する。固定テンプレートのままでは実運用に耐えない。

## Scope
- `spec/contexts/identity-management.yaml` (または既存の通知/email を扱う context) に
  `NotificationTemplate` (tenant-scoped、locale 別の件名・本文、許可された変数の集合) を追加する。
- `send_email` action の `template_key` を、テナント定義の `NotificationTemplate` または製品固定
  template のいずれかを参照できるようにする。
- 変数展開は許可された placeholder の集合 (例: user 表示名、workflow 名、テナント名) に限定し、
  任意の user 属性展開や HTML/script 注入を許さない sanitization/validation を設計する。
- 管理 UI にテンプレート編集・プレビュー画面を追加する。

## Out of Scope
- 自由記述の任意 HTML エディタやリッチテキスト全機能 (WYSIWYG)。初期はプレーンテキストまたは
  限定的な placeholder 付きテンプレートに留める。
- SMS/push 等、email 以外の通知チャネル。

## Plan
- 変数展開はテンプレートエンジンを新規導入せず、許可済み placeholder の文字列置換に限定し、任意
  コード実行や SSTI (Server-Side Template Injection) のリスクを避ける。
- 既存の固定 `template_key` はデフォルトテンプレートとして残し、テナントが未設定の場合の fallback に
  する。

## Tasks
- [ ] T001 [SCL] `NotificationTemplate` モデル、authorization、scenarios を追加する。
- [ ] T002 [Decision] placeholder 許可リストと sanitization 方針を ADR に記録する。
- [ ] T003 [App] テンプレート管理 usecase と `send_email` action の参照解決を実装する。
- [ ] T004 [UI] テンプレート編集・プレビュー画面を追加する。
- [ ] T005 [Verify] injection 耐性・fallback・locale 切り替えを検証する。

## Verification
- `just yaml-check`
- `just test-go`
- `just verify-ui`
- 自動: 許可外の placeholder や HTML/script を含むテンプレート保存が拒否される。
- 手動: テナント管理者がテンプレートを編集し、実際の workflow 実行メールに反映されることを確認する。

## Risk Notes
テンプレートのフリーフォーム編集はメール本文へのインジェクション (HTML injection、ヘッダ
インジェクション) リスクを伴うため、保存時・送信時の両方で sanitization と placeholder 検証を
fail-closed に行う。
