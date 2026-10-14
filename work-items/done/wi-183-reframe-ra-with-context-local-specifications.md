---
status: cancelled
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# RA を SCL 単一上流方式からコンテキストローカルな複数仕様形式へ再定義する

一度 RA 本文の改訂（§2・§3.1・§3.2〜§3.4・§3.8・§4）、`SPECIFICATION_CORE_LANGUAGE.md` の
仕様ポートフォリオ形式への刷新、agent skills・`AGENTS.md`・tooling framing の同期、そして
既存 13 context の SCL YAML のうち 8 context（claim-mapping・audit・scim・jobs・saml・
signing-keys・ws-federation・system・tenancy）を Markdown 仕様ポートフォリオへ実移行して
試した。
実施の過程で、元の SCL 単一形式と比べていろいろな不都合が生じることが分かった。
たとえば、関心ごとにファイルを分割すると `overview.md` が契約の名前を機械的に列挙するだけの
空疎な文書になりやすい、`invariants`／`user_experience` を蒸留すると規則や UI 仕様の一覧性・
検索性が SCL 台帳より劣化する、`states.mmd` を複数持つ context での区別や `contracts.tsp` の
型忠実度（`Record<unknown>` への逃げ）などパターンの一貫性を保つコストが高い、といった問題が
レビューで指摘された。1 個の exemplar（audit）を改訂した時点で、SCL 単一形式が持っていた
一覧性・機械検証性・保守コストの低さを、分割形式で上回れる見通しが立たなかった。
そのため、今回の変更方針は本 work item として**塩漬け（一旦見送り）**とし、着手前の状態へ
戻した。`REGENERATIVE_ARCHITECTURE.md`・`SPECIFICATION_CORE_LANGUAGE.md`・agent skills・
`AGENTS.md`・`tools/ra/src/main.ts`・`justfile`・`spec/scl.yaml`・`docs/domain/*.yaml`
（13 context 全て）を、本 work item 着手前の commit（`31bf9bd9`）の内容まで
`git reset --hard` で復元し、それに伴う 14 commit（wi-183・wi-207・wi-208・wi-209 系列）を
履歴から完全に削除した。`spec/templates/` など新規追加していたディレクトリ・ファイルも
併せて削除された。再挑戦する場合は、今回レビューで指摘された「overview の空疎化」「台帳蒸留に
よる一覧性低下」「パターンの一貫性維持コスト」を先に解消する設計を用意すること。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
