---
status: completed
---

# EmailSent の具象の型を、発行するモジュールの domain に置く

`EmailSent` の構造体を `backend/shared/spec` から消し、発行する Authentication と IdManagement の `domain` にそれぞれ置いた。
二つの構造体は同じフィールドとタグを持ち、ワイヤ表現の一次情報は TypeSpec の `EmailSent` 一つのままである。
IdManagement が Authentication の `domain` を import する案は、IdManagement の送信のイベントの所有者を Authentication に移し依存も増やすので採らなかった。
語彙の検査から、共有の木のイベントを TypeSpec の宣言元へ帰属させる規則を消した。帰属させるイベントがなくなったためである。
`mise run spec-diff` は規範の差分を報告しない。
EX-TENANCY-018-01 はテスト送信で `EmailSent` を発行すると述べるが、実装は発行しておらず、この食い違いは wi-95251 で扱う。

全文は `git log -1 -- work-items/active/wi-72857-move-email-sent-into-emitting-module-domains.md` が示すコミットの親、またはそのコミットで読む。
