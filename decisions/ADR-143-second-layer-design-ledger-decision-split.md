---
status: accepted
authors: [tn]
created_at: 2026-07-26
supersedes: [ADR-116]
---

# ADR-143: RA 第2層を設計正本・台帳・判断履歴の3つに分ける

## コンテキスト

RA 第2層（`REGENERATIVE_ARCHITECTURE.md` §3.2）は ADR と Architecture を「イベントとその射影」として
規定している。実測すると射影側が成立していない。

- `decisions/` は 139 本・散文 11,799 行。
- `ARCHITECTURE.md` は 3,699 行のうち 3,362 行（91%）が frontmatter 台帳で、散文は 337 行。
  `## Structural Decisions` は ADR へのリンク 13 本の箇条書きだけである。
- `frontend/ARCHITECTURE.md`（104 行）は `ARCHITECTURE_FORMAT.md` §1 が認める per-context 文書だが、
  `tools/ra` の discovery が root しか走査しないため検証対象に入っていない。
- 設計は README にも漏れている。`infra/README.md` の `## High Availability & Shared State` は 45 行の
  横断設計、`frontend/README.md` は 96 行中およそ 80 行が設計で、末尾に「詳細は `ARCHITECTURE.md`」と
  書かれている。frontend の設計は 2 ファイルに分裂している。

設計の散文はおよそ 12,500 行、うち Architecture 系は 441 行（3.5%）にとどまる。

原因は規律の緩みではなく、構造の欠陥が 4 つ重なったことにある。

1. **選択規則がない。** `SPECIFICATION_CORE_LANGUAGE.md` §1 は「構成や技術選択は ADR *または*
   `ARCHITECTURE.md` に置く」とだけ述べ、どちらかを決める規則を持たない。
2. **移送先が受け入れを拒否している。** `ADR_FORMAT.md`「役割の境界」は「現在の構成は
   `ARCHITECTURE.md` に置く」「既存 ADR が構成を抱え込んでいたら移せ」と命じているのに、
   `ARCHITECTURE.md` 自身が「索引である。人間向けの包括的な設計説明ではない」と宣言している。
   命令の宛先が受け取りを拒んでおり、設計は ADR と README へ流れるほかない。
3. **機械用の台帳と人間用の散文が同居している。** `ARCHITECTURE_FORMAT.md` §2 は「frontmatter には
   機械検証できる構造だけ、読み物は本文へ」と関心の分離を宣言しているが、同一ファイルに置いたため
   3,362 行をスクロールしないと散文に到達しない。
4. **per-context 文書が tooling から見えない。** `discoverArchitectureDocs` は root `spec/scl.yaml` から
   導出される単一 app（`root: '.'`）しか走査しない。

ADR-116 は Architecture map を検査可能な宣言にする判断であり、その判断自体は有効である。ただし map を
`ARCHITECTURE.md` の frontmatter に置くという形式の部分が、上記 3 の原因になっている。

## 決定

RA 第2層を、答える問いが異なる 3 つの成果物に分ける。

- **設計正本（`ARCHITECTURE.md`）**: いまどうなっているか / なぜそうなっているか。人間が読む散文。
  リポジトリ横断のものを root に、bounded context 固有のものを `<context>/ARCHITECTURE.md` に置く。
- **台帳（`architecture.yaml`）**: 構成を機械が検査できる形。ADR-116 が定めた context・RA layer・
  module role・宣言依存・runtime unit・complexity budget をそのまま持つ。ある module は、その `path` を
  含む最も近い祖先の `architecture.yaml` が宣言し、root が fallback になる。`contexts` /
  `runtime_units` / `complexity` は横断整合検査の対象なので root だけが持つ。
- **判断履歴（`decisions/ADR-NNN-*.md`）**: 何を却下してそう決めたか。追記型。

あわせて、この 3 つと隣接文書の境界を**答える動詞**で定義し、相互排他にする。`README.md` は
「どう使うか / どう動かすか」、`infra/runbooks/` は「何か起きたらどうするか」、`work-items/` は
「今回何をやったか」、`spec/` は「何を満たすべきか」を答える。設計は `ARCHITECTURE.md` 系だけが持つ。

ADR には SCL からも設計文書からも再導出できない「なぜ」だけを残す。ADR を起こすのは、実際に分岐が
あり却下した選択肢が実在するときに限る。設計を記述したいだけなら ADR を起こさず設計文書を更新する。
逆に、設計文書には形になった理由を 1〜2 文だけインラインで添え、却下案と当時の前提の全文は ADR へ
リンクする。読み手が設計を理解するために ADR を開かなくてよい状態を、両方向から保証する。

移行は段階的に行う。既に重複・分裂が実害を出している群（DB 設計ポリシー、durable job 実行基盤、
横断ランタイム方針、UI 設計、物理配置規約）を第 1 波で移し、以降は「ある context の設計に触れた
work item はその context の設計文書を現状に合わせる」を完了ゲートに組み込んで、触れた順に解消する。
139 本を一括で棚卸しするコミットは作らない。

## ADR-116 との関係

ADR-116 の「Architecture map を依存方向と複雑度を検査する実行可能な宣言にする」という判断は有効で、
map のフィールド定義と検査規則もそのまま引き継ぐ。本 ADR が上書きするのは、その map を
`ARCHITECTURE.md` の frontmatter に置くという配置だけである。したがって ADR-116 は `accepted` のまま
`superseded_by` を張る。

## 却下した代替案

- **設計文書を `DESIGN.md` として 2 系統に分ける案**: 構造・台帳を `ARCHITECTURE.md`、メカニズムと
  横断方針を `DESIGN.md` に分ける。「アーキテクチャ」の語と UI 一貫性ルールのような内容とのずれは
  解消するが、書くたびの判断が 1 つ増える。いま解こうとしている問題そのものを再生産する。規約は
  §3.2 が既に Architecture へ割り当てている「ディレクトリ・命名規約」の延長として扱えば足りる。
- **ADR を廃止し設計文書へ一本化する案**: 読み手の導線は最短になるが、却下案と当時の前提は単調増加
  する履歴であり、現状正本の文書に混ぜると必ず読めなくなる。RA の再生成前提（判断の保存）も弱まる。
- **台帳を単一の `architecture.yaml` に留める案**: tooling の変更は最小で済むが、3,362 行の単一
  ファイルが残り、parallel work item で別 context を並行させたときの衝突も残る。
- **台帳を `architecture/contexts/*.yaml` に集中させる案**: SCL の `spec/contexts/*.yaml` と同じ形に
  なり探しやすいが、台帳は実装を記述するものなのに実装ツリーから離れる。SCL が `spec/` に独立して
  いるのは実装非依存であることが理由なので、この類推は台帳には効かない。
- **`ARCHITECTURE.md` の憲章を変えず、`ADR_FORMAT.md` の役割の境界を機械検証で強制するだけの案**:
  移送先が受け入れを拒否している限りデッドロックは解けない。
- **README の設計内容を移送しない案**: 作業量は減るが、frontend の設計が README と `ARCHITECTURE.md`
  に分裂したまま残り、`## Cross-cutting Concerns` は中身が `infra/README.md` にあるため空のままになる。
- **139 本を一括で棚卸しする案**: 整合は一度で取れるが本体開発が止まり、レビュー不能な差分になる。

## 影響

- `REGENERATIVE_ARCHITECTURE.md` §3.2 が Architecture を「構成」から「設計」へ広げ、散文と台帳の分離、
  根拠のインライン化規則、文書体系の動詞表を持つ。§3.8 の物理配置図に `architecture.yaml` と
  per-context `ARCHITECTURE.md` が加わる。
- `ARCHITECTURE_FORMAT.md` が `ARCHITECTURE.md`（散文）と `architecture.yaml`（台帳）の 2 記法を定める。
  root の本文セクションは 9 つ、per-context は `## Overview` のみ必須で話題別の自由見出しを許す。
- `SPECIFICATION_CORE_LANGUAGE.md` §1 の「ADR または `ARCHITECTURE.md`」が選択規則に置き換わる。SCL 自身の
  スキーマと規範要素は変わらない。
- `ADR_FORMAT.md` に ADR を起こす条件と分量の目安が加わる。
- `tools/ra` が `architecture.yaml` をツリー走査で発見し、複数ファイルの module を 1 つのグラフへ合成する。
  `tools/check` の Architecture schema が台帳用と文書用の 2 本に分かれ、`arch-check` の本文見出し検査が
  root の 9 セクションを認識する。`decisions/ADR-*.md` へのリンクの実在検査を `check-ids` に加える。
- `frontend/ARCHITECTURE.md` が初めて検証対象に入る。
- `.agents/skills/implement-work-item` と `.agents/skills/new-architecture` の完了条件が更新される。
- SCL の規範要素・契約・非機能保証は変更しない。
