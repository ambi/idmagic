---
status: completed
---

# Docker を必要とする OpenLDAP 試験を通常の Go 検証から分離する

OpenLDAP の実物を使う結合試験と Connector の E2E を `openldap` タグで通常の Go 試験から分離し、専用の実行タスクとタグ付き lint を設けた。
専用タスクでは Docker 不在を診断付きの失敗とし、外部状態の変化を隠さないよう結果キャッシュを使わない。
CI と `verify-full` が両方の試験を実行して専用の証拠を保存する構成にし、通常の Go 試験群には Docker 不要の結合試験も含むことを明記した。
製品の仕様と公開契約は変えていない。

実在の OpenLDAP を使う結合試験 5 件と E2E 1 件はスキップなしで成功した。
Docker に接続できない設定でも通常の Go 試験は 8,278 件成功し、DB 検査、Go lint、リポジトリ検査と tools の検査も成功した。
通常実行の既存の Samba、Envoy と一部の fuzz 入力のスキップは残る。
`verify-changed -- main` は今回変更していない UI のテンプレート公開試験 1 件で失敗し、単独でも再現するため [[wi-63323-fix-initial-authorization-model-publication-test]] へ記録した。
追加の `lint-repo` は、今回変更していない `examples/mcp-client-walkthrough/walkthrough.ts` の 15 行目にある `generic-password` の検出で失敗した。
ShellCheck と GitHub Actions の静的検査は成功した。
全 Go 試験の水準別の再編と、既存の専用タスクを持つ外部環境試験の変更は対象外とした。
