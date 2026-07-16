---
status: accepted
authors: [tn]
created_at: 2026-07-17
---

# ADR-116: Architecture map を依存方向と複雑度を検査する実行可能な宣言にする

## コンテキスト

従来の Architecture frontmatter は少数のパスと SCL realization を列挙するだけで、全 context、
RA layer、module 間依存、runtime composition、実 import、複雑度上限を表現できなかった。そのため、
本文上の依存方針と実装が乖離しても、パスが存在する限り検証を通過した。

context map の論理的な依存とソースコードの import は同一ではない。公開 interface、binding、横断的な
technical shared、composition root を区別せず cross-context import を禁止すると、正当な実装まで
違反になる。一方、path wildcard や無期限の除外で例外を許すと、境界と複雑度の debt が恒久化する。

## 決定

`ARCHITECTURE.md` の frontmatter を、context、RA layer、module role、宣言依存、runtime unit、
complexity budget を持つ実行可能な Architecture map とする。module は
`specification_core -> decision_record -> domain -> use_cases -> adapters -> infrastructure ->
deploy_pipeline` の内層から外層への順序へ所属し、依存は同層または内側だけを許可する。

cross-context 依存は、`published_interface`、`binding`、`technical_shared`、`composition_root` の役割を
持つ module を介した宣言済み edge に限る。Architecture realization は ADR-114 の context-qualified
direct SCL element reference を使用し、参照先 context と module context の局所性を検査する。
runtime unit は `api`、`worker`、`relay`、`ui` の kind、実在する entrypoint、composition 対象 module を
宣言する。

source line 数と React local-state hook 数の budget を Architecture に宣言し、新規超過を失敗させる。
既存違反だけは ceiling ratchet の debt として許可し、現在値を超える増加を直ちに失敗させる。debt は
owner、reason、解消する work item、期限を必須とし、期限切れも失敗させる。import と複雑度の検査から
生成物、vendor、`node_modules`、Go/TypeScript の test source を除外し、製品コードの構造だけを測る。

## 却下した代替案

- context map の `depends_on` を source import と一対一に対応させる案: 公開面や binding の実装経路を
  表現できず、false positive を生む。
- cross-context import を path wildcard で例外化する案: 例外の責務と依存経路が map に残らない。
- budget 違反を一括して警告にする案: 新規違反と既存 debt の増加を変更時に止められない。
- 既存違反を無期限の ignore list に置く案: 解消責任と期限がなく ratchet が機能しない。
- test source の import も製品依存として扱う案: fixture や外部 package test が runtime 境界を表さない。

## 影響

- `ARCHITECTURE_FORMAT.md` と Architecture JSON Schema が map の正規形を定義する。
- `tools/yaml-check` は schema、path、SCL reference、dependency、runtime、budget/debt を検査する。
- `tools/ra` は workspace の Go/TypeScript import と Architecture 宣言を照合して report を返す。
- 既存の budget 超過は後続 work item と期限を持つ ceiling debt として root `ARCHITECTURE.md` に記録する。
