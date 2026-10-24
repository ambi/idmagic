---
status: completed
---

# Kerberos/SPNEGO によるログインと WS-Fed の Windows 統合認証

Authentication に keytab を使う直接 SPNEGO を追加し、検証済み principal を管理者が対応付けた同一テナントの Active User へ解決する。
TLS の channel binding、時刻差と共有の再送拒否を適用し、AMR は RFC 8176 の `wia` とする。
既存の MFA、サインインポリシーと必須操作を継続し、WS-Fed は統合 Windows 認証の要求を満たした Kerberos の署名済み Assertion を返す。

Windows の認証は通常ログインで明示的に選べ、自動試行は出口の公開 IP による任意設定とした。
チケットを提示できない端末はチャレンジを繰り返さず通常ログインへ戻し、未参加端末の既存のログインも維持する。
信頼設定のウィザード、AD 管理者用 PowerShell、ファイルの診断と鍵更新の運用手順を追加した。

AD の常時接続と利用者の同期は前提にしない。
後の同期管理への移管では、既存 User ID と外部の不変 ID を確認し、手動の管理から明示的に移す方針とした。
AD の無効化の自動同期、AD パスワードでのフォーム認証、NTLM と Hybrid Join は対象外であり、顧客 AD と Windows ブラウザーの受け入れは導入時の手順とする。
