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
| [System](../../modules/system/README.md) | 組み立て地点 | なし | `backend/cmd`, `backend/shared/http/server_http`, `backend/shared/http/testing_stack`, `backend/shared/testing/fixtures_postgres`, `frontend/` | 起動、機能選択、経路の組み立て、テスト前提の組み立て、健全性、画面 |
| [Tenancy](../../modules/tenancy/README.md) | `legacy` | `domain` または `ports` の区画 | `backend/tenancy` | Tenant と realm、テナント設定、属性スキーマ、制御面の管理 |
| [IdManagement](../../modules/identity-management/README.md) | `legacy` | `domain` または `ports` の区画 | `backend/idmanagement` | User、Group、Agent、プロフィール、アイデンティティのライフサイクル |
| [IdGovernance](../../modules/identity-governance/README.md) | `legacy` | `domain` または `ports` の区画 | `backend/idgovernance` | LifecycleWorkflow のポリシーとオーケストレーション |
| [Authentication](../../modules/authentication/README.md) | `legacy` | `domain` または `ports` の区画 | `backend/authentication` | 資格情報、MFA、ログインセッション、ステップアップ、認証イベント |
| [OAuth2](../../modules/oauth2/README.md) | `legacy` | `domain` または `ports` の区画 | `backend/oauth2` | OAuth 2.0 と OIDC のプロトコル、クライアント、同意、トークン |
| [Application](../../modules/application/README.md) | `legacy` | `domain` または `ports` の区画 | `backend/application` | Application、プロトコルのバインディング、割り当て、表示分類 |
| [Authorization](../../modules/authorization/README.md) | `internal` | なし | `backend/authorization` | 細粒度認可モデル、関係タプル、グラフ評価、整合トークン |
| [Audit](../../modules/audit/README.md) | `internal` | `backend/audit/ports` | `backend/audit` | 全モジュールの監査イベントを統合する Read Model と保持 |
| [ClaimMapping](../../modules/claim-mapping/README.md) | `legacy` | `domain` または `ports` の区画 | `backend/claimmapping` | プロトコル非依存のクレーム開示ポリシーとマッピング |
| [Provisioning](../../modules/provisioning/README.md) | `internal` | なし | `backend/provisioning` | IdMagic を正とする外向き SCIM プロビジョニング |
| [Sourcing](../../modules/sourcing/README.md) | `internal` | `backend/sourcing/scim/ports` | `backend/sourcing` | 上流の権威からの内向きアイデンティティ取り込み |
| [ApiTokens](../../modules/api-tokens/README.md) | `internal` | `backend/apitoken/domain`, `backend/apitoken/ports` | `backend/apitoken` | 管理 API と SCIM API のテナント単位アクセストークン |
| [Jobs](../../modules/jobs/README.md) | `legacy` | `domain` または `ports` の区画 | `backend/jobs` | テナント境界を保つ非同期ジョブ基盤 |
| [Seeding](../../modules/seeding/README.md) | `legacy` | `domain` または `ports` の区画 | `backend/seeding` | 環境構成のプレビュー、計画、初期データ適用 |
| [SigningKeys](../../modules/signing-keys/README.md) | `internal` | `backend/signingkeys/domain`, `backend/signingkeys/ports` | `backend/signingkeys` | 用途別署名鍵、証明書、ローテーション、提供元アダプター |
| [DataKeys](../../modules/data-keys/README.md) | `internal` | なし | `backend/datakeys` | 可逆な秘密情報を保護するテナント別 DEK のライフサイクル |
| [WsFederation](../../modules/ws-federation/README.md) | `legacy` | `domain` または `ports` の区画 | `backend/wsfederation` | WS-Federation、WS-Trust、メタデータ、RP の信頼 |
| [Saml](../../modules/saml/README.md) | `internal` | `backend/saml/domain`, `backend/saml/ports` | `backend/saml` | SAML 2.0 IdP、SP の信頼、SSO、SLO、メタデータ |
| [WorkloadIdentity](../../modules/workloadidentity/README.md) | `internal` | `backend/workloadidentity/domain` | `backend/workloadidentity` | 外部アテステーションと Agent の対応付け |
| [SharedSignals](../../modules/sharedsignals/README.md) | `legacy` | `domain` または `ports` の区画 | `backend/sharedsignals` | SSF、SET、CAEP による継続的アクセス評価と失効 |

## 品質要件の割り当て

数値目標の ID は[品質要件](../../requirements/quality.md)の宣言を参照する。
担当機能は HTTP 応答の測定境界を担い、署名鍵、主体の保存、共有の計装などの依存先は、その機能の設計と共通設計に従う。
JWKS の HTTP 応答と形式への直列化は OAuth2 が担当し、鍵素材の提供は SigningKeys が担当する。
複数の機能を含む母集団は同じ行へ担当を列挙し、目標値と測定境界を分割しない。

| 品質要件 | 担当機能 | 設計 | 検証 |
| --- | --- | --- | --- |
| [SLO-LOGIN-LATENCY](../../requirements/quality.md#サービス目標) | [ログイン](../../modules/authentication/sign-in/README.md#概要) | [負荷と律速点](../performance/scaling.md#スケーリング単位) | [SLO の証拠](../verification/README.md#要件の群ごとの証拠) |
| [SLO-AUTHORIZE-LATENCY](../../requirements/quality.md#サービス目標) | [認可](../../modules/oauth2/authorization/README.md#概要) | [負荷と律速点](../performance/scaling.md#スケーリング単位) | [SLO の証拠](../verification/README.md#要件の群ごとの証拠) |
| [SLO-PAR-LATENCY](../../requirements/quality.md#サービス目標) | [PAR](../../modules/oauth2/authorization/README.md#概要) | [負荷と律速点](../performance/scaling.md#スケーリング単位) | [SLO の証拠](../verification/README.md#要件の群ごとの証拠) |
| [SLO-TOKEN-LATENCY](../../requirements/quality.md#サービス目標) | [トークン](../../modules/oauth2/token/README.md#概要) | [負荷と律速点](../performance/scaling.md#スケーリング単位) | [SLO の証拠](../verification/README.md#要件の群ごとの証拠) |
| [SLO-INTROSPECT-LATENCY](../../requirements/quality.md#サービス目標) | [イントロスペクション](../../modules/oauth2/token/README.md#概要) | [負荷と律速点](../performance/scaling.md#スケーリング単位) | [SLO の証拠](../verification/README.md#要件の群ごとの証拠) |
| [SLO-REVOKE-LATENCY](../../requirements/quality.md#サービス目標) | [失効](../../modules/oauth2/token/README.md#概要) | [負荷と律速点](../performance/scaling.md#スケーリング単位) | [SLO の証拠](../verification/README.md#要件の群ごとの証拠) |
| [SLO-USERINFO-LATENCY](../../requirements/quality.md#サービス目標) | [UserInfo](../../modules/oauth2/token/README.md#概要) | [負荷と律速点](../performance/scaling.md#スケーリング単位) | [SLO の証拠](../verification/README.md#要件の群ごとの証拠) |
| [SLO-DISCOVERY-LATENCY](../../requirements/quality.md#サービス目標) | [メタデータ](../../modules/oauth2/protocol-endpoints/README.md#概要) | [負荷と律速点](../performance/scaling.md#スケーリング単位) | [SLO の証拠](../verification/README.md#要件の群ごとの証拠) |
| [SLO-JWKS-LATENCY](../../requirements/quality.md#サービス目標) | [JWKS](../../modules/oauth2/protocol-endpoints/README.md#概要) | [負荷と律速点](../performance/scaling.md#スケーリング単位) | [SLO の証拠](../verification/README.md#要件の群ごとの証拠) |
| [SLO-REGISTER-LATENCY](../../requirements/quality.md#サービス目標) | [動的登録](../../modules/oauth2/client/README.md#概要) | [負荷と律速点](../performance/scaling.md#スケーリング単位) | [SLO の証拠](../verification/README.md#要件の群ごとの証拠) |
| [SLO-DEVICE-AUTHORIZATION-LATENCY](../../requirements/quality.md#サービス目標) | [デバイス認可](../../modules/oauth2/device/README.md#概要) | [負荷と律速点](../performance/scaling.md#スケーリング単位) | [SLO の証拠](../verification/README.md#要件の群ごとの証拠) |
| [SLO-FEDERATION-CALLBACK-LATENCY](../../requirements/quality.md#サービス目標) | [外部連携](../../modules/authentication/federation/README.md#概要) | [負荷と律速点](../performance/scaling.md#スケーリング単位) | [SLO の証拠](../verification/README.md#要件の群ごとの証拠) |
| [SLO-PRIMARY-ERRORS](../../requirements/quality.md#サービス目標) | [ログイン](../../modules/authentication/sign-in/README.md#概要)、[認可と PAR](../../modules/oauth2/authorization/README.md#概要)、[トークンと失効と UserInfo](../../modules/oauth2/token/README.md#概要)、[動的登録](../../modules/oauth2/client/README.md#概要)、[デバイス認可](../../modules/oauth2/device/README.md#概要) | [負荷と律速点](../performance/scaling.md#スケーリング単位) | [SLO の証拠](../verification/README.md#要件の群ごとの証拠) |
| [SLO-LOOKUP-ERRORS](../../requirements/quality.md#サービス目標) | [イントロスペクション](../../modules/oauth2/token/README.md#概要)、[メタデータと JWKS](../../modules/oauth2/protocol-endpoints/README.md#概要) | [負荷と律速点](../performance/scaling.md#スケーリング単位) | [SLO の証拠](../verification/README.md#要件の群ごとの証拠) |
| [SLO-OAUTH2-AVAILABILITY](../../requirements/quality.md#サービス目標) | [認可と PAR](../../modules/oauth2/authorization/README.md#概要)、[トークンと参照](../../modules/oauth2/token/README.md#概要)、[メタデータと JWKS](../../modules/oauth2/protocol-endpoints/README.md#概要)、[動的登録](../../modules/oauth2/client/README.md#概要)、[デバイス認可](../../modules/oauth2/device/README.md#概要) | [可用性](../reliability/availability.md#可用性設計) | [SLO の証拠](../verification/README.md#要件の群ごとの証拠) |
| [SLO-TOKEN-AVAILABILITY](../../requirements/quality.md#サービス目標) | [トークン](../../modules/oauth2/token/README.md#概要) | [可用性](../reliability/availability.md#可用性設計) | [SLO の証拠](../verification/README.md#要件の群ごとの証拠) |
| [CAP-TOKEN-THROUGHPUT](../../requirements/quality.md#キャパシティ受入れ) | [トークン](../../modules/oauth2/token/README.md#概要) | [キャパシティ](../performance/capacity.md#想定ワークロード) | [品質受入れ](../verification/system-acceptance.md#品質受入れ) |
| [CAP-AUTHORIZE-THROUGHPUT](../../requirements/quality.md#キャパシティ受入れ) | [認可](../../modules/oauth2/authorization/README.md#概要) | [キャパシティ](../performance/capacity.md#想定ワークロード) | [品質受入れ](../verification/system-acceptance.md#品質受入れ) |
| [CAP-INTROSPECT-THROUGHPUT](../../requirements/quality.md#キャパシティ受入れ) | [イントロスペクション](../../modules/oauth2/token/README.md#概要) | [キャパシティ](../performance/capacity.md#想定ワークロード) | [品質受入れ](../verification/system-acceptance.md#品質受入れ) |

この表は目標設計の担当を示し、達成済みの証拠ではない。
数値目標の実測は未実施であり、SLO の達成とキャパシティ受入れには実環境が必要である。
可用性の記録ルールとアラートの不足は[監視設計](../observability/monitoring.md#検知していない事象)に示す。
数値を持たない品質特性は、[品質特性ごとの要件](../../requirements/quality.md#品質特性ごとの要件)にある共通設計と検証への参照を適用する。
