# Tenancy

## 責務と境界

`Tenant`（Realm）と、テナントごとの設定、外装、属性スキーマ、通知テンプレート、リソース上限を扱う。
`Tenant` は IdMagic のすべての Aggregate が属する境界である。
この Context は、HTTP リクエストからのテナントの解決と、制御面のテナント管理 API も扱う。

| 扱わないもの | 担当 |
| --- | --- |
| テナントの中に置かれる User と Group | `IdManagement` |
| テナントの中に置かれる Application | `Application` |
| 資格情報とセッション | `Authentication` |
| テナント分離の規則そのもの | [認可設計](../../design/security/authorization.md) |

この Context が決めるのは、それらの記録がどの境界に属し、その境界にどんな上限とデフォルトが効くかである。

## モデル

| Aggregate | 境界の内側にあるもの | 他の Aggregate との関係 |
| --- | --- | --- |
| `Tenant` | `realm`、表示名、状態、`endpoint_style`、セキュリティポリシーの上書き、通知のデフォルト言語 | すべての Context の Aggregate が `tenant_id` で参照する |
| `TenantBranding` | ブランド設定の項目とアセット | `Tenant` を `tenant_id` で参照する |
| `TenantUserAttributeSchema`、`TenantGroupAttributeSchema` | カスタム属性の定義 | `Tenant` を `tenant_id` で参照する |
| `NotificationTemplate` | 一つの `template_key` と `locale` の文面の上書き | `Tenant` を `tenant_id` で参照する |
| `TenantQuota` | リソースごとの上限の上書きと使用量 | `Tenant` を `tenant_id` で参照する |

`Tenant` は、不変の UUID の `id` と、変更できて一意な `realm` の二つの識別子を持つ。
URL の接頭辞、OIDC の発行者、Discovery Metadata など外部に示す識別子には `realm` を、`tenant_id` の外部キーなど内部の参照には `id` を使う。

起動時に自動で作成する `realm` が `default` のテナント（デフォルトテナント）は、制御面のテナントを兼ねる。
`system_admin` のロールを持つデフォルトテナントの User だけが、テナントを越える操作を行える。

## 公開する契約

HTTP の操作とモデルの形は TypeSpec の `Tenancy` のタグが定める。
次の表は、それ以外にほかの Context と結ぶ契約である。

| 契約 | 相手 | 向き | 内容 |
| --- | --- | --- | --- |
| リクエストコンテキストのテナント | すべての Context の HTTP ハンドラー | この Context が提供する | 解決した `Tenant`、発行者、URL の接頭辞を運ぶ |
| `TenantRepository` の読み取り | `Authentication`、`OAuth2`、`IdManagement`、`DataKeys`、`SigningKeys`、テナント解決のミドルウェア | この Context が提供する | `Tenant` と、その上書きの実効値を読む |
| `QuotaRepository` の `CheckAndIncrement` と `Decrement` | リソースを作成する Context（`IdManagement`、`Application`、`OAuth2`、`Authentication`、`Jobs`、`SharedSignals`） | この Context が提供する | 作成のトランザクションの中で Hard の上限を確認し、使用量を増減する |
| `TenantUserAttributeSchemaRepository`、`TenantGroupAttributeSchemaRepository` | `IdManagement` が実装する。`SAML`、`WS-Federation`、`Application`、`IdGovernance`、`Authentication` が読む | この Context が定める | 属性スキーマの永続化と読み取り |
| `TenantBrandingRepository` の読み取り | 通知メールの描画（`backend/shared/notification`）と `Authentication` のセキュリティ通知 | この Context が提供する | 差し込み変数 `product_name` などの外装を読む |
| ドメインイベント | 監査と下流が購読する | この Context が発行する | `Tenant…`、`TenantQuotaUpdated`、`QuotaExceeded`、`NotificationTemplate…` |

## 機能

| 機能群 | 機能 | 内容 |
| --- | --- | --- |
| テナント | [テナントのライフサイクル](lifecycle/README.md) | テナントの作成、無効化と再開、制御面の一覧 |
| テナント | [テナントの解決](resolution/README.md) | Host とパスからのテナントの解決、正規ロケーション、発行者 |
| テナント | [連携エンドポイント](integration-endpoints/README.md) | 管理者へ示すプロトコルと API の URL の一覧 |
| テナントの設定 | [テナント設定](settings/README.md) | 表示名、セキュリティポリシーの上書き、通知のデフォルト言語 |
| テナントの設定 | [ブランド設定](branding/README.md) | ホステッド UI のロゴ、配色、フッターと、その公開配信 |
| テナントの設定 | [属性スキーマ](attribute-schema/README.md) | ユーザー属性とグループ属性のカスタム定義 |
| テナントの設定 | [通知テンプレート](notification-template/README.md) | 通知メールの文面の上書き、プレビュー、試し送り |
| テナントの設定 | [リソース上限](quota/README.md) | 上限のデフォルト値、上書き、超過時の拒否 |

| 文書 | 内容 |
| --- | --- |
| [Tenancy の用語集](glossary.md) | この Context での語義 |
| [Tenancy の設計](design/README.md) | 話題ごとの設計と重要な判断 |
