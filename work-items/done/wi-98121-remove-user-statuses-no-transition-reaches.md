---
status: completed
archived_from: 18777aee4dfe33d07cf8bd1af40f9293515ef39c
---

# どの遷移も到達しない UserStatus の値を消す

`mise run spec-diff` は `spec/modules/identity-governance/models.tsp:UserStatus` の変更だけを示す。
`UserStatus` から `locked`、`staged`、`suspended` を消し、`Valid()` を状態の表の四つに限った。
起票時の前提と違い、三つの値は公開契約 `IdMagic.Contract.UserStatus` に載っていたので、契約、ワークフロー定義フォームの選択肢、翻訳からも消した。
利用者の確認を得て、契約から消す範囲で進めた。
データエクスポートの `status` の絞り込みと、ライフサイクルワークフローのトリガーの `from_status` と `to_status` は、三つの値を拒否するようになった。

全文は `git show 18777aee4dfe33d07cf8bd1af40f9293515ef39c:<このファイルのパス>` で読む。
