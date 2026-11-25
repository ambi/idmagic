---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-11-25
priority: p2
group: development-process
depends_on: []
change_kind: tooling
spec_impact: { kind: none, reason: "エージェントが作業中に受けた妨げをコミットのトレーラーに残し、集計するタスクと昇格の手順を足すだけで、製品の HTTP 応答、永続状態、イベント、規範要素は変えない。" }
---

# エージェントが作業中に受けた妨げをコミットに残し、繰り返したものを指示と検査へ昇格させる

## 動機

[エージェント向けの指示](../../docs/development/agent-guidance.md#書くものと書かないもの)は、同じ指摘を繰り返し受けたら所有する文書か skill へ書き、一度だけ起きた失敗は書かないと定める。
しかし、繰り返したかどうかを判断する材料がリポジトリにない。
エージェントのセッションは前のセッションを覚えておらず、個人メモリの指摘は wi-60157 でリポジトリの文書へ移したので、繰り返しに気付けるのは利用者の記憶だけである。
`implement-work-item` の手順 3 も、反復する作業が高コストなら起票すると定めるが、判断の材料は今のセッションの中の観測に限られる。

その結果、指示の改善は二方向とも偶然に頼っている。
古い節、誤ったタスク名、足りない前提でエージェントが迷っても、記録がなければ次のセッションも同じ所で迷う。
逆に、検査へ昇格させた後も文章に残った規則や、もう誰も迷わない注意は、削る根拠がないまま毎回読まれる。

Thoughtworks Technology Radar Vol.35（2026-11）は、セッションの終わりに共有の成果物を変えるべきかを問い、チームで更新を決める **feedback flywheel** を Trial に置いた。
同じ版は、エージェントが出会った問題を報告する場所を設ける agent suggestion box を Assess に、追加だけを続けて長く矛盾した指示になる agent instruction bloat を Caution に置いている。

## 対象範囲

- 妨げを記録するコミットのトレーラー `Agent-Friction:` の形式を `docs/development/agent-guidance.md` に定める。値は「対象の参照」と「起きたこと」の組で、対象はリポジトリ相対のパス（見出しのアンカーを付けてよい）か `mise run <task>` とする。
- `implement-work-item` の手順 7 と `new-work-item` の末尾に、正式コミットの前に一度だけ問う手順を足す。問いは「AGENTS.md、skill、開発文書、検査、`mise` タスクのうち、誤っていた、足りなかった、同じ作業を繰り返させたものはあったか」で、あれば一件ごとにトレーラーを書く。
- `mise run report-agent-friction` を `tools/` に足す。`git log` のトレーラーを対象ごとに集計し、件数、日付、コミットを示し、2 件以上の対象と、現在のツリーに存在しない対象を分けて表示する。形式に合わない行は捨てずに別に表示する。
- 集計を読んで変更を決める手順を `agent-guidance.md` に書く。2 件以上の対象は既存の「書くものと書かないもの」の基準で判断し、機械で判定できるものは検査へ昇格させ、昇格したら[毎回読む節を縮める](../../docs/development/agent-guidance.md#毎回読む節を縮める)の表に従って文章を縮める。決めた変更は作業項目として起票する。

## 対象外

- セッションの記録（会話、ツールの出力）の保存と解析。エージェントの実行環境ごとに形式が違い、どの環境でも成り立つ規約だけを書くという `agent-guidance.md` の基準に合わない。
- エージェントが集計から指示を自動で書き換える仕組み（Radar の self-improving harness）。変更の採否は人が決め、作業項目として通す。
- 代表的な依頼の評価シナリオ（`evals.json`）。wi-11469 で一度設計し、採らなかった。
- 特定のエージェントのサンドボックスや権限で起きた妨げ。`agent-guidance.md` が書かないものとして除いているので、トレーラーにも書かない。

## 設計

記録の置き場所を次の案から比べ、コミットのトレーラーを採る。

| 案 | 採否 | 理由 |
| --- | --- | --- |
| 正式コミットのトレーラー | 採る | 新しいファイルを足さない。`integrate-work-item` の rebase はメッセージを保つので、取り込み後も残る。並列の worktree でも衝突しない。`Spec-Impact:` と同じく `git log` から機械で集計できる |
| 追記していく台帳のファイル | 採らない | 並列の worktree で同じ行の付近に追記して衝突する。Radar Vol.35 が Caution に置く AI-driven documentation sprawl の一つになる |
| 妨げごとの作業項目 | 採らない | 一度だけ起きた失敗まで記録になり、未完了の項目の一覧が埋もれる。繰り返しが分かった後に起票すれば足りる |
| 個人メモリ | 採らない | 共有されない。wi-60157 でリポジトリの文書へ移した |

対象の参照をパスかタスク名に限るのは、集計で同じ対象を数えるためである。
自由記述だけでは同じ妨げを同じものとして数えられない。
対象を一つに決められない妨げ（どの文書にも書いていない前提など）は、最も近い文書のパスを書く。

トレーラーを書かないセッション（調査だけで終わる会話など）の妨げは集まらない。
正式コミットを作る作業から始め、取りこぼしが問題になれば別に起票する。

### 出典

| 出典 | 読んだ範囲 | 調査日 | 使う知見 |
| --- | --- | --- | --- |
| Thoughtworks Technology Radar Vol.35、Techniques（thoughtworks.com/radar/techniques） | 本文 | 2026-11-25 | feedback flywheel（Trial）、agent suggestion box（Assess）、agent instruction bloat（Caution）。指示は追加だけでなく削除もし、命令はできるものから検査へ移す |
| Böckeler「Harness engineering for coding agent users」（martinfowler.com） | 検索結果の要約 | 2026-11-25 | 同じ問題が繰り返したら、事前の指示と事後の検査を改める。既存の `agent-guidance.md` の規則と同じ方向なので、規則は変えない |

## 計画

1. トレーラーの形式と昇格の手順を `agent-guidance.md` に書く。
2. `report-agent-friction` のテストを RED にしてから実装し、`mise.toml` に宣言する。
3. `implement-work-item` と `new-work-item` に問いの手順を足す。
4. 過去のコミットにはトレーラーがないので、集計は空から始まる。

## タスク

- [ ] T001 [Docs] トレーラーの形式、対象の参照の書き方、集計から変更を決める手順を `agent-guidance.md` に書く。
- [ ] T002 [Tooling] 集計の単体テストを RED にし、`report-agent-friction` を実装して GREEN にする。
- [ ] T003 [Skills] `implement-work-item` と `new-work-item` に正式コミット前の問いを足す。
- [ ] T004 [Verify] 変更を検証する。

## 検証

- `mise run test-tools`：集計、形式外の行、存在しない対象の表示を確かめる。
- `mise run check-agent-guidance`：skill が呼ぶ `report-agent-friction` が `mise.toml` に宣言されていることを確かめる。
- `mise run check-links` と `mise run check-terminology`：変更した開発文書と skill のリンクと表記を確かめる。

## リスク

問いが毎回の手順になると、何もなくても何かを書く誘因になり、雑音のトレーラーが増える可能性がある。
手順では「なければ書かない」を明記し、集計の判断は 2 件以上の対象に限る。
