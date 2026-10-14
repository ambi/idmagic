---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# ApiTokens が宣言する標準 11 行にテストを対応付け、標準の被覆台帳から外す

`mise run spec-diff` は `no normative specification change against main` を返す。規範の変更は無い。
差分は `docs/domain/api-tokens/standards.md` の 12 行のうち、名指しを持たなかった 11 行に対する被覆の
状態である。**11 行すべて**がその行の `Statement` を区別できる入力と観測を持つテストを得て
`tools/check/standards-coverage-debt.json` から消え、台帳は 121 件から 110 件になった。
テストは 1 ファイル（`api_token_standards_test.go`、11 テスト）で、`Register` が組み立てたスタックへ
HTTP 要求を出す。製品コードは 1 行も変わっていない。
見つけた欠陥 1 件を [[wi-511-dpop-proof-htu-at-protected-resources-is-not-the-target-uri]] へ切り出した。
この記録は一度、`RFC7662-API-TOKEN-INACTIVE` を「製品が満たしていないので消化できない」と結論し、
欠陥の work item まで起票して完了した。**その結論はハーネスの配線の誤りによるもので、取り消した。**
経緯は Design の「ハーネスが本番とずれていた」節。ハーネスを `cmd/idmagic/server.go` と同じ形へ直し、
コンソール失効の事例を足して行を消化し、起票した work item は取り下げた。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
