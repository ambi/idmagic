---
status: completed
---

# リポジトリの編集で sd を使わず、ast-grep と編集ツールと jq/yq で書き換える

`mise run spec-diff` は規範の変更なしと報告した。
AGENTS.md のツール表から `sd` を外し、同じ構造または同じ文字列の複数箇所の書き換えを `ast-grep run -p -r -U` に、JSON と YAML の変更を `jq` と `yq` にまとめた。
`setup-cli-tools` の導入一覧からも `sd` を外した。
`sd` は一致が 0 件でも終了コード 0 で空振りが見えないためで、代わりの道具はいずれも一致しなければ失敗する。
ruplacer は、開発元の Codeberg に配布バイナリがなく保守の見通しに不安があるため採らなかった。

全文は `git log -1 -- work-items/active/wi-75753-stop-using-sd-for-repository-edits.md` が示すコミットの親、またはそのコミットで読む。
