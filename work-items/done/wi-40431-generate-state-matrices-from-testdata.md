---
status: completed
---

# 状態遷移表を testdata から生成し、テストに文書の Markdown を読ませない

`mise run spec-diff` は main に対して規範の差分なしを示す。
29 の状態機械の三つの表（状態の表、遷移の表、状態遷移表（マトリクス形式））の一次情報を `testdata/<Machine>.statemachine.json` へ移し、25 本の機能仕様の表を `mise run generate-spec-from-testdata`（`generate-spec-examples` を改名）で生成するようにした。
`check-spec` の `state-machines` は、生成の区間のない状態機械と一次情報とのずれを拒否する。
移行前後の表は、区切り行の書き方を除いて一行も変わらない。
DataEncryptionKeyLifecycle だけは遷移の表とマトリクスの間に説明の二文があったので、区間の前へ移した。
マトリクスの行は状態の表と違う順序で並ぶ文書があるため、一次情報の `matrix` の順序で生成する。
一次情報は、モデルテストのある User、Tenant、SigningKey はテストのパッケージの `testdata` に、ほかはモジュールの根のパッケージの `testdata` に置いた。
`testing_statematrix` は Markdown の解析をやめて `Load` で JSON を読み、三つのモデルテストはパッケージの外を読まなくなった。
`check-repository` の `test-document-reads` は、Go のテストの文字列リテラルが `docs/` を指すことを拒否する。コメントの中の出典の記載と URL は数えない。
`spec-diff` は、空行だけを挟んで続くマトリクスの行を遷移の表の行として数えていたので、意図どおり遷移の表だけを読むよう直した。
frontend と tools のテストは検査の対象外なので、`integrate-work-item` の暫定の規則は広げず、wi-57328 の着手時の確認事項に加えた。
