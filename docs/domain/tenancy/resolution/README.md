# テナントの解決

## 概要

この文書は、HTTP リクエストの Host とパスからテナントを決める規則、正規ロケーションと発行者の導出、正規ロケーションの切り替えの仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | Host とパスからのテナントの解決、正規ロケーションと発行者の導出、正規ロケーションの切り替え |
| 行為者 | プロトコルのエンドポイントを呼ぶ `OAuth2Client` と利用者、管理 API の呼び出し元、正規ロケーションを切り替える System 管理者 |
| 扱わないもの | 解決したテナントの中で何を許可するかは[認可設計](../../../design/security/authorization.md)が扱う |

## モデル

テナントの `endpoint_style` は、そのテナントの正規ロケーションを一つだけ指す。

| `endpoint_style` | 到達する経路 | 発行者 |
| --- | --- | --- |
| `Path` | パスの接頭辞 `/realms/{realm}/...` | `{base}/realms/{realm}` |
| `Subdomain` | Host `{realm}.{tenant_base_domain}` | `{scheme}://{realm}.{tenant_base_domain}` |

テナントは正規ロケーションからだけ到達でき、もう一方の経路では不在として扱う。
一つのテナントは、一つの正規ロケーションと一つの発行者を持つ。

- **判断**：正規ロケーションを一つにする理由は、[テナントの正規ロケーションを一つにする](../design/decisions.md#テナントの正規ロケーションを一つにする)。パスの接頭辞をデフォルトとする理由は、[テナントの解決はパスの接頭辞をデフォルトとする](../design/decisions.md#テナントの解決はパスの接頭辞をデフォルトとする)。

## 操作

### 要求からのテナントの解決

#### REQ-TENANCY-006 path style のテナントは realm prefix から解決される

- パスが `/realms/{realm}/` で始まる要求を受けたとき、Tenancy は、realm が `{realm}` で `endpoint_style` が `Path` のテナントに解決する。
- パスが realm の接頭辞を持たず、Host もテナントに解決しない要求を受けた場合、Tenancy は、404 と `tenant_not_found` で拒否し、default テナントを含むどのテナントにも解決しない。
- **例**：EX-TENANCY-006-01、EX-TENANCY-006-02、EX-TENANCY-006-03

#### REQ-TENANCY-007 subdomain style のテナントは Host から解決される

- `tenant_base_domain` を設定したデプロイでは、Host が `{realm}.{tenant_base_domain}` の要求を受けたとき、Tenancy は、realm が `{realm}` で `endpoint_style` が `Subdomain` のテナントに解決する。
- `Subdomain` のテナントでは、セッション Cookie を発行するとき、Tenancy は、名前に `__Host-` 接頭辞と `Path=/` を付け、`Domain` 属性を付けない。
- `Subdomain` のテナントでは、Tenancy は、そのテナントのホスト名 `{realm}.{tenant_base_domain}` を WebAuthn の RP ID として使う。
- **例**：EX-TENANCY-007-01

#### REQ-TENANCY-022 Host は大文字と小文字、ポート、末尾のドットを区別せず、単一のラベルだけを realm として読む

- Host からテナントを解決するとき、Tenancy は、Host の前後の空白、ポート番号、末尾のドットを除き、小文字にしてから `{label}.{tenant_base_domain}` と照合する。
- `{label}` が空か、`{label}` にドットを含む Host の要求を受けた場合、Tenancy は、どのテナントにも解決しない。
- `tenant_base_domain` を設定していないデプロイでは、Tenancy は、Host からテナントを解決しない。
- **例**：EX-TENANCY-022-01、EX-TENANCY-022-02

#### REQ-TENANCY-024 解決したテナントの応答は Host への依存を示し、発行者はデプロイの発行者から導出する

- テナントの解決に成功したとき、Tenancy は、応答に `Vary: Host` を付ける。
- `Path` のテナントでは、Tenancy は、デプロイに設定した発行者の URL `{base}` から `{base}/realms/{realm}` を発行者として導出する。
- `Subdomain` のテナントでは、Tenancy は、`{base}` の `{scheme}` と `{:port}` を引き継いだ `{scheme}://{realm}.{tenant_base_domain}{:port}` を発行者として導出する。
- **例**：EX-TENANCY-024-01、EX-TENANCY-024-02

#### REQ-TENANCY-010 Discovery Metadata の `issuer` は取得元 URL と一致する

- テナントの Discovery Metadata を取得したとき、Tenancy は、取得元 URL から `/.well-known/openid-configuration` を除いた URL と一致する、そのテナントの発行者を `issuer` に返す。
- テナントの Discovery Metadata を取得したとき、Tenancy は、すべてのエンドポイントの URL をそのテナントの正規ロケーションの下で返す。
- **例**：EX-TENANCY-010-01

#### REQ-TENANCY-008 未知のサブドメインは default テナントに解決されない

- `tenant_base_domain` を設定したデプロイでは、存在しない realm のサブドメインへの要求を受けた場合、Tenancy は、404 と `tenant_not_found` で拒否し、default テナントにもほかのどのテナントにも解決しない。
- **例**：EX-TENANCY-008-01

#### REQ-TENANCY-009 テナントは自分の正規ロケーション以外からは到達できない

- `Subdomain` のテナントへパスの接頭辞 `/realms/{realm}/` で到達した場合、Tenancy は、404 と `tenant_not_found` で拒否する。
- `Path` のテナントへ Host `{realm}.{tenant_base_domain}` で到達した場合、Tenancy は、404 と `tenant_not_found` で拒否する。
- Host がテナントに解決する要求のパスが `/realms/{realm}/` で始まる場合、Tenancy は、404 と `tenant_not_found` で拒否し、パスの realm のテナントにも Host のテナントにも解決しない。
- **例**：EX-TENANCY-009-01、EX-TENANCY-009-02、EX-TENANCY-009-03

#### REQ-TENANCY-023 解決の拒否は、テナントの存在と状態を正規ロケーションの外へ漏らさない

- 要求をテナントに解決できない場合、Tenancy は、状態コード 404 と本文 `{"error":"tenant_not_found"}` を返す。
- テナントは存在するが到達経路が `endpoint_style` と一致しない場合、Tenancy は、realm が存在しない要求と同じ応答を返す。
- 無効化されたテナントへ正規ロケーション以外から到達した場合、Tenancy は、同じ 404 を返す。
- テナントが `Disabled` の間、正規ロケーションへの要求を受けたとき、Tenancy は、プロトコルの経路か管理 API の経路かを問わず、状態コード 400 と `error` が `invalid_request` の本文を返し、要求を処理しない。
- **例**：EX-TENANCY-023-01、EX-TENANCY-023-02

### System 管理者による正規ロケーションの切り替え

#### REQ-TENANCY-011 System管理者はテナントの正規ロケーションを切り替えられる

- System 管理者がテナントの `endpoint_style` を切り替えたとき、Tenancy は、テナントの `endpoint_style` を指定した値にし、204 を返す。
- System 管理者がテナントの `endpoint_style` を切り替えたとき、Tenancy は、以後の要求を新しい正規ロケーションからだけ受け付け、発行者と WebAuthn の RP ID を新しい正規ロケーションから導出する。
- System 管理者がテナントの `endpoint_style` を切り替えたとき、Tenancy は、操作者、テナント、切り替える前と後の `endpoint_style` を載せた `TenantEndpointStyleChanged` を発行する。
- `endpoint_style` に `path` と `subdomain` のどちらでもない値を指定された場合、Tenancy は、400 と `invalid_request` で拒否し、`TenantEndpointStyleChanged` を発行しない。
- `tenant_base_domain` を設定していないデプロイでは、`subdomain` を指定された場合、Tenancy は、400 と `invalid_request` で拒否し、`endpoint_style` を変えず、`TenantEndpointStyleChanged` を発行しない。
- 存在しない realm を指定された場合、Tenancy は、404 と `tenant_not_found` で拒否し、`TenantEndpointStyleChanged` を発行しない。
- **判断**：切り替える前と同じ値を指定した要求にもイベントを発行する。イベントは操作者の操作を監査に残すためのものであり、前と後の値を比べれば、状態が動いた切替と再送を区別できる。
- **判断**：切替を `TenantUpdated` に含めず別のイベントにするのは、切替が発行済みトークンの `iss` の検証と既存のパスキーを無効にし、通常の属性の更新と影響の範囲も復旧の手順も異なるためである。型を分ければ、監査ログからこの操作だけを取り出せる。
- **例**：EX-TENANCY-011-01、EX-TENANCY-011-02

## セキュリティ上の考慮

テナントの解決は認可境界の一部である。
どの経路にも一致しないリクエストをデフォルトテナントへ退避させず、存在しない realm と正規ロケーション以外からの到達とで応答を変えない。
そのため、解決器の応答だけからテナントを列挙することはできない。
境界の規則そのものは[認可設計](../../../design/security/authorization.md)が定める。

正規ロケーションの切り替えと realm の変更は、発行者を変え、`Subdomain` ではホスト名も変える。
既存のクライアントの再設定、既存のパスキーの再登録、進行中のセッションの終了を伴うので、アイデンティティの移行として計画する。
