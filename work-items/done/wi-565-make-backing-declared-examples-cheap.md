---
status: completed
archived_from: a8e03c729bc0cb0163e004b8abe40c9d40181425
---

# 宣言済みの具体例にテストを対応付ける 1 件あたりの費用を、テスト基盤と台帳と索引の側から下げる

`mise run spec-diff` は `no normative specification change against main` を返す。規範も製品コードも 1 行も動いていない。
変わったのは、宣言済みの具体例へテストを対応付ける作業の 1 件あたりの費用である。
観測点を探す作業は `mise run spec-route` が計算するようになり、`Register` と同じ配線は
`backend/shared/http/testing_stack` の option 合成で建つようになり、読んで初めて分かった判断は
台帳の行そのもの (`blocked_by`、`finding`) に残るようになった。
wi-538 が量を理由に割った 5 記録は 1 つへ畳み直し、割らない方針を `implement-work-item` へ書いた。
Context ごとの兄弟記録 17 件は統合せず、新しい道具を使う形へ更新した。

全文は `git show a8e03c729bc0cb0163e004b8abe40c9d40181425:<このファイルのパス>` で読む。
