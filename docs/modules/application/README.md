# Application

## 責務と境界

運用者が「接続する業務アプリケーション」として扱う Application を扱う。
OIDC のクライアント、SAML の SP、WS-Fed の RP は、Application に関連付けるプロトコル設定である。
表示名、アイコン、ライフサイクル、割り当て、サインインポリシー、ポータルでの並び順とカテゴリはここに集約する。

| 扱わないもの | 担当 |
| --- | --- |
| プロトコルの通信の上の動作 | `OAuth2`、`SAML`、`WS-Federation` |
| クレームの公開の下限 | `ClaimMapping` |
| 下流のシステムへのプロビジョニング | `Provisioning` |
| ステップアップ認証と信頼済みデバイス | `Authentication` |

割り当てとサインインポリシーは、ポータルでの表示とフェデレーションの可否をフェイルクローズで制御する。
Application は、プロトコル設定を中身に依存しないキーで参照する。

## モデル

| Aggregate | 境界の内側にあるもの | 他の Aggregate との関係 |
| --- | --- | --- |
| `Application` | 名前、種類（`weblink`、`federated`、`service`）、起動の URL、状態、アイコン、カテゴリ、プロトコル設定への参照（`ApplicationProtocol`） | `Tenant` を `tenant_id` で参照する。プロトコル設定を最大一つ参照する |
| `ApplicationAssignment` | 主体（`user` または `group`）、`visibility` | `Application` と、User または Group を参照する |
| `AppSignInPolicy` | 順序付けた `SignInRule` の集合 | `Application` を参照する |
| `TenantDefaultSignInPolicy` | テナントのデフォルトの `SignInRule` の集合 | `Tenant` を `tenant_id` で参照する |
| `ApplicationCategory` | テナントの管理者が定義するカテゴリ | `Tenant` を `tenant_id` で参照する |
| `ApplicationOrdering` | `(tenant_id, user_sub)` ごとの `application_id` の並び | User を参照する |

- **判断**：テナントのデフォルトのサインインポリシーは、テナントの Aggregate ではなく、アプリケーションへのサインインの方法に関する概念なので、`Tenancy` ではなくこのモジュールに置く。
- **判断**：ポータルでの手動の並び順とカテゴリは、IdManagement の User の Aggregate ではなく、`Application` の表示に関する概念なので、このモジュールに置く。

## 公開する契約

HTTP の操作とモデルの形は TypeSpec の `Application` のタグが定める。
次の表は、それ以外にほかのモジュールと結ぶ契約である。

| 契約 | 相手 | 向き | 内容 |
| --- | --- | --- | --- |
| 割り当てとサインインポリシーの評価 | `OAuth2` の認可、`SAML` の SSO、`WS-Federation` のサインイン | このモジュールが提供する | フェデレーションを始めるたびに、トークンや Assertion の発行の前に、割り当てとポリシーを評価する |
| サインインポリシーの評価の計算（`EvaluateSignInPolicy`、`EffectiveSignInRules` ほか） | `OAuth2` のログイン、`Authentication` のアカウントのセキュリティ設定 | このモジュールが提供する | 保存先を読まず、ポリシー、認証の文脈、クライアントの IP、時刻から、許可、拒否、step-up の判定、信頼済みデバイスの可否、MFA の強制の開始日を計算する |
| `AssignApplicationDesiredState`、`UnassignApplicationDesiredState` | `IdGovernance` | このモジュールが提供する | あるべき状態として、直接の割り当てを冪等に作成と解除する |
| `ProvisioningNotifier` | `Provisioning` が実装する | このモジュールが定める | 割り当ての変化を下流のプロビジョニングへ伝える |
| `SubjectDirectory` | `IdManagement` が実装する | このモジュールが定める | 割り当ての主体が存在するかを確かめる |
| ドメインイベント | 監査と下流が購読する | このモジュールが発行する | `Application…`、`AppSignInPolicyUpdated`、`TenantDefaultSignInPolicyUpdated`、`AppAccessDeniedByPolicy`、`AppStepUpRequired`、`ApplicationCategory…` |

## 機能

| 機能 | 内容 |
| --- | --- |
| [アプリケーションのカタログ](catalog/README.md) | Application とプロトコル設定の作成、参照、更新、クライアントシークレット、アイコン |
| [割り当て](assignment/README.md) | 割り当てによるフェデレーションの関門と、あるべき状態としての割り当て |
| [サインインポリシー](sign-in-policy/README.md) | アプリケーションごとのポリシーとテナントのデフォルトのポリシー |
| [ポータルのアプリケーション](portal/README.md) | エンドユーザーのポータルでの一覧と並び順 |

| 文書 | 内容 |
| --- | --- |
| [Application の用語集](glossary.md) | このモジュールでの語義 |

## 設計

| 設計領域 | 区分 | 参照先 | 内容 |
| --- | --- | --- | --- |
| アーキテクチャ設計 | 固有設計 | [担当文書](design/architecture.md#文脈と範囲) | 文脈と範囲、解決戦略、構成要素、実行時の流れ |
| データ設計 | 固有設計 | [担当文書](design/data.md#application-とプロトコル設定の関係) | Application とプロトコル設定の関係、作成の原子性 |
| セキュリティ設計 | 委譲 | [割り当て](assignment/README.md#セキュリティ上の考慮)、[サインインポリシー](sign-in-policy/README.md#セキュリティ上の考慮) | フェデレーションの関門は割り当てとサインインポリシーのセキュリティ上の考慮の節が扱う |

| 文書 | 内容 |
| --- | --- |
| [設計判断](design/decisions.md) | 代替案を比べた重要な判断 |
