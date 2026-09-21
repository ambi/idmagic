---
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-19
priority: p3
depends_on: []
change_kind: docs
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: "既存の設計内容を変えず、設計文書間の一次情報源と参照関係だけを整理するため、利用者向けのリリース文書は不要である。"
  references: []
initial_context:
  specification:
    - DOCUMENTATION_GUIDE.md
    - SPECIFICATION_FORMAT.md
    - WORK_ITEM_FORMAT.md
    - docs/development/specification-first-workflow.md
    - docs/development/coding-style.md
    - docs/development/testing.md
    - docs/design/application/design-guidelines.md
    - docs/architecture/deployment.md
    - docs/design/infrastructure/platform.md
    - docs/design/observability/logging.md
    - docs/design/reliability/availability.md
    - docs/domain/jobs/internals.md
    - docs/runbooks/async-jobs.md
  typespec: []
  source:
    - infra/docker/docker-compose.dev.yaml
    - infra/k8s/base/worker.yaml
    - infra/k8s/overlays
  tests: []
  stop_before_reading: [backend, frontend, spec]
spec_impact:
  kind: none
  reason: "設計文書が同じ事実を複数箇所へ書いている状態を一次情報源へ集約するだけで、設計の内容も規範要素も変えない。"
---

# 複数の設計文書に書かれた同じ事実を、一次情報源へ集約する

## Motivation

`DOCUMENTATION_GUIDE.md` は、同じ定義、同じ数値、同じ一覧を二か所に置かず、一次情報源を一つ決めて他所からは参照すると定める。
`docs/design/infrastructure/platform.md` は冒頭で「この文書は配置先の表を繰り返さない」と宣言しており、表の単位では境界が引けている。
一方、散文の単位では同じ事実が複数の文書に書かれている。

| 事実 | 書かれている場所 |
| --- | --- |
| Alloy が Docker Engine API からコンテナログを読み、ホストのログドライバーに依存しない | `architecture/deployment.md:150`、`design/infrastructure/platform.md:30`、`platform.md:273`、`design/observability/logging.md:134` |
| 汎用 Kubernetes の PostgreSQL を CloudNativePG で持たせる（仮） | `deployment.md` 2 か所、`platform.md` 6 か所、`design/reliability/availability.md` 1 か所 |
| Google Cloud プロファイルは構成ファイルがまだ無いため全体が（仮）である | `deployment.md:190`、`platform.md:295` にほぼ同文 |
| レーンが与えるのは順序ではなくキャパシティの隔離である | `domain/jobs/internals.md:7`、`runbooks/async-jobs.md:19` にほぼ同文、`platform.md:282` に同趣旨 |

節の構成も重なっている。
`deployment.md` の「プロファイルごとの構成」（ローカル Docker Compose、汎用 Kubernetes、Google Cloud）と、`platform.md` の同名の節が、同じ三つのプロファイルを同じ順序で説明する。
前者は Mermaid の図、後者は散文という違いはあるが、どちらも「このプロファイルは何を提供し、何を提供しないか」を述べている。

重複の害は分量ではなく、片方だけが動くことである。
上の 4 件はいずれも（仮）を含む記述で、構成ファイルが入った時点で（仮）を外す作業が発生する。
いま外す場所が 4 か所、9 か所と散っているため、一つを外して残りが（仮）のまま残る経路が開いている。

なお、文単位で完全に一致する重複は `docs/` 全体で 2 件しかない。
問題は言い回しを変えた同じ事実であり、機械的な一致検出では見つからない。

## Scope

- 上表の 4 件について、事実ごとに一次情報源を一つ決め、他の場所を参照へ置き換える。
- `deployment.md` と `platform.md` の「プロファイルごとの構成」の担当を分け、どちらが何を述べるかを両文書の冒頭に書く。
- 決めた担当を、`docs/design/infrastructure/platform.md` の「対象範囲」と `docs/architecture/deployment.md` の「関連文書」へ反映する。

## Out of Scope

- `docs/design/infrastructure/` の 2 文書にある（仮）と未決定の行そのものの解消。`platform.md` に 60 か所、`network.md` に 49 か所あり、リポジトリ全体の 120 か所のうち 109 か所をこの 2 文書が占める。これは確定していない構成を確定させる作業であり、重複の解消とは別である。
- Mermaid の図に現れる同じ構成要素名。図は文と違い、要素を省くと読めなくなる。図が重複しているかどうかは、文の担当を決めた後に判断する。
- `docs/domain/` の Context 文書どうしの重複。Bounded Context は用語が文脈ごとに違う意味を取ることを前提としており、同じ語が現れることは重複ではない。
- 規約文書の重複の解消と、文書の言語の統一。[[wi-629-one-source-for-the-document-layout-and-format-rules]] と [[wi-631-write-the-human-readable-documents-in-japanese]] が扱う。

## Design

事実の種類で一次情報源を割り当てる。
`DOCUMENTATION_GUIDE.md` の「上位と下位の分かれ目」に従い、アーキテクチャは構成要素の割り当てを、領域別設計はその実現方式を担当する。

| 事実 | 一次情報源 | 理由 |
| --- | --- | --- |
| Alloy の収集経路 | `design/observability/logging.md` | ログの収集経路はオブザーバビリティ設計の担当であり、プロファイルの違いはそこで並べる |
| CloudNativePG を採る判断 | `design/infrastructure/platform.md` | 製品の選定理由はプラットフォーム設計の担当である。`deployment.md` は配置先の表に名前だけを書き、`availability.md` は可用性の観点から参照する |
| プロファイルの検証状況と（仮）の範囲 | `architecture/deployment.md` | プロファイルの一覧と構成ファイルの有無を既に表として並べている |
| レーンが与える隔離の意味 | `domain/jobs/internals.md` | レーンは Jobs の内部設計が決める概念であり、`platform.md` と `runbooks/` はその帰結を書く |

「プロファイルごとの構成」は、`deployment.md` が構成要素の配置と接続、`platform.md` が各構成要素の実現方式と選定理由を担当する、と分ける。
節を片方へ寄せる案は採らない。
配置の図を読みたい読者と、製品選定の理由を読みたい読者は別の入口から来るため、どちらかの文書に両方を置くと、もう片方の読者が目的の記述まで遠くなる。

重複の再発を検査で止める案は採らない。
言い回しを変えた同じ事実は字面で一致せず、字面の一致を見る検査は、同じ用語が正当に現れる箇所を落とす。
代わりに、両文書の冒頭に担当の宣言を置き、書き足すときに読む位置へ規則を出す。

## Plan

1. 4 件の事実それぞれについて、一次情報源に残す記述を書き、他の場所を参照へ置き換える。
2. `deployment.md` と `platform.md` の「プロファイルごとの構成」を、設計表の担当に沿って整理する。
3. 両文書の冒頭へ担当の宣言を書く。
4. `mise run check-links` で、置き換えた参照のアンカーが解決することを確かめる。

## Tasks

- [x] T001 [Acceptance] `deployment.md`、`platform.md`、`availability.md`、`async-jobs.md` に残る一次情報源と同義の説明を `rg` で検出し、文書間の重複が残っているため失敗することを確認する。規範要件は `N/A: 製品の振る舞いを変えない文書整理である`。
- [x] T002 [Unit] `deployment.md` と `platform.md` の担当宣言を `rg` で探し、配置と接続、実現方式と選定理由の境界がまだ書かれていないため失敗することを確認する。規範要件は `N/A: 実行可能な単体境界を持たない文書整理である`。
- [x] T003 [Docs] Alloy の収集経路を `logging.md` へ集約し、3 か所を参照へ置き換える。
- [x] T004 [Docs] CloudNativePG の判断を `platform.md` へ集約し、`deployment.md` と `availability.md` を参照へ置き換える。
- [x] T005 [Docs] （仮）の範囲の記述を `deployment.md` へ集約する。
- [x] T006 [Docs] レーンの隔離の意味を `jobs/internals.md` へ集約し、`deployment.md`、`platform.md`、`runbooks/async-jobs.md` を参照へ置き換える。
- [x] T007 [Docs] 「プロファイルごとの構成」の担当を分け、両文書の冒頭へ宣言を書く。
- [x] T008 [Verify] 変更を検証する。

## Verification

- `mise run check-links`
- `mise run check-terminology`
- `mise run verify`

## Risk Notes

参照へ置き換えると、その場で読めていた内容がリンク先へ移る。
runbook は障害対応の最中に読むため、リンクをたどらせると手が止まる。
`runbooks/async-jobs.md` は、判断に必要な一文はその場に残し、背景の説明だけを参照へ回す。

一次情報源へ集約する過程で、複数の記述の間にある食い違いが見つかる可能性がある。
食い違いは、どちらが正しいかを構成ファイルとコードで確かめてから書く。
確かめられない場合は（仮）または未決定として残し、文書どうしの多数決で決めない。

## Completion

- **Completed At**: 2026-09-21
- **Summary**:
  `mise run spec-diff` は `main` に対する規範仕様の変更がないことを報告した。
  Alloy の収集経路、CloudNativePG の選定、Google Cloud プロファイルの検証状況、実行レーンの隔離を、それぞれ一つの設計文書へ集約した。
  デプロイメントアーキテクチャは構成要素の配置と接続を、プラットフォーム設計は実現方式と選定理由を担当する形に分けた。
- **Acceptance RED Evidence**:
  - **Test**: `deployment.md`、`platform.md`、`availability.md`、`async-jobs.md` に残る、一次情報源と同義の説明を拒否する `rg` 検査
  - **Requirement**: N/A: 製品の振る舞いを変えない文書整理である。
  - **Observed Failure**: 検査は、ログドライバーへの非依存、CloudNativePG の採用、Google Cloud 全体の（仮）、レーンの隔離を重ねて説明する 9 箇所を報告して終了コード 1 になった。
  - **Detection Reason**: 一次情報源以外に説明が残ると検査対象の語句へ一致するため、参照へ置き換えず重複を残す変更を区別できる。
- **Unit RED Evidence**:
  - **Test**: `deployment.md` と `platform.md` から「配置と接続」「実現方式と選定理由」の担当宣言を探す `rg` 検査
  - **Requirement**: N/A: 実行可能な単体境界を持たない文書整理である。
  - **Observed Failure**: 役割の組み合わせを述べる行がなく、`rg` は一致なしの終了コード 1 になった。
  - **Detection Reason**: 両文書の冒頭に担当宣言がなければ一致しないため、節の重なりだけを直して責務を明示し忘れた変更を検出できる。
- **Change-Resistance Results**:
  `N/A: low-risk の文書変更であり、Go のロジックを変更していないため mutation testing は適用しない。RED に使った二つの検査を変更後に再実行し、重複説明が消え、担当宣言が両文書に現れることを確認した。`
- **Source Comparison Results**:
  `infra/docker/docker-compose.dev.yaml` は Alloy が Docker socket だけをマウントすることを、`infra/k8s/base/worker.yaml` は三つの Deployment がそれぞれ一つの `JOB_WORKER_LANES` を持つことを示していた。
  `infra/k8s/overlays/` には dev と prod だけがあり、GKE 向け overlay は存在しなかった。
- **Verification Results**:
  - 文書間の重複説明を拒否する `rg` 検査 - passed
  - 両文書の担当宣言を確認する `rg` 検査 - passed
  - `mise run check-links` - passed (881 documents)
  - `mise run check-terminology` - passed (231 documents)
  - `mise run check-work-items` - passed
  - `mise run verify` - passed
  - `mise run test-ui-e2e` - N/A: ブラウザーへ到達するプロダクト変更ではない。
