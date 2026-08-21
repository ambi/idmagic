# DataKeys Glossary

| Term | Definition | Aliases |
|---|---|---|
| DataEncryptionKey | レコード単位の可逆なシークレットを Tink AEAD で直接暗号化・復号する、テナントスコープの対称鍵 (DEK)。 | DEK |
| MasterKey | DEK をラップしてエンベロープ暗号化する KMS 側の鍵。プロバイダー (OpenBao Transit 互換、または開発環境とローカル環境で使う Tink の平文鍵セット) が管理し、アプリケーションデータベースには平文で残らない。 |  |
| Wrap | `MasterKey` で DEK を暗号化し、永続化できる `wrapped_dek` にする操作。 | wrap |
| FailClosed | アンラップの失敗、プロバイダーへの到達不能、AAD や改ざんの不一致などで復号できない場合に、平文へフォールバックせずアクセスを拒否する方針。 |  |
| System | `DataKeys` のライフサイクルユースケースと再暗号化ジョブそのものを指す、人間の操作者を伴わない技術的な主体。 |  |
