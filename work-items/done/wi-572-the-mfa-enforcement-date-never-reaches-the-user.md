---
status: completed
archived_from: 854f521daf0113319b826044107f04189d1f24c1
---

# MFA の強制開始日時が利用者へ届かない

`mise run spec-diff` は main に対する規範仕様の差分なしを報告した。規範は動かさず、`EX-AUTHENTICATION-019-01` が要求していた警告の表示を実装へ揃えた。
公開契約では `IdMagic.Contract.AccountSecurityResponse` に任意の `mfa_enforcement_start_at` が加わり、`check-api-compat` は破壊的変更なしと判定した。
`GET /api/account/v1/security` は、テナントデフォルトポリシーの MFA 強制開始がまだ来ておらず、利用者が MFA 要素もパスキーも持たないときだけ、その日時を返す。
アカウントのセキュリティ画面はその日時と事前登録を促す警告を表示し、画面上で認証アプリかパスキーを登録すると警告を消す。
`tools/check/example-coverage-debt.json` から `EX-AUTHENTICATION-019-01` を外した。

全文は `git show 854f521daf0113319b826044107f04189d1f24c1:<このファイルのパス>` で読む。
