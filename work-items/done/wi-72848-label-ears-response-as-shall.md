---
status: completed
---

# 生成サイトの要件文の表示を EARS の型に合わせる

`mise run spec-diff` は main に対して規範仕様の変更なしと報告する。
生成サイトの要件文の表示を EARS の型に合わせた。応答の名札を Then から Shall にし、主体を表示に残し、望まない振る舞いの型では If と対になる Then を Shall の前に付ける。
前置きの名札と節の順序は EARS と合っていたので変えていない。

全文は `git log -1 -- work-items/active/wi-72848-label-ears-response-as-shall.md` が示すコミットの親、またはそのコミットで読む。
