---
name: new-adr
description: Create a new Architecture Decision Record (ADR) following the canonical format. Use when recording an important decision and its rationale, or when the user asks to draft/write an ADR or ADR-NNN.
---

# 新規 ADR（決定記録）の作成

正本書式は `ADR_FORMAT.md`。**既存ファイルを開いて書式を逆算しない**。
ADR は SCL から派生し SCL に反映される決定記録。**ADR が持つのは SCL から再導出できない「なぜ」だけ**——規範要件は SCL、現在の構成は `ARCHITECTURE.md` に置き、ADR はそれらを*参照*する。境界と移管の扱いは `ADR_FORMAT.md`「役割の境界」を見る。

## 手順

1. **採番する**
   - ファイル名は `decisions/ADR-NNN-kebab-title.md`。`NNN` はコンテキスト内の 3 桁ゼロ詰め連番。
   - **連番は再利用しない。廃止した ADR も削除せず残す**（過去の決定経緯は再生成の文脈になる）。
   - 並行採番で被っても `just check-ids` が検出するので、被ったら採り直す。
2. 下記スケルトンの **5 つの必須章をこの順で**書く。「決定」と「却下した代替案」の間に補足トピック節（例 `## 鍵のサイズと曲線`）を置いてよいが、必須章は置き換えない。
3. 決定が SCL に反映されているときは、**「ステータス」または「影響」で対応する SCL 要素**（`interfaces.Xxx` / `models.Yyy` 等）とソースを相互参照する。
4. 過去の決定を置き換えるなら、frontmatter で双方向に張る。新しい側に `supersedes: [ADR-NNN]`、古い側に `superseded_by: [ADR-NNN]`（全体廃止なら古い側 `status: superseded`、一部上書きなら `accepted` のまま）。詳細は `ADR_FORMAT.md`「廃止・置換（supersede）」。`just check-ids` が相互整合を検証する。

## スケルトン

```markdown
---
status: suggested       # suggested | accepted | rejected
authors: [name]
created_at: 2026-01-01  # YYYY-MM-DD
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
