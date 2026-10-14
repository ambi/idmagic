---
status: completed
archived_from: 0fca46fb861aad86ad8b000c8feebf5172954e84
---

# フェデレーション仕様範囲と受信要求検証を整合させる

SAML / WS-Federation SCL に standards、invariants、scenarios、permissions、
WS-Fed RP CRUD interface、WS-Trust RST の検証対象フィールドを追加した。
実装では SAML ForceAuthn / Destination / AuthnRequest 署名 / LogoutRequest 署名 /
LogoutResponse、WS-Trust To / KeyType を fail-closed に検証し、README の対応範囲表現を更新した。
Application 管理 UI でも SAML AuthnRequest / LogoutRequest 署名必須設定と検証用証明書 PEM を
作成・詳細・編集画面に反映した。

全文は `git show 0fca46fb861aad86ad8b000c8feebf5172954e84:<このファイルのパス>` で読む。
