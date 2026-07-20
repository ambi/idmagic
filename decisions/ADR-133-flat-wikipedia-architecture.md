---
status: accepted
authors: [tn]
created_at: 2026-07-20
supersedes: [ADR-047, ADR-070, ADR-090, ADR-130]
---

# ADR-133: Adapter を役割と技術で命名して feature 直下へフラット配置する

## コンテキスト

ADR-047、ADR-090、ADR-130 により context ownership、context-local persistence、feature vertical
slice は確立したが、`adapters/persistence/postgres` のような分類用ディレクトリが各 feature に
反復している。深い階層はファイルを開くまで同じ語が続き、エディタの一覧と検索結果から package の
具体的責務を読み取りにくい。ADR-070 の technical shared context も、複数技術を同一 Adapter
package に集約する箇所では同じ問題を残している。

## 決定

context ownership と feature vertical slice は維持しつつ、現在の配置規約は
[ARCHITECTURE.md](../ARCHITECTURE.md) の Flat Wikipedia Architecture に置き換える。
Core は feature 直下の `domain`、`ports`、`usecases` とし、Adapter は分類用の
`adapters` / `persistence` を介さず、役割と技術を表す snake_case package として feature 直下へ置く。

この変更は物理配置と Go import path の変更に限定する。異なる feature の同名 Core package や
同種 Adapter を同時に使う側は named import で区別し、所有 feature や Core package 名へ冗長な
接頭辞を加えない。technical shared context は capability ごとの feature を設け、技術に依存しない
契約を `ports` に、実装を技術別 Adapter に分ける。

## 却下した代替案

- 現在の `adapters/persistence/<technology>` を維持する: 所有境界は保てるが、反復する分類階層と
  package 名だけでは役割が分からない問題を解消しない。
- `user_domain` や `group_ports` のように全 package 名へ feature 接頭辞を付ける: import alias で
  解決できる衝突を永続的な冗長名へ転嫁し、Core の共通語彙を崩す。
- Adapter を context 横断の技術ディレクトリへ再集約する: 階層は浅くなるが context ownership と
  feature locality を失い、ADR-047/090 で解消した変更波及を再導入する。
- 旧 import path の shim を残す: 移行は容易になるが、禁止する中継ぎ階層を正規構造として残す。

## 影響

- Go import path、package 宣言、composition root、sqlc query/output path、Architecture module path が変わる。
- ADR-047/070/090/130 の context ownership、shared capability、context-local persistence、feature
  vertical slice の判断は維持し、それらが規定した Adapter の物理階層だけを本 ADR で置き換える。
- HTTP、SCL、DB schema、認可、運用上の振る舞いは変わらない。
