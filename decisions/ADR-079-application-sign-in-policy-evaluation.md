---
status: accepted
authors: [tn]
created_at: 2026-07-04
---

# ADR-079: Application sign-in policy evaluation

## コンテキスト
idmagic は Application への割当を protocol binding ごとに fail-closed で確認するが、アプリケーションごとの認証強度や再認証条件は持っていない。高リスクな業務アプリに MFA や短い再認証間隔を求めるには、OIDC、SAML、WS-Fed の各開始経路で同じ条件を評価する必要がある。

初期実装 (wi-71) では要求 ACR / factor を自由文字列で入力させ、network / device 条件も自由文字列の入力点だけ保持して評価時に fail-closed 拒否していた。これは「設定できるのに実際には評価されない」見せかけの入力欄になり、管理者の意図と実際の評価が食い違う。また "sign-on policy" は Okta / Ping レガシー寄りの語で、アプリ内の支配的語彙「ログイン / サインイン」とずれていた。

## 決定

Application context の `models.AppSignInPolicy`、`models.RequiredAuthnStrength`、`interfaces.GetAppSignInPolicy`、`interfaces.UpdateAppSignInPolicy`、`invariants.AppPolicyFailClosed`、`invariants.AppPolicyEvaluatedAcrossProtocols` に反映。wi-114 で「サインオンポリシー」から「サインインポリシー」へ改称し (UI 文言から DB テーブル名まで `SignOn` → `SignIn` へ統一)、条件を実際に評価できる構造化条件へ制約した。

ApplicationCatalog が `AppSignInPolicy` を tenant/application 単位で所有し、`SignInRule` の順序付き
集合として保存する。要求認証強度は自由文字列ではなく制約 enum `RequiredAuthnStrength`
(`Password` / `Mfa`) とし、アクセス条件は実際に評価できるもの (`reauth_max_age_seconds` の
recency 評価、`network_allow_cidrs` の CIDR 突き合わせ) だけを残す。旧 `network` / `device` の
自由文字列入力は、設定できるのに実際には評価されない見せかけの項目だったため廃止した ——
free-text のまま残す代替案は fail-open/fail-closed いずれでも管理者の意図とズレるため却下し、
各 protocol context が独自に policy を持つ代替案は評価条件や失敗挙動が分岐し迂回を招くため
却下した。評価は既存の Application 割当ゲートと同じ federation 開始経路で fail-closed に行う。

評価点の配置、OIDC/SAML/WS-Fed 各経路での挙動、CIDR 未一致時の扱いの詳細は
[`backend/application/ARCHITECTURE.md`](../backend/application/ARCHITECTURE.md) に置く。

## 却下した代替案
- 名称・識別子を "sign-on policy" のまま温存する: Okta / Ping レガシー寄りの語で、アプリの支配的語彙「ログイン / サインイン」とずれ、管理者の理解負荷が残る。
- network / device を自由文字列の入力点として残す: 評価できない条件を設定できてしまい、fail-open か fail-closed のどちらでも管理者の意図とズレる。
- 要求強度を自由文字列 acr / factor のまま入力させる: 実在する acr は pwd / mfa の 2 値のみで factor と冗長。制約されない入力は誤設定を招く。
- 各 protocol context が独自に policy を持つ: 条件や失敗挙動が分岐し、迂回や設定不整合が起きやすい。
- OAuth2 client / SAML SP / WS-Fed RP に直接 policy を埋め込む: 管理者が扱う単位である Application とズレ、複数 binding を持つアプリで一貫性を保ちにくい。

## 影響
- `application_sign_in_policies` テーブルを使用し、tenant/application 境界で保存する。既存の `application_sign_on_policies` の内容は移行する。
- 管理 API と UI に Application 配下の sign-in policy 編集面を持ち、要求強度は選択肢 (パスワードのみ / MFA 必須)、network は許可 CIDR リストの入力とする。
- federation 開始時の Application gate は割当結果だけでなく policy 評価結果を返し、クライアント IP を入力に含める。
