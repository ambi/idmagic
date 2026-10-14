---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# 退職日 (termination_date) を組み込みユーザー属性として追加する

`backend/identitymanagement/domain/attributes.go` の組み込み属性カタログに
`termination_date` (`AttributeTypeDate`、`hire_date` と同じ `org()` 定義:
`EditableByUser: false` / `AttrVisibilitySelfReadable`) と日本語ラベル
「退職日」を追加した。SCL (`spec/scl.yaml` / `docs/domain/*.yaml`) には
個別の組み込み属性キーを列挙する箇所が存在しない (`UserAttributeDef` は
型としてのみ SCL に現れ、実データは Go 側が正本) ため、SCL 変更は不要と
判断した。フロントエンドは attribute defs を動的に取得して描画するため
UI コード変更も不要。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
