# Feature: テナントの解決のシナリオ

## 結果

### Rule: REQ-TENANCY-006 path style のテナントは realm prefix から解決される

Primary actor: `OAuth2Client`

#### Example: EX-TENANCY-006-01 通常経路

- Given テナント "default" の endpoint_style は Path である
- When "/realms/default/authorize" にリクエストを送る
- Then 解決されたテナントは "default"
- Then iss claim はベースURL + /realms/default

#### Example: EX-TENANCY-006-02 対象テナントが無効化されている

- Given テナント "default" の endpoint_style は Path である
- When "/realms/default/authorize" にリクエストを送る
- But 対象テナントが無効化されている
- Then  tenant_id "acme" を作成して無効化する
- And  無効化済みテナントの "/realms/acme/authorize" にリクエストを送る
- And  テナントの存在を漏らさずエラー "InvalidRequestError"

#### Example: EX-TENANCY-006-03 realm prefix を持たない "/authorize" にリクエストを送る

- Given テナント "default" の endpoint_style は Path である
- When "/realms/default/authorize" にリクエストを送る
- But realm prefix を持たない "/authorize" にリクエストを送る
- Then テナントは解決されず 404 tenant_not_found になる
- And 任意のリクエストが default テナントへ落ちることはない

### Rule: REQ-TENANCY-007 subdomain style のテナントは Host から解決される

Primary actor: `EndUser`

#### Example: EX-TENANCY-007-01 通常経路

- Given `tenant_base_domain` が設定されている
- And テナント "acme" の endpoint_style は Subdomain である
- When Host "acme.{tenant_base_domain}" の "/authorize" にリクエストを送る
- Then 解決されたテナントは "acme" で、その branding のログイン画面が表示される
- Then セッション cookie は __Host- prefix と Path=/ を持ち Domain 属性を持たない
- Then WebAuthn RP ID は "acme.{tenant_base_domain}" である

### Rule: REQ-TENANCY-010 Discovery Metadata の `issuer` は取得元 URL と一致する

Primary actor: `OAuth2Client`

#### Example: EX-TENANCY-010-01 通常経路

- Given `tenant_base_domain` が設定されている
- And テナント "default" の endpoint_style は Path、テナント "acme" の endpoint_style は Subdomain である
- When "{base}/realms/default/.well-known/openid-configuration" を取得する
- Then issuer は "{base}/realms/default" であり、取得元 URL の prefix と一致する
- When "https://acme.{tenant_base_domain}/.well-known/openid-configuration" を取得する
- Then issuer は "https://acme.{tenant_base_domain}" であり、取得元 URL の prefix と一致する
- Then どちらのレスポンスもエンドポイントURLを自分の正規ロケーション配下だけで組み立てる

## 拒否

### Rule: REQ-TENANCY-008 未知のサブドメインは default テナントに解決されない

Primary actor: `OAuth2Client`

#### Example: EX-TENANCY-008-01 通常経路

- Given `tenant_base_domain` が設定されている
- And realm "unknown" のテナントは存在しない
- When Host "unknown.{tenant_base_domain}" の "/authorize" にリクエストを送る
- Then 404 tenant_not_found になり、default テナントにも他のどのテナントにも到達しない

### Rule: REQ-TENANCY-009 テナントは自分の正規ロケーション以外からは到達できない

Primary actor: `OAuth2Client`

#### Example: EX-TENANCY-009-01 通常経路

- Given `tenant_base_domain` が設定されている
- And テナント "acme" の endpoint_style は Subdomain である
- And テナント "beta" の endpoint_style は Path である
- When "/realms/acme/authorize" にリクエストを送る
- Then acme は Subdomain なので path prefix 経路では不在として扱われ 404 になる

#### Example: EX-TENANCY-009-02 Host "beta.{tenant_base_domain}" の "/authorize" にリクエストを送る

- Given `tenant_base_domain` が設定されている
- And テナント "acme" の endpoint_style は Subdomain である
- And テナント "beta" の endpoint_style は Path である
- When "/realms/acme/authorize" にリクエストを送る
- But Host "beta.{tenant_base_domain}" の "/authorize" にリクエストを送る
- Then beta は Path なのでサブドメイン経路では不在として扱われ 404 になる

#### Example: EX-TENANCY-009-03 Host "acme.{tenant_base_domain}" の "/realms/beta/authorize" にリクエストを送る

- Given `tenant_base_domain` が設定されている
- And テナント "acme" の endpoint_style は Subdomain である
- And テナント "beta" の endpoint_style は Path である
- When "/realms/acme/authorize" にリクエストを送る
- But Host "acme.{tenant_base_domain}" の "/realms/beta/authorize" にリクエストを送る
- Then acme の origin から beta へ到達することはできず 404 になる

## 作用

### Rule: REQ-TENANCY-011 System管理者はテナントの正規ロケーションを切り替えられる

Primary actor: `SystemAdministrator`

#### Example: EX-TENANCY-011-01 通常経路

- Given system_admin ロールを持つ "sysadmin" が認証済みである
- And `tenant_base_domain` が設定されている
- And テナント "acme" の endpoint_style は Path である
- When "sysadmin" が `SetTenantEndpointStyle` で acme を `Subdomain` に切り替える
- Then acme は "acme.{tenant_base_domain}" からのみ到達できるようになる
- Then "{base}/realms/acme/..." は 404 になる
- Then issuer と WebAuthn RP ID が新しい正規ロケーション由来の値に変わる

#### Example: EX-TENANCY-011-02 `tenant_base_domain` が設定されていない環境で `Subdomain` を指定する

- Given system_admin ロールを持つ "sysadmin" が認証済みである
- And `tenant_base_domain` が設定されている
- And テナント "acme" の endpoint_style は Path である
- When "sysadmin" が `SetTenantEndpointStyle` で acme を `Subdomain` に切り替える
- But `tenant_base_domain` が設定されていない環境で `Subdomain` を指定する
- Then `InvalidRequestError` で拒否され、`endpoint_style` は変わらない
