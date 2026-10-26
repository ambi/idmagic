# Tenancy

## 責務と境界

`Tenant`（Realm）と、テナントごとの設定、外装、通知テンプレート、リソース上限を扱う。
`Tenant` は IdMagic のすべての Aggregate が属する境界である。
このモジュールは、HTTP リクエストからのテナントの解決と、制御面のテナント管理 API も扱う。

| 扱わないもの | 担当 |
| --- | --- |
| テナントの中に置かれる User と Group | `IdManagement` |
| User と Group の属性定義の管理、検証と保存 | `IdManagement`。公開 API の仕様と規範 ID は[属性スキーマ](features/attribute-schema.md)に置く |
| テナントの中に置かれる Application | `Application` |
| 資格情報とセッション | `Authentication` |
| テナント分離の規則そのもの | [認可設計](../../design/security/authorization.md) |

このモジュールが決めるのは、それらの記録がどの境界に属し、その境界にどんな上限とデフォルトが効くかである。

## 主な流れ

| 流れ | 契機 | 実行する場所 | 詳細 |
| --- | --- | --- | --- |
| テナントの解決 | すべての HTTP の要求 | `api` のミドルウェアが、ハンドラーより先に解決する | [テナントの解決の設計](features/resolution.md#実現方式) |
| テナント単位の管理 API | 解決済みのテナントの `/api/admin/v1/...` への要求 | `api` が、ハンドラーからユースケースを同期的に呼ぶ | 各機能仕様 |
| 制御面のテナント管理 API | 解決済みのテナントの `/api/admin/v1/tenants/...` への要求 | `api` が、`RequireControlPlaneUser` で `system_admin` と制御面テナントへの所属を確かめてからユースケースを呼ぶ | [テナントのライフサイクル](features/lifecycle.md) |
| 上限の確認 | ほかのモジュールのリソースの作成 | 作成する側のユースケースが、作成のトランザクションの中で `CheckAndIncrement` を呼ぶ | [リソース上限](features/quota.md) |
| User の使用量の再集計 | 保持期限バッチ | System が IdManagement の件数を渡し、Tenancy が使用量の行を固定したまま件数で置き換える | [User の使用量の再集計](features/quota.md#req-tenancy-049-user-の使用量の再集計は使用量を占める-user-の件数で使用量を置き換える) |
| Soft Quota の照合 | 保持期限バッチ | System が各保存先の使用量を測り、Tenancy が使用量の保存と超過の警告を行う | [Soft Quota の照合](features/quota.md#req-tenancy-048-soft-quota-の照合は実データの使用量を保存し超過を警告する) |
| デフォルトテナントの作成 | プロセスの起動 | 起動処理が、存在しなければ作成する | [テナントのライフサイクル](features/lifecycle.md) |

## 設計の要点

| 方針 | 理由 |
| --- | --- |
| テナントの識別子を UUID と realm に分ける | realm の改名で外部キーに触れない。詳細は[判断](decisions.md#テナントのキーを不変な-uuid-と可変な-realm-に分ける) |
| 一つのテナントに一つの正規ロケーションと一つの発行者を対応させる | 発行者と Discovery Metadata を標準に合わせる。詳細は[判断](decisions.md#テナントの正規ロケーションを一つにする) |
| 解決済みの設定を持つ文脈と ID だけの文脈を区別する | 背景処理が未解決の設定をデフォルト値と取り違えない。格納先と ID の拒否は[テナントの解決の設計](features/resolution.md#アーキテクチャ)の一か所に集める |
| パスワードのデフォルト値を設定の型と同じ公開パッケージに置く | `backend/tenancy/domain` の定数を、設定 API と Authentication が共に使う。上書きの解決とパスワードの検証は Authentication が行う |
| 外装と属性スキーマを `Tenant` とは別の Aggregate にする | 認可と realm の解決が依存する `Tenant` を太らせない。詳細は[判断](decisions.md#外装と属性スキーマを-tenant-から独立した-aggregate-にする) |
| Hard の上限の確認を、作成する側のトランザクションに委ねる | 作成と使用量の加算を同時に確定させる |
| User の使用量のずれを実数の再集計で回収する | 加算と減算を作成と削除のトランザクションに入れない経路の失敗を、経路ごとの補償なしに次のバッチで直す |
| Soft の警告を実データの照合で行う | 記録の作成を止めず、保存済みデータと削除による減少を次のバッチで反映する |

## 機能一覧

| 機能群 | 機能 | 内容 |
| --- | --- | --- |
| テナント | [テナントのライフサイクル](features/lifecycle.md) | テナントの作成、無効化と再開、制御面の一覧 |
| テナント | [テナントの解決](features/resolution.md) | Host とパスからのテナントの解決、正規ロケーション、発行者 |
| テナント | [連携エンドポイント](features/integration-endpoints.md) | 管理者へ示すプロトコルと API の URL の一覧 |
| テナントの設定 | [テナント設定](features/settings.md) | 表示名、セキュリティポリシーの上書き、通知のデフォルト言語 |
| テナントの設定 | [ブランド設定](features/branding.md) | ホステッド UI のロゴ、配色、フッターと、その公開配信 |
| テナントの設定 | [属性スキーマ](features/attribute-schema.md) | ユーザー属性とグループ属性のカスタム定義 |
| テナントの設定 | [通知テンプレート](features/notification-template.md) | 通知メールの文面の上書き、プレビュー、試し送り |
| テナントの設定 | [リソース上限](features/quota.md) | 上限のデフォルト値、上書き、超過時の拒否 |

属性スキーマの管理 API は、IdManagement が定義の型、検証、参照保護と保存をまとめて実装する。
API のパスと仕様の分類は維持し、Tenancy の管理用の型と組み立て地点での相互変換は持たない。

## 文書

| 設計領域 | 文書 | 内容 |
| --- | --- | --- |
| データ設計 | [Tenancy のデータ](data.md#集約) | Aggregate とその関係、テナントの識別子、`tenant_id` のカラム、Aggregate の保存先 |
| インフラストラクチャ設計 | [テナントの解決の設計](features/resolution.md#インフラストラクチャ) | `tenant_base_domain` の要否はテナントの解決の設計が扱い、それ以外はシステムのインフラストラクチャ設計に従う |

| 文書 | 内容 |
| --- | --- |
| [Tenancy の用語集](glossary.md) | このモジュールでの語義 |
| [設計判断](decisions.md) | 代替案を比べた重要な判断 |
| [リスク](risks.md) | 既知の欠陥、技術的負債、受容したリスク |
