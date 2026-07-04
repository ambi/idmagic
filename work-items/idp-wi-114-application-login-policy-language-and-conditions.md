---
id: idp-wi-114-application-login-policy-language-and-conditions
title: "アプリケーションのサインオンポリシーをログインポリシーとして再設計する"
created_at: 2026-07-04
authors: [tn]
status: pending
risk: high
---

# Motivation
現在の「サインオンポリシー」は、他の画面で使っている「ログイン」語彙とずれており、管理者に余計な理解負荷を与えている。
さらに ACR、factor、Password、MFA、再認証最大経過秒数、ネットワーク条件、デバイス条件が UI 上で何を意味するのか分かりづらい。
network / device に任意文字列を入力できる形は、実際に評価できる条件なのか、将来の入力点なのかが曖昧で、ポリシーがあるように見えて実効性が不明になる。
管理者が自然な語彙と制約された選択肢で、実際に評価されるログイン要件を設定できるようにする。

# Scope
- `spec/contexts/application.yaml`
  - `AppSignOnPolicy` / `SignOnRule` / `RequiredAuthnLevel` / `AccessCondition` の用語を見直し、UI と API の表示名を「ログインポリシー」へ寄せる。
  - ACR や factor をそのまま入力させる契約ではなく、管理者向けの選択肢に写像する value object / enum を定義する。
  - Password / MFA / step-up / 再認証 max age の意味を SCL 上で明確にし、既存 AuthenticationContext との対応を記述する。
  - network / device 条件を、初期実装で評価できる構造化条件に限定する。未実装の条件は UI で入力できないか、明示的な disabled / future として扱う。
- ADR
  - 必要なら ADR-079 を更新または新規 ADR を作成し、ログインポリシーの語彙、管理者向け選択肢、内部 ACR/AMR への写像、未評価条件の扱いを決める。
- Go / HTTP
  - 既存の sign-on policy API との後方互換または移行方針を決め、評価器が構造化条件だけを fail-closed に評価するようにする。
- UI
  - 管理者向けの表示文言を ACR/factor などの内部語彙から切り離し、「パスワードのみ」「MFA 必須」「再認証を要求」など理解できる表現にする。
  - ネットワーク/デバイス条件は自由入力ではなく、実装済み条件の選択 UI にする。

# Out of Scope
- 全アプリケーションにまたがる既定ログインポリシーの導入。
- リスクスコアリング、UEBA、MDM/デバイス証明などの高度な条件評価エンジン。
- 新しい認証要素そのものの追加。

# Verification
- `just yaml-check-scl`
- `just verify-go`
- `just verify-ui`
- 手動: 既存ポリシーを読み込み、管理者向け語彙で同じ意味に表示・保存できること。
- 手動: 未実装の network / device 条件を任意文字列で保存できないこと。
- 手動: MFA 必須、再認証要求、条件不一致時の拒否または step-up が既存の federation 開始経路で一貫して評価されること。

# Risk Notes
ポリシー語彙を変えるだけに見えて、内部の ACR/AMR 判定や fail-closed 評価に直結するためリスクは高い。
既存データの移行、API 互換、UI 表示名、監査イベント名のどれかがずれると、管理者が意図した強度と実際の評価が食い違う。
