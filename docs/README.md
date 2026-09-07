# システム文書

このディレクトリは、IdMagic の目的、要求、アーキテクチャ、設計、検証、運用をシステムの目的からトップダウンでたどるための正本文書を収める。個別のモデル、API、認証機構は `spec/contexts/<context>/` の TypeSpec が、一つの Bounded Context で閉じる振る舞いと設計は `docs/contexts/<context>/` が所有する。

## 読み順

1. [プロダクト概要](product-overview.md)で、解く問題、利用者、対象外を確認する。
2. [要求](requirements/)で、機能、品質、制約を確認する。
3. [アーキテクチャ](architecture/)で、外部境界、論理構成、実行時構成、配備構成、全体判断を確認する。
4. [設計](design/)で、アプリケーション、データ、基盤、セキュリティ、信頼性、性能、観測可能性の実現方式を確認する。
5. [検証](verification/)で、要求をどの証拠で受け入れるかを確認する。
6. [運用](operations/)と[運用手順](runbooks/)で、平常時の管理と障害時の具体的な操作を確認する。

この順序は文書の依存方向でもある。下位文書は上位の要求と判断を参照し、上位文書へ実装詳細を逆流させない。数値、境界、判断には一つの所有者を定め、別の文書は値を写さずリンクまたは安定 ID で参照する。

## 文書体系

| 層 | 正本の内容 |
| --- | --- |
| システムの目的 | [プロダクト概要](product-overview.md)、[用語集](glossary.md)、[全体規範](standards.md) |
| 要求 | [機能要求](requirements/functional.md)、[品質要求](requirements/quality.md)、[システム制約](requirements/constraints.md) |
| アーキテクチャ | [システムコンテキスト](architecture/system-context.md)、[論理アーキテクチャ](architecture/logical.md)、[実行時アーキテクチャ](architecture/runtime.md)、[配備アーキテクチャ](architecture/deployment.md)、[アーキテクチャ上の判断](architecture/decisions.md) |
| 設計 | [アプリケーション](design/application/)、[データ](design/data/)、[インフラストラクチャ](design/infrastructure/)、[セキュリティ](design/security/)、[信頼性](design/reliability/)、[性能](design/performance/)、[観測可能性](design/observability/) |
| Context 横断の振る舞い | [システム横断シナリオ](scenarios.feature.md) |
| 検証 | [システム受入れ設計](verification/system-acceptance.md) |
| 運用 | [サービス管理](operations/service-management.md)、[保守](operations/maintenance.md)、[運用手順](runbooks/) |
| ドメインの詳細 | [Context 別仕様](contexts/)と隣接する `spec/contexts/` |
| リポジトリ構造 | [構造](structure.md) |

## 執筆上の境界

人が書く現在状態の正本は `docs/` に置く。機械が読むモデルと API 契約は `spec/` に置き、生成物は追跡しない `spec/generated/` に出力する。変更固有の分析、代替案、実装履歴は `work-items/` が、開発手順は[開発文書](development/)が、リリース固有の差分は `releases/` が所有する。

文書配置、正本の種類、仕様先行の変更手順は、ルートの [文書ガイド](../DOCUMENTATION_GUIDE.md)、[Specification Format](../SPECIFICATION_FORMAT.md)、[Work Item Format](../WORK_ITEM_FORMAT.md) が定める。開発ツールのバージョンとコマンドは `mise.toml` に集約し、基本操作は `mise run <task>` から実行する。
