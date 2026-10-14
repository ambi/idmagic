---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# SCL flows を views/sees/does 記法へ再設計し CEL root binding を形式定義する

`flows` の記法を `entry`+`transitions` のフラット配列から、`entry`+`views`(view 名を
キーとする map、各 view が `sees` と `does`(action の配列)を持つ)へ再設計した。
JSON Schema(`Flow`/`FlowView`/`FlowAction`)、`scl-semantics.ts` の到達可能性 BFS
(view キー全件が `entry` から到達可能であることを検証するよう強化)・action 重複検出・
interface 参照解決、`scl-to-html` の型定義・レンダラー(view ごとに sees を見出しにして
does の表を並べる形へ変更)・テストを更新し、既存 10 context ファイル・32 flow 全件と
`scl-to-html` 自身の自己記述 SCL(1 flow)を新記法へ移行して実際の画面内容・操作内容を
記述した。あわせて `SPECIFICATION_CORE_LANGUAGE.md` を全セクション書き直し、CEL の
各 root binding(`context`/`subject`/`resource`/`principal`/`input`/`output`/`response`/
`measurement`/`request`/`event`/`emitted`)の実体を新設の §5.1 で定義し、
`interfaces` のネスト `fields` 形、`bindings.kind` 別必須 field、`states.*.guard` の
無接頭辞例外などドキュメントの構造的な抜けを埋めた。ADR-112 でこの決定
(ADR-103 §5 の「view 別台帳を持たない」を部分的に上書きすること含む)を記録した。
`spec_version` は `"3.0"` のまま据え置いた(単一 work item 内の一括移行のため)。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
