---
status: pending
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-11-14
change_kind: bugfix
priority: p1
depends_on: []
affected_spec:
  - { path: docs/modules/oauth2/features/authorization.md, requirement: REQ-OAUTH2-005 }
  - { path: docs/modules/authentication/features/trusted-device.md, requirement: REQ-AUTHENTICATION-026 }
---

# 再認可が求めた MFA の水準を、記憶済みの端末で省略せず満たすまで発行しない

## 動機

RFC 9470 の再認可では、リソースサーバーが `acr_values` と `max_age` で認証の水準と新しさを求める。
IdMagic の `/authorize` には、その水準を守り切らない欠陥が三つある（2026-11-14 のコード読解）。

1. 信頼済みデバイス（`tdev`）は `acr` を `urn:idmagic:acr:mfa` に上げるので、`acr_values` で MFA を求めた再認可も、パスワードだけで第二要素を省いたまま MFA の水準のトークンを発行する。
2. `acr_values` は `/authorize` の時点で既存のセッションと比べるだけであり、`max_age` を過ぎてログインし直した後は、要求した水準を満たしたかを確かめない。サインインポリシーがパスワードだけを求めるテナントでは、`acr=pwd` のトークンを発行する。
3. 既存のセッションに追加の認証を求める条件が「TOTP を登録している」であり、WebAuthn や復旧コードだけを登録した利用者はログインからのやり直しに回される。第二要素の画面自体は、それらの方式を扱える。

どれも、リソースサーバーが同じ要求を拒否し続け、再認可が往復する原因になる。
[[wi-468-system-console-privileged-session-assurance]] の特権操作の再認証は、この往復を避けられない。

## 対象範囲

- MFA の水準を求める再認証（`acr_values` が MFA を求め、`max_age` を過ぎているか `prompt=login`）では、サインインポリシーの「毎回 MFA」と同じく、信頼済みデバイスを第二要素の充足と認めない。
- ログインを終えた後も、要求した `acr_values` の水準を満たすまで第二要素の手順へ進める。
- 既存のセッションに追加の認証を求める条件を、登録済みの第二要素のいずれかがあることに広げる。
- 登録済みの第二要素がなく水準を満たせない場合は、`unmet_authentication_requirements` を RP へ返し、トークンを発行しない。
- 通常のログインでの信頼済みデバイスの扱い（サインインポリシーの `allow_trusted_device`）は変えない。

## 対象外

- `acr` の語彙を増やすこと、アプリのポリシーと `acr_values` を結ぶこと。[[wi-63330-define-built-in-assurance-levels]] が扱う。
- `acr_values` を OIDC の `claims` パラメーターの必須の要求として扱うこと。

## 設計

主要な IdP の扱いを比べた（2026-11-14 に利用者と検討）。
Okta は既存のセッションで満たした認証器を考慮してから不足の要素を求め、満たせなければ `unmet_authentication_requirements` を返す。`max_age=0` で毎回の再認証を求められる。
Ping は要求された `acr_values` で認証ポリシーの経路を選び、記憶済みの端末を通すかどうかは管理者が経路で組む。
記憶済みの端末の省略をステップアップで止めるかは、どちらも管理者の設定に委ねている。
IdMagic には経路を組む仕組みがないので、MFA の水準を求める再認証では省略しないことを製品の規則にする。
設定によらず守る規則にしないと、リソースサーバーの判定と食い違い、再認可が往復する。

`unmet_authentication_requirements` は、OpenID Connect Core とは別の OpenID の小さな仕様がエラーコードとして定義している。適用の前に一次資料で定義と返し方を確かめる。

## 計画

1. 三つの欠陥を受け入れ境界で観測する。
2. 規範を更新し、`/authorize` の手順を直す。

## タスク

- [ ] T001 [Acceptance] 信頼済みデバイスを持つ利用者が `acr_values=urn:idmagic:acr:mfa` と `max_age` で再認可すると、第二要素なしで MFA の水準のトークンが出ることを観測し、RED にする。
- [ ] T002 [Acceptance] パスワードだけのポリシーのテナントで、ログインし直した後に `acr=pwd` のトークンが出ることと、WebAuthn だけの利用者がログインからやり直しになることを観測する。
- [ ] T003 [Spec] REQ-OAUTH2-005 と REQ-AUTHENTICATION-026 に、再認証での省略の禁止、水準を満たすまでの手順、満たせないときのエラーを書く。
- [ ] T004 [App] `/authorize` の手順を直す。
- [ ] T005 [Verify] 変更を検証する。

## 検証

- HTTP 境界の受け入れテストで、三つの欠陥がそれぞれ直ったことを観測する。
- 最終検査は `mise run verify-changed -- <着手時の基準リビジョン>` とする。

## リスク

MFA を求める RP で、これまで信頼済みデバイスで省略できていた再認証が、第二要素を求めるようになる。
通常のログインは変えないので、影響は `acr_values` と再認証を同時に求める要求に限られる。
