# IdManagement Internals

## Just-in-time provisioning from federation

フェデレーションのログイン時に User を作る経路は、通常の作成経路と同じ不変条件を通る。テナントのクォータ、ユーザー名とメールアドレスの一意性、属性スキーマ、`UserCreated` イベントは、上流からの作成であっても緩まない。この経路のためだけの近道は存在しない。

作られる User はパスワード資格情報を持たない。上流が認証の権威である以上、ローカルの資格情報を同時に持たせると、上流を無効にした後もローカルのパスワードでサインインできる経路が残るからである。資格情報を後から追加するかどうかは、テナントの明示的な設定に委ねる。

## User Lifecycle: Deletion and Anonymization

削除は物理的な除去ではなく匿名化である。`User.lifecycle.status` は `Deleted` へ遷移する。これはどの状態からも到達でき、戻る遷移を持たない終端状態である。Aggregate は破棄せず、その場で書き換える。`AdminAuditEvent` をはじめとする追記専用の記録が `sub` を参照しており、物理削除はその参照を壊すうえ、「削除済み」と単なる「停止中」の運用上の区別も消してしまうからである。`sub` は永久に保持し、再利用しない。

墓標への置き換えは、ユーザーを再識別または再認証しうるすべての項目を不可分に消す。`preferred_username` は `deleted:<sub>` になり、`name` / `given_name` / `family_name` / `email` は空になり、`email_verified` と `mfa_enrolled` は `false` に戻り、`password_hash` は空になり、`roles` は空になり、疎な `attributes` の対応表は丸ごと消え、`lifecycle.status` は `Deleted` になる。`preferred_username` は墓標を立てた時点で再利用のために解放される。削除されていない行に範囲を限った部分的な一意インデックスにより、墓標の値は将来のユーザーと衝突せず、解放された名前は再び使える。

削除は、削除したユーザーから以後たどれてはならないすべての Aggregate へ同期的にカスケードする。その `sub` に対する `Consent`、`RefreshTokenRecord`、`LoginSession`、`PasswordHistory`、`MfaFactor`、有効な `DeviceAuthorization` の記録をすべて取り除く。PostgreSQL を使うカスケード処理は 1 つのトランザクション内で行う。Valkey に置くセッションやデバイスコードの状態はストアごとに削除する。これらの状態は本来揮発的であり、短い不整合期間はトランザクションを複雑にしないこととのトレードオフとして許容できるためである。

削除は冪等である。すでに Tombstone 化したユーザーに対して再び呼び出しても、監査イベントを再発行せず成功を返す `no_op` になる。そのため、再試行や管理者の並行操作が失敗として現れたり、監査記録を重複させたりしない。自己破壊を防ぐため、操作者と対象が同じプリンシパルで、対象が `admin` または `system_admin` を持つ場合は削除を拒否する。管理者が自身の特権アカウントを削除する経路は、どの対話フローにも必要ないためである。削除のたびに `actorSub` / `targetSub` / `reason` / `occurredAt` を載せた `UserDeleted` 監査イベントを発行する。`sub` と Tombstone が残るため、匿名化後も「誰が何をいつ削除したか」を再構成できる。

## User Profile: Thin Core and Attribute Bag

`User` は型として持つ中核を、アイデンティティ、認証、RBAC に必要なものだけに限る。`sub`、`tenant_id`、`preferred_username`、`password_hash`、`email`、`email_verified`、`mfa_enrolled`、`roles`、`name` / `given_name` / `family_name`、`lifecycle`、各種の時刻である。滅多に使わない OIDC や SCIM の任意項目 25 個ほどを、すべてのユーザーに型と保存の水準で持たせると、それらを使わないテナントにとってモデルが肥大するだけである。それ以外のプロフィール属性 — 残りの OIDC §5.1 の任意クレーム (`middle_name`、`nickname`、`picture`、`phone_number`、`address_*` など)、SCIM 相当の組織属性 (`title`、`department`、`manager_sub` など)、テナントが定義する独自項目 — は、単一の疎な `attributes: Map<String, AttributeValue>` に置き、実際に値を持つキーだけが領域を消費する。OIDC の `address` クレームは入れ子の構造ではなく平坦なキー (`address_formatted`、`address_locality` など) として保存し、`AttributeValue` を素直な直和型 (文字列、数値、真偽値、日付、文字列の配列) に保つ。入れ子の `address` へ組み直すのは、UserInfo や ID Token のクレームを作るときだけである。

ライフサイクルの正は `User.lifecycle.status`（`Active` / `Disabled` / `Locked` / `Staged` / `Suspended` / `Deleted`）と `status_changed_at` の 1 組だけである。遷移時刻の監査記録は、時刻を持つ `UserDisabled` / `UserDeleted` イベントに残す。認証を許可するのは `status == Active` だけであり、それ以外の状態は、デフォルトで `Active` に解決されるゼロ値も含めて認証を拒否する。

#### Attribute Definitions (`UserAttributeDef`)

OIDC と SCIM の組み込みの属性も、テナントが定義する独自の属性も、同じ `UserAttributeDef` の仕組みが統べるので、管理者が設定するスキーマの形は 2 つではなく 1 つで済む。定義は 2 つの段から来て、1 つの実効的なスキーマに合わさる。

- **組み込みのカタログ** `BuiltinUserAttributeDefs`。コードで定義しすべてのテナントで共有する。OIDC §5.1 の任意の claim と、SCIM の `enterprise:User` に相当する組織の属性である。
- **テナントのスキーマ** `TenantUserAttributeSchema`。`Tenant` Aggregate に埋め込まず、`tenant_id` をキーとする独立した Aggregate とする。スキーマがテナント設定より速く変化すること、将来独自のテーブルへ分ける候補であること、テナント削除時に明示的なカスケード経路が必要なことが理由である。

実効的な定義は組み込みとテナントの和である。組み込みの鍵を再定義するテナントのスキーマはその場で拒否する。各 `UserAttributeDef` は `key` (snake_case、先頭は英字)、`type` (`string` / `number` / `boolean` / `date` / `string_array`)、`required`、`editable_by_user`、`visibility == claim_exposed` のときにのみ効く任意の `claim_name` と `oidc_scope` の組、そして `visibility` 自体を持つ。`visibility` は `private` / `self_readable` / `admin_readable` / `claim_exposed` のいずれかで、relying party へ開示されるのは `claim_exposed` だけである。`pii` のデフォルトは `true` である。定義が明示的に外さない限り、保存され監査される値は平文ではなく SHA-256 で要約される。テナントの利便よりも開示の上限を優先する、安全側のデフォルトである。

`ValidateAttributes` は `User.attributes` の対応表を、保存する前に実効的なスキーマと照合する。定義のない鍵、欠けている必須の値、型の不一致を拒否し、各 `AttributeValue` が宣言された `type` の選ぶ項目だけを埋めていることを強制する。利用者自身の経路 (`UpdateUserProfile` と `/api/account/profile`) はさらに、書き込みを `editable_by_user == true` の属性に限り、対応表全体を置き換えるのではなく鍵ごとに併合する。これにより利用者自身の編集が、触れる理由のない管理者管理の属性を上書きすることはない。またユーザーへ開示するのも `self_readable` と `claim_exposed` の属性だけである。削除の際は型として持つ中核とともに `attributes` の対応表も丸ごと消えるので、疎な入れ物が墓標より長生きすることはない。

## User CSV Round Trip

`User` CSV は 2 つ目のプロビジョニング権威ではなく、IdManagement が所有する部分更新の窓口である。機械処理用の列名の語彙と可逆なセル変換はユーザーのドメインに置き、エクスポートとインポートが HTTP のラベルや UI のロケールに依存せず 1 つの定義を共有する。組み込みの書き込み可能列、読み取り専用列、禁止列は閉じた集合とする。テナント定義の列は、ユーザーのユースケースポートを介して実効属性スキーマを解決した後にだけ `custom:<key>` として追加する。これにより解析を決定的に保ちながら、テナントのスキーマを CSV 処理とは独立して発展させられる。

ドメインの解析器は列の有無とセルの内容を別々に保持する。列がないことは Aggregate を変更しないことを意味する一方、存在する空の列は項目を消す意味を持ちうるからである。どの行も計画する前に、未知のヘッダー、重複するヘッダー、シークレットを含むヘッダーを拒否する。数式に対して安全な変換器は、表計算で危険な先頭文字と既存の先頭アポストロフィーに可逆な形で接頭辞を付け、レコードの引用処理を RFC 4180 CSV の変換器に委ねる。したがってエクスポートとインポートは、報告専用のエクスポートで使われる情報を失うアポストロフィーのエスケープを受け入れず、カンマ、引用符、複数行の値を含めて `decode(encode(value)) == value` という不変条件を共有する。

`User` のエクスポートとインポートは、設定可能な 1 つの転送ポリシーを共有する。デフォルトの上限はデータ行 100,000 件、成果物ごとに 64 MiB、項目ごとに 64 KiB とする。これは 1 個の非同期成果物に対するリソース保護の上限であり、テナントのユーザー数の上限ではない。実効ポリシーを超える `User` エクスポートは、再インポートできない成功済み成果物を作らず失敗する。容量の契約には、10,000 ユーザーとすべての組み込み列を持つ統合フィクスチャーを含め、小さな移行単位を超える規模で往復の保証を検証する。

解析と直列化は `io.Reader` と `io.Writer` に対して動作し、CSV 全体を文字列、2 次元のレコードスライス、ジョブ JSON 内の base64 として実体化しない。解析器は byte、行、項目の上限を段階的に強制する。計画ではページ単位のリポジトリ読み取りから上限付きの ID と username のインデックスを構築し、適用では行ごとのトランザクション境界を保ちながら上限付きのまとまりで進める。これにより `worker` のメモリを成果物全体の大きさから独立させ、CSV の行ごとにリポジトリを検索することを避ける。

ユーザーのユースケース層は プレビュー と 適用 に共通する 1 個の決定的な計画器を所有する。既存の集約を不変な ID、次に優先 username で解決し、型付きの独自属性と行間の衝突を検証し、状態を変更せず `created`、`updated`、`unchanged`、`rejected` のいずれかを生成する。適用 は プレビュー 時の古い変更計画を実行せず、現在のリポジトリ状態に対して同じ計画器を再び実行する。これにより同時変更を 適用 時に可視化し、プレビュー が暗黙の楽観的ロックの迂回路になることを防ぐ。

プレビュー が CSV を受け取るのは 1 回だけである。テナント単位の不変な成果物ストアが stream を受け取り、中身の見えない参照、サーバーが計算した SHA-256、byte 数、行数を返す。ジョブのパラメーターと結果はそのメタデータと要約だけを保存し、CSV の本文や base64 を決して含まない。永続化アダプターは上限付きのまとまりを使うため、永続化と読み取りに成果物と同じ大きさの単一のデータベース値やプロセスのバッファーを必要としない。メモリアダプターはテストとローカルの組み立てに同じポートを提供する。

適用が受け入れるのは、成功したプレビュージョブの ID だけである。別の CSV や、クライアントが提示するダイジェストは受け入れない。ジョブが認可、テナント、ライフサイクルの境界を与え、SHA-256 は実行対象をプレビュー時に保存した正確なペイロードへバインドする内部の完全性検査となる。これは署名ではなく、認可を置き換えない。適用ジョブはプレビューへの参照とダイジェストの両方を記録するため、`worker` はキュー投入後のペイロードの置換や破損も検出できる。ペイロードは固定するが、変更計画は現在の状態から意図的に再生成する。大量の行エラーは、同じ不変かつチャンク化したアーティファクトストア内の固定件数のページへ直列化し、リソース固有のエラーテーブルやジョブの JSON には置かない。公開する結果 API は、管理 API のテナントとクエリにバインドした署名済みカーソル、`limit`、RFC 8288 の `Link`、`Pagination-*` ヘッダーを使う。エラーの通し番号はアーティファクトのページとページ内の位置へ直接対応するため、深い位置を前後に読み取る場合も先行するエラーを走査しない。

受理した各行は、プロフィールフィールド、直接付与するロール、必須操作、カスタム属性、永続化、監査イベントの発行をまとめた Aggregate 単位の変更境界を通る。その境界内で失敗した場合は先行する部分的な変更を残さず行を拒否する。ある行の失敗によって、他の受理済み行はロールバックしない。これにより、行単位の復旧可能性と予測可能な再試行を両立する。

2 つのユースケースポートにより、Context をまたぐ知識を CSV ドメインの外に保つ。実効ユーザー属性スキーマのリーダーは型付きのテナント定義を提供し、取り込み元の所有権ガードは既存の User が外部管理かどうかを返す。これらのアダプターはアプリケーション境界で組み立てる。所有権を確認できない場合や確認に失敗した場合は、更新対象を外部管理として扱う。CSV という便宜的な経路が、より強い上流の権威を暗黙に上書きしてはならないためである。

## Group Aggregate and Effective Roles

`Group` はテナント単位の Aggregate であり、`(id, tenant_id, name, description?, roles[], created_at, updated_at?)` を持つ。組織変更のたびに影響する全 User の `roles` を個別に編集せず、ロールの組（「営業チーム = `catalog:read` + `invoice:read`」）を 1 単位として付与・取り消しできるよう導入した。`id` は生成後に変わらない `group_<uuid>` である。`name` はテナント内で一意な編集可能な表示名であり、`(tenant_id, name)` の一意インデックスで強制する。テナントをまたぐメンバーシップは無条件に拒否する。`AddMember` は対象の `User` を読み込み、存在しない場合や別テナントに属する場合は拒否する。

User の実効ロールは `user.roles ∪ ⋃_{g ∈ user.groups} g.roles` である。単純な和集合を整列して重複を除き、減算や優先順位の規則は持たない。平らな和集合で十分なところへ deny や minus の演算子を導入すると、評価順の複雑さが増すためである。User がどの Group にも属さない場合、実効ロールは `user.roles` に戻るため、`Group` の導入によって既存アカウントの挙動は変わらない。管理コンソールの RBAC 制御と `/account` の自己管理ビューでは、生の `user.roles` ではなく実効ロールを解決する。User は自身の実効権限のうち、グループメンバーシップに由来するものを確認できる。`User.roles` 自体は、どの Group にも覆われない User 個別の上書き経路として残す。ロールはデフォルトではトークンクレームへ射影しない。個別ロールにも Group 由来のロールにも、そのマッピングはまだ存在せず、ここでは意図的に対象外とする。

メンバーシップ操作は冪等である。既存メンバーの追加や、メンバーではない User の削除はドメインイベントを再発行しない `no_op` とし、Okta と Keycloak のメンバーシップ API の扱いに合わせる。Group の CRUD とメンバーシップの変更では、`AdminAuditEvent` と、`GroupCreated` / `GroupUpdated` / `GroupDeleted` / `GroupMemberAdded` / `GroupMemberRemoved` のいずれかを発行する。Group の削除ではメンバーシップをカスケード削除し、最後の `GroupDeleted` より前にメンバーごとの `GroupMemberRemoved` を発行する。

## Group Contact and Custom Attributes

`Group` は任意の `email` (部署のメーリングリストなど単純な連絡先) と、テナントが定義する任意の項目を入れる疎な `attributes` も持つ。スキーマを持たない自由形式のキー・値ではなく、管理者が定義したスキーマでグループのプロフィールを拡張する方式を採り、`User` の属性と同じ統治の姿勢を保つ。`email` は `User.email` と同じ形式検査だけを行い、検証済みフラグ、変更要求のフロー、一意性の制約は持たない。グループには受信箱を支配していることを示せる本人がおらず、それに依存する認証経路もないからである。

`Group.attributes` は、`GroupAttributeDef` で定義する `TenantGroupAttributeSchema` に対して検証する。これは `Tenancy` が所有するテナント単位の Aggregate である。どのプリンシパルを統治するスキーマであっても、テナント単位のスキーマ管理は `Tenancy` の関心事であるため、`TenantUserAttributeSchema` と同じ場所に置く。`GroupAttributeDef` は `UserAttributeDef` と異なり、`key`、`label`、`type`、`multi_valued`、`required` だけを持ち、`editable_by_user`、`claim_name` / `oidc_scope`、`visibility` は持たない。`Group` にはセルフサービスの編集画面がなく、その属性を OIDC / SAML クレームへ射影しないためである。和集合にする組み込みカタログもない。`User` の組み込み層は、OIDC §5.1 と SCIM `enterprise:User` が多数の任意プロフィールクレームを固定的に定めるため存在するが、Group には同様の標準語彙がない。そのため `TenantGroupAttributeSchema.attributes` だけが実効定義の集合になる。未定義キーの拒否、型の一致、`multi_valued` の整合性、`required` の充足という `ValidateAttributes` 型の検査は概念として再利用する。一方で、2 つの定義がすべてのフィールドを共有するわけではないため、`GroupAttributeDef` に対する Group 固有の処理として実装する。管理者は、ユーザースキーマと同じ形の 2 つのエンドポイント `GetTenantGroupAttributeSchema` / `UpdateTenantGroupAttributeSchema` (`/api/admin/v1/tenant/group_attribute_schema`) を通じてスキーマを管理する。

## Agent Principal

`Agent` は、`User` と OAuth2 が所有する資格情報プリミティブに並ぶ、第 3 の第一級プリンシパル型である。この Context が所有するのは、アイデンティティ、所有権、ライフサイクル、資格情報のバインディングを含む Aggregate 自体である。エージェントがトークン交換のチェーンで actor として振る舞うための委譲機構は `OAuth2` が所有する。

`Agent` Aggregate は `(id, tenant_id, display_name, kind, status, owner, purpose, created_at, updated_at, disabled_at?, killed_at?)` を持つ。`id` は URL セーフなスラッグであり、`kind` はエージェントの行為に人間がどの程度関与するかを宣言するため、`autonomous` と `supervised` を区別する。登録、検索、変更はすべてテナント単位とし、IdManagement の他の Aggregate と同じテナント境界に従う。

`Agent` は独自の資格情報プリミティブを持たず、`AgentCredentialBinding` を通じて 1 個以上の既存 `OAuth2Client` 登録にバインドする。これにより、1 つの資格情報と鍵管理のインターフェースを一般的な M2M クライアントとエージェントの両方で利用し、`Agent` はその上に所有権、目的、ライフサイクルの層だけを追加する。すべての Agent は所有者（`User` または所有する `Group`）を持たなければならず、所有者のない Agent は登録できない。所有者のオフボーディングは、孤立した非人間アイデンティティを残さないよう、その所有者が所有する Agent へ伝播させる。

ライフサイクルの状態は `active` / `disabled` / `killed` である。`disabled` は復元可能な運用停止、`killed` は一方向の緊急停止を表す。どちらも、各バインディングの `OAuth2Client` が通るトークン発行境界でフェイルクローズに強制する。ステータスが `active` ではない Agent には新しいトークンを発行せず、検査に曖昧さがあれば発行しない側へ倒す。これはキルスイッチに共通する意図的な方針である。`AgentRegistered` / `AgentUpdated` / `AgentDisabled` / `AgentEnabled` / `AgentDeleted` / `AgentOwnerChanged` は、既存の監査・アウトボックス経路へ発行する。Agent の CRUD とキルスイッチは一般的な管理者ロールを再利用せず、専用の `AdminAgentsManage` 権限で制御する。
