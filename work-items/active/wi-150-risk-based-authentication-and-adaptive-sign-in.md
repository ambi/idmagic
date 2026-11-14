---
depends_on: []
status: pending
authors: ["tn"]
risk: high
created_at: 2026-07-10
priority: p3
group: authentication
change_kind: feature
affected_spec:
  - { path: docs/modules/authentication/features/sign-in.md, requirement: REQ-AUTHENTICATION-007 }
  - { path: docs/modules/application/features/sign-in-policy.md, requirement: REQ-APPLICATION-009 }
---

# 認証イベントからリスクを算出し、アプリのサインインポリシーで MFA の要求とブロックに使う

## 動機

認証イベントの TypeSpec のモデルは `countryCode` と `riskScore` の項目を持つが、`riskScore` を算出する処理も、それを使う認証の制御もない。
アプリのサインインポリシーが評価できる条件は、要求元の CIDR、認証の強度、再認証の経過時間、記憶済みの端末の許可である（[REQ-APPLICATION-009](../../docs/modules/application/features/sign-in-policy.md)）。
匿名の IP、国の急な変化、パスワードスプレー、見たことのない端末といった信号を判断に使えないと、管理者は MFA を一律に求めるか、危険なサインインも通すかの二択になる。
Microsoft Entra ID Protection と Okta の adaptive policy は、こうした信号から MFA の要求、ブロック、パスワードの変更の要求を選ぶ。

## 対象範囲

- 既存の認証イベントから、規則に基づいてリスクの信号と水準を算出し、認証イベントの `riskScore` に保存する。
- サインインポリシーの規則に、リスクの水準の条件と、満たしたときの処置（MFA の要求、ブロック、パスワードの変更の要求）を加える。
- ログイン、認可、ステップアップの経路で評価し、評価できないときは fail-closed にする。
- 判定をドメインイベントとして記録し、管理者がリスクの高いサインインを一覧できるようにする。

## 対象外

- 機械学習や UEBA による異常検知。
- 外部の脅威情報、Tor の出口の一覧、商用の IP の評判との連携。
- 管理された端末の状態の条件。wi-151 が扱う。
- アカウントの自動の復旧。

## 設計

着手時に、信号の一覧と算出の規則、水準の段階、処置の優先順位、偽陽性で利用者を締め出したときの復旧の経路を決める。
算出は認証イベントの列を入力とするデータ上の計算に保ち、評価の時刻と過去のイベントの読み出しはユースケースが渡す。

## 検証

- 信号ごとの算出と、水準ごとの処置を固定するテスト。
- ログイン、認可、ステップアップの各経路で、リスクの高いサインインが処置されることを観測する受け入れのテスト。
- `mise run verify`

## リスク

偽陽性は正当な利用者を締め出す。
処置の既定はブロックではなく MFA の要求にし、ブロックは管理者が明示したときだけ使う。
