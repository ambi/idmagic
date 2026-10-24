# Sourcing のデータ

この文書は、Sourcing の Aggregate を扱う。

## 集約

| Aggregate | 境界の内側にあるもの | 他の Aggregate との関係 |
| --- | --- | --- |
| `ScimUserRef` | テナント、SCIM の ID、内部の User の ID | `User` を参照する |
| `ScimGroupRef` | テナント、SCIM の ID、内部の Group の ID | `Group` を参照する |
| `DirectoryConnector` | テナント、ID、表示名、状態、登録トークンのハッシュと有効期限、公開鍵と鍵の ID、作成、登録、最終接続、無効化の時刻 | なし |
| `DirectoryUserCorrelation` | テナント、Connector、外部の不変 ID、内部の User の ID、最後に現れた実行 | `DirectoryConnector` と `User` を参照する |
| `IngestionRun` | テナント、ID、Connector、種類、状態、開始と完了の時刻、無効にした User の数、記録したバッチ（バッチの ID、項目のダイジェスト、項目ごとの結果、記録の時刻） | `DirectoryConnector` を参照する |

- **判断**：取り込み元との対応（`ScimUserRef`、`ScimGroupRef`）は `User` と `Group` の Aggregate に埋め込まず、このモジュールの独立した Aggregate とする。埋め込むと、IdManagement の主体のモデルが取り込み元の存在を知ることになり、取り込み元の変更が IdManagement へ波及するからである。

用語集の `IdentitySource`、`IngestionRun`、`SourceCursor` は、取り込み元の種類が増えたときの概念であり、現在の `scim` の機能は実体を持たない。
`directory` の機能では `DirectoryConnector` が `IdentitySource` を兼ね、`DirectoryUserCorrelation` が `SourceCorrelation` に当たり、`IngestionRun` が実体を持つ。`SourceCursor` は Connector の手元だけにあり、実体を持たない。

- **判断**：`DirectoryUserCorrelation` は `ScimUserRef` とテーブルを共有しない。SCIM の対応は IdMagic が採番した ID を鍵とし、Directory の対応は Connector と外部の不変 ID の組を鍵とし、完全な再同期のために最後に現れた実行を持つ。共通するのは User を指すことだけである。
モジュールのルートにはファサードと組み立てだけを置き、複数の取り込み元に実在する共通点が判明するまでは、共通の機構を作らない。
