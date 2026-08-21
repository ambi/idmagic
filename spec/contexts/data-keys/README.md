# DataKeys

テナントごとの `DataEncryptionKey` (DEK) のライフサイクル (`bootstrap`、`rotate`、`disable`、`destroy`) とメタデータを所有する。DEK はマスターキープロバイダーでラップし、データベースに保存する TOTP seed などの可逆なシークレットをエンベロープ暗号化する。マスターキープロバイダーには OpenBao Transit 互換の実装を使い、開発環境とローカル環境では Tink の平文鍵セットも使用できる。

暗号処理そのものは所有しない。`EnvelopeCrypto` ポートとその実装は技術的な共有アダプター (`backend/shared/security`) に置き、この Context が外部へ公開するのは `EncryptedSecret` と鍵のライフサイクルメタデータだけである。署名鍵（`private_jwk`）も管理せず、`SigningKeys` の責務とする。

| File | Content |
|---|---|
| [glossary.md](glossary.md) | この Context での語義 |
| [states.md](states.md) | 状態と遷移 |
| [decisions.md](decisions.md) | 設計判断 |
| [internals.md](internals.md) | 機構の説明 |
| [scenarios.md](scenarios.md) | 受け入れシナリオ |
