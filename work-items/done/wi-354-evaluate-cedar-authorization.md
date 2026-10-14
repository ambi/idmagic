---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# Cedar を IdMagic の実行時認可正本として採用するか再評価する

cedar-go v1.8.0 の token-exchange pilot で grant 宣言、scope 縮小、resource/audience 一致を
Cedar policy/schemaへ移し、既存 evaluatorとの decision parity、schema validation、undefined action・
不正projection・evaluation errorのfail-closedを確認した。runtime判断は表現可能だったが、validatorが
互換性保証外の実験APIで、role-policy inspection用Go tableも撤去できないためCedarは不採用とした。
pilot、Cedar dependency、schema、policyは撤去済みで、wi-53は既存Go evaluator方針で進める。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
