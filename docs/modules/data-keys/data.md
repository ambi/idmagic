# DataKeys のデータ

この文書は、DataKeys の Aggregate を扱う。

## 集約

| Aggregate | 境界の内側にあるもの | 他の Aggregate との関係 |
| --- | --- | --- |
| `TenantDataEncryptionKey` | テナントとバージョンの組ごとの `wrapped_dek`、マスターキーの ID、状態、有効化、退役、無効化、破棄の時刻 | `Tenant` を `tenant_id` で参照する |

テナントごとに、新規の暗号化に使える DEK は常に高々 1 本である。
復号できるバージョンは複数ありうる。
暗号文は、暗号化に使った DEK のバージョンとともに、項目を持つモジュールが保存する。
