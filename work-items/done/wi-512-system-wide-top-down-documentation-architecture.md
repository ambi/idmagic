---
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-08
priority: p1
depends_on: []
change_kind: tooling
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: "文書体系と正準文書の配置を変更する開発者向け文書変更であり、プロダクト利用者へ告知する機能、互換性、移行操作はない。"
  references: []
initial_context:
  specification:
    - DOCUMENTATION_GUIDE.md
    - SPECIFICATION_FORMAT.md
    - WORK_ITEM_FORMAT.md
    - docs/README.md
    - docs/structure.md
    - docs/development/specification-first-workflow.md
  typespec: []
  source:
    - tools/check/src/specification-doc.ts
    - tools/check/src/canonical-document-set.ts
    - tools/workspace/src/workspace.ts
    - tools/render-spec-docs/src/main.ts
  tests:
    - tools/check/src/specification-doc.test.ts
    - tools/check/src/canonical-document-set.test.ts
    - tools/workspace/src/workspace.test.ts
    - tools/render-spec-docs/src/render.test.ts
  stop_before_reading:
    - backend
    - frontend/src
spec_impact:
  kind: none
  reason: "文書の配置と日本語表記だけを変更し、既存の規範 ID、要求の意味、TypeSpec 契約、目標値は変更しない。"
---

# システム要求から設計と検証へ分解する文書体系を定め、ガイドと実際の文書を移行する

## Motivation

現在の文書体系はアプリケーションの Bounded Context を中心に構成されている。`docs/README.md` は全体文書の対象を、複数のコンテキストが同じ従い方をする仕様と設計に限定する。この分類だけでは、利用目的、外部システム、インフラ、ネットワーク、運用体制を含むシステム全体から、要求を各構成要素へ割り当てる順序が見えない。

`DOCUMENTATION_GUIDE.md` §1 はアーキテクチャ文書や品質要件文書を設けないと定め、§2 は現在の仕様と設計を直下へ配置し、`architecture/` などの階層を禁止している。`SPECIFICATION_FORMAT.md` と文書検査もこの平坦な配置を前提にする。トップダウンの体系を導入するには、これらの規則と実際の文書を一緒に変更する必要がある。

インフラに関する情報がすべて欠落しているわけではない。`docs/architecture/deployment.md` には実行単位、水平拡張、共有状態、ヘルスチェックがあり、`docs/design/performance/capacity.md` には品質目標と容量の前提がある。ただし、`docs/design/observability/README.md` は相関、指標、ログをまとめ、監視基盤自体の設計や運用の責任分界へ進む導線が弱い。ガイドが SLO と復旧目標の正本に指定する `reliability.md` と `recovery.md` は実際の直下文書にも許可ファイル集合にもなく、ガイドとリポジトリの配置も一致していない。

本項目の成果は、新しい目次の提案に加えて、ガイドへの反映、既存文書の移行、根拠のある不足記述の補完、継続して検査できる状態までとする。「完璧な構成」は、対象システムに適用する関心事の担当が一意に決まり、親要求から設計と検証をたどれ、欠落と未確定事項が識別できる構成として検証する。

## Scope

- SWEBOK、SEBoK、ISO/IEC/IEEE 42010、ISO/IEC/IEEE 15289、ISO/IEC 25010 の一次資料に基づき、文書体系の分類原則、階層、各文書の責務、最小記載項目、参照規則、作成順序を定める。
- `DOCUMENTATION_GUIDE.md` をプロダクト非依存の文書体系として再構成する。機能要求と非機能要求、アーキテクチャ、機能設計、データ、インフラ、ネットワーク、セキュリティ、可用性、冗長性、災害復旧、拡張性、性能、監視、ログ、開発、運用、検証を配置する。
- idmagic の現在の正準文書、`infra/` 以下の説明文書、開発文書、運用手順を棚卸しし、節単位で移行先を決め、実際に移行する。`docs/README.md` と `docs/structure.md` を同期する。
- ソース、構成資材、既存仕様、実際に得られた検証結果から確認できる設計を補完する。確認できない本番環境の構成や測定値には、必要な調査と判断を明記する。
- `SPECIFICATION_FORMAT.md`、開発ワークフロー、必要な範囲の `WORK_ITEM_FORMAT.md`、`AGENTS.md` とリポジトリ内スキルを新体系へ同期する。
- 文書探索、形式検査、規範差分、要求の参照解決、SLO 参照、HTML 生成を新階層へ対応させる。既存の要求 ID、TypeSpec シンボル、監視資材、作業項目からの参照を維持する。
- 大きな調査、設計判断、実装、実環境試験が必要な不足は、簡潔な現状と制約を正準文書に残し、既存または新規の作業項目へ割り当てる。

## Out of Scope

- クラウド事業者、リージョン数、ネットワーク製品、監視製品などの新規採用、本番環境の操作、インフラの配備、アプリケーションの振る舞いの変更。
- 根拠のない SLO、SLA、RPO、RTO、負荷条件、容量、費用の新設や既存目標の変更。必要な意思決定は、対象要求を参照する別項目で扱う。
- 高可用性構成の構築、負荷試験環境の用意、障害注入、復旧演習、未実装の計装の追加。本項目は、それらの設計文書上の位置、現状、未検証の範囲、受入条件を明らかにする。
- ISO 規格への正式な適合宣言や認証取得。公開資料で確認できる概念と、本リポジトリで選ぶ文書構成を区別する。
- 独立した ADR 保管庫、アーキテクチャ台帳、手書きの構成資産一覧、独自の仕様言語の追加。文書の階層化に合わせたソースコードや Bounded Context の再編。
- 既存の詳細仕様を別の機能仕様書へ複製すること、文書数やページ数を埋めるためだけの空ファイルの作成。

## Design

### Source basis

次の資料を出発点とし、実装着手時に採用する版、参照節、借りる概念を確認する。調査時点は 2026-09-08 とする。規格の公開概要から確認した範囲を超えて、特定のディレクトリ名やテンプレートが規格で義務付けられているとは記述しない。

| Source | Role in this change |
| --- | --- |
| [IEEE Computer Society, SWEBOK Guide V4](https://www.computer.org/education/bodies-of-knowledge/software-engineering?source=resources) と [Software Engineering Operations](https://www.computer.org/resources/software-operations-guide/) | 要求、アーキテクチャ、設計、テスト、運用、保守、構成管理、品質、セキュリティ、開発管理の関心事を点検する。知識領域の目次をそのまま文書ツリーに変換しない。 |
| [SEBoK, System Requirements Definition](https://sebokwiki.org/wiki/System_Requirements) と [System Detailed Design Definition](https://sebokwiki.org/wiki/System_Detailed_Design_Definition) | 利害関係者の必要性からシステム要求を導き、システム要素へ再帰的に割り当て、設計と検証につなぐ順序を参考にする。 |
| [ISO/IEC/IEEE 42010:2022](https://www.iso.org/standard/74393.html) | アーキテクチャ記述、関心事、視点、ビュー、その対応関係を整理する。文書の構成方法そのものは本ガイドで定める。 |
| [ISO/IEC/IEEE 15289:2019](https://www.iso.org/standard/74909.html) | ライフサイクルを通じた情報項目の目的と内容を点検し、設計、手順、計画、記録の役割を区別する。 |
| [ISO/IEC 25010:2023](https://www.iso.org/standard/78176.html) | ソフトウェアだけでなく ICT 製品を対象とする品質モデルを、品質要求の適用性と抜けの点検に使う。版による用語の違いを混ぜない。 |

SWEBOK は知識領域の確認、SEBoK はシステムとしての分解、42010 はアーキテクチャの表現、15289 は情報項目の区別、25010 は品質要求の点検に用いる。ガイドではこの使い分けと出典を説明し、idmagic 固有の構成や数値は実際の文書に置く。

### Decomposition and ownership

上位の作成順序は「目的と利用状況 → システム要求と制約 → アーキテクチャと要求の割当 → 各領域の設計 → 実現と検証 → 運用と保守」とする。実現可能性や試験から新たな制約が分かった場合は、親要求と設計へ戻って更新する。トップダウンは読む順序と責務の分解を定めるもので、工程の一回限りの実施を要求しない。

ツリーは正本の所有関係を示し、領域をまたぐ関係はリンクで表す。可用性、セキュリティ、性能の要求は複数の構成要素に影響するため、要求をインフラやアプリケーションの各枝へコピーすると、同じ数値と条件の正本が増えてしまう。品質要求は一か所で定義し、アーキテクチャで割当を示し、設計はその要求をどう満たすかを記述する。

機能と非機能は要求を分類する軸であり、インフラとネットワークは実現する対象、監視とログは運用を支える仕組みである。これらを要求の同じ階層の兄弟として並べない。設計の各枝は、同じシステムに対して答える問いが異なる設計領域として定義する。

### Target document tree

次の物理配置を採用する。`docs/contexts/` と `spec/contexts/` の対応は保ち、アプリケーション設計の索引から既存の詳細仕様へ進めるようにする。ツリー中のファイルは各領域の正本候補を明示したものであり、対象外の領域では親の索引に理由を記し、空ファイルは作らない。

```text
docs/
├── README.md                         # システム文書全体の入口と読み順
├── product-overview.md               # 目的、利用者、利用状況、対象範囲
├── glossary.md                       # システム共通語彙
├── standards.md                      # 採用する外部規範
├── structure.md                      # リポジトリ配置と実装構造への案内
├── requirements/
│   ├── README.md                     # 要求の分類、出典、詳細仕様への案内
│   ├── functional.md                 # システム機能の分解と担当の割当
│   ├── quality.md                    # 非機能要求、品質目標、測定境界
│   └── constraints.md                # 外部条件、制約、適用する運用環境
├── architecture/
│   ├── README.md                     # 関心事、視点、ビュー間の対応
│   ├── system-context.md             # 外部システム、利用者、責任分界
│   ├── logical.md                    # 機能分割、Context Map、データ所有
│   ├── runtime.md                    # 実行単位、相互作用、主要処理の流れ
│   ├── deployment.md                 # 実行単位の配置と環境別の構成
│   └── decisions.md                  # 現在有効な全体設計判断
├── design/
│   ├── README.md                     # 設計領域の責務と相互参照
│   ├── application/                  # 機能設計とソフトウェア設計
│   │   ├── README.md                 # contexts と UI 設計への案内
│   │   ├── api-rules.md              # TypeSpec を補う契約の規則
│   │   ├── design-rules.md           # モジュール、型、依存、効果の設計規則
│   │   └── user-interface.md         # 画面、操作、アクセシビリティ、国際化
│   ├── data/                         # データ設計
│   │   ├── README.md
│   │   ├── database.md               # 永続化、整合性、トランザクション
│   │   └── lifecycle.md              # 保持、削除、移行、データ分類
│   ├── infrastructure/               # 実行基盤の設計
│   │   ├── README.md
│   │   ├── platform.md               # 計算資源、ストレージ、環境差、IaC
│   │   └── network.md                # 通信経路、名前解決、入口と出口、隔離
│   ├── security/                     # 保護機構の設計
│   │   ├── README.md
│   │   ├── threat-model.md           # 資産、信頼境界、脅威、制御
│   │   ├── authorization.md          # 主体、認証と認可の責任分界
│   │   └── secrets.md                # 秘密、鍵、証明書の管理機構
│   ├── reliability/                  # 障害を扱う設計
│   │   ├── README.md
│   │   ├── availability.md           # 障害モデル、冗長性、切替、縮退
│   │   └── recovery.md               # バックアップ、復元、災害復旧
│   ├── performance/                  # 資源と負荷を扱う設計
│   │   ├── README.md
│   │   ├── capacity.md               # 負荷の前提、資源見積もり、費用の前提
│   │   └── scaling.md                # 拡張単位、律速点、過負荷保護
│   └── observability/                # システムを観測する設計
│       ├── README.md                 # 相関、信号間の関係、共通の設計規則
│       ├── monitoring.md             # 監視対象、指標、検知、通知
│       ├── logging.md                # ログの生成、収集、保存、検索、保護
│       └── tracing.md                # トレース伝播、収集、標本化
├── contexts/<context>/              # 既存の種類別詳細仕様を維持
├── scenarios.feature.md             # システム横断の規範シナリオを維持
├── verification/
│   ├── README.md                     # 要求ごとの検証方法と証拠への案内
│   └── system-acceptance.md          # 統合、非機能、障害、復旧の受入設計
├── development/                     # 開発環境、工程、CI、テスト、リリース
├── operations/
│   ├── README.md                     # 運用責任と手順への案内
│   ├── service-management.md         # SLO 評価、当番、障害、変更、保守
│   └── maintenance.md                # 定期作業、更新、廃止と引渡し
├── runbooks/                        # 手を動かす運用手順
└── releases/                        # 利用者向けの変更と移行の告知

spec/contexts/<context>/             # モデル、API、認証の機械可読契約
infra/                              # 配備と基盤の構成資材
work-items/                         # 変更固有の分析、計画、証拠、残課題
```

### Boundaries and coverage

| Concern | Canonical responsibility | References and exclusions |
| --- | --- | --- |
| 機能要求と機能設計 | `requirements/functional.md` は利用者の目的から主要機能へ分解し、`architecture/logical.md` が担当を割り当てる。詳細は既存のコンテキスト文書が持つ。 | 既存の規範本文と TypeSpec のフィールド一覧を上位文書へ再掲しない。UI は `design/application/user-interface.md` が共通設計を持つ。 |
| 非機能要求 | `requirements/quality.md` が品質目標、測定境界、適用環境、検証方法を持つ。 | 性能だけでなく、信頼性、セキュリティ、互換性、利用と操作、保守、変更と移行、安全性などを採用版の品質モデルで点検する。対象外にも理由を残す。 |
| 全体構造と配置 | `architecture/` が論理構成、実行時の協調、配置の対応を持つ。 | `structure.md` はコードと文書の配置に絞る。実行単位の詳細な資産一覧はコードと構成資材から導く。 |
| インフラとネットワーク | `design/infrastructure/` が環境、計算資源、ストレージ、名前解決、経路、ポート、TLS 終端、出口制御、管理経路を持つ。 | セキュリティ設計の信頼境界を参照し、構成値は資材を参照する。事業者が提供する範囲と運用者が用意する範囲を明記する。 |
| 可用性と冗長性 | `design/reliability/availability.md` が障害単位、状態の共有、複製、切替、単一障害点、縮退時の動作を持つ。 | 冗長性は可用性を実現する手段として記述する。複製だけで論理破損や災害からの復元が成立するとは扱わない。 |
| 復旧 | `requirements/quality.md` が既存 RPO/RTO の目標と条件を、`design/reliability/recovery.md` が実現方式と復元順序を持つ。 | 実行手順は `runbooks/`。バックアップ対象に鍵と構成を含めて点検し、目標と演習結果を区別する。 |
| 拡張性、容量、性能 | `design/performance/` が負荷分布、資源予算、レイテンシーの配分、律速点、拡張、同時実行、待ち行列、再試行、受付制限を持つ。 | 目標は `requirements/quality.md`。障害時の縮退順序は可用性設計を参照し、性能設計にはその適用機構を書く。費用の未算定を架空の試算で埋めない。 |
| 監視とログ | `design/observability/` がアプリケーション、基盤、ネットワーク、外部依存、観測基盤自体の観測を持つ。 | 検知条件と通知経路は監視設計、発報後の判断と責任分界は運用管理、実作業は運用手順。ログの保持、個人情報の除外、閲覧権限、容量超過、収集停止も扱う。 |
| 監査と診断ログ | 監査イベントの意味と保持要求は監査コンテキスト、診断ログの出力と収集方式はログ設計が持つ。 | イベント名と公開属性は既存 TypeSpec を参照する。監査証跡を診断ログで代替しない。 |
| データとセキュリティ | データ設計は所有、整合性、ライフサイクル、セキュリティ設計は資産の保護と信頼境界を持つ。 | 同じ保持期間や制御を複数の正本に定義しない。各コンテキストの鍵や認可の詳細を参照する。 |
| 開発、検証、運用 | `development/` は作り方と開発手順、`verification/` はシステム要求の受入設計、`operations/` は稼働後の管理を持つ。 | テストの実装はテストソース、実行結果は作業項目や CI の記録、個別リリースの変更説明は `releases/` が持つ。 |

### Minimum content and evidence

ガイドは各設計領域について、親要求と対象範囲、入力となる前提、構成と相互作用、正常時と障害時の動作、現在の判断と理由、検証方法、制約と未確定事項という最小記載項目を定める。領域固有の項目は上表をもとに追加し、意味のない共通欄の埋め合わせは要求しない。

現在の目標、設計上の仮定、実装済みの構成、測定や演習の結果、将来の案を区別する。既存の `Specification target`、`Planning assumption`、`Measurement` の意味を保ち、本番環境への適用を確認していない参照構成はそのように記す。構成資材に存在することだけでは、稼働や障害耐性の検証済み証拠としない。

未確定事項には、判明している現状、欠けている判断または証拠、影響する要求と設計、確定に必要な情報、対応する作業項目を持たせる。単独の `TBD` や見出しだけでは補完済みと数えない。適用対象外の場合は、システムの境界や利用条件に基づく理由を親文書へ記載する。

### Migration and reference preservation

移行は節単位で行う。`deployment.md` の実行単位は実行時アーキテクチャ、配置は配備アーキテクチャ、冗長性は可用性設計、HTTP 保護は対応するアプリケーションまたはセキュリティ設計へ分ける。`capacity.md` の SLO と受入目標は品質要求へ、負荷の前提と資源算出は容量設計へ移す。`observability.md` は相関、監視、ログの所有先へ分ける。

既存の `REQ-*`、`EX-*`、`SLO-*`、`CAP-*` と TypeSpec シンボルを改番しない。配置だけを変えた場合に規範の削除と新設として誤判定しないよう、着手前の基準コミットに対して差分を確認する。作業項目の `affected_spec` は完了済みも含めて参照先を追随させる。過去の検証結果や完了時の判断は書き換えない。

本文を新旧二か所に残す移行は行わない。リポジトリ内のリンクとアンカーを更新し、公開済み URL が判明した場合のみ、旧参照先に内容を複製しない転送案内を置く。新階層は Markdown と生成 HTML の双方で入口から到達できるようにする。

### Deferred work and alternatives

既存の関連項目は、本項目の着手を待たせる依存ではなく、不足の引継ぎ先として扱う。移行時に本文と現在状態を再確認し、文書整理と実装の担当範囲を明示する。

| Work item | Boundary |
| --- | --- |
| [wi-165](../wi-165-high-availability-and-failover-resilience-topology.md) | 高可用性トポロジの具体化、切替、障害試験。 |
| [wi-164](../wi-164-data-tier-scalability-partitioning-read-replica-pooling.md) | データ層の拡張方式と実装。 |
| [wi-282](../wi-282-staging-load-testing-and-capacity-validation.md) | 実負荷による容量と性能の検証。 |
| [wi-107](../wi-107-opentelemetry-distributed-tracing.md) | 未実装のトレース伝播と計装。 |
| [wi-290](../wi-290-alert-runbook-catalog-and-on-call-operations.md) | アラートに対応する運用手順と索引の完成。 |
| [wi-419](../wi-419-quantification-beyond-performance.md) | 品質の量化と予算消費時の方針。新体系への配置変更と新しい要求の決定を分ける。 |
| [wi-449](../wi-449-deployment-update-compatibility-preflight.md)、[wi-450](../wi-450-mixed-version-release-acceptance.md) | 更新前検査と異なる版が混在する配備の受入試験。 |

既存項目で扱えない不足だけを追加起票する。新規項目には親要求または適用する規範、成果物、完了条件、依存を記し、本項目の完了を待つ必要がある場合にのみ `depends_on` を設定する。正準文書は現在の制約と設計を持ち、具体的な未決事項と対応先の対応表は本項目に残す。

却下する案は次のとおりとする。

- 平坦な文書のまま目次だけを階層化する案。要求と設計の異なる内容が同じファイルに残り、今回の分類と責務の問題を解消しない。
- SWEBOK の知識領域を同名のディレクトリへ写す案。知識体系の分類と、個別システムの要求から設計を作る順序が一致しない。
- 非機能設計という一冊へインフラ、可用性、性能、監視を集める案。品質目標と複数領域の実現方式が混在し、元の大きな内部文書の問題が再発する。
- 品質属性ごとに全体構成を複製する案。各文書が配備構成やデータ所有を別々に持ち、変更時の整合維持が困難になる。
- テンプレートをすべて作って空欄を埋める案。未確定の設計が確定済みに見えるため、適用性と証拠を伴う最小記述で扱う。

## Migration record

着手時の基準は `main` の `3c9a4cee` とする。`docs/` 直下にあった 8 文書は、次の対応で節ごとに移した。移行先が複数ある文書は、節の所有先を分けた。

| 旧文書 | 節 | 移行先 |
| --- | --- | --- |
| `docs/api-rules.md` | 全節 | `docs/design/application/api-rules.md` |
| `docs/authorization.md` | 全節 | `docs/design/security/authorization.md` |
| `docs/threat-model.md` | 全節 | `docs/design/security/threat-model.md` |
| `docs/database.md` | 全節 | `docs/design/data/database.md`。保持と削除は `docs/design/data/lifecycle.md` |
| `docs/design-rules.md` | 全節 | `docs/design/application/design-rules.md` |
| `docs/capacity.md` | Evidence classes、Measurement boundary、Service level objectives | `docs/requirements/quality.md` |
| `docs/capacity.md` | Reference operating profile、Peak request profile、Sizing rules、Degradation order | `docs/design/performance/capacity.md` |
| `docs/deployment.md` | Runtime units、Domain event delivery | `docs/architecture/runtime.md` |
| `docs/deployment.md` | Horizontal scaling reference topology | `docs/architecture/deployment.md` と `docs/design/performance/scaling.md` |
| `docs/deployment.md` | Load shedding under saturation、Endpoint rate limiting | `docs/design/performance/scaling.md` |
| `docs/deployment.md` | Health probes and graceful drain、Availability and shared state | `docs/design/reliability/availability.md` |
| `docs/deployment.md` | HTTP server hardening、Security response headers | `docs/design/security/threat-model.md` と `docs/design/infrastructure/network.md` |
| `docs/observability.md` | Request correlation | `docs/design/observability/README.md` と `tracing.md` |
| `docs/observability.md` | Metrics | `docs/design/observability/monitoring.md` |
| `docs/observability.md` | Logging | `docs/design/observability/logging.md` |

`docs/README.md`、`docs/product-overview.md`、`docs/glossary.md`、`docs/standards.md`、`docs/structure.md`、`docs/scenarios.feature.md` は直下に残し、参照先だけを新しい配置へ更新した。`docs/contexts/`、`docs/development/`、`docs/runbooks/`、`docs/releases/` の配置は変えていない。

移行で本文が二重にならないよう、旧ファイルは削除し、`backend/`、`frontend/`、`infra/`、`load/` のコメントにある正本パスと、監視資材および完了済み作業項目からの参照を同じ変更で更新した。`REQ-*`、`EX-*`、`SLO-*`、`CAP-*`、TypeSpec シンボルは改番していない。

新設した文書のうち、`docs/design/observability/tracing.md`、`docs/design/reliability/recovery.md`、`docs/design/infrastructure/platform.md` は、確認できる現状と未確定の範囲を書き、実装と実測は [wi-107](../wi-107-opentelemetry-distributed-tracing.md)、[wi-165](../wi-165-high-availability-and-failover-resilience-topology.md)、[wi-282](../wi-282-staging-load-testing-and-capacity-validation.md) が持つものとして引継ぎ先を明記した。担当のない不足は見つからなかったため、新規の作業項目は起票していない。既存 Markdown 全体の表題と固定書式を日本語へ移す作業は、本項目の文書体系移行とは分離して扱う。

## Plan

1. 着手時の基準コミット、既存文書、検査と生成の状態を記録し、作業項目に着手時の必須属性を追加する。
2. 文書と構成資材を棚卸しし、既存の各節から新体系への移行表と、設計領域ごとの充足状況を本項目へ記録する。一次資料の版と概念を確認する。
3. 上記の責務と階層を `DOCUMENTATION_GUIDE.md` の規定、書式、作成順序へ反映し、続いてリポジトリの形式規約と開発ワークフローを更新する。
4. 新階層の文書を検査と生成が認識するようにする。規範を含む文書が探索から脱落する不具合を検出する試験を先に用意する。
5. 目的、要求、アーキテクチャ、領域別設計、検証、開発と運用の順に本文を移行し、不足を補完する。詳細なコンテキスト仕様は既存の正本へ接続する。
6. 既存の関連項目と不足を照合し、別項目へ委ねる内容を具体化する。新しい規範や目標が必要になった場合は、その変更を本項目の文書移行に混ぜない。
7. 参照の更新、規範差分の確認、生成サイトの確認、検査を行い、移行表と充足状況で完了条件を確認する。

Acceptance RED は、標準の文書検査が新しい固定階層にある正準文書を探索し、未知のファイルを拒否することを要求する `tools/workspace/src/workspace.test.ts` の試験で記録する。現在の探索は `docs/` 直下と `contexts/` だけなので失敗する。

Unit RED は、`documentKind` が新しい正準パスを文書種別へ解決し、固定ディレクトリごとの許可ファイル集合が入れ子の誤配置を拒否することを要求する `tools/check/src/specification-doc.test.ts` と `canonical-document-set.test.ts` の試験で記録する。プロダクトの受入境界と単体境界は本項目に適用されないため、この二つを代替証拠とする。

階層、責務の分離、コンテキスト配置の維持、規範を変更しない移行という設計境界は本項目で選択済みとする。実装着手後に境界の変更が必要になった場合は、ガイドと形式規約の設計へ戻り、移行を先に進めない。

## Tasks

- [x] T001 [Design] 既存文書と基盤資材を棚卸しし、節単位の移行表と領域別の不足を本項目へ記録する。
- [x] T002 [Design] 一次資料の版、該当箇所、採用する概念を確認し、要求と設計領域の網羅性を点検する。
- [x] T003 [Spec] `DOCUMENTATION_GUIDE.md` を新体系へ改訂し、責務、ツリー、最小書式、参照規則、作成順序、適用対象外と未確定事項の扱いを定める。
- [x] T004 [Spec] `SPECIFICATION_FORMAT.md`、開発ワークフロー、必要な作業項目規約、エージェント向け規則とスキルを同期する。
- [x] T005 [Acceptance] 新階層の規範文書、破損した参照、生成サイトから脱落した設計文書を検出する検証を用意し、変更前の失敗を記録する。
- [x] T006 [Tools] 文書の種類と探索、形式検査、規範差分、参照解決、SLO 参照、HTML の階層表示を対応させる。誤った配置を検出する既存検査を保つ。
- [x] T007 [Docs] プロダクト概要、システムの境界、機能要求、品質要求、制約、アーキテクチャを移行して補完する。
- [x] T008 [Docs] アプリケーション、データ、インフラ、ネットワーク、セキュリティの設計を移行して補完する。
- [x] T009 [Docs] 可用性、冗長性、復旧、性能、容量、拡張性、監視、ログ、トレースの設計を移行して補完する。
- [x] T010 [Docs] システム受入設計、開発と運用の文書を整理し、正準設計、運用手順、実行証拠の参照をつなぐ。
- [x] T011 [Docs] 大きな不足を既存項目へ割り当て、担当のない不足だけを新規起票する。現状と制約を正準文書へ残す。
- [x] T012 [Docs] `docs/README.md`、`docs/structure.md`、各入口文書、作業項目、監視資材などからの参照を更新し、旧本文の重複を除く。
- [x] T013 [Verify] 規範差分、参照整合、領域の充足状況、生成 HTML の到達性、代表的な誤変更の検出を確認する。
- [x] T014 [Verify] 所定の検査を通し、完了記録を追加して `work-items/done/` へ移す。

## Verification

起票時は `mise run check-work-items` と `mise run check-ids` を通す。実装完了時は以下を満たす。

- ガイドの構成を実際の文書が満たす。適用する各設計領域には、本文のある正本、親要求、関連設計、検証方法があり、未確定の部分には不足の内容と引継ぎ先がある。
- 棚卸ししたすべての節が、移行、維持、根拠付きの統合または廃止のいずれかになっている。重要な制約や運用手順を移動に伴って落としていない。
- ログイン、外部連携、管理操作の代表的な機能と、可用性、性能、ログ保護、復旧の代表的な品質要求について、入口から要求、設計、実装または資材、検証方法までをたどれる。
- 同じ SLO、容量目標、RPO/RTO、保持期間、信頼境界、規範本文を別の文書へ再定義していない。参照のための図や要約は正本へのリンクを持ち、新たな義務を加えない。
- 本番適用済み、実装済み、試験済み、参照構成、設計上の仮定、未確定事項の区別が、根拠と一致する。高可用性や性能を本文だけで保証済みにしていない。
- `mise run spec-diff` を着手時の基準に対して確認し、規範的な意味の差分がない。文書の移動を理由に要求を削除扱いにしたり、検査対象から除外したりしていない。
- `mise run check-spec`、`mise run check-links`、`mise run check-work-items`、`mise run check-ids`、`mise run check-slo-references`、`mise run check-agent-guidance` を通す。
- `mise run spec-render` で TypeSpec と HTML の未追跡成果物を再生成し、新しい各領域を入口から開けること、要求とコンテキストと API の参照が解決することを確認する。
- 検査と生成を変更した場合は `mise run test-tools` と `mise run verify-spec` を通す。入れ子の規範文書を探索から外す、要求参照を壊す、SLO の正本を見失わせるという代表的な誤変更を検査が検出することを記録する。
- `mise run verify` を通す。文書本文の質や分類の適切さは機械検査だけで保証できないため、移行表と領域別の充足確認を別に記録する。

## Risk Notes

新しい階層が検査や生成の対象から漏れると、文書は存在していても規範の検証が実行されなくなる。`tools/check/src/specification-doc.ts`、`tools/check/src/canonical-document-set.ts`、`tools/workspace/`、`tools/render-spec-docs/` などの探索経路を確認し、ファイル名の許可だけでなく標準タスクからの到達性を試験する。SLO 検査には旧 `docs/capacity.md` の固定参照があるため、正本の移動と同時に対応する。

分類軸を細かくしすぎると、複数文書を開かないと一つの判断が理解できない構成になる。各設計領域の索引に対象、正本、関連領域を示し、機構とその理由は同じ正本に置く。上位文書は詳細を複写せず、要求の割当と設計間の関係を説明する。

目標値の移動は意味の変更を隠しやすい。測定境界、母集団、時間窓、単位、適用環境、証拠区分を値と一緒に保持し、単なる文書整理を理由に達成水準を引き下げない。

不足の補完を急ぐと、未確認の本番構成や根拠のない設計を事実として書いてしまう。確認できる現状と確定に必要な作業を分け、別項目への委譲も完了条件と対応先を伴わせる。新体系の導入自体を、関連するすべてのインフラ実装の完了まで待たせない。

文書移動は完了済み作業項目や運用資材の参照にも及ぶ。本文の所有先を先に確定し、参照更新と検査を同じ移行単位で行う。過去の記録は参照修復に必要な範囲だけを変更し、当時の結果を現在の設計へ書き換えない。

## Completion

- **Completed At**: 2026-09-08
- **Summary**:
  要求、アーキテクチャ、領域別設計、検証、開発、運用へトップダウンに分解した文書体系を定め、既存文書を実際の階層へ移行した。
  インフラストラクチャ、ネットワーク、信頼性、復旧、性能、容量、スケーリング、監視、ログ、トレースを独立した責務として補完し、親要求と検証方法への参照を設けた。
  文書探索、正準ファイル集合、SLO 参照、生成 HTML を入れ子の配置へ対応させた。
  生成 HTML はサイドバーの不要な一段を除き、コンテキスト別ナビゲーションを既定で閉じ、API リファレンスを全幅化した。
  Swagger UI は OpenAPI を Blob URL から読み込むことで `file://` 表示時の内部参照を解決し、固定 UI 文言を日本語化した。
  日本語の使用範囲は `.claude/rules/japanese-writing.md` に集約し、`AGENTS.md` は mise、ツール、参照先だけに縮小した。
  既存 Markdown と TypeSpec の大量の英語本文は [wi-515](../wi-515-localize-existing-markdown-and-typespec-prose.md) に分離した。
- **Acceptance RED Evidence**:
  - **Test**: `discoverWorkspaceConfig > discovers the standard layout without a registry file`（`tools/workspace/src/workspace.test.ts`）
  - **Requirement**: N/A: 製品機能ではなく、正準文書を標準タスクから探索できるというリポジトリ内の文書基盤を検証する。
  - **Observed Failure**: 新設した `requirements/`、`architecture/`、`design/`、`verification/`、`operations/` が探索結果に含まれず、期待した正準文書集合と一致しなかった。
  - **Detection Reason**: 実際の標準配置を一時ワークスペースに構築し、検査入口が返す全文書を比較するため、ファイルだけ存在して検査対象から脱落する欠陥を検出する。
- **Unit RED Evidence**:
  - **Test**: `documentKind > classifies nested whole-system documents`（`tools/check/src/specification-doc.test.ts`）と `verifyCanonicalDocumentSet > holds each level to its own set of names`（`tools/check/src/canonical-document-set.test.ts`）
  - **Requirement**: N/A: 製品機能ではなく、文書パスの分類と階層ごとの閉じた正準ファイル集合を検証する。
  - **Observed Failure**: 入れ子の正準パスを文書種別へ分類できず、同じファイル名を誤った階層へ置いた場合も階層別の許可集合で拒否できなかった。
  - **Detection Reason**: パスの分類とディレクトリごとの許可集合を別々に検査するため、新しい枝の追加漏れと誤配置を区別して検出する。
- **Change-Resistance Results**:
  旧来の平坦な探索実装に対して、上記 Acceptance と Unit の試験が失敗することを確認した。
  さらに、利用者から報告された生成 HTML の不具合について、開発文書とシステム文書の階層平坦化、コンテキスト別ナビゲーションの初期状態、Swagger UI の Blob URL 読込み、全幅表示を個別の表明に分け、変更前には 5 件が失敗したことを記録した。
  各表明は担当する出力断片を直接検査するため、固定階層の削除、`spec` による埋込みへの逆戻り、余白または既定展開の再導入を検出する。
- **Verification Results**:
  - `mise run format-tools` - passed
  - `mise run test-tools` - passed（459 件）
  - `mise run check-links` - passed（747 文書）
  - `mise run check-work-items` - passed（513 件、完了前の状態）
  - `mise run check-ids` - passed（513 件）
  - `mise run check-agent-guidance` - passed
  - `mise run spec-render` - passed（180 文書、336 操作、19 API タグ、889 TypeSpec シンボル、1073 ページ）
  - `mise run spec-diff` - passed（29 シナリオの「リクエスト本文／レスポンス本文」を「リクエストボディ／レスポンスボディ」へ統一した文章差分だけを確認）
  - `mise run verify-spec` - passed
  - `mise run verify` - passed（Go の競合検査と 683 件の UI 単体試験を含む）
  - `git diff --check` - passed
