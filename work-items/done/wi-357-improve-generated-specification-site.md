---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# 生成仕様 HTML を分割し、図・scenario・API・model を人が読める仕様サイトにする

単一の長大 HTML を 767-page の静的 specification site に置き換え、Method、whole-system、各 bounded
context、Swagger UI API reference、検索可能な TypeSpec model catalog を独立して閲覧できるようにした。
DDD Context Map と transition table 由来の Mermaid diagram、scenario keyword の semantic styling、
responsive/light/dark/print layout、全 generated link の到達性検査を追加した。
root 文書は application 名に依存しない `Whole-System Specification` とし、current specification から
historical ADR link を除去して現行 rationale を Design に残した。モデル監査では `Operations` transport
wrapper 1,805件を Swagger UI に委譲し、それ以外の宣言型740件を property 型・必須性・constraint・参照と
ともに catalog 化した。既存 TypeSpec には539 symbol page、合計2,220箇所の description 欠落があったため、
推測文は生成せず `No description.` として明示した。
`tools/` から application 名と固定 module/OpenAPI path を除去した。TypeSpec ownership は compiler の
project source location、OpenAPI current/baseline は標準 directory の一意 artifact、Go import root は
`go.mod` から導出する。別名 fixture と曖昧 artifact の fail-closed test を追加した。
目視 feedback 後、Mermaid の edge/arrow と node border を light/dark theme ごとの高 contrast 色へ
固定し、無条件遷移の Guard 91件を `""` から `—` へ正規化した。Swagger UI は dark page 内でも
自身の配色が崩れない light color-scheme surface に隔離した。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
