---
status: completed
---

# Directory Connector を登録し、その呼び出しを署名付き JWT で認証する

Sourcing に `directory` 機能を設け、閉域の AD と LDAP を取り込み元とする Directory Connector の登録と、その呼び出しの認証を加えた（REQ-SOURCING-008〜014、状態機械 DirectoryConnectorLifecycle）。
管理者は管理 API で Connector を作成、一覧、取得、無効化し、作成と再発行で一回だけ使える 1 時間有効の登録トークンを受け取る。IdMagic はトークンの SHA-256 だけを保存する。
Connector は Connector API（`/directory-connector/v1/`）で手元の公開鍵を登録し、以後は秘密鍵で署名した 5 分以内の JWT を `Authorization: Bearer` で提示する。`aud` はテナントの発行者識別子に `/directory-connector/v1` を続けた値で、`jti` は OAuth2 の記録先に `directory_connector:` の接頭辞を付けて一度だけ受け付ける。
mTLS はエッジの設定を求めるので採らなかった。再発行は登録済みの鍵を破棄して登録待ちに戻し、鍵の更新と失効を兼ねる。無効化は終端である。
Connector API を汎用 API の一つとして API の区分に加え、主体の種類に DirectoryConnector を加えた。ユーザーの取り込みは wi-23043 が、この認証を上位の要件として使う。
