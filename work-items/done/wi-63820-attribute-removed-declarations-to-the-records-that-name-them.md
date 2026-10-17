---
status: completed
---

# 取り除いた規範要素による文書化水準の要求を、その要素を名指しする記録だけに課す

`check-work-items` は、取り除いたシナリオ、標準行、TypeSpec の宣言による `removal_notice` の要求を、`affected_spec` でその要素を名指しする記録だけに課す。
変更された記録のどれも名指ししなければ、従来どおり差分を持つすべての記録に課す。
照合には追加の帰属と同じ規則を使い、関数名を `claimsSpecificationElement` へ改めた。
改名する記録は新名と旧名を両方挙げ、追加と除去の両方の要求を受ける。
旧名は作業ツリーで解決できないので、`main` との差分が取り除いた要素として示す間は、その参照の解決を求めない。
旧名を別の欄に書かせる案と、同じファイルへの追加の記録に帰属させる案は採らなかった。
初回公開前は告知の最低水準が常に `none` なので、wi-68415 で起きた失敗は公開後に再び現れるはずだったものである。
`mise run spec-diff` は規範の差分を報告しない。

全文は `git log -1 -- work-items/active/wi-63820-attribute-removed-declarations-to-the-records-that-name-them.md` が示すコミットの親、またはそのコミットで読む。
