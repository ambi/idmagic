# Saml Decisions

- SP と IdP プロファイルの登録・参照・削除は `AdminFederationTrustsManage` 権限 (AuthZEN action `admin:federation_trusts_manage`) を要し、`admin` ロールを持つ、有効かつ認証済みのユーザーが所属テナントに対して行える。
- プロトコルのエンドポイントは管理者の認可を通らない。IdP メタデータと証明書の取得は認証を要さない公開 Discovery であり、公開するのは entityID、エンドポイント、署名証明書に限る。SSO と SLO はブラウザーのログインセッションで主体を決め、SP の entityID、`AssertionConsumerServiceURL`、Destination、対象ユーザーの Application 割り当てをすべて検証してから発行する。1 つでも一致しなければ SAMLResponse を発行しない。
- テナントとプロファイルはどちらも信頼境界である。SSO と SLO では、リクエスト先のルートが指すプロファイルと、対象 SP に割り当てられたプロファイルが一致することを確認する。ある信頼境界に対する正当なリクエストを、同じテナントの別のプロファイルへ送り直しても通らない。
- 対応範囲を Web Browser SSO Profile に限り、ECP、暗号化 Assertion、SAML SP としての動作は含めない。対応するバインディングと形式を減らすほど、SAML で知られた署名ラッピング攻撃への露出が小さくなるからである。
- SAML IdP プロファイルは共有可能な 1 つのモデルとし、専用プロファイルを別の型にしない。`dedicated` は同じモデルに SP を 1 つだけ関連付けた状態として表す。信頼境界の規則をプロトコル、永続化、管理のすべての経路で 1 つに保つためである。
- XML の構文解析、正規化、署名は自作せず、検証済みの第三者製ライブラリに委ねる。
- この Context を `Generic` に分類する。SAML 2.0 への準拠が価値のすべてであり、独自の語彙やモデルを足す理由が無い。
