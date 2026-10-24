# Kerberos による無音サインイン

## 概要

HTTP Negotiate の Kerberos チケットを検証し、管理者が対応付けた既存 User のセッションを作る。
チケットの検証と対応付けは Authentication が、WS-Federation の要求と Assertion の発行は WsFederation が担当する。
LDAP 同期、NTLM、WS-Trust の `windowstransport`、Hybrid Join と Entra Seamless SSO は扱わない。

## モデル

運用管理者は、テナント ID、Kerberos realm、HTTP の SPN、keytab のファイル、TLS 終端証明書、社内 CIDR と、principal の名前から内部 User ID への対応を設定する。
同じ realm と SPN の組を複数のテナントに設定しない。
realm と principal の名前は、大文字と小文字を変換せずに完全一致で比較する。
メールアドレスやユーザー名からの自動リンクと、新しい User の作成は行わない。

利用開始には、AD 管理者がサービスアカウントと HTTP の SPN を登録し、そのサービス鍵を持つ keytab を API にデプロイする必要がある。
ドメイン参加 PC が AD の KDC からチケットを取得し、API はサービス鍵で検証する。
利用者情報の同期がなくても検証できるが、AD の無効化を自動で取り込む保証はない。
同期導入前は、管理者が idmagic の User または対応付けを無効化する。

対応付けは、管理者が同一人物と確認して登録するものであり、AD から同期した属性という保証を持たない。
設定ファイルの更新と再起動で追加、変更と削除を反映し、デプロイの監査にはファイルの SHA-256 を記録する。
ディレクトリー同期へ移管する際は、既存 User ID と外部の不変 ID を確認した明示的な関連付けが必要である。
認証処理は、チケットを検証する操作と、検証済み principal を対応付ける操作を分ける。

## 操作

### 利用者による Kerberos のログイン

#### REQ-AUTHENTICATION-042 検証した Kerberos principal に対応する利用者だけがログインできる

- 指定した信頼ファイルが不正か、鍵と証明書に不整合がある場合、Authentication は、API の起動を失敗させる。
- 設定した realm と SPN の有効な Kerberos チケットを受け、principal が同じテナントの Active User に明示的に対応付けられているとき、Authentication は、RFC 8176 の `wia` の AMR のセッションを作り、`NegotiateAuthSucceeded` を発行する。
- Kerberos のセッションを作るとき、Authentication は、既存のサインインのポリシーと必須操作を適用する。
- Kerberos による本人確認後に追加の要素が必要な状態の間、Authentication は、認証を保留して該当する画面へ進ませる。
- Kerberos による認証がポリシーを満たして完了したとき、Authentication は、`UserAuthenticated` を発行する。
- 復号、SPN、realm、TLS の channel binding、有効期間、5 分の許容時刻差の検証に失敗したか、NTLM のトークンを受けた場合、Authentication は、401 で拒否し、`NegotiateAuthRejected` を発行し、セッションを作らない。
- principal に対応がないか、対応する User が存在しないか、別テナントに属するか、Active でない場合、Authentication は、401 で拒否し、`NegotiateAuthRejected` を発行し、User と対応付けを作成も変更もしない。
- 同じサービスへの同じ principal と時刻とマイクロ秒の authenticator を再度受けた場合、Authentication は、プロセスと再起動をまたいでも拒否し、セッションを増やさない。
- 再送拒否の共有ストアへ到達できない場合、Authentication は、ログインを失敗させ、セッションを作らない。
- Kerberos による本人確認を完了したとき、Authentication は、MFA を充足したものとして扱わず、OAuth と SAML の既存の認可継続へ認証事実を渡す。
- ブラウザーのログイン API の Origin または CSRF の検証に失敗した場合、Authentication は、既存のブラウザー要求の検証と同じ 403 で拒否し、チケットを検証せず、セッションを作らない。
- Negotiate のログイン要求がログイン用のレート制限を超えたか、共有カウンターに到達できない場合、Authentication は、429 と `Retry-After` で拒否し、チケットを検証せず、セッションを作らない。

#### REQ-AUTHENTICATION-043 任意の Kerberos 認証はチャレンジを繰り返さず通常ログインへ戻せる

- 設定済みテナントの HTTPS のログイン画面を開いたとき、Authentication は、通常ログインに加えて Windows のサインインを選択させる。
- 利用者が Windows のサインインを選択したか、管理者が自動試行を有効にした出口の公開 IP からログインを求めたとき、Authentication は、401 と `WWW-Authenticate: Negotiate` を一度提示し、5 分間有効な署名付きの試行 Cookie を発行する。
- チャレンジを提示するとき、Authentication は、試行 Cookie をテナントと継続先へ束縛し、Secure、HttpOnly、SameSite=Lax と `Cache-Control: no-store` を設定する。
- 同じ試行で資格情報が送られないか、Negotiate 以外の方式だけが送られた場合、Authentication は、チャレンジを繰り返さず、通常のログインへ戻す。
- 設定がないか、HTTPS の発行者でない状態の間、Authentication は、Negotiate を提示せず、通常のログインを使わせる。
- Kerberos の利用可否を取得している間、Authentication は、通常のログインフォームを利用可能に保つ。
- 自動試行が無効か、アクセス元が登録済みの出口の公開 IP でない状態の間、Authentication は、利用者が選ぶ前に Negotiate を提示せず、通常のログインを表示する。
- 試行の署名が不正か、期限を過ぎているか、テナントか継続先が異なる Cookie を添えた資格情報を受けた場合、Authentication は、401 で拒否し、セッションを作らない。
- 自動試行のアクセス元を判定するとき、Authentication は、構成した信頼プロキシ数に従う既存のクライアント IP の解決を使い、利用者名を示すプロキシヘッダーは認証に使わない。
- 利用者が通常ログインを選択したとき、Authentication は、そのログイン試行の自動 Negotiate を省略し、フォームと別の IdP でのログインを継続できるようにする。

## セキュリティ上の考慮

TLS の終端証明書を設定し、`tls-server-end-point` の channel binding を必須とする。
出口の公開 IP は自動試行の条件だけに使い、本人確認やドメイン参加の証拠として扱わない。
ブラウザーが許可するサービス名を GPO などで設定する必要があり、非対応の端末にチャレンジを送ると OS の資格情報ダイアログが出る場合がある。
keytab のサービス鍵を持つ者はチケットを偽造できるため、読み取り権限を API プロセスだけに限り、ログと管理 API に内容を出さない。
AD アカウントの削除と名前の再利用の前に対応付けを失効させる。

## 実現方式

導入と鍵の更新は [Kerberos の信頼設定と導入確認](../../../runbooks/kerberos-setup.md)に従う。

チケット、SPNEGO、相互認証の応答と channel binding は `github.com/otuschhoff/gokrb5/v8` に委ねる。
要求ごとに keytab を読み直し、旧新 KVNO を持つファイルへ原子的に置き換えてローテーションする。
ライブラリのプロセス内の再送拒否に加え、認証応答の共有ストアへ `krb5:` の名前空間のキーを 10 分間予約する。
許容時刻差の両端を含む再送を、複数の API プロセスで拒否するためである。
