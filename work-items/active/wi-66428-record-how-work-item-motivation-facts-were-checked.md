---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-11-28
priority: p2
group: development-process
depends_on: []
change_kind: docs
spec_impact: { kind: none, reason: "作業項目の動機に確かめ方を添え、確かめていない前提を利用者に確認する起票の手順と形式の変更だけで、製品の HTTP 応答、永続状態、イベント、規範要素は変えない。" }
---

# 作業項目の動機に書く事実に確かめ方を添え、確かめていない前提を起票時に利用者へ確認する

## 動機

`new-work-item` の手順 3 は、動機に書く事実（未実装である、検査やタスクがある、ファイルや節がある、件数）を書く前に `rg`、`fd`、`mise tasks` で確かめると定める。
しかし、どのコマンドで確かめたかは記録に残らない。
`implement-work-item` の手順 2 は、起票の後に動機が名指すパスが変わったときに前提を確かめ直すが、何を確かめれば前提が成り立つかを起票時の記録から読み取れないので、同じ調べ方を推測し直すことになる。

また、エージェントが確かめられなかった前提（利用者の意図、外部の製品の振る舞い、要約しか読めなかった知見）は、確かめた事実と同じ書き方で動機に並ぶ。
後から読む人は、どれが確かめた事実で、どれが仮定かを区別できない。

AWS の AI-DLC は、最初の段階（Intent Capture）の成果物の主張に出典の印を付け、仮定と未解決の問いを別に示す。
残す仮定は、レビューと承認の前に利用者が明示的に確認しなければならず、`claim-sources` センサーがそれを検査する。

## 対象範囲

- `work-item-format.md` に、動機の事実に確かめ方を添える形を定める。
  - 件数、有無、実在の主張には、確かめたコマンドかパスを添える（例: 「`rg -l 'Agent-Friction' work-items` で 0 件」）。
  - 文書の記述の主張は、節へのリンクで足りる。
- 確かめていない前提を、動機の中で前提であると分かる形で書く規則を定める。
- `new-work-item` に、確かめていない前提が作るものを変え得るときは、起票の前に利用者へ確認する手順を足す。

## 対象外

- 確かめ方の有無を機械で検査すること。主張かどうかの判定は誤検出しうるので、[書くものと書かないもの](../../docs/development/agent-guidance.md#書くものと書かないもの)の方針どおり文章に留める。
- 既存の作業項目の動機を書き直すこと。新しく起票する項目から適用する。
- 外部の知見の出典と調査日。`new-work-item` の手順 4 がすでに定める。

## 設計

出典: AWS AI-DLC のドキュメント「Phases and Stages」の Intent Capture と「Rules and the Learning Loop」の `claim-sources` センサー（https://awslabs.github.io/aidlc-workflows/guide/）。本文を読んだ。調査日は 2026-11-28。

AI-DLC から採る点は、確かめた主張と仮定を分けて書くこと、残す仮定を利用者に確認することである。
採らない点は、すべての主張に出典の印を付けて機械で検査することである。
IdMagic の動機は短く、件数や有無の主張だけに確かめ方を添えれば足りる。

確かめ方を添えると、`implement-work-item` の手順 2 で前提を確かめ直すときに同じコマンドを再実行できる。

## 計画

1. `work-item-format.md` の動機の節に、確かめ方と前提の書き方を足す。
2. `new-work-item` の手順 3 を、確かめ方を添える形に書き換え、前提を確認する手順を足す。
3. `implement-work-item` の手順 2 から、添えた確かめ方を再実行する一文を足す。

## タスク

- [ ] T001 [Docs] `work-item-format.md` に確かめ方と前提の書き方を書く。
- [ ] T002 [Docs] `new-work-item` の手順を更新する。
- [ ] T003 [Docs] `implement-work-item` の手順 2 を更新する。
- [ ] T004 [Verify] 検査を通す。

## 検証

- `mise run check-agent-guidance`: skill から呼ぶタスクとリンクの形式を確かめる。
- `mise run check-links`: 文書間のリンクを確かめる。
- `mise run check-work-items`: 記録の形式を確かめる。

## リスク

確かめ方を添えると動機が長くなる。
件数、有無、実在の主張だけに限ることで抑える。
