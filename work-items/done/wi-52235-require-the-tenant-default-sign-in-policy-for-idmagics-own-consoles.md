---
status: completed
---

# IdMagic 自身の画面へのサインインにテナントのデフォルトのポリシーを適用することを要件にする

`mise run spec-diff` は REQ-APPLICATION-016 の追加を報告する。
IdMagic 自身の管理コンソールとアカウントポータルへのサインインにテナントのデフォルトのサインインポリシーを適用することを、今の実装の振る舞いのまま要件にした。第二要素の要求、第二要素がない場合とネットワークの拒否、`prompt=none`、失効したセッションの各文を含む。
既存の 4 件のテストに要件を引かせ、テストのなかったログインの API の first-party の経路と、`/authorize/resume` と `prompt=none` のテストを足した。
評価を呼ぶ 4 か所をそれぞれ無効にすると、要件を引くテストが落ちる。
振る舞いは変えていない。

全文は `git log -1 -- work-items/active/wi-52235-require-the-tenant-default-sign-in-policy-for-idmagics-own-consoles.md` が示すコミットの親、またはそのコミットで読む。
