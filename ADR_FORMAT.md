# 決定記録（ADR）

重要な決定とその理由を残すレコード。SCL から派生し SCL に反映される。配置は `decisions/` に置く。

- ファイル名は `decisions/ADR-NNN-kebab-title.md`。`NNN` はそのコンテキスト内の 3 桁ゼロ詰め連番。
- 廃止した ADR も**削除せず**残す（過去の決定経緯は再生成の文脈になる）。

ADR は次のような構成となる。「決定」と「却下した代替案」の間に、決定を補足する任意のトピック節を置いてよい。

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
