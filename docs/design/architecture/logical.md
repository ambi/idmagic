# 論理アーキテクチャ

この文書は、IdMagic を構成するモジュールとその責務、公開範囲、組み立て地点を示す。

## アーキテクチャ様式

IdMagic は、一つの Go モジュールの中でモジュールの境界を保つ Modular Monolith である。独立したデータ所有権、担当チーム、サービス目標が必要になるまではサービスへ分割しない。ドメイン層とユースケース層は外部技術へ依存せず、ポートを通じて PostgreSQL、HTTP、通知などのアダプターへ接続する。

コードの依存方向、公開範囲、配置は[コード構成](../../domain/structure.md)で、境界を選ぶ手順とモジュールを評価する規則は[設計ガイドライン](../application/design-guidelines.md)で定める。
論理単位をプロセスへ割り当てる方法は[ランタイムアーキテクチャ](runtime.md)で定める。
画面は利用者の操作単位で分け、その構成は[フロントエンド設計](../application/frontend.md#機能スライスとモジュールの対応)で定める。

モジュール間の依存の辺は、この文書に手で書かない。
どのモジュールがどのモジュールを import するかはコードから抽出でき、`mise run check-boundaries` が[コード構成](../../domain/structure.md#モジュール間の依存規則)の規則で判定する。
ドメインイベントは、発行するモジュールが共通のワイヤ表現を使い、組み立て地点にある一つの配信点を通じて Audit と Authentication の通知機構へ事実を渡すため、モジュール間の import を生まない。契約となる語彙は [モジュール間イベント](../../domain/structure.md#モジュール間イベント) で定める。

## モジュールの責務

**モジュール**は、この表に宣言した責務と、その実装、仕様の文書、TypeSpec の対応を一つの単位として呼ぶ名前である。
モジュールの名前、Go パッケージ、仕様のディレクトリの対応は、この表だけで定める。
表の各列は、境界検査の入力として次の意味を持つ。

| 列 | 意味 |
| --- | --- |
| モジュール | モジュールの名前と、仕様の文書の入口 |
| 公開方式 | `legacy` は `domain` または `ports` の区画を含むパッケージを公開とみなす移行中の方式、`internal` は公開パッケージの列に挙げたパッケージだけを公開する最終形。System 行は組み立て地点を表す |
| 公開パッケージ | `internal` のモジュールがほかのモジュールへ公開するパッケージの完全なパス。配下のパッケージは自動的には公開しない |
| Go パッケージ | モジュールに属するパッケージの接頭辞。System 行では組み立て地点の接頭辞 |
| 責務 | モジュールが担う状態と操作 |

公開方式と公開パッケージの判定、組み立て地点と共有ライブラリの扱いは[コード構成](../../domain/structure.md#モジュール間の依存規則)が定める。

| モジュール | 公開方式 | 公開パッケージ | Go パッケージ | 責務 |
| --- | --- | --- | --- | --- |
| [System](../../domain/system/README.md) | 組み立て地点 | なし | `backend/cmd`, `backend/shared/http/server_http`, `backend/shared/http/testing_stack`, `frontend/` | 起動、機能選択、経路の組み立て、健全性、画面 |
| [Tenancy](../../domain/tenancy/README.md) | `legacy` | `domain` または `ports` の区画 | `backend/tenancy` | Tenant と realm、テナント設定、属性スキーマ、制御面の管理 |
| [IdManagement](../../domain/identity-management/README.md) | `legacy` | `domain` または `ports` の区画 | `backend/idmanagement` | User、Group、Agent、プロフィール、アイデンティティのライフサイクル |
| [IdGovernance](../../domain/identity-governance/README.md) | `legacy` | `domain` または `ports` の区画 | `backend/idgovernance` | LifecycleWorkflow のポリシーとオーケストレーション |
| [Authentication](../../domain/authentication/README.md) | `legacy` | `domain` または `ports` の区画 | `backend/authentication` | 資格情報、MFA、ログインセッション、ステップアップ、認証イベント |
| [OAuth2](../../domain/oauth2/README.md) | `legacy` | `domain` または `ports` の区画 | `backend/oauth2` | OAuth 2.0 と OIDC のプロトコル、クライアント、同意、トークン |
| [Application](../../domain/application/README.md) | `legacy` | `domain` または `ports` の区画 | `backend/application` | Application、プロトコルのバインディング、割り当て、表示分類 |
| [Authorization](../../domain/authorization/README.md) | `legacy` | `domain` または `ports` の区画 | `backend/authorization` | 細粒度認可モデル、関係タプル、グラフ評価、整合トークン |
| [Audit](../../domain/audit/README.md) | `legacy` | `domain` または `ports` の区画 | `backend/audit` | 全モジュールの監査イベントを統合する Read Model と保持 |
| [ClaimMapping](../../domain/claim-mapping/README.md) | `legacy` | `domain` または `ports` の区画 | `backend/claimmapping` | プロトコル非依存のクレーム開示ポリシーとマッピング |
| [Provisioning](../../domain/provisioning/README.md) | `legacy` | `domain` または `ports` の区画 | `backend/provisioning` | IdMagic を正とする外向き SCIM プロビジョニング |
| [Sourcing](../../domain/sourcing/README.md) | `legacy` | `domain` または `ports` の区画 | `backend/sourcing` | 上流の権威からの内向きアイデンティティ取り込み |
| [ApiTokens](../../domain/api-tokens/README.md) | `legacy` | `domain` または `ports` の区画 | `backend/apitoken` | 管理 API と SCIM API のテナント単位アクセストークン |
| [Jobs](../../domain/jobs/README.md) | `legacy` | `domain` または `ports` の区画 | `backend/jobs` | テナント境界を保つ非同期ジョブ基盤 |
| [Seeding](../../domain/seeding/README.md) | `legacy` | `domain` または `ports` の区画 | `backend/seeding` | 環境構成のプレビュー、計画、初期データ適用 |
| [SigningKeys](../../domain/signing-keys/README.md) | `legacy` | `domain` または `ports` の区画 | `backend/signingkeys` | 用途別署名鍵、証明書、ローテーション、提供元アダプター |
| [DataKeys](../../domain/data-keys/README.md) | `legacy` | `domain` または `ports` の区画 | `backend/datakeys` | 可逆な秘密情報を保護するテナント別 DEK のライフサイクル |
| [WsFederation](../../domain/ws-federation/README.md) | `legacy` | `domain` または `ports` の区画 | `backend/wsfederation` | WS-Federation、WS-Trust、メタデータ、RP の信頼 |
| [Saml](../../domain/saml/README.md) | `legacy` | `domain` または `ports` の区画 | `backend/saml` | SAML 2.0 IdP、SP の信頼、SSO、SLO、メタデータ |
| [WorkloadIdentity](../../domain/workloadidentity/README.md) | `legacy` | `domain` または `ports` の区画 | `backend/workloadidentity` | 外部アテステーションと Agent の対応付け |
| [SharedSignals](../../domain/sharedsignals/README.md) | `legacy` | `domain` または `ports` の区画 | `backend/sharedsignals` | SSF、SET、CAEP による継続的アクセス評価と失効 |
