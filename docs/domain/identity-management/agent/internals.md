# エージェントの内部設計

## エージェント主体

`Agent` は、`User` と、OAuth2 で定義する資格情報プリミティブに並ぶ、第 3 の第一級プリンシパル型である。この Context では、アイデンティティ、所有者、ライフサイクル、資格情報のバインディングを含む Aggregate 自体を扱う。エージェントがトークン交換のチェーンで actor として振る舞うための委譲機構は `OAuth2` が担う。

`Agent` Aggregate は `(id, tenant_id, display_name, kind, status, owner, purpose, created_at, updated_at, disabled_at?, killed_at?)` を持つ。`id` は URL セーフなスラッグであり、`kind` はエージェントの行為に人間がどの程度関与するかを宣言するため、`autonomous` と `supervised` を区別する。登録、検索、変更はすべてテナント単位とし、IdManagement の他の Aggregate と同じテナント境界に従う。

`Agent` は独自の資格情報プリミティブを持たず、`AgentCredentialBinding` を通じて 1 個以上の既存 `OAuth2Client` 登録にバインドする。これにより、1 つの資格情報と鍵管理のインターフェースを一般的な M2M クライアントとエージェントの両方で利用し、`Agent` はその上に所有権、目的、ライフサイクルの層だけを追加する。すべての Agent は所有者（`User` または所有する `Group`）を持たなければならず、所有者のない Agent は登録できない。所有者のオフボーディングは、孤立した非人間アイデンティティを残さないよう、その所有者が所有する Agent へ伝播させる。

ライフサイクルの状態は `active` / `disabled` / `killed` である。`disabled` は復元可能な運用停止、`killed` は一方向の緊急停止を表す。どちらも、各バインディングの `OAuth2Client` が通るトークン発行境界でフェイルクローズに強制する。ステータスが `active` ではない Agent には新しいトークンを発行せず、検査に曖昧さがあれば発行しない側へ倒す。これはキルスイッチに共通する意図的な方針である。`AgentRegistered` / `AgentUpdated` / `AgentDisabled` / `AgentEnabled` / `AgentDeleted` / `AgentOwnerChanged` は、既存の監査・アウトボックス経路へ発行する。Agent の CRUD とキルスイッチは一般的な管理者ロールを再利用せず、専用の `AdminAgentsManage` 権限で制御する。
