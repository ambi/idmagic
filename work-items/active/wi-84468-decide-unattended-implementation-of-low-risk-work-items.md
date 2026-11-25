---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-11-25
priority: p3
group: development-process
depends_on: [wi-11123-analyze-untrusted-input-flows-in-development-agents]
change_kind: tooling
spec_impact: { kind: none, reason: "低リスクの作業項目をエージェントに無人で実装させる経路を試し、採否を決めるだけで、製品の HTTP 応答、永続状態、イベント、規範要素は変えない。" }
---

# 採否を判断したうえで、低リスクの作業項目を人が見ていない間にエージェントへ実装させる

## 動機

2026 年には、エージェントがコードを書き、検査し、取り込むまでを人がコードを読まずに回す **dark factory**（暗い工場、無人の工場）という呼び名が広まった。
StrongDM は、仕様とシナリオからエージェントがコードを書き、人のレビューなしに収束させる開発を公表した。
一方で、独立した検査のないまま人を外すことには反論も多い。
Thoughtworks Technology Radar Vol.35（2026-11）は、全自動ではなく、機能の大きさの作業を夕方にエージェントへ渡して翌朝に結果を見る unattended overnight agentic builds を Trial に置き、その前提をサンドボックス、事前に書いた仕様、エージェントが自分で誤りを直すための検査、行ごとのレビューに代わる検証の仕組みとしている。

このリポジトリは、その前提の多くをすでに満たしている。
作業項目は作るものを変え得る問いを実装前に解決すると定め、`implement-work-item` は RED、局所検査、最終検査、要約への縮約までを手順にし、`integrate-work-item` は rebase 後のツリーだけを `verify-integration` で検証する。
[証拠の要件](../../docs/development/specification-first-workflow.md#証拠の要件)はリスクごとに必要な証拠を定め、作業の権限は起票で与えると定める。

足りないのは、人が見ていない間に実装させる手順である。
どの項目を無人で実装させてよいかの条件、作業を始める入口、結果を後から人が確かめる報告がない。
現在、未完了の項目のうち `risk: low` は 21 件ある。

## 対象範囲

- 無人で実装させてよい項目の条件を決める。候補は、`risk: low`、`reversibility: reversible`、`depends_on` がすべて完了、`change_kind` が `docs`、`tooling`、`refactor`、`maintenance` のいずれか、`brief` が `--concern security` と `--concern boundary` を求めない対象である。
- 条件を満たす 3 件を選び、1 件ずつ別の worktree で、`implement-work-item` の正式コミットまで人が介入せずに実装させる。push と取り込みはしない。
- 3 件の結果を、利用者が後から読んで判断する。見る点は、正式コミットまで止まらずに進んだか、必要な検査が通ったか、取り込む前に人が直した箇所、エージェントが途中で止まった理由である。
- 結果から採否を決める。採るなら、条件、入口、報告の形を開発文書と skill に書く別の項目を起票する。採らないなら、止まった理由ごとに直すべき前提を起票するか、理由を要約に書いて中止する。

## 対象外

- 人が確かめないままの push と取り込み。この項目は正式コミットまでを無人にし、取り込みは人が判断する。
- `feature`、`bugfix`、`operations` の項目と、`risk` が `medium` 以上の項目。IdMagic は ID プロバイダーであり、認証、認可、テナント境界の誤りは利用者の被害になる。
- 複数のエージェントを同時に走らせて取り込む構成（Radar Vol.35 が Caution に置く coding agent swarms）。`integrate-work-item` は一つずつ取り込む前提であり、試行も 1 件ずつ行う。
- StrongDM のように、受け入れのシナリオを実装するエージェントから隠す構成（holdout）。このリポジトリは規範 ID を引用するテストとミューテーションテストで、テストを通すだけの実装を検出する。試行でテストの迂回が見つかったときに改めて検討する。
- 特定のクラウドのエージェント実行基盤の選定。

## 設計

### 採らない案

| 案 | 理由 |
| --- | --- |
| 全項目を対象にした dark factory | 製品の性質上、`risk` が高い変更を人が確かめずに入れる利益が被害の見込みに見合わない。Radar Vol.35 も risk-tiered human oversight（Trial）で、取り消しにくい変更を人に残すよう勧めている |
| 試行なしで手順を書く | 止まる理由が分からないまま条件と入口を決めることになる。3 件の結果から決める |

### 先に完了させる項目

wi-11123 を先に完了させる。
無人の実装では、エージェントが依存の文書や外部のページを読んだ後に外部へ送る操作を、途中で人が止められないためである。

### 出典

| 出典 | 読んだ範囲 | 調査日 | 使う知見 |
| --- | --- | --- | --- |
| Thoughtworks Technology Radar Vol.35、Techniques（thoughtworks.com/radar/techniques） | 本文 | 2026-11-25 | unattended overnight agentic builds（Trial）とその前提、risk-tiered human oversight（Trial）、human-in-the-loop theater（Caution）、coding agent swarms（Caution） |
| Simon Willison「How StrongDM's AI team build serious software without even looking at the code」（simonwillison.net、2026-02-07） | 本文の前半 | 2026-11-25 | 仕様とシナリオからエージェントが人のレビューなしに収束させる。シナリオをコードの外に置いて holdout にし、エージェントが常に真を返すテストで通すことを防ぐ |
| Addy Osmani「Software Factories, Light and Dark」、O'Reilly Radar の論考 | 検索結果の要約 | 2026-11-25 | 人が読まないコードを機械だけで検証して出す構成と、独立した検査なしに人を外すことへの反論。用語の説明にだけ使う |

## 計画

1. wi-11123 の完了後に、条件を満たす項目を一覧にし、3 件を選ぶ。条件を満たす項目が 3 件に満たなければ、満たす件数で行う。
2. 1 件ずつ、別の worktree で実装させ、結果を確かめる。
3. 採否を決め、採るなら手順を書く項目を起票する。

## タスク

- [ ] T001 [Plan] 条件を満たす項目を一覧にし、試行の 3 件を選ぶ。
- [ ] T002 [Trial] 3 件を無人で実装させ、結果を確かめる。
- [ ] T003 [Decide] 採否を決め、後続の項目を起票するか中止する。

## 検証

- 試行した各項目の正式コミットについて、`implement-work-item` が選んだ検査を人が再実行して結果が同じであることを確かめる。
- `mise run check-work-items`：後続の項目を起票した場合に形式を確かめる。

## リスク

無人の作業が、作業場所の外のファイルや外部のサービスへ書き込む可能性がある。
試行は push 先の資格情報を渡さない作業場所で行い、wi-11123 の規則を前提にする。
