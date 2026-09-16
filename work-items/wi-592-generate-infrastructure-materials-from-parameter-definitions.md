---
status: pending
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-09-16
priority: p2
depends_on: [wi-584-ground-deployment-design-in-reference-profiles]
change_kind: tooling
spec_impact:
  kind: none
  reason: "構成ファイルの一次情報源を移すだけで、プロダクトが公開する契約も観測できる振る舞いも変わらない。生成後の構成ファイルは現行と同じものを出す。"
---

# 構成ファイルの一次情報源をパラメーター定義へ移し、三つのデプロイ先へ生成する

## Motivation

[プラットフォーム設計](../docs/design/infrastructure/platform.md)と[ネットワーク設計](../docs/design/infrastructure/network.md)は、どの資源が何を担い、なぜそう選んだかを持つ。
そこから `docker-compose.dev.yaml`、Kubernetes の構成ファイル、Terraform を機械的に生成できるかを確かめると、できない。

生成に必要な量と、設計書が持つ量が釣り合っていない。
Kubernetes の構成ファイルはスカラー値として設定されるフィールドを 473 個持ち（`worker.yaml` で 126、`networkpolicy.yaml` で 84）、Docker Compose はサービス定義に加えて設定ファイル 8 本 485 行を参照する。
一方、設計書 3 文書に現れる具体値は `psqldef`、`latency_sensitive`、`REGIONAL`、`TLS 1.2`、`distroless/static-debian12`、`asia-northeast1` の 6 個である。
ラベルとセレクターの体系、Probe の経路、ポート番号、イメージのタグ、CronJob のスケジュール、CIDR、IAM のロール文字列、組織ポリシーの制約名、規則の優先度、保持日数は、設計書にひとつも現れない。

これは [DOCUMENTATION_GUIDE.md](../DOCUMENTATION_GUIDE.md) §2 の「同じ数値を二か所に置かない」の帰結である。
値の一次情報源を構成ファイルに置くと決めたので、設計書は決定だけを持つ。
方針としては一貫しているが、代償が三つある。

第一に、同じ決定が三つのデプロイ先で別々に表現され、突き合わせる仕組みがない。
`docker-compose.dev.yaml` の `worker` は `JOB_WORKER_LANES` を設定せず一プロセスで全レーンを取り、`infra/k8s/base/worker.yaml` はレーンごとに Deployment を分ける。
この差は意図的だが、意図的であることはコメントが主張しているだけで、検査がない。

第二に、Google Cloud には構成ファイルそのものが無い。
設計は GKE Autopilot を採っているが、GKE 向けの overlay も、クラスターと周辺リソースを作る Terraform も無い。
`infra/deploy/gcp/` にあるのは採らなかった Cloud Run 案の shell のひな型である。
プロジェクト ID、CIDR、ロール割り当て、Cloud SQL のバックアップ窓、Cloud Armor の規則式、[プラットフォーム設計](../docs/design/infrastructure/platform.md)が（仮）としたリソース階層、ポリシー制約、監査ログの集約は、どの構成ファイルにも無い。
値がリポジトリのどこにも無いため、設計と構成ファイルの差分を取ることすらできない。

第三に、同じ概念を指す名前がデプロイ先ごとに手で書かれている。
実行単位の名前、レーンの名前、ポートの役割、シークレットの参照名がそれぞれの構成ファイルに散り、`idp` から `api` への改名のような変更が 6 ファイルへ波及した。

## Scope

- 三つのデプロイ先が共有する決定を持つ、機械可読なパラメーター定義を新設する。
- そこから `infra/docker/docker-compose.dev.yaml`、`infra/k8s/` の base と overlay（GKE 向けを含む）、GKE クラスターと周辺リソースの Terraform を生成する生成器を作る。
- 生成物と現行の構成ファイルの差分を検査するタスクを作り、生成物が現行と等価であることを移行の受け入れ条件にする。
- 生成物を Git で追跡するかどうかを決め、追跡するなら生成物が最新であることを検査する。
- [DOCUMENTATION_GUIDE.md](../DOCUMENTATION_GUIDE.md) §2 の一次情報源の割り当てを改訂し、パラメーター定義を「機械が食う契約」として位置づける。
- [プラットフォーム設計](../docs/design/infrastructure/platform.md)と[ネットワーク設計](../docs/design/infrastructure/network.md)の「設定値は構成ファイルに書いた値を正しい値とする」という記述を、新しい割り当てへ合わせる。
- Google Cloud の（仮）の項目を、Terraform を生成できる粒度のパラメーターへ落とす。

## Out of Scope

- Google Cloud への実際の適用。生成した Terraform を本番へ当てることは含めない。
- 監視の構成ファイルの生成。`prometheus-rules.yml`、`grafana-dashboard.json`、`alloy-config.alloy` は、対象のドメインが違うため別に扱う。
- デプロイ先の選定。どのプロファイルを本番へ採るかはこの work item が決めない。
- 現行の構成ファイルの設定内容の変更。生成物は現行と等価なものを出すことを条件にする。設定を変えるなら別の記録で行う。

## Design

### 何をパラメーター定義が持つか

境界は「三つのデプロイ先のうち二つ以上が同じ決定を別々に表現しているもの」とする。
一つのデプロイ先にしか現れない設定は、そのデプロイ先の生成器が持つデフォルト値にする。
共有していないものを共有の定義へ上げると、定義が三つの和集合になり、読んでもどのデプロイ先の話か分からなくなる。

この境界で分けると、パラメーター定義が持つのは次になる。

| 区分 | 例 |
| --- | --- |
| 実行単位 | 名前、エントリーポイント、HTTP を持つか、実行レーン、Probe の経路 |
| ポートの役割 | アプリケーション用、メトリクス用 |
| 設定とシークレット | 環境変数のキー、値の出どころ、シークレットとして扱うか |
| 依存の順序 | スキーマ適用が API とワーカーより先、など |
| ファイアウォールルール | [ネットワーク設計](../docs/design/infrastructure/network.md#ファイアウォールルール)の表と同じ行 |
| 環境差 | 環境の名前、レプリカ数、リソースの要求量と上限 |
| イメージ | ベースイメージ、収録する実行ファイル、レジストリ、バージョンの指定方法 |
| 命名とラベル | リソース名の組み立て規則、付けるラベルの軸 |

`prometheus.yml` のメトリクスの取得先のように、実行単位とポートから導けるものは定義へ書かず生成する。

### 生成物を追跡するか

生成物を Git で追跡し、最新であることを検査する。
追跡しない案（`spec/generated/` と同じ扱い）を採らない理由は、Kubernetes の構成ファイルが Pull Request のレビュー対象であり、レビューアーが差分として読める必要があるからである。
OpenAPI と違って、これらは適用すると副作用を持つ。

代わりに、生成物の先頭へ「生成物である。手で編集しない」と書き、`mise run check-generated-infra` が定義から作り直した結果と一致することを確かめる。

### 生成器の実装

`tools/` に TypeScript で置く。
既存の検査とコード生成（`tools/check/`、`tools/generate-contract/`）が同じ場所にあり、`mise` タスクの並びも揃う。

Terraform を HCL の文字列として組み立てるのではなく、Terraform が読む `.tf.json` を出す。
HCL を文字列で組み立てると、生成器が整形とエスケープの責任を負い、構文の検査を実行するまで誤りが分からない。

### 移行の順序

Kubernetes から始める。
三つのうち最もフィールド数が多く（473）、生成で得るものが大きい。
`mise run check-k8s` がレンダリングとスキーマを検査するため、生成物の正しさを既存のタスクで確かめられる。

次に Docker Compose、最後に Terraform とする。
Terraform は現行の構成ファイルが無いため、差分で等価を確かめる相手がいない。
先に二つで生成器の形を固めてから、比較対象の無いものへ進む。

### 採らない案

**設計書の散文からの生成。** Markdown の表を構造化データとして読む案は採らない。表の列が増減するたびに解析器が壊れ、文書の書きやすさと機械可読性のどちらも損なう。

**Kustomize と Helm で済ませる案。** Kubernetes の中では環境差を扱えるが、Docker Compose と Terraform を同じ定義から出せない。今ある問題は、三つのデプロイ先のあいだで決定が重複していることである。

**Terraform の module だけを整える案。** Google Cloud の内側は整うが、実行単位とポートとファイアウォールルールが Kubernetes 側と別に書かれたままになる。

## Plan

1. 現行の構成ファイルの全フィールドを棚卸しし、共有する決定、デプロイ先固有の設定、生成で導けるものへ分類する。
2. パラメーター定義のスキーマを決め、Kubernetes 相当の内容を書く。
3. Kubernetes の生成器を作り、生成物が現行の構成ファイルと等価であることを差分で確かめる。
4. Docker Compose へ広げる。`JOB_WORKER_LANES` の扱いのように意図的な差は、定義の中で差として表現する。
5. Google Cloud の（仮）の項目をパラメーターへ落とし、Terraform を生成する。
6. 生成物の鮮度を検査するタスクを作り、`mise run verify` へ組み込む。
7. `DOCUMENTATION_GUIDE.md` §2 と、設計書の「設定値は構成ファイルに書いた値を正しい値とする」という記述を改訂する。

## Tasks

- [ ] T001 [Design] 現行の構成ファイルのフィールドを棚卸しし、共有・固有・導出の三区分へ分類する。
- [ ] T002 [Design] パラメーター定義のスキーマを決める。
- [ ] T003 [Tooling] Kubernetes の生成器を作る。
- [ ] T004 [Verify] 生成物が現行の Kubernetes 構成ファイルと等価であることを確かめる。
- [ ] T005 [Tooling] Docker Compose の生成器を作り、等価性を確かめる。
- [ ] T006 [Design] Google Cloud の（仮）の項目をパラメーターへ落とす。
- [ ] T007 [Tooling] Terraform の生成器を作る。
- [ ] T008 [Tooling] 生成物の鮮度を検査するタスクを作り、`verify` へ組み込む。
- [ ] T009 [Docs] `DOCUMENTATION_GUIDE.md` §2 と設計書の一次情報源の記述を改訂する。

## Verification

- `mise run check-generated-infra`
- `mise run check-k8s dev`
- `mise run check-k8s prod`
- `mise run check-compose`
- `mise run check-monitoring`
- `mise run dev-compose` でスタックを起動し、生成した構成が動くことを確かめる
- `mise run check-links`
- `mise run check-work-items`
- `mise run verify`

## Risk Notes

生成器は、現行の構成ファイルが持つ設定を落としても静かに通る。
等価性の確認を「生成物が妥当な YAML である」ではなく「現行の構成ファイルとの差分が空である」に置く。
差分が空にならない箇所は、意図的な変更か落としたかを一件ずつ判断して記録する。

一次情報源を移すと、構成ファイルだけを見て変更する経路が壊れる。
生成物へ手で書いた変更は次の生成で消えるため、生成物の先頭に警告を置き、鮮度の検査を `verify` に入れる。検査が無いまま移すと、手で編集した構成ファイルが黙って巻き戻る。

Terraform は比較対象の現行の構成ファイルを持たない。
生成できたことは、その構成が正しいことも適用できることも意味しない。Google Cloud の部分は、生成物を未検証と明示し、[プラットフォーム設計](../docs/design/infrastructure/platform.md)が（仮）としている状態を引き継ぐ。

パラメーター定義が三つのデプロイ先の和集合へ育つと、読み手はどの項目がどこへ効くのか追えなくなる。
Design の境界（二つ以上が同じ決定を別々に表現しているもの）を、項目を足すたびに当てる。
