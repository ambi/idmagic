---
status: completed
archived_from: eb810c8705279bec43d2256f289c9f43db487f8e
---

# 開いたままの親 work item が、子の規範差分を自分のものとして受け取ってしまう

`mise run spec-diff` は `REQ-AUTHENTICATION-036` の追加と `RFC8176-AMR-VOCABULARY` の変更を返すが、
どちらも [[wi-508-amr-vocabulary-declaration-and-implementation-disagree]] のものであり、本項目は
規範を 1 行も変えていない。差分は `documentation_impact` の推論の内側にある。規範差分が追加した
要素は、それを `affected_spec` で宣言している記録にだけ帰属するようになった。宣言している記録が
1 件も無いときは従来どおりすべての所有者へ帰属する。

全文は `git show eb810c8705279bec43d2256f289c9f43db487f8e:<このファイルのパス>` で読む。
