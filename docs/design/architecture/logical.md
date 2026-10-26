# 論理アーキテクチャ

この文書は、IdMagic のモジュールの構成を選んだアーキテクチャ様式と、品質要件のモジュールへの割り当てを示す。

## アーキテクチャ様式

この文書では二つの「モジュール」を区別する。
**Go モジュール**は、`go.mod` が定める依存管理とビルドの単位である。
**モジュール**は、責務と公開契約を分ける論理単位であり、[責務表](../../modules/README.md#モジュール)の一行に当たる。

IdMagic は、リポジトリのルートの `go.mod` が定める一つの Go モジュールの中に複数のモジュールを実装し、複数の実行単位が実装を共有する Modular Monolith である。
モジュールの境界は Go モジュールの境界ではなく、公開パッケージと依存規則で保つ。
モジュール間は公開パッケージに置いた型、操作、ポートで接続する。ドメイン層とユースケース層は外部技術へ依存せず、ポートを通じて PostgreSQL、HTTP、通知などのアダプターへ接続する。

通常は複数のモジュールを一つの API プロセスに組み合わせ、リソースやレイテンシーの特性が異なるジョブと横断的なバッチ処理だけを別の実行単位にする。独立したデータ所有権、担当チーム、サービス目標が必要になるまではサービスへ分割しない。この記述は現在の設計を示すものであり、将来も同じ構成を義務付けるものではない。

モジュールの分割とは別に、同じ実装のまま API の Deployment を用途別の種別へ分けるかどうかという軸がある。こちらは [System の設計判断](../../modules/system/decisions.md#api-のプレーンを分けない)が扱う。

コードの依存方向、公開範囲、配置は[バックエンド設計](../application/backend.md)で、境界を選ぶ手順とモジュールを評価する規則は[設計ガイドライン](../application/design-guidelines.md)で定める。
論理単位をプロセスへ割り当てる方法は[ランタイムアーキテクチャ](runtime.md)で定める。
画面は利用者の操作単位で分け、その構成は[フロントエンド設計](../application/frontend.md#機能スライスとモジュールの対応)で定める。

コード上のモジュール間の参照はソースから抽出し、`mise run check-boundaries` が[依存規則](../application/backend.md#モジュール間の依存規則)で判定する。
[モジュールの間の依存の図](../../modules/README.md#モジュールの間の依存)は、その参照に、各モジュールの設計が定めるモデルの利用や変換のラベル、HTTP とポート越しの呼び出しを重ねて示す。
イベントの語彙と配信点は [モジュール間イベント](../application/backend.md#モジュール間イベント) が定める。

## モジュールの構成

モジュールの一覧、責務、公開パッケージ、Go パッケージとの対応は、[モジュール設計文書の責務表](../../modules/README.md#モジュール)が定める。

## 品質要件の割り当て

数値目標の ID は[品質要件](../../requirements/quality.md)の宣言を参照する。
担当機能は HTTP 応答の測定境界を担い、署名鍵、主体の保存、共有の計装などの依存先は、その機能の設計と共通設計に従う。
JWKS の HTTP 応答と形式への直列化は OAuth2 が担当し、鍵素材の提供は SigningKeys が担当する。
複数の機能を含む母集団は同じ行へ担当を列挙し、目標値と測定境界を分割しない。

| 品質要件 | 担当機能 | 設計 | 検証 |
| --- | --- | --- | --- |
| [SLO-LOGIN-LATENCY](../../requirements/quality.md#サービス目標) | [ログイン](../../modules/authentication/features/sign-in.md#概要) | [負荷と律速点](../performance/scaling.md#スケーリング単位) | [SLO の証拠](../verification/README.md#要件の群ごとの証拠) |
| [SLO-AUTHORIZE-LATENCY](../../requirements/quality.md#サービス目標) | [認可](../../modules/oauth2/features/authorization.md#概要) | [負荷と律速点](../performance/scaling.md#スケーリング単位) | [SLO の証拠](../verification/README.md#要件の群ごとの証拠) |
| [SLO-PAR-LATENCY](../../requirements/quality.md#サービス目標) | [PAR](../../modules/oauth2/features/authorization.md#概要) | [負荷と律速点](../performance/scaling.md#スケーリング単位) | [SLO の証拠](../verification/README.md#要件の群ごとの証拠) |
| [SLO-TOKEN-LATENCY](../../requirements/quality.md#サービス目標) | [トークン](../../modules/oauth2/features/token.md#概要) | [負荷と律速点](../performance/scaling.md#スケーリング単位) | [SLO の証拠](../verification/README.md#要件の群ごとの証拠) |
| [SLO-INTROSPECT-LATENCY](../../requirements/quality.md#サービス目標) | [イントロスペクション](../../modules/oauth2/features/token.md#概要) | [負荷と律速点](../performance/scaling.md#スケーリング単位) | [SLO の証拠](../verification/README.md#要件の群ごとの証拠) |
| [SLO-REVOKE-LATENCY](../../requirements/quality.md#サービス目標) | [失効](../../modules/oauth2/features/token.md#概要) | [負荷と律速点](../performance/scaling.md#スケーリング単位) | [SLO の証拠](../verification/README.md#要件の群ごとの証拠) |
| [SLO-USERINFO-LATENCY](../../requirements/quality.md#サービス目標) | [UserInfo](../../modules/oauth2/features/token.md#概要) | [負荷と律速点](../performance/scaling.md#スケーリング単位) | [SLO の証拠](../verification/README.md#要件の群ごとの証拠) |
| [SLO-DISCOVERY-LATENCY](../../requirements/quality.md#サービス目標) | [メタデータ](../../modules/oauth2/features/protocol-endpoints.md#概要) | [負荷と律速点](../performance/scaling.md#スケーリング単位) | [SLO の証拠](../verification/README.md#要件の群ごとの証拠) |
| [SLO-JWKS-LATENCY](../../requirements/quality.md#サービス目標) | [JWKS](../../modules/oauth2/features/protocol-endpoints.md#概要) | [負荷と律速点](../performance/scaling.md#スケーリング単位) | [SLO の証拠](../verification/README.md#要件の群ごとの証拠) |
| [SLO-REGISTER-LATENCY](../../requirements/quality.md#サービス目標) | [動的登録](../../modules/oauth2/features/client.md#概要) | [負荷と律速点](../performance/scaling.md#スケーリング単位) | [SLO の証拠](../verification/README.md#要件の群ごとの証拠) |
| [SLO-DEVICE-AUTHORIZATION-LATENCY](../../requirements/quality.md#サービス目標) | [デバイス認可](../../modules/oauth2/features/device.md#概要) | [負荷と律速点](../performance/scaling.md#スケーリング単位) | [SLO の証拠](../verification/README.md#要件の群ごとの証拠) |
| [SLO-FEDERATION-CALLBACK-LATENCY](../../requirements/quality.md#サービス目標) | [外部連携](../../modules/authentication/features/federation.md#概要) | [負荷と律速点](../performance/scaling.md#スケーリング単位) | [SLO の証拠](../verification/README.md#要件の群ごとの証拠) |
| [SLO-PRIMARY-ERRORS](../../requirements/quality.md#サービス目標) | [ログイン](../../modules/authentication/features/sign-in.md#概要)、[認可と PAR](../../modules/oauth2/features/authorization.md#概要)、[トークンと失効と UserInfo](../../modules/oauth2/features/token.md#概要)、[動的登録](../../modules/oauth2/features/client.md#概要)、[デバイス認可](../../modules/oauth2/features/device.md#概要) | [負荷と律速点](../performance/scaling.md#スケーリング単位) | [SLO の証拠](../verification/README.md#要件の群ごとの証拠) |
| [SLO-LOOKUP-ERRORS](../../requirements/quality.md#サービス目標) | [イントロスペクション](../../modules/oauth2/features/token.md#概要)、[メタデータと JWKS](../../modules/oauth2/features/protocol-endpoints.md#概要) | [負荷と律速点](../performance/scaling.md#スケーリング単位) | [SLO の証拠](../verification/README.md#要件の群ごとの証拠) |
| [SLO-OAUTH2-AVAILABILITY](../../requirements/quality.md#サービス目標) | [認可と PAR](../../modules/oauth2/features/authorization.md#概要)、[トークンと参照](../../modules/oauth2/features/token.md#概要)、[メタデータと JWKS](../../modules/oauth2/features/protocol-endpoints.md#概要)、[動的登録](../../modules/oauth2/features/client.md#概要)、[デバイス認可](../../modules/oauth2/features/device.md#概要) | [可用性](../reliability/availability.md#可用性設計) | [SLO の証拠](../verification/README.md#要件の群ごとの証拠) |
| [SLO-TOKEN-AVAILABILITY](../../requirements/quality.md#サービス目標) | [トークン](../../modules/oauth2/features/token.md#概要) | [可用性](../reliability/availability.md#可用性設計) | [SLO の証拠](../verification/README.md#要件の群ごとの証拠) |
| [CAP-TOKEN-THROUGHPUT](../../requirements/quality.md#キャパシティ受入れ) | [トークン](../../modules/oauth2/features/token.md#概要) | [キャパシティ](../performance/capacity.md#想定ワークロード) | [品質受入れ](../verification/system-acceptance.md#品質受入れ) |
| [CAP-AUTHORIZE-THROUGHPUT](../../requirements/quality.md#キャパシティ受入れ) | [認可](../../modules/oauth2/features/authorization.md#概要) | [キャパシティ](../performance/capacity.md#想定ワークロード) | [品質受入れ](../verification/system-acceptance.md#品質受入れ) |
| [CAP-INTROSPECT-THROUGHPUT](../../requirements/quality.md#キャパシティ受入れ) | [イントロスペクション](../../modules/oauth2/features/token.md#概要) | [キャパシティ](../performance/capacity.md#想定ワークロード) | [品質受入れ](../verification/system-acceptance.md#品質受入れ) |

この表は目標設計の担当を示し、達成済みの証拠ではない。
数値目標の実測は未実施であり、SLO の達成とキャパシティ受入れには実環境が必要である。
可用性の記録ルールとアラートの不足は[監視設計](../observability/monitoring.md#検知していない事象)に示す。
数値を持たない品質特性は、[品質特性ごとの要件](../../requirements/quality.md#品質特性ごとの要件)にある共通設計と検証への参照を適用する。
