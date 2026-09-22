---
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-16
priority: p1
depends_on: [wi-583-normalize-design-document-terminology]
change_kind: docs
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: "既存の設計文書の記述粒度を上げるだけで、利用者が観測する振る舞い、設定項目、移行手順はどれも変わらない。"
  references: []
spec_impact:
  kind: none
  reason: "規範 ID、TypeSpec のシンボル、設定キー、プロダクトの振る舞いは変えない。spec-diff は REQ-SYSTEM-001、REQ-SYSTEM-016、REQ-SYSTEM-017、REQ-IDMANAGEMENT-009、REQ-OAUTH2-048、REQ-PROVISIONING-005、REQ-TENANCY-021、RFC8693-DELEGATION-DEFAULT、RFC8693-DELEGATION-DEPTH、RFC7643-OUT-CORE-RESOURCES、RFC7643-OUT-GROUP-RESOURCES を挙げるが、変わったのは「既定」から「デフォルト」への表記と、REQ-SYSTEM-001 の Given にあった実在しない実行単位名を実在の名前へ直したことだけで、条件も要求する結果も同じである。"
initial_context:
  specification:
    - docs/design/architecture/deployment.md
    - docs/design/architecture/system-boundary.md
    - docs/design/architecture/runtime.md
    - docs/design/infrastructure/README.md
    - docs/design/infrastructure/platform.md
    - docs/design/infrastructure/network.md
    - DOCUMENTATION_GUIDE.md
  typespec: []
  source:
    - infra/docker/docker-compose.dev.yaml
    - infra/k8s/base
    - infra/k8s/overlays/prod
    - infra/deploy/gcp
    - infra/README.md
    - frontend/Caddyfile
    - tools/check/src/terminology.ts
  tests: []
  stop_before_reading:
    - backend
    - spec
    - infra/schema/postgres.sql
---

# デプロイとインフラの設計を Docker Compose と Google Cloud の具体で書く

## Motivation

[システムコンテキスト](../../docs/design/architecture/system-boundary.md)の図には「ゲートウェイ」と「デプロイ、シークレット、監視のプラットフォーム」が現れるが、それが何であるかの記述がどこにもない。
IdMagic 単体では確定しない部分だが、`infra/docker/`、`infra/k8s/`、`infra/deploy/gcp/` に三つの具体があり、Google Cloud なら Cloud Load Balancing と Cloud Run、ローカルなら Caddy と Docker Compose だと書ける。

[デプロイアーキテクチャ](../../docs/design/architecture/deployment.md)は共通トポロジーを ASCII で描いており、同じリポジトリの他文書が Mermaid を使っているのと揃っていない。
本文も「ゲートウェイ」「ワーカー」の抽象のままで、Docker Compose と Google Cloud で何がその役を担うかを書いていない。

[プラットフォーム設計](../../docs/design/infrastructure/platform.md)は 23 行、[ネットワーク設計](../../docs/design/infrastructure/network.md)は 22 行で、計算資源、ストレージ、入口、TLS 終端、名前解決、シークレット供給のいずれも、環境ごとの具体を持たない。

一方で `infra/deploy/gcp/README.md` は、Cloud Run Service と worker pools の使い分け、Cloud SQL の REGIONAL、Cloud Armor、Secret Manager、デプロイ工程での `psqldef` 適用といった設計判断を持っている。
設計判断が構成ファイルの隣にあり、`docs/` から到達できない。

## Scope

- デプロイプロファイルを三つ定義する。ローカル Docker Compose、汎用 Kubernetes、Google Cloud。
- [デプロイアーキテクチャ](../../docs/design/architecture/deployment.md)の共通トポロジーを Mermaid にし、プロファイルごとに実行単位と配置先の対応を書く。
- [システムコンテキスト](../../docs/design/architecture/system-boundary.md)の「ゲートウェイ」と「プラットフォーム」が、各プロファイルで何であるかを書く。
- [プラットフォーム設計](../../docs/design/infrastructure/platform.md)に、計算資源、ストレージ、シークレット供給、スキーマ適用、スケール単位をプロファイル別に書く。
- [ネットワーク設計](../../docs/design/infrastructure/network.md)に、入口、TLS 終端、名前解決、送信先の許可、管理接続をプロファイル別に書く。
- `infra/deploy/gcp/README.md` が持つ設計判断を `docs/` へ移し、構成ファイル側は手順と値に寄せる。

## Out of Scope

- 新しい overlay、Terraform、Cloud Run の設定変更。
- 本番環境の確定。どのデプロイプロファイルを採用するかはこのリポジトリが決めない。
- 可用性とフェイルオーバーの設計。[[wi-588-reliability-and-performance-design]] が扱う。
- 監視基盤の構成。[[wi-589-observability-design]] が扱う。
- （仮）とした項目の構成ファイルへの反映。汎用 Kubernetes の PostgreSQL と Job、GKE 向けの overlay、イメージへの `idmagic-seed` の追加もこれに含む。本 work item は案を文書へ置くところまでを担い、Terraform、overlay、Cloud Run の設定は書かない。
- 設計書からの構成ファイルの生成。本書は決定を持ち、値は構成ファイルが持つという [DOCUMENTATION_GUIDE.md](../../DOCUMENTATION_GUIDE.md) §2 の割り当てを維持するため、この文書から `docker-compose.dev.yaml`、Kubernetes マニフェスト、Terraform は生成できない。一次情報源をパラメーター定義へ移す作業は [[wi-592-generate-infrastructure-materials-from-parameter-definitions]] が持つ。

## 途中で広げた範囲

実装中の指摘により、当初の Out of Scope から次を範囲へ移した。

- **Docker Compose の `idp` サービス名の変更**。IdP 以外の機能も持つ実行単位に `idp` という名は合わない。[ランタイムアーキテクチャ](../../docs/design/architecture/runtime.md)が宣言する実行単位名に合わせて `api` へ変更した。構成ファイルの変更を含まないという当初の線は、この一点について外した。
- **費用見積もりの移動**。`infra/deploy/gcp/README.md` が持っていた費用表を[プラットフォーム設計](../../docs/design/infrastructure/platform.md)へ移した。見積もりの値を新たに確定したわけではなく、置き場所を判断の側へ移しただけである。
- **決めるべき論点の追加と仮決定**。プロファイル別の観点表だけでは、決めるべき論点そのものが欠けていた。リソース階層、IAM、イメージのサプライチェーン、監査、セグメンテーション、通信の許可規則を項目として追加し、状態を確定・仮決定・委譲・未決定の四値で示した。未決定を並べるだけでは反対する対象が無いため、外部の入力を待たない項目はこの文書の中で仮決定した。
- **規範シナリオの訂正**。`docs/domain/system/scenarios.feature.md` の EX-SYSTEM-001-01 から 03 の Given が、実行単位として存在しない「イベントリレー」を宣言していた。[ランタイムアーキテクチャ](../../docs/design/architecture/runtime.md)が宣言する実行単位名へ揃えた。振る舞いの要求は変えていないので `REQ-SYSTEM-001` は据え置く。`infra/README.md` の同じ記述と、構成ファイルと食い違っていた本番レプリカ数の記述も直した。
- **Google Cloud プロファイルのコンピュートを GKE Autopilot へ変更**。Cloud Run では、実行レーンを PostgreSQL のキューの滞留で増減できない（worker pools の自動スケールは CPU か Pub/Sub の滞留だけ）、HPA の振る舞いと PodDisruptionBudget を表現できない、停止猶予が 10 秒で固定される、`/metrics` のプルにサイドカーが要る、経路の許可リストが URL マップと二重になる。汎用 Kubernetes の構成ファイルが表現する設計をそのまま動かせる GKE Autopilot を採り、エッジは Cloud CDN に対応する GKE Ingress とした。Cloud Run のひな型は採らなかった案として `infra/deploy/gcp/` に残し、README をその位置付けへ書き換えた。費用表は Pod の要求量から見積もり直した。
- **用語の具体化**。「計算資源」「シークレット供給」「信号の収集」「区画」のような、概念が普通名詞へ吸収される語を、コンピューティング、シークレットの注入、メトリクスとログの収集経路、セグメンテーションへ改めた。途中で「コンピュート」と置いた語も、通じにくいため「コンピューティング」へ直した。
- **文書全体の用語の統一と検査への追加**。「資材」を「構成ファイル」（Kubernetes に限るなら「マニフェスト」）、「既定」を「デフォルト」へ、`docs/` と root の文書全体で置き換え、`tools/check/src/terminology.ts` の規則へ「参照トポロジー」「参照プロファイル」「既定」「コンピュート」「資材」を加えた。「既定」の置き換えは規範シナリオの文言にも及ぶが、要求の意味は変えていない。
- **汎用 Kubernetes プロファイルの不足の補完**。PostgreSQL とスキーマ適用を持たないままではプロファイル単体で IdMagic が動かないため、CloudNativePG によるデータベースと、`psqldef` と `idmagic-seed` の Job を（仮）として設計に加えた。

## Design

デプロイプロファイルは、実行単位と配置先の対応表を三つ並べる形にする。
行は[ランタイムアーキテクチャ](../../docs/design/architecture/runtime.md)が宣言済みの実行単位（API、Worker、Batch、Seed、フロントエンドゲートウェイ）に固定し、列にプロファイルを置く。
実行単位の一覧を新しく作らない。

各プロファイルについて、次の観点を同じ順序で書く。
観点を揃えることで、プロファイル間の差が読み取れる形になる。

| 観点 | 書くこと |
| --- | --- |
| 実行基盤 | 各実行単位を何が動かすか |
| 入口と TLS 終端 | 公開入口は何か、TLS をどこで終端するか |
| 静的アセット | SPA をどこから配信するか |
| 名前解決と証明書 | DNS と証明書を誰が所有するか |
| シークレット供給 | 起動時シークレットの供給元と注入方法 |
| スキーマ適用 | `psqldef` をどの工程で実行するか |
| データベース | PostgreSQL を何が提供するか |
| ログとメトリクス | 収集経路 |
| スケール単位 | 何を独立に増減できるか |
| 既知の制限 | このプロファイルが提供しないもの |

Google Cloud のプロファイルは、`infra/deploy/gcp/README.md` が既に持つ内容を設計として書き直す。
Cloud Load Balancing と Cloud CDN と Cloud Armor が公開入口、GCS + Cloud CDN が静的アセット、Cloud Run Service が API、Cloud Run worker pools が HTTP を持たない Worker、Cloud Run Job が Seed、Cloud SQL for PostgreSQL の REGIONAL がデータベース、Secret Manager と Cloud KMS がシークレットと鍵である。
Cloud Run の Service が `$PORT` への応答を要求するために Worker を worker pools へ置くという判断は、コードからは復元できないのでプラットフォーム設計が持つ。

Docker Compose のプロファイルは `infra/docker/docker-compose.dev.yaml` の現行サービス（`postgres`、`schema`、`otel-collector`、`prometheus`、`loki`、`alloy`、`grafana`、`idp`、`frontend`、`worker`）を根拠にする。
Caddy が同一オリジンでゲートウェイを担い、`schema` サービスがスキーマ適用を担い、Alloy が Docker Engine API からログを読む点が、他のプロファイルとの差になる。

**デプロイプロファイルは適用済み構成ではない。** 現在の文書が守っているこの区別は維持し、各プロファイルの冒頭で明示する。

構成ファイルと文書の責任分界を次のとおりにする。
判断と理由は `docs/` が持ち、コマンド、変数名、値、実行順は構成ファイルの README が持つ。
`infra/deploy/gcp/README.md` は設計の再掲をやめ、`docs/design/infrastructure/platform.md` を参照する。
根拠は [DOCUMENTATION_GUIDE.md](../../DOCUMENTATION_GUIDE.md) §2 の「人が読む文書は `docs/` に集める」である。

採らない案は、`docs/` から `infra/deploy/gcp/README.md` へリンクするだけで済ませる案である。
設計判断が構成ファイルの隣に散り、プロファイルどうしを同じ観点で比較できなくなる。

## Plan

1. 三つのプロファイルの観点表を、現行の構成ファイルから読み取って埋める。
2. デプロイアーキテクチャの共通トポロジーを Mermaid にし、プロファイル別の図を置く。
3. システムコンテキストのゲートウェイとプラットフォームに、プロファイル別の実体を与える。
4. プラットフォーム設計とネットワーク設計を観点順に書き直す。
5. `infra/deploy/gcp/README.md` から設計判断を移し、手順と値に寄せる。

## Tasks

- [x] T001 [Design] 三つのデプロイプロファイルの観点表を現行の構成ファイルから作る。
- [x] T002 [Docs] デプロイアーキテクチャの共通トポロジーを Mermaid へ移し、プロファイル別に書く。デプロイの不変条件に理由の列を付ける。
- [x] T003 [Docs] システムコンテキストのゲートウェイとプラットフォームを具体化する。
- [x] T004 [Docs] プラットフォーム設計をプロファイル別に書き直し、決めるべき論点を状態付きで並べる。
- [x] T005 [Docs] ネットワーク設計をプロファイル別に書き直し、決めるべき論点を状態付きで並べる。
- [x] T006 [Docs] `infra/deploy/gcp/README.md` の設計判断と費用表を `docs/` へ移す。
- [x] T008 [Terminology] 「参照」を含む語（reference topology、reference profile の訳）を「共通トポロジー」「デプロイプロファイル」へ改め、用語集へ登録する。未完了の他 work item の本文も揃える。
- [x] T009 [Infra] Docker Compose の `idp` サービスを `api` へ改名し、Caddy、Prometheus、Alloy、`infra/schema/README.md` の参照を追随させる。
- [x] T010 [Spec] 規範シナリオの Given から「イベントリレー」を除き、実行単位名を揃える。`infra/README.md` の同じ記述も直す。
- [x] T011 [Docs] 外部の入力を待たない未決定項目を、理由付きで仮決定する。状態を四値にし、仮決定が適用済みでないことを明示する。
- [x] T012 [Docs] 用語を具体化する。コンピュート、シークレットの注入、メトリクスとログの収集経路、セグメンテーション。
- [x] T013 [Docs] Mermaid 図を足す。リソース階層、リリースの流れ、受信の流れ、トークン取得の経路、セグメンテーション、プロファイル別の詳細。
- [x] T014 [Docs] 構成ファイルから生成できるかを基準に不足を洗い、値ではなく決定として欠けていたものを足す。命名とラベルの体系、プローブの種別と分ける理由、実行単位ごとに必要な操作、規則の評価順序と発信元の指し方、アドレス範囲の割り当て規則。
- [x] T015 [Docs] 本書から構成ファイルを生成できないことと、その一次情報源の移動を [[wi-592-generate-infrastructure-materials-from-parameter-definitions]] が持つことを明記する。
- [x] T016 [Docs] ネットワーク設計の語を技術用語へ改める。「通信の許可規則」をファイアウォールルール、「通信経路の規則」を共通の不変条件（同一オリジンへの集約と Egress のデフォルト拒否）、「公開入口」をエッジ、「〜の所有」を管理する場所や実施する主体へ。ファイアウォールルールの表を各プロファイルで何が実施するかの対応表を足し、Google Cloud の階層型ファイアウォールポリシー、VPC ファイアウォールルール、Cloud Armor、VPC Service Controls、Cloud Run の受信制限と呼び出し権限の分担を書く。
- [x] T018 [Docs] Google Cloud プロファイルを GKE Autopilot へ切り替える。選定理由と採らなかった Cloud Run 案、GKE 向けの overlay で変える箇所（NetworkPolicy の Cloud SQL と Managed Service for Prometheus とメタデータサーバー、ServiceAccount の分割、PodMonitoring、Ingress と BackendConfig と FrontendConfig）、エッジの判断（CDN が保存する応答、ヘルスチェックの経路、接続ドレイン）、証明書の制約（Google マネージド証明書はワイルドカード非対応）、三段のファイアウォールを書く。「egress」を英語のまま使う。worker pools を preview と書いていた誤りを直す。
- [x] T017 [Docs] 「ポートと管理アクセス」を、リスナーの公開範囲と管理アクセスの二節へ分けて中身を書く。API が `/metrics` を同じリスナーで出し、ゲートウェイの経路の許可リストがそれを公開経路から外していること、管理操作ごとの経路と記録、シェルを持たないイメージが診断に課す要求。
- [x] T019 [Docs] デプロイメントアーキテクチャの共通トポロジーを、構成要素名（`idmagic-frontend`、`idmagic-api`、`idmagic-worker`、`idmagic-batch`、`idmagic-seed`、`psqldef`）で描き直し、図の読み方と構成要素の表を付ける。プロファイルごとの図に小見出しを付ける。
- [x] T020 [Docs] プラットフォーム設計とネットワーク設計の状態列をやめ、（仮）の注記へ改める。Probe の表を Kubernetes の Probe 名で書き、Google Cloud の節を全体図、構成要素、リリースの流れ、アーキテクチャの選択、マニフェストの差分、既知の制限で組み直す。Cloud CDN と Cloud Armor の評価順序（バックエンドセキュリティポリシーはキャッシュミスだけに効く）を受信の流れへ反映する。
- [x] T022 [Docs] 現在状態の文書から work item への参照を除き、未完了の事柄を現在の状態として書き直す。対象は設計、要求、検証、運用の 10 行と、本 work item が足した 1 行。
- [x] T023 [Tooling] 現在状態の文書が work item を参照したら拒否する検査 `work-item-references` を加える（`mise run check-work-item-references`）。RED: `mise run test-tools-file -- check/src/docs-work-item-links.test.ts` がモジュール不在で失敗した。GREEN: 単体 4 件と受け入れ 2 件が通った。故障注入: 検査の登録を外すと受け入れ 2 件が失敗し、語の境界の判定を外すと単体 1 件が失敗した。
- [x] T024 [Docs] 表の列見出しを名詞の見出しへ改め（保持する状態、実行形態、違反時の影響、判定内容、失敗時の動作、提供内容、許可する接続元、アクセス経路、監査記録など）、セルの文言も揃える。Google Cloud の全体図を、線の交差しないリソース階層の木、共通プロジェクトとの関係の表、production プロジェクトの構成図に分ける。
- [x] T021 [Tooling] 用語検査へ「参照トポロジー」「参照プロファイル」「既定」「コンピュート」「資材」を加え、文書全体を置き換える。RED: `mise run test-tools-file -- check/src/terminology.test.ts` が新しい 2 件の試験で失敗した（規則が無く指摘が空）。GREEN: 規則を加えて 12 件すべてが通り、`mise run check-terminology` が 224 文書で通った。
- [x] T007 [Verify] リンク、用語、構成ファイル、監視の構成ファイル、仕様、全体検証を通す。スタックを起動して改名後の配線を確認する。

## Verification

- `mise run check-links`
- `mise run check-k8s`
- `mise run check-compose`
- `mise run check-spec`
- `mise run verify`

## Risk Notes

構成ファイルから読み取った値を文書へ写すと、構成ファイルが変わった時点で文書が古くなる。レプリカ数、`minScale`、タイムアウトなどの値は文書へ写さず、構成ファイルを正本として参照する。文書が持つのは、どの資源がどの役を担い、なぜそう選んだかだけである。

Google Cloud のプロファイルは実際に稼働している構成ではない。検証済みと読めるとこの文書は無いより悪くなるため、プロファイルごとに検証状況を明示する。

`infra/deploy/gcp/README.md` から内容を移すと、構成ファイルを読む人が設計の理由を失う。移した先へのリンクを構成ファイル側に残す。

## Completion

- **Completed At**: 2026-09-17
- **Summary**:
  デプロイとインフラの設計が、三つのデプロイプロファイル（ローカル Docker Compose、汎用 Kubernetes、Google Cloud）ごとの具体で読めるようになった。
  共通トポロジーは構成要素名（`idmagic-frontend`、`idmagic-api`、`idmagic-worker`、`idmagic-batch`、`idmagic-seed`、`psqldef`）で描かれ、プロファイルごとの配置先、構成図、プラットフォームとネットワークの設計項目を持つ。
  Google Cloud プロファイルのコンピューティングは GKE Autopilot とし、エッジは GKE Ingress、静的アセットは `idmagic-frontend` のイメージと Cloud CDN とした。Cloud Run 案は採らなかった選択肢として比較とともに残り、`infra/deploy/gcp/` のひな型もその位置付けになった。
  汎用 Kubernetes プロファイルには、単体で動くよう CloudNativePG の PostgreSQL と `psqldef` と `idmagic-seed` の Job が（仮）として加わった。構成ファイルへの反映はまだ無い。
  ネットワーク設計は、ファイアウォールルールの表とプロファイルごとの実施の仕組み、Cloud CDN と Cloud Armor の評価順序、リスナーの公開範囲、管理アクセスを持つ。
  Docker Compose の `idp` サービスは `api` になり、Caddy、Prometheus、Alloy の参照も追従した。
  規範文書は語だけが変わった。`mise run spec-diff` は REQ-SYSTEM-001、016、017、REQ-IDMANAGEMENT-009、REQ-OAUTH2-048、REQ-PROVISIONING-005、REQ-TENANCY-021 と四つの標準要求を挙げるが、変わったのは「既定」から「デフォルト」への表記と、REQ-SYSTEM-001 の Given にあった実在しない「イベントリレー」を実在の実行単位名へ直したことだけで、条件も要求する結果も同じである。
  文書全体で「資材」「既定」「コンピュート」「参照トポロジー」「参照プロファイル」を使わなくなり、`mise run check-terminology` が再発を拒否する。
  現在状態の文書は work item を参照しなくなり、`mise run check-work-item-references` が再発を拒否する。
  Google Cloud を構成ファイルから生成できる形にする作業は、別の記録 [[wi-592-generate-infrastructure-materials-from-parameter-definitions]] として起票した。
- **Acceptance RED Evidence**:
  - **Test**: `tools/check/src/repository-checks.acceptance.test.ts` の「設計文書の work item 参照を、位置つきで拒否する」「work item を参照しない作業ツリーを通す」
  - **Requirement**: N/A: 文書の設計内容と検査ツールの変更であり、規範のプロダクト要求を持たない
  - **Observed Failure**: 検査を registry から外した状態で、この 2 件が失敗し、残る 17 件は通った
  - **Detection Reason**: runner を仮の作業ツリーに対して起動し、終了コードと `file:line:column` の出力を確かめる。検査が配線されていなければ、拒否すべき文書を通す形で落ちる
  - **Test**: `mise run dev-compose` で起動したスタック（`idp` から `api` への改名の確認）
  - **Requirement**: N/A: ローカル構成のサービス名の変更
  - **Observed Failure**: RED は観測していない。改名後に、`frontend` から `http://api:8081/health` が応答し、Prometheus の `idmagic-api` が `http://api:8081/metrics` で up、Loki に `service` ラベルが付いたことを確認した。Alloy のセレクターが一致しなければ `service` ラベルは付かない
  - **Detection Reason**: 三つの参照先（Caddy の中継先、Prometheus の取得先、Alloy のセレクター）を、それぞれ実際の通信の成否で確かめた
- **Unit RED Evidence**:
  - **Test**: `tools/check/src/terminology.test.ts` の「構成の名前としての「参照」だけを対象にする」「既定、コンピュート、資材を、採用語へ寄せる」
  - **Requirement**: N/A: 文書の用語の統一
  - **Observed Failure**: 規則を加える前、2 件とも指摘が空の配列になり失敗した
  - **Detection Reason**: 指摘された語の並びを完全一致で比べる。「別の文書を参照する」の動詞の「参照」を落とさないことも同じ行で固定する
  - **Test**: `tools/check/src/docs-work-item-links.test.ts`
  - **Requirement**: N/A: 検査ツールの追加
  - **Observed Failure**: `./docs-work-item-links.ts` が存在せず、import が解決しなかった
  - **Detection Reason**: 指摘の列位置を完全一致で比べ、`Wi-Fi`、`kiwi-42`、地の文の `work-items/` を通すことも固定する
- **Change-Resistance Results**:
  `risk: low` なので契約上は不要だが、新しい検査に故障を注入した。
  検査を registry から外すと、受け入れ試験 19 件中 2 件が落ちた。
  `wi-` の直前が英数字でないことの判定を外すと、単体試験 4 件中 1 件（「語の一部としての wi- は指摘しない」）が落ちた。
  どちらも検出された。TypeScript 側に変異器は無いため、系統的な変異は行っていない。
  作業中、新しい検査を既存の `tools/check/src/work-item-references.ts` と同じ名前で作り、既存のファイルと試験を上書きした。受け入れ試験の全件失敗で気付き、`git restore` で戻してから別名で作り直した。
- **Verification Results**:
  - `mise run check-links` - passed（833 文書）
  - `mise run check-terminology` - passed（224 文書）
  - `mise run check-work-item-references` - passed（181 文書）
  - `mise run check-spec` - passed（Mermaid の構文解析を含む）
  - `mise run check-work-items` - passed
  - `mise run check-compose` - passed
  - `mise run check-k8s dev` - passed（21 リソース）
  - `mise run check-monitoring` - passed
  - `mise run test-tools` - passed（43 ファイル 547 件）
  - `mise run lint-tools` - passed
  - `mise run verify` - passed
