# 決定記録（ADR）

重要な決定とその理由を残すレコード。SCL から派生し SCL に反映される。配置は `decisions/` に置く。

- ファイル名は `decisions/ADR-NNN-kebab-title.md`。`NNN` はそのコンテキスト内の 3 桁ゼロ詰め連番。
- 廃止した ADR も**削除せず**残す（過去の決定経緯は再生成の文脈になる）。

## 役割の境界

ADR が持つのは **SCL から再導出できない「なぜ」**——採用 / 却下した選択肢、判断の前提、当時の制約、見直し条件、運用上の学習——だけである。次は ADR に書かない。

- **規範要件・振る舞い・契約・データ形状**は SCL（`spec/`）に置く。ADR は結果として変わる SCL 要素を「影響」節で*参照*する（本文を転記しない）。
- **現在の構成**（採用スタック、モジュール台帳、ディレクトリ規約、依存方向）は `ARCHITECTURE.md` に置く。ADR はそこへリンクする。

既存の ADR が仕様や構成そのものを抱え込んでいると気づいたら、その内容を SCL / `ARCHITECTURE.md` へ移し、ADR 本文には移管先へのポインタと「なぜ」だけを残す。**ADR は削除しない**（上の削除禁止と同じ理由）。移管そのものが非自明な判断を含むなら新しい ADR を起こす（例: ADR-105〜109）。

## 廃止・置換（supersede）

後の ADR が過去の決定を置き換えたら、両者の frontmatter で関係を**双方向**に張る。**古い側のファイルは削除・改変しない**（本文の該当箇所への一文注記は可）。

- 新しい ADR（置換する側）: `supersedes: [ADR-NNN]`
- 古い ADR（置換される側）: `superseded_by: [ADR-NNN]`
- 決定の**全体が無効化された**ら、古い ADR の `status` を `superseded` にする。
- 決定の**一部だけが上書き**され残りが有効なら、古い ADR の `status` は `accepted` のまま `superseded_by` だけ張り、本文の該当箇所に何がどの ADR へ移ったかを一文添える。

`just check-ids` が次を検証する: `supersedes` と `superseded_by` の相互整合、参照先 ADR の実在、`status: superseded` に後継（`superseded_by`）が張られていること。

ADR は次のような構成となる。「決定」と「却下した代替案」の間に、決定を補足する任意のトピック節を置いてよい。

```markdown
---
status: suggested       # suggested | accepted | rejected | superseded
authors: [name]
created_at: 2026-01-01  # YYYY-MM-DD
# supersedes: [ADR-NNN]     # この ADR が置き換える過去の決定（任意）
# superseded_by: [ADR-NNN]  # この ADR を置き換えた後の決定（任意）
---

# ADR-NNN: 一文で表す決定

## コンテキスト
なぜ今この決定が要るか。

## 決定
何を決めたか。

## 却下した代替案
- 案 A: なぜ採らないか。

## 影響
- 新規 / 変更される SCL 要素、契約、データ、運用。
```
