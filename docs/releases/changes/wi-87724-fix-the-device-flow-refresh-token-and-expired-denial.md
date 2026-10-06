# WI-87724: Fix the device flow refresh token and expired denial

作業項目は `wi-87724-fix-the-device-flow-refresh-token-and-expired-denial` である。

WI-87724 は、デバイス認可の `device_code` の交換で、スコープに `offline_access` を含むときだけリフレッシュトークンを返すようにする。

これまでは、`offline_access` を要求しないデバイス認可の交換もリフレッシュトークンを返し、`RefreshTokenIssued` を発行していた。
変更後は、認可コードの交換と同じく、`offline_access` を含まない交換の応答に `refresh_token` を含めない。
リフレッシュトークンを使うデバイス認可のクライアントは、デバイス認可の要求の `scope` に `offline_access` を含め、クライアントの許可するスコープにも `offline_access` を登録する必要がある。

また、有効期間を過ぎた `user_code` の拒否は、承認と同じく 400 と `expired_token` で拒否され、記録は `Denied` にならない。
規範上の条件は[トークン](../../domain/oauth2/token/README.md)の REQ-OAUTH2-021 と、[デバイス認可](../../domain/oauth2/device/README.md)の REQ-OAUTH2-027 が定める。
