# Tenancy

Tenant (Realm) の Aggregate、ライフサイクル、HTTP リクエストからのテナント解決、テナント単位の設定・外装・属性スキーマ・通知テンプレート・リソース上限、そして制御面のテナント管理 API を担う。Tenant は IdMagic のあらゆる Aggregate が属する境界である。

テナントの中に置かれる記録そのものは扱わない。User と Group は `IdManagement` が、Application は `Application` が、資格情報とセッションは `Authentication` が管理する。この Context が決めるのは、それらがどの境界に属し、その境界にどんな上限とデフォルトが効くかである。

テナント分離の規則そのものはプロダクト全体の関心事なので [認可設計](../../design/security/authorization.md) が正であり、ここではその境界を決める側の記録を扱う。

| 文書 | 内容 |
|---|---|
| [Tenancy の用語集](glossary.md) | この Context での語義 |
| [Tenancy の設計判断](decisions.md) | 複数の機能にまたがる設計判断 |
| [Tenancy の内部設計](internals.md) | 複数の機能にまたがる機構の説明 |

一つの機能だけの規則、状態遷移、設計判断、機構の説明は、次の機能ノードに置く。

| 機能ノード | 内容 |
|---|---|
| [テナントの解決](resolution/README.md) | Host とパスからのテナントの解決、正規ロケーション、発行者 |
| [テナントのライフサイクル](lifecycle/README.md) | テナントの作成、無効化と再開、制御面の一覧 |
| [テナント設定](settings/README.md) | 表示名、セキュリティポリシーの上書き、通知のデフォルト言語 |
| [ブランド設定](branding/README.md) | ホステッド UI のロゴ、配色、フッターと、その公開配信 |
| [属性スキーマ](attribute-schema/README.md) | ユーザー属性とグループ属性のカスタム定義 |
| [リソース上限](quota/README.md) | 上限のデフォルト値、上書き、超過時の拒否 |
| [通知テンプレート](notification-template/README.md) | 通知メールの文面の上書き、プレビュー、試し送り |
| [連携エンドポイント](integration-endpoints/README.md) | 管理者へ示すプロトコルと API の URL の一覧 |
