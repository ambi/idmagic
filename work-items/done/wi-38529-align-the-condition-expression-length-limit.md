---
status: completed
---

# 条件の式の長さの上限を、契約、サーバー、画面で一つの数え方にそろえる

条件の `deny_when` と `require_approval_when` の上限を、契約、サーバー、画面で Unicode のコードポイント数による 1024 文字にそろえた。
REQ-OAUTH2-064 のバイト数による拒否を是正し、作成と更新の超過は REQ-SYSTEM-028 に従って 422 と `field_length_exceeded`、フィールド名と上限を返し、保存しない。
TypeSpec の `ResourceAccessCondition` は数え方を明記し、作成と更新の操作へ 422 の応答を加えた。
REQ-OAUTH2-090 に送信前の長さ検証を加え、両方の欄にエラーと文字数を表示する。式の空白は文字数に含める。
ASCII、日本語、BMP 外の文字の 1023、1024、1025 文字を単体で検査し、管理 API で保存と拒否後の状態、ブラウザーで送信可否と保存後の読み戻しを確かめるテストを加えた。
1024 という上限値は維持する。
CEL のコンパイル費用の測定は未実施であり、[wi-31151](wi-31151-measure-resource-condition-compilation-cost.md) で費用と追加対策の要否を確かめる。
