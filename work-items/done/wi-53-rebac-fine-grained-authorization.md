---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# エージェントのデータアクセス向け関係ベース細粒度認可 (ReBAC / FGA)

新しい Bounded Context `Authorization` を追加し、リソース 1 件ごとの関係ベース認可 (ReBAC) を通した。規範的シナリオ REQ-AUTHORIZATION-001〜009 と、認可モデル・関係タプル・判定・列挙の TypeSpec 契約 (`AuthorizationModel` / `RelationTuple` / `FgaCheckRequest` / `FgaCheckResult` ほか計 22 モデルと 6 操作) が新設された。判定は Zanzibar 風の書き換え規則 3 種 (`direct` / `computed_userset` / `tuple_to_userset`) の和で表し、深さ制限つきの探索で評価する。深さ超過・循環・未知の関係・ストア障害・事実の欠落はいずれも許可へ退避しない。
合成規則は新しい経路を作らず、既存 AuthZEN の `Authorizer` ポートへ寄せた。`resource:access` アクションと 4 つの新規規則 (`relationship_facts_present` / `relationship_permits_subject` / `relationship_permits_actor_chain` / `actor_chain_principals_active`) を `backend/shared/spec` に追加し、Authorization は関係の事実と代行チェーンを判定 context に載せるだけになった。これによりエージェントは代行するユーザーの権限を超えられない。
永続化はテナント境界を主キーに持つ `authorization_relation_tuples`、追記のみの `authorization_models`、テナントごとの `authorization_write_versions` の 3 テーブルで、メモリと PostgreSQL が同じ契約テストを共有する。書き込みが返す整合トークンはテナントを束縛しており、他テナントのトークンや先行する版の提示は fail-closed で拒否する。監査は `AuthorizationModelPublished` / `RelationTupleWritten` / `RelationTupleDeleted` / `FgaCheckEvaluated` / `FgaResourcesEnumerated` を発行し、リソース識別子はテナントと型を混ぜたダイジェストに落とす。列挙は 1 件ごとの判定を監査へ展開せず、走査全体を 1 件にまとめる。
権限 `AdminAuthorizationModelManage` (`admin:authorization_model_manage`) を追加し、`/api/admin/v1/authorization/*` の 6 エンドポイントを管理者に限定した。ルート `SPECIFICATION.md` の Context Map、責務表、構造上の決定、データベース設計も同期した。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
