---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# WebAuthn / Passkey と recovery codes を phishing-resistant MFA として導入する

WebAuthn / Passkey と recovery codes を Authentication context の phishing-resistant MFA として追加した。
SCL には WebAuthn credential / recovery code / ceremony I/O / account security 射影 / step-up method /
account・browser API を追加し、派生 HTML / JSON Schema / OpenAPI を再生成した。設計判断は
ADR-087 に記録した。
Go 側は go-webauthn による登録・assertion 検証、credential / recovery code repository
(memory / PostgreSQL)、Valkey 対応 challenge store、account security / step-up / browser login
HTTP endpoint、WebAuthn RP env config を実装した。Recovery codes はハッシュのみ保存し、
one-time consume と残数算出にした。
UI 側は account security のパスキー一覧・登録・削除、recovery code 生成・再生成・失効・一度だけの平文表示、
login 第二要素の TOTP / passkey / recovery code 選択、step-up dialog の passkey / recovery code 対応を追加した。
README と compose dev stack に WebAuthn RP ID / origin / HTTPS 注意を追記した。

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。
