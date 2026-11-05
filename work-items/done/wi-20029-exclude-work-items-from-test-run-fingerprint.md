---
status: completed
---

# テストの実行記録の fingerprint から作業項目を除く

検証の入力の同一性に範囲を設けた。
テストの実行器は範囲 `test-execution` で fingerprint を記録し、`work-items/` を除くファイルの内容だけを入力にして、revision を含めない。
完了時の `primary_use_cases` の照合と `report-verification` の表示も同じ範囲で現在の値を求め、範囲を記録していない従来の記録を受け入れない。
静的検査の実行器は範囲 `workspace` のまま、revision と全ファイルを入力にする。

これにより、テストの後に要約を書く、完了記録を移す、作業項目だけをコミットする、のいずれでもテストの再実行は要らなくなった。
最終検査の選択にあった、`input.revision` と `git diff` で作業項目だけの差分を手で確かめる手順は、この規則の説明へ置き換えた。
`mise run spec-diff` に規範仕様の差分はない。

既存の `.verification/` の記録は範囲を持たないので、一度テストを実行し直すまで照合に使えない。
除外は Go のテストが `work-items/` を読まないことを前提とする。将来その配下を読むテストを加える場合は、この除外を見直す。
