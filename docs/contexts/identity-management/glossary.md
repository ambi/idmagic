# IdManagement Glossary

| Term | Definition | Aliases |
|---|---|---|
| Administrator | `User.roles` に `admin` を持ち、所属テナント内の管理 API の利用を許可された認証済みユーザー。テナント境界を越える操作は SystemAdministrator に限定する。 | admin, 管理者, TenantAdmin |
| SystemAdministrator | `User.roles` に `system_admin` を持つ認証済みユーザー。テナント管理（CRUD・無効化・有効化）とテナント横断操作を許可され、`system_admin` 専用のシステムコンソール (`/system`) から `/api/admin/tenants/*` や `/api/admin/keys/health` を呼び出せる。テナント境界を越えるため、パスではなくロールで制御する。 | system_admin, システム管理者 |
| EndUser | 認証済みまたは認証を試みる一般利用者。管理ロールを持たない、アカウントポータルでのセルフサービス操作の主体。 | 利用者, エンドユーザー |
| UserDisablement | `User.status` を `Disabled` に遷移させ、認証とセッション利用を停止する復元可能な管理操作。削除や個人識別情報の消去とは異なる。 | disable ユーザー |
| UserImport | 管理者が UTF-8 CSV でユーザーの作成と部分更新を行う操作。まずプレビューで事前検証し、成功したプレビューを指定して非同期に適用する。CSV のヘッダーは安定した機械可読なキーで、順序と部分集合は任意だが、`password` と `password_hash` は含められない。 | CSV ユーザーインポート, ユーザー一括インポート |
| UserDeletion | User の Tombstone 化と関連 Aggregate のカスケード削除。`status` を `Deleted` に遷移させて個人識別情報のフィールドを匿名化し、監査のため `id` だけを保持する。`Deleted` は終端状態であり復元できない。 | delete ユーザー, anonymize ユーザー, アカウント削除 |
| Deleted | User の終端状態。個人識別情報は匿名化済みで、ログイン、トークン発行、UserInfo のいずれも無効なプリンシパルとして扱う。 | deleted |
| Delete | User を Deleted に遷移させる管理操作。Tombstone 化と関連 Aggregate のカスケード削除を 1 回の操作で行う。 | delete |
| PendingDeletion | User の削除予約状態。`status == PendingDeletion` では個人識別情報を保持するが、`Disabled` と同様に認証を拒否する。猶予期間（`states.UserLifecycle` の `PendingDeletion` → `Deleted` 遷移のガード）内であれば Restore で `Active` に戻せる。猶予期間を過ぎると Purge により `Deleted` へ遷移し、匿名化する。 | pending_deletion, 削除予約中 |
| SoftDelete | User を `Active` / `Disabled` から `PendingDeletion` へ遷移させる管理操作。個人識別情報、同意、リフレッシュトークン、セッションを残したまま削除を予約するため、誤操作を猶予期間内に取り消せる。 | soft_delete, soft-delete |
| Restore | `PendingDeletion` の User を `Active` に戻す管理操作。猶予期間内だけ実行でき、個人識別情報と資格情報は保持しているため通常どおりログインを再開できる。 | restore |
| Purge | User を `Active` / `Disabled` / `PendingDeletion` から `Deleted` に遷移させる確定削除操作。匿名化をカスケードし、猶予期間経過後の自動消去と、管理者による明示的な完全削除の双方から呼び出す。 | purge |
| Group | テナント単位の Aggregate。再利用できるロールの束を持ち、所属する User へまとめて付与する。階層、拒否規則、属性による自動所属は持たず、和集合だけで構成する。連絡先メールアドレスと、`TenantGroupAttributeSchema` に対して検証する独自属性も持つ。 | group, グループ, ロールグループ |
| GroupMembership | User と Group の所属関係 (`GroupMember`)。`manual` は管理者操作、`dynamic` は有効な CEL 規則の評価結果だけから変更する。`effective_roles(user) = user.roles ∪ ⋃ membership.group.roles`。 | group membership, グループ所属, membership |
| DynamicGroupRule | User の中核属性と TenantUserAttributeSchema で定義した属性だけを参照し、所属可否を Boolean で返す制限付き CEL 式。規則のバージョンが一致する動的メンバーシップだけを有効とする。 | dynamic membership rule, 動的グループルール |
| EffectiveRoles | 認可判断で使う User の有効ロール集合。`User.roles` と所属 Group のロールの和集合であり、管理コンソールの RBAC 制御とアカウントポータルの双方が参照する。 | 有効ロール |
| Agent | テナント単位の非人間プリンシパル。自身の資格情報は持たず、AgentCredentialBinding で既存の OAuth2Client にバインドしてトークンを得る。所有者（User または Group の ID）は必須。 | agent, エージェント, AI エージェント, 非人間アイデンティティ |
| Killed | Agent の緊急停止 (kill-switch) による一方向終端状態。Killed からは復帰できず、新規トークンを一切発行しない (fail-closed)。 | killed |
| DisableAgent | Agent を Active から Disabled に遷移させる可逆な運用停止。`/api/admin/agents/{agent_id}/disable` から発火。 | disable agent |
| EnableAgent | Disabled の Agent を Active に戻す。`/api/admin/agents/{agent_id}/enable` から発火。 | enable agent |
| KillAgent | Agent を Killed (一方向終端) に遷移させる緊急停止。`/api/admin/agents/{agent_id}/kill` から発火し、以後復帰不能。 | kill agent |
| Autonomous | Agent が人間の都度承認なしに自律実行する区分。 | autonomous |
| Supervised | Agent が人間の監督下で実行する区分。 | supervised |
