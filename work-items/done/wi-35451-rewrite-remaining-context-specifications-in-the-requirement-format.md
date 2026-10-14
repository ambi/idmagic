---
status: completed
archived_from: b9479e8440deae37b58df07e00ec726661de9ea0
---

# 残りの Context の機能仕様を、EARS 形式の要件だけで書く方式へ書き直す

`mise run spec-diff` によれば、要件を 9 件加え（REQ-APPLICATION-015、REQ-AUTHENTICATION-038、REQ-AUTHENTICATION-039、REQ-OAUTH2-051〜054、REQ-SAML-009、REQ-WSFEDERATION-006）、IdManagement と Tenancy 以外の 19 Context の要件 262 件の本文を、日本語の EARS の要件文に書き直した。
20 個の状態機械の遷移の表を、実装が通る遷移に合わせて改め（同意の再同意と再撤回、IdP の接続の信頼の根拠の更新と削除、信頼済みデバイスの再失効、ジョブの再取得など）、すべての状態機械に状態遷移表（マトリクス形式）を加えた。
`check-spec` は全 Context の要件文の構文と、すべての状態機械のマトリクスの存在を検査し、`check-unspecified-vocabulary` は全 21 Context を対象にした。
未記載の振る舞いは 35 件を分類し、(c) の 11 件を 6 件の記録（wi-12645、wi-83307、wi-37490、wi-87724、wi-54733、wi-61672）に起票した。
モデルベースのテストは、利用者の判断で wi-51017 へ移した。

全文は `git show b9479e8440deae37b58df07e00ec726661de9ea0:<このファイルのパス>` で読む。
