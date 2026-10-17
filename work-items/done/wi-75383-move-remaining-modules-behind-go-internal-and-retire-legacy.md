---
status: completed
---

# 残りのモジュールを Go の internal/ へ移し、legacy の公開方式を撤去する

`mise run spec-diff` は、TypeSpec の `FapiProfile` の説明文にある Go のパスを移動先へ直した 1 件だけを規範の変更として報告した。
Application、OAuth2、IdManagement、Authentication、WsFederation を Go の `internal/` へ移し、責務表のすべてのモジュールを `internal` にした。
組み立て地点が名指していた非公開パッケージは、各モジュールのルートパッケージの操作（永続層ごとの生成、経路とジョブの登録、ライフサイクル操作、通知、保持期間の掃除）と型の別名に置き換えた。
ほかのモジュールの結合テストのために、IdManagement のルートに `AdminUsers` を置いた。
`fixtures_postgres` は、所有モジュールの内側のテストとの循環を避けるため、User、Group、OAuth2 クライアントの保存先を引数で受け取る。
境界検査は作業ツリーの責務表で `internal` だけを受け付け、`legacy` は負債の比較の基準 revision を読むときだけ受け付ける。
その互換は、この項目が `main` に入った後に wi-11332 が外す。

全文は `git log -1 -- work-items/active/wi-75383-move-remaining-modules-behind-go-internal-and-retire-legacy.md` が示すコミットの親、またはそのコミットで読む。
