---
status: in_progress
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-17
priority: p2
depends_on: []
change_kind: docs
spec_impact:
  kind: none
  reason: "モジュール設計文書の配置と索引の形を変えるだけで、規範 ID、要件の文言、TypeSpec の契約、製品の振る舞い、永続状態、イベント、外向きの呼び出しは変更しない。"
---

# モジュール設計文書を対象ごとのファイルに分け、`design/` と機能群の段を廃止する

## 動機

モジュール設計文書（`docs/modules/`）の構成は、次の点で読み手に伝わりにくい。

- モジュール設計文書全体の概観がない。モジュールの一覧は[論理アーキテクチャ](../../docs/design/architecture/logical.md)にあり、[モジュール設計文書の入口](../../docs/modules/README.md)は一覧を持たない。
- モジュールの中に `design/` の段があり、「モジュール設計文書」のうち `design/` 以外は設計でないように読める。
- 規則にある機能群（`<group>/`）の段を使うモジュールは一つもなく、機能群は README の機能の表の列としてだけ使われている。
- 機能ごとに `README.md` が並び、`docs/modules/<module>/<feature>/README.md` の深さになる。

根本には、仕様と設計の区別の基準が見えないことがある。
現行の規則は「仕様は要素が外へ約束すること、設計はその実現方法」と定め、この区別をディレクトリ（`design/`）、ファイル（`README.md`）、節（機能仕様の「実現方式」）の三通りで表している。
しかし区別は相対的であり、モジュールの責務と契約はシステムから見れば全体設計の割り当て、モジュールから見れば外への約束である。
実際にモジュールの `README.md` には仕様と設計の索引が同居し、機能仕様にも実現方式が入っている。

## 対象範囲

- 仕様と設計の区別を、ファイルの分け方ではなく文の性質で定める。仕様は検証の対象になる規範（`REQ-*` の要件、TypeSpec の契約、横断シナリオ、`standards.md` の準拠と逸脱、品質の割り当て）、設計はそれを実現する構造、仕組み、判断の説明とする。両者は一つのファイルに同居してよい。
- モジュールの文書を対象（モジュール全体、設計領域、機能）で分ける。
  - `design/<area>.md`、`design/decisions.md`、`design/risks.md` をモジュールの直下へ移す。
  - 横断的概念の文書（`design/<concept>.md`）を廃止し、モジュールの直下を固定の名前だけにする。CSV の往復変換は機能「CSV の転送」の実現方式へ、IdManagement と Authentication のイベントの記録は各モジュールの `observability.md` へ、WS-Federation のメタデータは `architecture.md` と `security.md` へ移す。
  - 機能仕様 `<feature>/README.md` を `features/<feature>.md`、受け入れ例 `<feature>/acceptance.feature.md` を `features/<feature>.feature.md`、章 `<feature>/<chapter>.md` を `features/<feature>.<chapter>.md` へ移す。
  - 機能群の段を規則から削る。
- モジュールの `README.md` の末尾の `## 設計` を `## 文書` に替え、設計領域の索引の表と、用語集、標準、品質、設計判断、リスクの表を並べる。用語集と標準の表は機能の節からこの節へ移す。モジュールの直下の文書がこの節から辿れなければ検査が拒否する。
- 論理アーキテクチャの責務表をモジュール設計文書の入口へ移し、区分の列を足す。論理アーキテクチャは様式と品質要件の割り当てに絞る。
- モジュール設計文書の入口に、モジュール間の関係の図と、読み始める順序を置く。
  - 図はモジュール間の依存を表し、手で書かずに境界検査が抽出する import の関係から生成して、入口の印の間へ書き込む。図が古ければ検査が拒否する。
  - 読む順序は、区分ごとに最初に読むモジュールとその理由を手で書く。
- 上の配置を強制する検査、生成サイト、`brief` の文脈導出、境界検査の入力の読み取りを新しい配置へ合わせる。
- 基準 revision と比べる検査（仕様差分と境界の負債の比較）が、移す前の配置の revision も読めるようにする。読めなければ、移動だけで全要件が削除と追加に見え、責務表も見つからない。
- 移動したファイルへのリンクを、`docs/`、`work-items/`、`spec/`、コードのコメント、`.agents/` で直す。

## 対象外

- 規範 ID、要件の文言、機能の分割、モジュールの境界は変えない。
- システムの全体設計文書（`docs/design/`）と要件文書の構成は変えない。
- `work-items/done/` の縮めた記録は、形だけを検査し、移動に追従させない規則に従う。

## 設計

### 目標の配置

```text
docs/modules/
  README.md                 # 入口。区分ごとのモジュール一覧、文書の構成、読み方
  glossary.md
  scenarios.feature.md
  <module>/
    README.md               # 概要：責務と境界、モデル、公開する契約、機能の索引、文書の索引
    glossary.md
    standards.md
    quality.md
    architecture.md         # 設計領域の文書。九領域の語彙と順序は現行どおり
    data.md …
    decisions.md            # 設計の記録
    risks.md
    features/
      <feature>.md          # 機能仕様。最後の節「実現方式」は現行どおり
      <feature>.feature.md  # 受け入れ例
      <feature>.<chapter>.md
```

- モジュールの直下には固定の名前だけを置く。機能の段の名前は機能名で始め、`glossary`、`design` などのファイル種別の名前は機能名にも章の名前にも使えない。
- 機能スライスとコード `backend/<module>/<feature>/` の対応は、`features/<feature>.md` のファイル名で判定する。ハイフンを除いた名前の一致と別名の規則は現行どおり。

### 横断的概念の廃止

当初は横断的概念を `<concept>.md` としてモジュールの直下に残す計画だった。
しかし任意の名前の文書が設計領域の文書と同じ段に並ぶと何が入るのかが伝わらず、`csv-transfer.md` と `features/csv-transfer.md` のように同じ対象を扱う文書が二つの段に並んでいた。
既存の 4 件は、どれも共有の仕組みを担う機能か一つの設計領域に収まったので、任意の名前の文書を廃止し、モジュールの直下を閉じた集合にした。

### 基準 revision の読み替え

仕様差分（`tools/check/src/spec-diff.ts`）は、機能ごとのディレクトリに置いた過去の機能仕様と受け入れ例を、現在の機能仕様と同じ種別として読む。
境界の負債の比較（`tools/check/src/boundary-debt-ratchet.ts`）は、その revision にある入力だけを取り出し、責務表を入口か論理アーキテクチャのどちらかから読む。
既存の共通シナリオの旧配置の読み替えと同じ扱いであり、移す前の revision との比較が不要になったら外せる。

### 採用しない案

| 案 | 採らない理由 |
| --- | --- |
| `design/` を残し、名前を `internals/` などへ替える | 仕様と設計をディレクトリで分ける構造が残り、区別の基準が見えない問題を解かない |
| 機能仕様をモジュールの直下に `<feature>.md` で置く | 一段浅くなるが、設計領域の文書と機能が同じ段に混ざり、名前の衝突を常に検査する必要がある |
| 横断的概念を `<concept>.md` として残し、呼び名だけを改める | 任意の名前の文書が固定の名前の文書と同じ段に並び、何が入るのかを名前から読めない状態が残る |
| モジュールの `README.md` の索引を一つの表にまとめる | 設計領域の行は区分と参照先の列を持ち、記録の行は持たないので、一つの表では空のセルが並ぶ |
| `<feature>/README.md` を維持する | 変更は小さいが、機能ごとに `README.md` が並び、エディターのタブと検索結果で区別できない状態が残る |
| 責務表を論理アーキテクチャに残し、入口には一覧を置かない | arc42 の構成要素ビューには沿うが、モジュール設計文書の概観がないという指摘に応えない |

### 区分

区分ごとに責務表を分け、H3 の見出しを区分の名前にする。境界検査は見出しの一致する表をすべて読む。

| 区分 | モジュール |
| --- | --- |
| 基盤 | System、Tenancy、Jobs、Seeding、SigningKeys、DataKeys、ApiTokens、Audit |
| アイデンティティ | IdManagement、IdGovernance、Sourcing、Provisioning、WorkloadIdentity |
| 認証と認可 | Authentication、Authorization、Application、ClaimMapping |
| 連携プロトコル | OAuth2、Saml、WsFederation、SharedSignals |

### モジュール間の関係の図

[論理アーキテクチャ](../../docs/design/architecture/logical.md#アーキテクチャ様式)は、依存の辺を手で書かず、コードから抽出して検査すると定めている。
図もこの規則に従い、`analyzeModuleBoundaries`（`tools/check/src/check-boundaries.ts`）が求めるモジュール間の import の関係から Mermaid の `flowchart` を生成する。
区分を部分グラフにし、組み立て地点の System は図に含めない。
生成した図は入口の `<!-- generated:module-dependencies:start -->` と `end` の印の間に追跡したまま置き、GitHub でもそのまま読めるようにする。
生成は `mise run generate-module-dependencies`、古さの検出はリポジトリ検査の `module-dependencies` が受け持つ。
境界検査の受け入れテストの作業ツリーが図を持たなくても済むよう、境界検査とは別の検査にする。
ドメインイベントによる関係は import を生まないため図に現れない。図の下にその旨と[モジュール間イベント](../../docs/design/application/backend.md#モジュール間イベント)への参照を書く。

| 案 | 採らない理由 |
| --- | --- |
| 依存の図を手で描く | 依存の辺を手で書かない規則に反し、モジュールを足したときに古くなる |
| 生成サイトだけに図を出す | リポジトリ上で入口を読む人に図が見えない |

### 読む順序

入口に、最初に読むモジュールと理由を書く。
Tenancy（すべての業務データの境界）、IdManagement（主体の Aggregate）、Authentication（ログインとセッション）、OAuth2（代表的な連携プロトコル）の順とし、残りは区分の表と依存の図から辿る。
文書ガイドは `README.md` に読む順序を書かないと定めているので、モジュール設計文書の入口だけを例外として明記する。

## 計画

1. 規則文書を新しい配置へ書き換える：[文書ガイド](../../docs/formats/documentation-guide.md#配置)、[仕様フォーマット](../../docs/formats/specification-format.md#仕様の木)、[設計フォーマット](../../docs/formats/design-format.md#1-設計領域)、フォーマットの理由、モジュール設計文書の入口、全体設計文書の入口、論理アーキテクチャ。
2. 検査とツールを新しい配置へ合わせ、テストを直す。
3. ファイルを `git mv` で移し、一度だけの移行スクリプトでリンクを直す。各モジュールの `## 設計` を `## 文書` へ書き換える。
4. 検証する。

## タスク

- [x] T001 [Docs] 規則文書とモジュール設計文書の入口を書き換え、責務表を移す。
- [x] T002 [Tooling] 配置、機能スライス、索引、責務表の読み取り、生成サイト、`brief` を新しい配置へ合わせる。基準 revision の旧配置を読めるようにする。
- [x] T003 [Docs] ファイルを移し、リンクとモジュールの索引を直す。横断的概念の 4 件を機能と設計領域の文書へ移す。
- [x] T004 [Tooling] モジュール間の依存の図を生成するタスクと、図の古さを拒否する検査を足し、入口に図と読む順序を書く。
- [x] T005 [Verify] 検査と生成サイトで確かめる。

## 検証

- `mise run check-repository`、`mise run check-links`、`mise run check-boundaries`：配置、索引、リンク、責務表の読み取りを確かめる。
- `mise run check-rendered-docs`：生成サイトの機能地図とモジュール索引を確かめる。
- ツールのテスト：配置と機能スライスの判定を変えるため。
- `mise run verify`：最後に全体を確かめる。

## リスク

- 多数のリンクの書き換えで、アンカーやコードのコメントの参照を取りこぼすおそれがある。`check-links` とコメント内のパスの検索で確かめる。
- 未完了の作業項目が旧パスの `affected_spec` を持つ。移動に合わせて参照先を更新する。

## 要約

`mise run spec-diff` は main に対して規範仕様の変更なしと報告する。
モジュール設計文書を対象ごとのファイルに分けた：設計領域の文書と記録をモジュールの直下へ、機能仕様と例と章を `features/<feature>.md`、`<feature>.feature.md`、`<feature>.<chapter>.md` へ移し、機能群の段と任意名の横断的概念の文書を廃止した。
仕様と設計はファイルではなく文の性質で区別すると文書ガイドに定め、モジュールの `README.md` の最後の節を「文書」にした。
責務表を区分ごとにモジュール設計文書の入口へ移し、読み始める順序と、import から生成して古さを検査する依存の図を置いた。
仕様差分と境界の負債の比較は、移す前の配置の revision も読む。移す前の revision と比べる必要がなくなれば、この読み替えは外せる。
