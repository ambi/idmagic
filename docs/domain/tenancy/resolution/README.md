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

#### REQ-TENANCY-007 subdomain style のテナントは Host から解決される

#### REQ-TENANCY-022 Host は大文字と小文字、ポート、末尾のドットを区別せず、単一のラベルだけを realm として読む

- Host は、前後の空白、ポート番号、末尾のドットを除き、小文字にしてから `{label}.{tenant_base_domain}` と照合する。
- `{label}` が空の Host と、`{label}` にドットを含む Host は、どのテナントにも解決しない。
- `tenant_base_domain` を設定していないデプロイでは、Host からテナントを解決しない。
- **担保手段**：`Deps.ResolveHostTenant`

#### REQ-TENANCY-024 解決したテナントの応答は Host への依存を示し、発行者はデプロイの発行者から導出する

- 解決に成功したすべての応答に `Vary: Host` を付ける。
- `Path` のテナントの発行者は `{base}/realms/{realm}` とする。`{base}` はデプロイに設定した発行者の URL である。
- `Subdomain` のテナントの発行者は `{scheme}://{realm}.{tenant_base_domain}{:port}` とする。`{scheme}` と `{:port}` は `{base}` のものを引き継ぐ。
- **担保手段**：`Deps.CanonicalLocation`

#### REQ-TENANCY-010 Discovery Metadata の `issuer` は取得元 URL と一致する

#### REQ-TENANCY-008 未知のサブドメインは default テナントに解決されない

#### REQ-TENANCY-009 テナントは自分の正規ロケーション以外からは到達できない

#### REQ-TENANCY-023 解決の拒否は、テナントの存在と状態を正規ロケーションの外へ漏らさない

- 解決できないリクエストには、状態コード 404 と本文 `{"error":"tenant_not_found"}` を返す。
- realm が存在しない場合と、テナントは存在するが到達経路が `endpoint_style` と一致しない場合とで、応答を変えない。
- 無効化されたテナントへ正規ロケーション以外から到達したリクエストにも、同じ 404 を返す。
- 無効化されたテナントへ正規ロケーションから到達したリクエストには、プロトコルの経路か管理 API の経路かを問わず、状態コード 400 と `error` が `invalid_request` の本文を返す。
- **担保手段**：`Deps.ResolvePathTenant`、`Deps.ResolveHostTenant`

### System 管理者による正規ロケーションの切り替え

#### REQ-TENANCY-011 System管理者はテナントの正規ロケーションを切り替えられる

## セキュリティ上の考慮

テナントの解決は認可境界の一部である。
どの経路にも一致しないリクエストをデフォルトテナントへ退避させず、存在しない realm と正規ロケーション以外からの到達とで応答を変えない。
そのため、解決器の応答だけからテナントを列挙することはできない。
境界の規則そのものは[認可設計](../../../design/security/authorization.md)が定める。

正規ロケーションの切り替えと realm の変更は、発行者を変え、`Subdomain` ではホスト名も変える。
既存のクライアントの再設定、既存のパスキーの再登録、進行中のセッションの終了を伴うので、アイデンティティの移行として計画する。
