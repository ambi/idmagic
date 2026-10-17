# Audit のデータ

この文書は、Audit の Aggregate を扱う。

## 集約

| Aggregate | 境界の内側にあるもの | 他の Aggregate との関係 |
| --- | --- | --- |
| `AuditEventRecord` | ID、テナント、イベントの種類、発生時刻、ペイロード、検索属性 | ペイロードの `tenantId` でテナントを参照する。外部キーで `tenants` を参照しない |

監査の記録は追記だけで扱い、削除やアーカイブの API は提供しない。
記録を消す経路は、種別ごとの保持期間を過ぎたものの一括削除だけである。

| 概念 | 内容 |
| --- | --- |
| 操作者（`AuditActor`） | 操作を行った本人。ペイロードの `userId`。`actor.id` が指す |
| 対象ユーザー（`AuditTargetUser`） | 別のユーザーに対する操作の対象。ペイロードの `targetUserId`。`target.id` が指す |
| 行為者の種別（`AuditActorType`） | `user` または `agent`。エージェントを行為者とするイベントでは、`actor.id` をエージェントの識別子のまま保持し、`userId` へ読み替えない |
| 委譲チェーン（`AuditDelegationChain`） | `act` チェーンに現れる主体の並び。多値の検索属性 `delegation.actor` として保存する |

- **判断**：実在するアカウントが必ず特定できるイベントのペイロードには、平文かハッシュかを問わずユーザー名を含めない（`ResolvableUserEventPayloadPolicy`）。各モジュールのイベントのモデルにユーザー名に当たる項目を設けないことで、構造的に保証する。ユーザー名による検索は、`username` を User の Repository で `user_id` へ解決し、`user_id` の検索に帰着させる。実在しないアカウント名も追跡する必要がある認証の失敗のイベントに限り、平文の検索属性 `actor.username` を持たせる。
- **判断**：委譲チェーンに ID は振らない。参加者は `act` から導出でき、ID を採番すれば委譲のモードの導出と食い違いうる第二の真実になるためである。
