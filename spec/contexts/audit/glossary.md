# Audit Glossary

| Term | Definition | Aliases |
|---|---|---|
| AuditEventSearchRegistry | 監査イベントで使用できる検索属性（`AuditEventSearchAttribute`）の定義一覧。属性ごとにフィールド、演算子、変換方法を宣言し、任意の SQL や JSONPath を受け付けない閉じた検索文法を定める。Go の `AuditSearchRegistry` マップを唯一の正とする。 | search-attribute-registry, 検索属性定義 |
| AuditEventFilterExpression | `AuditEventQuery.filter` を構成する論理積 (AND) の 1 要素。レジストリの許可リストにあるフィールド、演算子、値からなる。個人識別情報の属性は平文入力をサーバー側で変換してから照合する。 | filter-expression, フィルター式 |
| AuditActor | 操作を行った本人。ペイロードの `userId` に相当する。認証、OAuth2 フロー、本人による操作では、常に本人が操作者になる。`AuditEventQuery.user_id`、検索時に `user_id` へ解決する `username`、フィルターの `actor.id` は、いずれも操作者を指す。UI では「ユーザー ID（操作者）」「ユーザー名（操作者）」「ログイン試行のユーザー名」と表記する。 | actor, 操作者 |
| AuditTargetUser | 別のユーザーに対する操作の対象。ペイロードの `targetUserId` に相当する。管理者が実行する UserCreated / UserDisabled / GroupMemberAdded の対象ユーザーがこれにあたる。フィルターの `target.id` が指すのはこちらであり、操作者とは区別する。UI では「対象ユーザー」と表記する。 | target, 対象ユーザー |
| ResolvableUserEventPayloadPolicy | 実在するアカウントが必ず特定できるイベント (UserAuthenticated、ConsentGranted、AuthorizationCodeIssued、AuthorizationCodeRedeemed、AccessTokenIssued、RefreshTokenIssued など) のペイロードには、平文かハッシュかを問わずユーザー名を含めないという方針。各 Context のイベントモデルにユーザー名相当のフィールドを設けないことで構造的に保証する。ユーザー名による検索は `AuditEventQuery.username` を User Repository で `user_id` へ解決し、既存の `user_id` 検索に帰着させる。実在しないアカウント名も追跡する必要がある認証失敗イベントに限り、平文の検索属性 `actor.username` を持たせる。 | resolvable-user-event-payload-policy |
| TenantAdministrator | 所属テナントまたは (system_admin の場合) 全テナント横断で監査イベントを読み出す権限を持つ管理者。 |  |
