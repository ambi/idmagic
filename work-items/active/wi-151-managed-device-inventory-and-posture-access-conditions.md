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
  - { path: docs/modules/application/features/sign-in-policy.md, requirement: REQ-APPLICATION-009 }
  - { path: docs/modules/authentication/features/trusted-device.md, requirement: REQ-AUTHENTICATION-026 }
---

# 管理された端末の台帳を持ち、端末の状態をアプリのサインインポリシーの条件にする

## 動機

アプリのサインインポリシーは要求元の CIDR と再認証の経過時間を評価できるが、端末が管理されているかを条件にできない。
Microsoft Entra の準拠済みデバイス、Google の Context-Aware Access、Okta の device assurance のように、企業向けの IdP では、管理された端末、OS、証明書、紛失や侵害の状態をアクセスの判断に使う需要が高い。
記憶済みの端末（[REQ-AUTHENTICATION-026](../../docs/modules/authentication/features/trusted-device.md)）は MFA を省略するための本人の端末の記憶であり、管理者が承認した端末の台帳ではない。

## 対象範囲

- テナントの端末の台帳を持つ。端末と User の対応、承認、拒否、失効、状態の更新をドメインイベントとして記録する。
- サインインポリシーの規則に、管理された端末、承認された端末、端末の状態の条件を加える。
- アカウントポータルで本人の端末を一覧して失効させ、管理コンソールで端末を検索、承認、拒否できるようにする。

## 対象外

- 特定の MDM 製品との本番の連携と、端末のエージェントの配布、OS の水準のアテステーション。
- 記憶済みの端末による MFA の省略。既存の機能のまま変えない。
- 端末の指紋による追跡。

## 設計

着手時に、端末を識別する手段（クライアント証明書、MDM が発行する識別子など）と、状態をだれが更新するか（管理者、外部の MDM からの取り込み）を決める。
識別の手段が決まらなければ、条件だけを加えても評価できないので、この判断を実装の前に済ませる。

## 検証

- 端末の状態ごとに、サインインポリシーの判定が変わることを固定するテスト。
- 承認していない端末からのサインインが、条件を持つアプリで拒否されることを観測する受け入れのテスト。
- `mise run verify`

## リスク

端末の識別子を本人が偽れる手段を選ぶと、条件は防御にならない。
識別の手段の判断で、偽れる場合の扱いを明記する。
