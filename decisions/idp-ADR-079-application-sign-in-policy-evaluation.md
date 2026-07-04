# idp-ADR-079: Application sign-in policy evaluation

## ステータス
採用。Application context の `models.AppSignInPolicy`、`models.RequiredAuthnStrength`、`interfaces.GetAppSignInPolicy`、`interfaces.UpdateAppSignInPolicy`、`invariants.AppPolicyFailClosed`、`invariants.AppPolicyEvaluatedAcrossProtocols` に反映。idp-wi-114 で「サインオンポリシー」から「サインインポリシー」へ改称し、条件を実際に評価できる構造化条件へ制約した。

## コンテキスト
idmagic は Application への割当を protocol binding ごとに fail-closed で確認するが、アプリケーションごとの認証強度や再認証条件は持っていない。高リスクな業務アプリに MFA や短い再認証間隔を求めるには、OIDC、SAML、WS-Fed の各開始経路で同じ条件を評価する必要がある。

初期実装 (idp-wi-71) では要求 ACR / factor を自由文字列で入力させ、network / device 条件も自由文字列の入力点だけ保持して評価時に fail-closed 拒否していた。これは「設定できるのに実際には評価されない」見せかけの入力欄になり、管理者の意図と実際の評価が食い違う。また "sign-on policy" は Okta / Ping レガシー寄りの語で、アプリ内の支配的語彙「ログイン / サインイン」とずれていた。

## 決定
1. ApplicationCatalog が `AppSignInPolicy` を tenant/application 単位で所有する。「サインインポリシー」に改称し、UI 文言・API description に加えて内部識別子 (SCL entity / interface / DTO 名、HTTP パス `/sign-in-policy`、DB テーブル `application_sign_in_policies`、監査イベント `AppSignInPolicyUpdated`) まで `SignOn` → `SignIn` に統一する。
2. policy は `SignInRule` の順序付き集合として保存する。要求認証強度は自由文字列ではなく制約 enum `RequiredAuthnStrength` (`Password` / `Mfa`) とし、内部の acr URN (`urn:idmagic:acr:pwd` / `urn:idmagic:acr:mfa`) と amr へ 1:1 で写像する。`Password` は追加要求なし、`Mfa` は第二要素必須。
3. 静的アクセス条件は初期実装で実際に評価できる構造化条件のみとする。`reauth_max_age_seconds` は認証または step-up の recency として評価し、`network_allow_cidrs` はクライアント IP を許可 CIDR に突き合わせて評価する。CIDR の妥当性は保存時に検証する。
4. 旧 `network` / `device` の自由入力は廃止する。device 条件の実体 (MDM / attestation) は将来 WI とし、入力点も設けない。
5. 評価点は既存の Application 割当ゲートと同じ federation 開始経路に置き、token / assertion 発行前に必ず通す。クライアント IP も全経路 (OIDC / SAML / WS-Fed) で評価器に渡す。
6. 評価は fail-closed とする。要求認証強度を step-up で満たせる不足は既存の step-up 導線へ誘導する (OIDC)。SAML / WS-Fed は初期実装では protocol transaction を停止し、明確な拒否理由を返す。許可 CIDR 非空でクライアント IP が一致しない、または IP を取得できない場合は step-up ではなく拒否する。

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
