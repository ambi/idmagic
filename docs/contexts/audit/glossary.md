# Audit Glossary

| Term | Definition | Aliases |
|---|---|---|
| AuditEventSearchRegistry | 監査イベントで使用できる検索属性（`AuditEventSearchAttribute`）の定義一覧。属性ごとにフィールド、演算子、変換方法を宣言し、任意の SQL や JSONPath を受け付けない閉じた検索文法を定める。Go の `AuditSearchRegistry` マップを唯一の正とする。 | search-attribute-registry, 検索属性定義 |
| AuditEventFilterExpression | `AuditEventQuery.filter` を構成する論理積 (AND) の 1 要素。レジストリの許可リストにあるフィールド、演算子、値からなる。個人識別情報の属性は平文入力をサーバー側で変換してから照合する。 | filter-expression, フィルター式 |
| AuditActor | 操作を行った本人。ペイロードの `userId` に相当する。認証、OAuth2 フロー、本人による操作では、常に本人が操作者になる。`AuditEventQuery.user_id`、検索時に `user_id` へ解決する `username`、フィルターの `actor.id` は、いずれも操作者を指す。UI では「ユーザー ID（操作者）」「ユーザー名（操作者）」「ログイン試行のユーザー名」と表記する。 | actor, 操作者 |
| AuditTargetUser | 別のユーザーに対する操作の対象。ペイロードの `targetUserId` に相当する。管理者が実行する UserCreated / UserDisabled / GroupMemberAdded の対象ユーザーがこれにあたる。フィルターの `target.id` が指すのはこちらであり、操作者とは区別する。UI では「対象ユーザー」と表記する。 | target, 対象ユーザー |
| ResolvableUserEventPayloadPolicy | 実在するアカウントが必ず特定できるイベント (UserAuthenticated、ConsentGranted、AuthorizationCodeIssued、AuthorizationCodeRedeemed、AccessTokenIssued、RefreshTokenIssued など) のペイロードには、平文かハッシュかを問わずユーザー名を含めないという方針。各 Context のイベントモデルにユーザー名相当のフィールドを設けないことで構造的に保証する。ユーザー名による検索は `AuditEventQuery.username` を User Repository で `user_id` へ解決し、既存の `user_id` 検索に帰着させる。実在しないアカウント名も追跡する必要がある認証失敗イベントに限り、平文の検索属性 `actor.username` を持たせる。 | resolvable-user-event-payload-policy |
| AuditDelegationChain | 1 つの監査イベントの `act` チェーンに現れる主体の並び。現在の行為者、入れ子の `act.sub`、subject をすべて含む。多値の検索属性 `delegation.actor` として保存し、チェーンのどの段の参加者からでも同じ連なりを引けるようにする。チェーンに ID は振らない。参加者は `act` から導出でき、ID を採番すれば委譲モードの導出と食い違いうる第二の真実になるためである。参加者は識別子だけを保持し、`ResolvableUserEventPayloadPolicy` に従いユーザー名は含めない。 | delegation-chain, 委譲チェーン |
| AuditActorType | 監査イベントの行為者が人間の利用者かエージェントかの区別。検索属性 `actor.type` が `user` または `agent` の値を取る。エージェントを行為者とするイベントでは `actor.id` を `userId` へ読み替えず、エージェントの識別子をそのまま保持する。この読み替えを残すと、エージェントによる代行が本人の操作と検索上区別できない。 | actor-type, 行為者種別 |
| TenantAdministrator | 所属テナントまたは (system_admin の場合) 全テナント横断で監査イベントを読み出す権限を持つ管理者。 |  |
