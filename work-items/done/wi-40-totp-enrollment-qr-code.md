---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# 認証アプリ登録時に otpauth URI の QR コードを表示する

AccountSecurityPage の TOTP 登録ステップに qrcode.react の QRCodeSVG で
otpauth URI の QR を表示。セットアップキー / otpauth URI 文字列はスキャン
できない場合のフォールバックとして併置。SCL / Go / 永続化は無変更で、
contract (/api/account/mfa/totp/enroll/*) も不変。

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。
