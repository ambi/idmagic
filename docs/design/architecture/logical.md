# 論理アーキテクチャ

この文書は、IdMagic を構成するモジュールとその責務、公開範囲、組み立て地点を示す。

## アーキテクチャ様式

IdMagic は、一つの Go モジュール（`go.mod`）の中でモジュールの境界を保ち、複数の実行単位が実装を共有する Modular Monolith である。モジュール間は公開パッケージに置いた型、操作、ポートで接続する。ドメイン層とユースケース層は外部技術へ依存せず、ポートを通じて PostgreSQL、HTTP、通知などのアダプターへ接続する。

通常は複数のモジュールを一つの API プロセスに組み合わせ、リソースやレイテンシーの特性が異なるジョブと横断的なバッチ処理だけを別の実行単位にする。独立したデータ所有権、担当チーム、サービス目標が必要になるまではサービスへ分割しない。この記述は現在の設計を示すものであり、将来も同じ構成を義務付けるものではない。

モジュールの分割とは別に、同じ実装のまま API の Deployment を用途別の種別へ分けるかどうかという軸がある。こちらは [System の設計判断](../../modules/system/design/decisions.md#api-のプレーンを分けない)が扱う。

コードの依存方向、公開範囲、配置は[バックエンド設計](../application/backend.md)で、境界を選ぶ手順とモジュールを評価する規則は[設計ガイドライン](../application/design-guidelines.md)で定める。
論理単位をプロセスへ割り当てる方法は[ランタイムアーキテクチャ](runtime.md)で定める。
画面は利用者の操作単位で分け、その構成は[フロントエンド設計](../application/frontend.md#機能スライスとモジュールの対応)で定める。

モジュール間の依存の辺は、この文書に手で書かない。
どのモジュールがどのモジュールを import するかはコードから抽出でき、`mise run check-boundaries` が[バックエンド設計](../application/backend.md#モジュール間の依存規則)の規則で判定する。
ドメインイベントは、発行するモジュールが共通のワイヤ表現を使い、組み立て地点にある一つの配信点を通じて Audit と Authentication の通知機構へ事実を渡すため、モジュール間の import を生まない。契約となる語彙は [モジュール間イベント](../application/backend.md#モジュール間イベント) で定める。

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

公開方式と公開パッケージの判定、組み立て地点と共有ライブラリの扱いは[バックエンド設計](../application/backend.md#モジュール間の依存規則)が定める。

| モジュール | 公開方式 | 公開パッケージ | Go パッケージ | 責務 |
| --- | --- | --- | --- | --- |
| [System](../../modules/system/README.md) | 組み立て地点 | なし | `backend/cmd`, `backend/shared/http/server_http`, `backend/shared/http/testing_stack`, `frontend/` | 起動、機能選択、経路の組み立て、健全性、画面 |
| [Tenancy](../../modules/tenancy/README.md) | `legacy` | `domain` または `ports` の区画 | `backend/tenancy` | Tenant と realm、テナント設定、属性スキーマ、制御面の管理 |
| [IdManagement](../../modules/identity-management/README.md) | `legacy` | `domain` または `ports` の区画 | `backend/idmanagement` | User、Group、Agent、プロフィール、アイデンティティのライフサイクル |
| [IdGovernance](../../modules/identity-governance/README.md) | `legacy` | `domain` または `ports` の区画 | `backend/idgovernance` | LifecycleWorkflow のポリシーとオーケストレーション |
| [Authentication](../../modules/authentication/README.md) | `legacy` | `domain` または `ports` の区画 | `backend/authentication` | 資格情報、MFA、ログインセッション、ステップアップ、認証イベント |
| [OAuth2](../../modules/oauth2/README.md) | `legacy` | `domain` または `ports` の区画 | `backend/oauth2` | OAuth 2.0 と OIDC のプロトコル、クライアント、同意、トークン |
| [Application](../../modules/application/README.md) | `legacy` | `domain` または `ports` の区画 | `backend/application` | Application、プロトコルのバインディング、割り当て、表示分類 |
| [Authorization](../../modules/authorization/README.md) | `internal` | なし | `backend/authorization` | 細粒度認可モデル、関係タプル、グラフ評価、整合トークン |
| [Audit](../../modules/audit/README.md) | `legacy` | `domain` または `ports` の区画 | `backend/audit` | 全モジュールの監査イベントを統合する Read Model と保持 |
| [ClaimMapping](../../modules/claim-mapping/README.md) | `legacy` | `domain` または `ports` の区画 | `backend/claimmapping` | プロトコル非依存のクレーム開示ポリシーとマッピング |
| [Provisioning](../../modules/provisioning/README.md) | `legacy` | `domain` または `ports` の区画 | `backend/provisioning` | IdMagic を正とする外向き SCIM プロビジョニング |
| [Sourcing](../../modules/sourcing/README.md) | `legacy` | `domain` または `ports` の区画 | `backend/sourcing` | 上流の権威からの内向きアイデンティティ取り込み |
| [ApiTokens](../../modules/api-tokens/README.md) | `legacy` | `domain` または `ports` の区画 | `backend/apitoken` | 管理 API と SCIM API のテナント単位アクセストークン |
| [Jobs](../../modules/jobs/README.md) | `legacy` | `domain` または `ports` の区画 | `backend/jobs` | テナント境界を保つ非同期ジョブ基盤 |
| [Seeding](../../modules/seeding/README.md) | `legacy` | `domain` または `ports` の区画 | `backend/seeding` | 環境構成のプレビュー、計画、初期データ適用 |
| [SigningKeys](../../modules/signing-keys/README.md) | `legacy` | `domain` または `ports` の区画 | `backend/signingkeys` | 用途別署名鍵、証明書、ローテーション、提供元アダプター |
| [DataKeys](../../modules/data-keys/README.md) | `legacy` | `domain` または `ports` の区画 | `backend/datakeys` | 可逆な秘密情報を保護するテナント別 DEK のライフサイクル |
| [WsFederation](../../modules/ws-federation/README.md) | `legacy` | `domain` または `ports` の区画 | `backend/wsfederation` | WS-Federation、WS-Trust、メタデータ、RP の信頼 |
| [Saml](../../modules/saml/README.md) | `legacy` | `domain` または `ports` の区画 | `backend/saml` | SAML 2.0 IdP、SP の信頼、SSO、SLO、メタデータ |
| [WorkloadIdentity](../../modules/workloadidentity/README.md) | `legacy` | `domain` または `ports` の区画 | `backend/workloadidentity` | 外部アテステーションと Agent の対応付け |
| [SharedSignals](../../modules/sharedsignals/README.md) | `legacy` | `domain` または `ports` の区画 | `backend/sharedsignals` | SSF、SET、CAEP による継続的アクセス評価と失効 |
