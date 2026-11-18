---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-11-19
priority: p2
group: development-process
depends_on: []
change_kind: docs
spec_impact:
  kind: none
  reason: "エージェント向けの指示（AGENTS.md、.agents/skills）と開発文書を直すだけで、外部から観測できる振る舞い、TypeSpec の契約、規範要素は変えない。"
---

# 繰り返し受けた指摘と外部の知見を、エージェント向けの指示と開発文書へ反映し、指示の変更を確かめる手順を持つ

## 動機

エージェントが働く前提となる文書と skills を 2026-11-19 に棚卸しし、AI 駆動開発と仕様駆動開発の公開知見と照らした。
次の不足を確認した。

1. **繰り返し受けた指摘が、一つのエージェントの個人メモリにしかない。**
   Claude Code の利用者ごとのメモリに、このリポジトリの作業手順に関わる指摘が 6 件ある。
   作業項目の前提をコードで確かめる、起票だけのコミットは `main` に置く、統合後に `.git/config` に残るブランチ設定を片付ける、サンドボックスの外で実行する操作（worktree の `setup` と削除、`install-ui` に依存するタスク）、名前の自動検査は誤検出なら規則を削る、機能の有無を断言する前に索引から確かめる、である。
   どれも `rg` で探した範囲ではリポジトリの文書と skills に書かれていない（AGENTS.md はブラウザーと localhost を使うタスクだけを挙げる）。
   Codex など別のエージェントと、ほかの開発者の環境では、同じ失敗を繰り返す。
2. **skills の言語と description が規則に合っていない。**
   [文章の言語](../../docs/development/writing-language.md)は人が読む文章を日本語と定め、AGENTS.md は `.agents` をその対象に含める。
   しかし `new-work-item` と `integrate-work-item` は本文と description が英語、`spec-change` は本文の大半と description が英語である。
   `implement-work-item` の description は何をするかだけを書き、いつ使うかを書いていない。
   Skill の選択は description だけを読んで行われるため、使う場面の欠落は起動漏れにつながる。
3. **`new-work-item` が、起票の質を左右する手順を持たない。**
   未完了の作業項目は 156 件ある。
   重複と隣接の項目の探索、動機に書く事実のコードでの確認、外部の知見に依る項目での出典と調査日の記録が手順にない。
   今回の起票でも、[wi-99632](wi-99632-separate-reusable-practices-from-idmagic-specifics.md) と [wi-85394](wi-85394-remove-implementation-names-from-design-docs.md) との重なりを手で探した。
4. **指示の変更を確かめる手段が静的検査だけである。**
   `.agents/` と AGENTS.md には 135 件のコミットがあるが、`check-agent-guidance` が確かめるのは、未定義の `mise` タスク、廃止した文書の参照、必須の印の有無だけである。
   指示を変えた後に、代表的な依頼でエージェントの行動が意図どおり変わったかを確かめる場面と手順がない。
5. **文書と作業項目の陳腐化を定期的に見つける手順がない。**
   2026-11-19 の走査で、設計文書が存在しないパスを名指していた（`docs/design/data/database.md` の `backend/datakeys/ports`、`docs/design/application/backend.md` の `backend/workloadidentity/usecases`）。
   設計文書の実装名は [wi-85394](wi-85394-remove-implementation-names-from-design-docs.md) が取り除く。
   一方、開発文書、skills、未完了の作業項目の前提の陳腐化は、作業の途中で偶然見つかるのを待っている。
   2026-11-14 の整理では、未完了の 4 件の前提が解消済みか逆転していた。
6. **エージェント向けの指示をどこへ何を書くかの規則がない。**
   [開発文書の索引](../../docs/development/README.md)には、AGENTS.md、skills、開発文書の役割分担、長さの方針、繰り返す指摘の昇格先を定める文書がない。
   また `docs/development/e2e-runner-comparison.md` が索引から漏れている。

開発の検証の網を広げ、ほかの全グループの作業の質に効くため、グループの基準どおり p2 とする。

## 対象範囲

- エージェント向けの指示の保守を定める開発文書を一つ追加し、開発文書の索引と AGENTS.md から参照する。
- 動機 1 の 6 件を、それぞれを所有する文書または skill へ移す。
- `new-work-item`、`integrate-work-item`、`spec-change` の本文と description を日本語にし、全 skill の description を「何をするか」と「いつ使うか」の形にそろえる。
- `new-work-item` に、重複の探索、前提の確認、外部の知見の出典と調査日の記録を加える。
- 文書と作業項目の陳腐化を探して直す skill を一つ追加する。
- repo skills ごとに代表的な依頼の評価シナリオを置き、skill を変えるときの確認手順を新しい開発文書に書く。
- `docs/development/e2e-runner-comparison.md` を開発文書の索引へ載せる。

## 対象外

- 設計文書から実装のパスと名前を取り除く作業。
  [wi-85394](wi-85394-remove-implementation-names-from-design-docs.md) が扱い、本項目は動機 5 の 2 件を同項目の対象として扱う。
- 汎用の方法論と IdMagic 固有の内容の分離。
  [wi-99632](wi-99632-separate-reusable-practices-from-idmagic-specifics.md) が扱う。
  本項目で書く規則は IdMagic の文書として書き、分離の判断を先取りしない。
- 評価シナリオを LLM で自動実行する CI とタスク。
  費用と非決定性に見合う効果がまだ示せないため、手順は人が起動するエージェントで行う。
- 利用者のマシンにある個人メモリの削除。
  リポジトリへ移した後に、利用者が要否を判断する。
- `~/.agents/skills/` にあるマシン全体の skills の変更。
- 仕様駆動開発の外部ツール（Spec Kit、Kiro、OpenSpec、Tessl）の導入。
  理由は設計に書く。

## 設計

### 調査の出典

2026-11-19 に調べた。
一次情報を読めたものと、二次情報の要約しか得られなかったものを分ける。

| 出典 | 読んだ範囲 | 本項目で使う知見 |
| --- | --- | --- |
| Anthropic「Skill authoring best practices」（platform.claude.com） | 本文 | description は三人称で何をするかといつ使うかを書く。本文は 500 行未満。参照は SKILL.md から 1 段。評価を先に作り、skill なしの基準と比べる。最低 3 件のシナリオ。指示を書くインスタンスと試すインスタンスを分ける。繰り返す確認は検証の手順として書く |
| Gloaguen ほか「Evaluating AGENTS.md」（arXiv:2602.11988） | 本文 | LLM が生成した文脈ファイルは解決率を上げず費用を約 20% 上げる。開発者が書いたものは平均で約 2.4% 上げるが有意ではない。リポジトリの概要は役に立たない。名指したツールはよく使われる。README などにない固有の規約だけを書くよう勧める |
| Vasilopoulos「Codified Context」（arXiv:2602.20478） | 要旨 | 常時読む短い規約と、必要時に読む仕様文書の二層に分けると、セッションをまたいだ一貫性を保てた |
| OpenAI「Harness engineering」（2026-02） | 二次情報の要約 | AGENTS.md は約 100 行の地図にし、知識は `docs/` を一次情報とする。繰り返す指摘は文書から検査や道具へ昇格させる。陳腐化した文書を探す作業を定期的に回す |
| Böckeler「Understanding Spec-Driven-Development」（martinfowler.com、2025-10） | 二次情報の要約 | spec-first、spec-anchored、spec-as-source の 3 段階を区別する |
| 2026 年の仕様駆動開発の解説記事（dev.to ほか） | 検索結果の要約 | 実装前の曖昧さの解消、対象外の明記、仕様とコードを同じ変更で更新する |

2026-10 以降の一次情報で、上の知見を覆すものは見つからなかった。
二次情報だけの知見は、既存の規則と矛盾しない範囲でだけ使う。

### 現状との照合

外部の知見の多くは、すでにこのリポジトリが採っている。

| 知見 | 現状 | 判断 |
| --- | --- | --- |
| 仕様を実装より先に書き、仕様とコードを同じ変更で更新する | 仕様先行のワークフロー、`check-spec-impact`、完了時の検査 | 変えない |
| 実装前に曖昧さを解消する | [仕様の漏れを探す観点](../../docs/development/specification-first-workflow.md#仕様の漏れを探す観点)と作業項目の計画 | 変えない |
| 対象外を明記する | 作業項目の形式の必須節 | 変えない |
| 文脈を必要なときだけ読む | `brief` が条件付きの参照表から節を返す | 変えない |
| 繰り返す指摘を道具へ昇格させる | 検査は多いが、個人メモリにある指摘の昇格先がない | 動機 1 として扱う |
| 指示の変更を評価で確かめる | 静的検査だけ | 動機 4 として扱う |
| 陳腐化を定期的に掃除する | 偶然見つかるのを待つ | 動機 5 として扱う |

このリポジトリは TypeSpec から実行時の経路の定義を生成し、規範 ID をテストから引用するので、Böckeler の分類では spec-anchored に当たり、契約については spec-as-source に近い。
外部ツールの導入は採らない。
作業項目の形式が仕様、計画、タスク、検証をすでに一つの記録で扱い、ツールの形式へ移すと既存の検査と `brief` が働かなくなるためである。

### 新しい開発文書

`docs/development/agent-guidance.md`（題は「エージェント向けの指示」）を追加する。
[開発文書の形式](../../docs/formats/documentation-guide.md#開発文書)に従い、次の節を持つ。

- **置き場所**：AGENTS.md は作業ごとの参照先の表と、どの作業にも効く短い規則だけを持つ地図にする。手順は skills、規則と判断は開発文書と設計文書が持つ。製品の概要とリポジトリの構成の説明を AGENTS.md に足さない（上の Gloaguen ほかの結果）。
- **繰り返す指摘の昇格**：同じ指摘を 2 回受けたら、所有する文書か skill へ書く。機械で確実に判定できるものだけを検査にし、名前の検査のように誤検出しやすいものは文章に留める。
- **skill の書き方**：description の形、本文の長さ、参照の深さ、言語。
- **skill の評価**：シナリオの形式と、変更時の確認手順（下記）。

AGENTS.md の表と `brief` の仕組みは変えない。
表は `brief` の一次情報であり、地図の役割をすでに果たしているためである。

### 指摘の移し先

| 指摘 | 移し先 |
| --- | --- |
| 作業項目の前提をコードで確かめる | `new-work-item` と `implement-work-item` の着手前の確認、`garden-work-items`（下記） |
| 機能の有無を断言する前に索引から確かめる | `new-work-item`。比較と提案の結論を左右するときに限る |
| 起票だけのコミットは `main` に置く | [作業項目](../../docs/development/specification-first-workflow.md#作業項目)の節 |
| 統合後に残るブランチ設定を片付ける | `integrate-work-item` の手順 5 |
| サンドボックスの外で実行する操作 | AGENTS.md の `mise` の節。既存の列挙へ、worktree の `setup` と削除、`install-ui` に依存するタスクを足す |
| 名前の自動検査は誤検出なら規則を削る | [リポジトリ検査](../../tools/check/README.md#検査の判定基準)の判定基準 |

UUID の version を「バージョン」と書く指摘は、[wi-89473](wi-89473-apply-version-vocabulary-to-modules-and-backend.md) が扱う version の表記の規則に含まれるので移さない。

### 陳腐化を探す skill

`garden-work-items` を追加する。
利用者が「整理して」「棚卸しして」と頼んだときと、一定数の作業項目を完了した後に使う。
次の順に進み、直せるものは直し、判断が要るものは利用者へ一覧で示す。

1. `mise run check-links`、`check-agent-guidance`、`check-work-items` を実行する。
2. 文書と skills のコード表記のうち、リポジトリのパスと `mise run` のタスク名に見えるものを列挙し、存在しないものを候補にする。
   `tools/list` のような MCP のメソッド名も一致するため、候補は人が判定し、検査にはしない。
3. 未完了の作業項目のうち作成日の古いものから、動機が名指すコード、タスク、節を確かめ、解消済みなら中止、前提が崩れていれば日付付きで書き直す。

### 評価シナリオ

各 repo skill のディレクトリに `evals.json` を置き、3 件以上のシナリオを書く。
各シナリオは依頼文、前提の状態、期待する行動の列からなる。

```json
[
  {
    "query": "wi-12345 を実装して",
    "state": "作業項目が pending で、既定のチェックアウトが main でクリーン",
    "expected_behavior": [
      "既定のチェックアウトで work-item/wi-12345 ブランチを切る",
      "最初のコード編集の前に brief を実行する"
    ]
  }
]
```

skill を変えるときは、指示を書くエージェントとは別の新しいセッションで、変更前と変更後の skill に同じシナリオを与え、期待する行動の達成を比べる。
結果は記録に書かない。
実行で確かめられない自己申告を残さない[完了](../../docs/formats/work-item-format.md#完了)の方針に合わせるためである。
`check-agent-guidance` は、各 repo skill に `evals.json` があり、JSON として読めることだけを検査に加える。

採らない案：

- シナリオを CI で LLM に実行させる。費用と非決定性があり、失敗の判定にも LLM が要るため、検査の信頼性が既存の検査より低い。
- 結果を作業項目に記録させる。上の理由で採らない。

### skill の言語と description

本文と description を日本語にする。
`name` は既存の英語のケバブケースを保つ。
起動の名前と、ほかの文書からの参照を変えないためである。
description は「〜する。〜のときに使う。」の二文にする。
本文の書き換えでは手順の中身を変えず、移した指摘の追加だけを中身の変更とする。

`update-all-dependencies/agents/openai.yaml` は Codex の表示用メタデータであり、ほかの skill へは広げない。
効果を確かめる手段がないためである。

## 計画

1. 新しい開発文書を書き、索引と AGENTS.md から参照する。
2. 指摘を移し先へ書く。
3. 3 つの skill を日本語にし、全 skill の description をそろえ、`new-work-item` に手順を足す。
4. `garden-work-items` を追加する。
5. 評価シナリオを置き、`check-agent-guidance` に存在の検査を足す。
6. 新しいセッションで、変えた skill のシナリオを変更前後で比べる。

未解決の問いはない。

## タスク

- [ ] T001 [Docs] `docs/development/agent-guidance.md` を追加し、開発文書の索引（`e2e-runner-comparison.md` を含む）と AGENTS.md から参照する。
- [ ] T002 [Docs] 指摘を移し先の文書、skill、AGENTS.md へ書く。
- [ ] T003 [Skills] `new-work-item`、`integrate-work-item`、`spec-change` を日本語にし、全 repo skill の description を二文の形にそろえる。
- [ ] T004 [Skills] `new-work-item` に、重複と隣接の探索、前提の確認、外部の知見の出典と調査日の記録を足す。
- [ ] T005 [Skills] `garden-work-items` を追加する。
- [ ] T006 [Tooling] 各 repo skill に `evals.json` を置き、`check-agent-guidance` に存在と形式の検査を足す。先に検査のテストを RED にする。
- [ ] T007 [Verify] 変えた skill のシナリオを、新しいセッションで変更前後に与えて比べる。
- [ ] T008 [Verify] 最終検査を実行する。

## 検証

- `mise run check-agent-guidance`：skill と AGENTS.md の変更、`evals.json` の検査を直接確かめる。
- `mise run test-tools-file -- <check-agent-guidance のテスト>`：T006 の検査の追加を RED と GREEN で確かめる。
- `mise run lint-tools`：検査の変更。
- `mise run check-links`：新しい文書と書き換えた skill のリンクとアンカー。
- `mise run check-work-items`：本項目の形式。
- `mise run check`：上を含む集約。Go と UI のコードは変えないので、`verify` は選ばない。

## リスク

- `.agents/skills` はエージェントのサンドボックスで書き込みが禁止されている。
  実装時は利用者の許可を得て書き込む。
- skill の本文を訳す過程で手順の意味が変わるおそれがある。
  訳す変更と手順を足す変更をコミットの中で分け、訳だけの差分は原文と一文ずつ対応させて読む。
- AGENTS.md へ規則を足すと、全作業で読む量が増える。
  足すのはサンドボックスの外で実行する操作の列挙だけにし、ほかは skill と開発文書へ置く。
- 評価シナリオは実行が人任せなので、使われずに古くなるおそれがある。
  `garden-work-items` の手順に、変更された skill のシナリオが本文と食い違っていないかの確認を含める。
