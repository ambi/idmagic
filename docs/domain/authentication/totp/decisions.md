# TOTPの設計判断

- MFA の TOTP seed を含め、アプリケーションのデータベースに残る可逆なシークレットは平文で保存せず、`DataKeys` Context と `EnvelopeCrypto` ポートによるエンベロープ暗号化へ移す。
