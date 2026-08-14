---
context: identity-management
updated_at: 2026-08-11
---

# IdManagement Specification

## Overview

人間の `User`、`Group`、非人間の `Agent` というアイデンティティプリンシパルと、そのプロフィール、ロール、ライフサイクル、管理 API、自己管理 API を所有する。資格情報の検証、MFA、ログインセッションは Authentication に分離する。

`IdManagement` コンテキストは、テナント単位のプリンシパル台帳である `User`、`Group`、`Agent` の各集約と、ユーザープロフィールの検証に使う属性スキーマを所有する。資格情報の検証とログインセッションは `Authentication` が、OAuth2 クライアントの資格情報とトークン発行は `OAuth2` が所有し、IdManagement はそれらのコンテキストが認証とトークン発行の対象にするプリンシパルのレコードを所有する。`User`、`Group`、`Agent` はそれぞれ `user/`、`group/`、`agent/` に置く別々の機能スライスであり、各スライスが自身のドメイン、ポート、ユースケース、アダプターを持つ。本書はユーザーのライフサイクル、ユーザープロフィールを構成する属性モデル、`Group`、`Agent` の順に読む。

## Glossary

| Term | Definition | Aliases |
|---|---|---|
| Administrator | `User.roles` に `admin` を持ち、所属テナント内の管理 API の利用を許可された認証済みユーザー。テナント境界を越える操作は SystemAdministrator に限定する。 | admin, 管理者, TenantAdmin |
| SystemAdministrator | `User.roles` に `system_admin` を持つ認証済みユーザー。テナント管理（CRUD・無効化・有効化）とテナント横断操作を許可され、`system_admin` 専用のシステムコンソール (`/system`) から `/api/admin/tenants/*` や `/api/admin/keys/health` を呼び出せる。テナント境界を越えるため、パスではなくロールで制御する。 | system_admin, システム管理者 |
| EndUser | 認証済みまたは認証を試みる一般利用者。管理ロールを持たない自己サービス操作 (account portal) の主体。 | end ユーザー, 利用者, エンドユーザー |
| UserDisablement | `User.status` を `Disabled` に遷移させ、認証とセッション利用を停止する復元可能な管理操作。削除や個人識別情報の消去とは異なる。 | disable ユーザー |
| UserImport | 管理者が UTF-8 CSV を使ってユーザーの作成・部分更新を事前検証し、成功済み プレビュー に結合して非同期適用する操作。CSV は安定した機械キーのヘッダーを任意順・任意部分集合で持ち、パスワードや password_hash を含めない。 | CSV ユーザーインポート, bulk ユーザーインポート, ユーザー一括インポート |
| UserDeletion | User の Tombstone 化と関連 Aggregate のカスケード削除。`status` を `Deleted` に遷移させて個人識別情報のフィールドを匿名化し、監査のため `id` だけを保持する。`Deleted` は終端状態であり復元できない。 | delete ユーザー, anonymize ユーザー, アカウント削除 |
| Deleted | User の終端状態。`status == Deleted` で PII が anonymize 済み。login / トークン / userinfo は active=false 相当。 | deleted |
| Delete | User を Deleted に遷移させる管理操作。tombstone 化と cascade を 1 オペレーションで実施する。 | delete |
| PendingDeletion | User の削除予約状態。`status == PendingDeletion` では個人識別情報を保持するが、`Disabled` と同様に認証を拒否する。猶予期間（`states.UserLifecycle` の `PendingDeletion` → `Deleted` 遷移のガード）内であれば Restore で `Active` に戻せる。猶予期間を過ぎると Purge により `Deleted` へ遷移し、匿名化する。 | pending_deletion, 削除予約中 |
| SoftDelete | User を Active / Disabled から PendingDeletion に遷移させる管理操作。PII / Consent / RefreshToken / Session を残したまま削除を予約し、誤操作を猶予期間内で救済できる。 | soft_delete, soft-delete |
| Restore | `PendingDeletion` の User を `Active` に戻す管理操作。猶予期間内だけ実行でき、個人識別情報と資格情報は保持しているため通常どおりログインを再開できる。 | restore |
| Purge | User を `Active` / `Disabled` / `PendingDeletion` から `Deleted` に遷移させる確定削除操作。匿名化をカスケードし、猶予期間経過後の自動消去と、管理者による明示的な完全削除の双方から呼び出す。 | purge |
| Group | テナント単位集約。再利用可能なロール束 (ロール[]) を持ち、所属する User にそのロールを一斉付与する。階層・deny ルール・属性自動所属は持たない (union のみ)。連絡先 email と、TenantGroupAttributeSchema に対して検証される custom attributes も持つ。 | group, グループ, ロール group, ロールグループ |
| GroupMembership | User と Group の所属関係 (`GroupMember`)。`manual` は管理者操作、`dynamic` は有効な CEL 規則の評価結果だけから変更する。`effective_roles(user) = user.roles ∪ ⋃ membership.group.roles`。 | group membership, グループ所属, membership |
| DynamicGroupRule | User の中核属性と TenantUserAttributeSchema で定義した属性だけを参照し、所属可否を Boolean で返す制限付き CEL 式。規則のバージョンが一致する動的メンバーシップだけを有効とする。 | dynamic membership rule, 動的グループルール |
| EffectiveRoles | 認可判断で使う User の有効ロール集合。`User.roles` と所属 Group のロールの和集合。管理者向け RBAC の制御と `/account` の自己管理 Context で参照する。 | effective ロール, 有効ロール |
| Agent | テナント単位の非人間アイデンティティプリンシパル。自身の資格情報は持たず、AgentCredentialBinding で既存の OAuth2Client にバインドしてトークンを得る。所有者（User または Group の ID）は必須。 | agent, エージェント, AI エージェント, 非人間アイデンティティ |
| Killed | Agent の緊急停止 (kill-switch) による一方向終端状態。Killed からは復帰できず、新規トークンを一切発行しない (fail-closed)。 | killed |
| DisableAgent | Agent を Active から Disabled に遷移させる可逆な運用停止。`/api/admin/agents/{agent_id}/disable` から発火。 | disable agent |
| EnableAgent | Disabled の Agent を Active に戻す。`/api/admin/agents/{agent_id}/enable` から発火。 | enable agent |
| KillAgent | Agent を Killed (一方向終端) に遷移させる緊急停止。`/api/admin/agents/{agent_id}/kill` から発火し、以後復帰不能。 | kill agent |
| Autonomous | Agent が人間の都度承認なしに自律実行する区分。 | autonomous |
| Supervised | Agent が人間の監督下で実行する区分。 | supervised |

## State Transitions

### UserLifecycle

User Aggregate のライフサイクル。`Active` は通常稼働、Disable は復元可能な無効化を表す。SoftDelete で削除予約 (`PendingDeletion`) に入り、猶予期間内は Restore で `Active` に戻せる。Purge では Tombstone 化し、匿名化をカスケードする。`Deleted` は終端状態であり復元できない。`PendingDeletion` から `Deleted` への遷移ガードは、猶予期間（業界で一般的な 7〜30 日に合わせたデフォルト 30 日）が経過したこと、または管理者が明示的に `purge=true` を指定したことのいずれかを要求する。猶予期間の経過前に `purge=false` で PendingDeletion のユーザーに対して呼び出した場合、`DeleteAdminUser.requires` が InvalidRequestError で拒否し、この遷移は発生しない。

Initial: `Active` Terminal: `Deleted`

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Active | UserDisabled | — | Disabled |  |
| Disabled | UserEnabled | — | Active |  |
| Active | UserSoftDeleted | — | PendingDeletion |  |
| Disabled | UserSoftDeleted | — | PendingDeletion |  |
| PendingDeletion | UserRestored | — | Active |  |
| PendingDeletion | UserDeleted | input.purge == true \|\| duration_since(status_changed_at) >= duration('2592000s') | Deleted | UserDeleted |
| Active | UserDeleted | — | Deleted |  |
| Disabled | UserDeleted | — | Deleted |  |

### DynamicMembershipEvaluationLifecycle

全件再評価は queued から running を経て succeeded または failed へ終端する。

Initial: `queued` Terminal: `succeeded`, `failed`

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| queued | DynamicMembershipEvaluationStarted | — | running |  |
| running | DynamicMembershipEvaluated | — | succeeded |  |
| running | DynamicMembershipEvaluationFailed | — | failed |  |

### AgentLifecycle

Agent Aggregate のライフサイクル。`Active` は通常稼働、Disable は復元可能な運用停止、Kill は一方向の終端となる緊急停止を表す。`Killed` は終端状態で復元できず、`Active` 以外には新しいトークンを発行しない（フェイルクローズ）。

Initial: `Active` Terminal: `Killed`

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Active | AgentDisabled | — | Disabled |  |
| Disabled | AgentEnabled | — | Active |  |
| Active | AgentKilled | — | Killed |  |
| Disabled | AgentKilled | — | Killed |  |

### DataExportLifecycle

リソースエクスポートのライフサイクル。`queued` で受理され、`worker` プロセスが `running` で CSV を生成する。成功するとダウンロード可能な `succeeded`、失敗すると不完全なファイルをダウンロードできない `failed` で終了する。終了前は `canceled` で取り消せる。`succeeded` は保持期限を過ぎると `expired` へ遷移し、ファイル本体を完全削除してメタデータと監査記録だけを残す。`succeeded` から `expired` への遷移ガードは、Jobs のデフォルトの記録保持期間と同じ 30 日の経過を要求する。

Initial: `queued` Terminal: `failed`, `canceled`, `expired`

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| queued | DataExportStarted | — | running |  |
| running | DataExportSucceeded | — | succeeded |  |
| running | DataExportFailed | — | failed |  |
| queued | DataExportCanceled | — | canceled |  |
| running | DataExportCanceled | — | canceled |  |
| succeeded | DataExportExpired | duration_since(completed_at) >= duration('2592000s') | expired |  |

## Authorization Boundary

認可の意味づけはアプリケーションとそのテストが強制する。本仕様は API の認証を記録するが、ポリシーの DSL は意図的に定義しない。ポリシーの言語を採用する前に、別の work item で Cedar を評価する。

## Design

### Internal Interfaces

#### ProvisionFederatedUser
Authentication Context が、検証済みの上流アイデンティティとテナントの明示的な JIT ポリシーおよびクレームマッピングに基づいて、パスワード資格情報を持たない `Active` の User を作成するための内部公開インターフェース。テナントのクォータ、ユーザー名とメールアドレスの一意性、属性スキーマ、`UserCreated` イベントには、通常の作成と同じ契約を適用する。
- Result invariant: output.user.password_hash == null
- Result invariant: output.user.lifecycle.status == Active

### User Lifecycle: Deletion and Anonymization

削除は物理的な除去ではなく匿名化である。`User.lifecycle.status` は `Deleted` へ遷移する。これはどの状態からも到達でき、戻る遷移を持たない終端状態である。Aggregate は破棄せず、その場で書き換える。`AdminAuditEvent` をはじめとする追記専用の記録が `sub` を参照しており、物理削除はその参照を壊すうえ、「削除済み」と単なる「停止中」の運用上の区別も消してしまうからである。`sub` は永久に保持し、再利用しない。

墓標への置き換えは、ユーザーを再識別または再認証しうるすべての項目を不可分に消す。`preferred_username` は `deleted:<sub>` になり、`name` / `given_name` / `family_name` / `email` は空になり、`email_verified` と `mfa_enrolled` は `false` に戻り、`password_hash` は空になり、`roles` は空になり、疎な `attributes` の対応表は丸ごと消え、`lifecycle.status` は `Deleted` になる。`preferred_username` は墓標を立てた時点で再利用のために解放される。削除されていない行に範囲を限った部分的な一意インデックスにより、墓標の値は将来のユーザーと衝突せず、解放された名前は再び使える。

削除は、削除したユーザーから以後たどれてはならないすべての Aggregate へ同期的にカスケードする。その `sub` に対する `Consent`、`RefreshTokenRecord`、`LoginSession`、`PasswordHistory`、`MfaFactor`、有効な `DeviceAuthorization` の記録をすべて取り除く。PostgreSQL を使うカスケード処理は 1 つのトランザクション内で行う。Valkey に置くセッションやデバイスコードの状態はストアごとに削除する。これらの状態は本来揮発的であり、短い不整合期間はトランザクションを複雑にしないこととのトレードオフとして許容できるためである。

削除は冪等である。すでに Tombstone 化したユーザーに対して再び呼び出しても、監査イベントを再発行せず成功を返す `no_op` になる。そのため、再試行や管理者の並行操作が失敗として現れたり、監査記録を重複させたりしない。自己破壊を防ぐため、操作者と対象が同じプリンシパルで、対象が `admin` または `system_admin` を持つ場合は削除を拒否する。管理者が自身の特権アカウントを削除する経路は、どの対話フローにも必要ないためである。削除のたびに `actorSub` / `targetSub` / `reason` / `occurredAt` を載せた `UserDeleted` 監査イベントを発行する。`sub` と Tombstone が残るため、匿名化後も「誰が何をいつ削除したか」を再構成できる。

### User Profile: Thin Core and Attribute Bag

`User` は型として持つ中核を、アイデンティティ、認証、RBAC に必要なものだけに限る。`sub`、`tenant_id`、`preferred_username`、`password_hash`、`email`、`email_verified`、`mfa_enrolled`、`roles`、`name` / `given_name` / `family_name`、`lifecycle`、各種の時刻である。滅多に使わない OIDC や SCIM の任意の項目 25 個ほどを、すべてのユーザーに型と保存の水準で持たせると、ほとんど使わないテナントにとって模型が肥大することが分かった。それ以外のプロフィールの属性 — 残りの OIDC §5.1 の任意の claim (`middle_name`、`nickname`、`picture`、`phone_number`、`address_*` など)、SCIM 風の組織の属性 (`title`、`department`、`manager_sub` など)、テナントが定義する独自の項目 — は、単一の疎な `attributes: Map<String, AttributeValue>` に置き、実際に値を持つ鍵だけが領域を消費する。OIDC の `address` の claim は入れ子の構造ではなく平坦な鍵 (`address_formatted`、`address_locality` など) として保存し、`AttributeValue` を素直な直和の型 (文字列、数値、真偽値、日付、文字列の配列) に保つ。入れ子の `address` の物へ組み直すのは、UserInfo や ID Token の claim を作るときだけである。

ライフサイクルの正は `User.lifecycle.status`（`Active` / `Disabled` / `Locked` / `Staged` / `Suspended` / `Deleted`）と `status_changed_at` の 1 組だけである。遷移時刻の監査記録は、時刻を持つ `UserDisabled` / `UserDeleted` イベントに残す。認証を許可するのは `status == Active` だけであり、それ以外の状態は、デフォルトで `Active` に解決されるゼロ値も含めて認証を拒否する。

#### Attribute Definitions (`UserAttributeDef`)

OIDC と SCIM の組み込みの属性も、テナントが定義する独自の属性も、同じ `UserAttributeDef` の仕組みが統べるので、管理者が設定するスキーマの形は 2 つではなく 1 つで済む。定義は 2 つの段から来て、1 つの実効的なスキーマに合わさる。

- **組み込みのカタログ** `BuiltinUserAttributeDefs`。コードで定義しすべてのテナントで共有する。OIDC §5.1 の任意の claim と、SCIM の `enterprise:User` に相当する組織の属性である。
- **テナントのスキーマ** `TenantUserAttributeSchema`。`Tenant` Aggregate に埋め込まず、`tenant_id` をキーとする独立した Aggregate とする。スキーマがテナント設定より速く変化すること、将来独自のテーブルへ分ける候補であること、テナント削除時に明示的なカスケード経路が必要なことが理由である。

実効的な定義は組み込みとテナントの和である。組み込みの鍵を再定義するテナントのスキーマはその場で拒否する。各 `UserAttributeDef` は `key` (snake_case、先頭は英字)、`type` (`string` / `number` / `boolean` / `date` / `string_array`)、`required`、`editable_by_user`、`visibility == claim_exposed` のときにのみ効く任意の `claim_name` と `oidc_scope` の組、そして `visibility` 自体を持つ。`visibility` は `private` / `self_readable` / `admin_readable` / `claim_exposed` のいずれかで、relying party へ開示されるのは `claim_exposed` だけである。`pii` のデフォルトは `true` である。定義が明示的に外さない限り、保存され監査される値は平文ではなく SHA-256 で要約される。テナントの利便よりも開示の上限を優先する、安全側のデフォルトである。

`ValidateAttributes` は `User.attributes` の対応表を、保存する前に実効的なスキーマと照合する。定義のない鍵、欠けている必須の値、型の不一致を拒否し、各 `AttributeValue` が宣言された `type` の選ぶ項目だけを埋めていることを強制する。利用者自身の経路 (`UpdateUserProfile` と `/api/account/profile`) はさらに、書き込みを `editable_by_user == true` の属性に限り、対応表全体を置き換えるのではなく鍵ごとに併合する。これにより利用者自身の編集が、触れる理由のない管理者管理の属性を上書きすることはない。またユーザーへ開示するのも `self_readable` と `claim_exposed` の属性だけである。削除の際は型として持つ中核とともに `attributes` の対応表も丸ごと消えるので、疎な入れ物が墓標より長生きすることはない。

### User CSV Round Trip

`User` CSV は 2 つ目のプロビジョニング権威ではなく、IdManagement が所有する部分更新の窓口である。機械処理用の列名の語彙と可逆なセル変換はユーザーのドメインに置き、エクスポートとインポートが HTTP のラベルや UI のロケールに依存せず 1 つの定義を共有する。組み込みの書き込み可能列、読み取り専用列、禁止列は閉じた集合とする。テナント定義の列は、ユーザーのユースケースポートを介して実効属性スキーマを解決した後にだけ `custom:<key>` として追加する。これにより解析を決定的に保ちながら、テナントのスキーマを CSV 処理とは独立して発展させられる。

ドメインの解析器は列の有無とセルの内容を別々に保持する。列がないことは Aggregate を変更しないことを意味する一方、存在する空の列は項目を消す意味を持ちうるからである。どの行も計画する前に、未知のヘッダー、重複するヘッダー、シークレットを含むヘッダーを拒否する。数式に対して安全な変換器は、表計算で危険な先頭文字と既存の先頭アポストロフィーに可逆な形で接頭辞を付け、レコードの引用処理を RFC 4180 CSV の変換器に委ねる。したがってエクスポートとインポートは、報告専用のエクスポートで使われる情報を失うアポストロフィーのエスケープを受け入れず、カンマ、引用符、複数行の値を含めて `decode(encode(value)) == value` という不変条件を共有する。

`User` のエクスポートとインポートは、設定可能な 1 つの転送ポリシーを共有する。デフォルトの上限はデータ行 100,000 件、成果物ごとに 64 MiB、項目ごとに 64 KiB とする。これは 1 個の非同期成果物に対するリソース保護の上限であり、テナントのユーザー数の上限ではない。実効ポリシーを超える `User` エクスポートは、再インポートできない成功済み成果物を作らず失敗する。容量の契約には、10,000 ユーザーとすべての組み込み列を持つ統合フィクスチャーを含め、小さな移行単位を超える規模で往復の保証を検証する。

解析と直列化は `io.Reader` と `io.Writer` に対して動作し、CSV 全体を文字列、2 次元のレコードスライス、ジョブ JSON 内の base64 として実体化しない。解析器は byte、行、項目の上限を段階的に強制する。計画ではページ単位のリポジトリ読み取りから上限付きの ID と username のインデックスを構築し、適用では行ごとのトランザクション境界を保ちながら上限付きのまとまりで進める。これによりworkerのメモリを成果物全体の大きさから独立させ、CSV の行ごとにリポジトリを検索することを避ける。

ユーザーのユースケース層は プレビュー と 適用 に共通する 1 個の決定的な計画器を所有する。既存の集約を不変な ID、次に優先 username で解決し、型付きの独自属性と行間の衝突を検証し、状態を変更せず `created`、`updated`、`unchanged`、`rejected` のいずれかを生成する。適用 は プレビュー 時の古い変更計画を実行せず、現在のリポジトリ状態に対して同じ計画器を再び実行する。これにより同時変更を 適用 時に可視化し、プレビュー が暗黙の楽観的ロックの迂回路になることを防ぐ。

プレビュー が CSV を受け取るのは 1 回だけである。テナント単位の不変な成果物ストアが stream を受け取り、中身の見えない参照、サーバーが計算した SHA-256、byte 数、行数を返す。ジョブのパラメーターと結果はそのメタデータと要約だけを保存し、CSV の本文や base64 を決して含まない。永続化アダプターは上限付きのまとまりを使うため、永続化と読み取りに成果物と同じ大きさの単一のデータベース値やプロセスのバッファーを必要としない。メモリアダプターはテストとローカルの組み立てに同じポートを提供する。

適用が受け入れるのは、成功したプレビュージョブの ID だけである。別の CSV や、クライアントが提示するダイジェストは受け入れない。ジョブが認可、テナント、ライフサイクルの境界を与え、SHA-256 は実行対象をプレビュー時に保存した正確なペイロードへバインドする内部の完全性検査となる。これは署名ではなく、認可を置き換えない。適用ジョブはプレビューへの参照とダイジェストの両方を記録するため、`worker` はキュー投入後のペイロードの置換や破損も検出できる。ペイロードは固定するが、変更計画は現在の状態から意図的に再生成する。大量の行エラーは、同じ不変かつチャンク化したアーティファクトストア内の固定件数のページへ直列化し、リソース固有のエラーテーブルやジョブの JSON には置かない。公開する結果 API は、管理 API のテナントとクエリにバインドした署名済みカーソル、`limit`、RFC 8288 の `Link`、`Pagination-*` ヘッダーを使う。エラーの通し番号はアーティファクトのページとページ内の位置へ直接対応するため、深い位置を前後に読み取る場合も先行するエラーを走査しない。

受理した各行は、プロフィールフィールド、直接付与するロール、必須操作、カスタム属性、永続化、監査イベントの発行をまとめた Aggregate 単位の変更境界を通る。その境界内で失敗した場合は先行する部分的な変更を残さず行を拒否する。ある行の失敗によって、他の受理済み行はロールバックしない。これにより、行単位の復旧可能性と予測可能な再試行を両立する。

2 つのユースケースポートにより、Context をまたぐ知識を CSV ドメインの外に保つ。実効ユーザー属性スキーマのリーダーは型付きのテナント定義を提供し、取り込み元の所有権ガードは既存の User が外部管理かどうかを返す。これらのアダプターはアプリケーション境界で組み立てる。所有権を確認できない場合や確認に失敗した場合は、更新対象を外部管理として扱う。CSV という便宜的な経路が、より強い上流の権威を暗黙に上書きしてはならないためである。

### Group Aggregate and Effective Roles

`Group` はテナント単位の Aggregate であり、`(id, tenant_id, name, description?, roles[], created_at, updated_at?)` を持つ。組織変更のたびに影響する全 User の `roles` を個別に編集せず、ロールの組（「営業チーム = `catalog:read` + `invoice:read`」）を 1 単位として付与・取り消しできるよう導入した。`id` は生成後に変わらない `group_<uuid>` である。`name` はテナント内で一意な編集可能な表示名であり、`(tenant_id, name)` の一意インデックスで強制する。テナントをまたぐメンバーシップは無条件に拒否する。`AddMember` は対象の `User` を読み込み、存在しない場合や別テナントに属する場合は拒否する。

User の実効ロールは `user.roles ∪ ⋃_{g ∈ user.groups} g.roles` である。単純な和集合を整列して重複を除き、減算や優先順位の規則は持たない。平らな和集合で十分なところへ deny や minus の演算子を導入すると、評価順の複雑さが増すためである。User がどの Group にも属さない場合、実効ロールは `user.roles` に戻るため、`Group` の導入によって既存アカウントの挙動は変わらない。管理コンソールの RBAC 制御と `/account` の自己管理ビューでは、生の `user.roles` ではなく実効ロールを解決する。User は自身の実効権限のうち、グループメンバーシップに由来するものを確認できる。`User.roles` 自体は、どの Group にも覆われない User 個別の上書き経路として残す。ロールはデフォルトではトークンクレームへ射影しない。個別ロールにも Group 由来のロールにも、そのマッピングはまだ存在せず、ここでは意図的に対象外とする。

メンバーシップ操作は冪等である。既存メンバーの追加や、メンバーではない User の削除はドメインイベントを再発行しない `no_op` とし、Okta と Keycloak のメンバーシップ API の扱いに合わせる。Group の CRUD とメンバーシップの変更では、`AdminAuditEvent` と、`GroupCreated` / `GroupUpdated` / `GroupDeleted` / `GroupMemberAdded` / `GroupMemberRemoved` のいずれかを発行する。Group の削除ではメンバーシップをカスケード削除し、最後の `GroupDeleted` より前にメンバーごとの `GroupMemberRemoved` を発行する。

### Group Contact and Custom Attributes

`Group` は任意の `email` (部署の distribution list などの単純な連絡先) と、テナント定義の custom metadata を入れる疎な `attributes` も持つ。Keycloak の schema のない自由形式の key / value model ではなく、管理者定義の schema で group profile を拡張する Okta と Microsoft Entra ID の方式に従う。これは既存の `User` attribute の仕組みと同じ姿勢である。`email` は形式だけを検証し (`User.email` と同じ address format の検査)、`User.email` と異なり verification flag、変更要求の flow、一意性の constraint を持たない。group には inbox の支配を証明する self-service actor がおらず、それに依存する認証経路もないからである。

`Group.attributes` は、`GroupAttributeDef` で定義する `TenantGroupAttributeSchema` に対して検証する。これは `Tenancy` が所有するテナント単位の Aggregate である。どのプリンシパルを統治するスキーマであっても、テナント単位のスキーマ管理は `Tenancy` の関心事であるため、`TenantUserAttributeSchema` と同じ場所に置く。`GroupAttributeDef` は `UserAttributeDef` と異なり、`key`、`label`、`type`、`multi_valued`、`required` だけを持ち、`editable_by_user`、`claim_name` / `oidc_scope`、`visibility` は持たない。`Group` にはセルフサービスの編集画面がなく、その属性を OIDC / SAML クレームへ射影しないためである。和集合にする組み込みカタログもない。`User` の組み込み層は、OIDC §5.1 と SCIM `enterprise:User` が多数の任意プロフィールクレームを固定的に定めるため存在するが、Group には同様の標準語彙がない。そのため `TenantGroupAttributeSchema.attributes` だけが実効定義の集合になる。未定義キーの拒否、型の一致、`multi_valued` の整合性、`required` の充足という `ValidateAttributes` 型の検査は概念として再利用する。一方で、2 つの定義がすべてのフィールドを共有するわけではないため、`GroupAttributeDef` に対する Group 固有の処理として実装する。管理者は、ユーザースキーマと同じ形の 2 つのエンドポイント `GetTenantGroupAttributeSchema` / `UpdateTenantGroupAttributeSchema` (`/api/admin/v1/tenant/group_attribute_schema`) を通じてスキーマを管理する。

### Agent Principal

`Agent` は、`User` と OAuth2 が所有する資格情報プリミティブに並ぶ、第 3 の第一級プリンシパル型である。エージェント固有の関心事を `OAuth2Client` に後付けすると、監査とポリシーにおいて自律型・監督型の AI エージェントを一般的な M2M クライアントと区別できない。一方、エージェントに独立した資格情報と暗号機能のインターフェースを与えると、`OAuth2Client` にすでに存在する攻撃面が倍増する。IdManagement は、アイデンティティ、所有権、ライフサイクル、資格情報のバインディングを含む Aggregate 自体を所有する。エージェントがトークン交換チェーンのアクターとして行為するための委任機構は OAuth2 が所有し、その Context の設計記録で扱う。

`Agent` Aggregate は `(id, tenant_id, display_name, kind, status, owner, purpose, created_at, updated_at, disabled_at?, killed_at?)` を持つ。`id` は URL セーフなスラッグであり、`kind` はエージェントの行為に人間がどの程度関与するかを宣言するため、`autonomous` と `supervised` を区別する。登録、検索、変更はすべてテナント単位とし、IdManagement の他の Aggregate と同じテナント境界に従う。

`Agent` は独自の資格情報プリミティブを持たず、`AgentCredentialBinding` を通じて 1 個以上の既存 `OAuth2Client` 登録にバインドする。これにより、1 つの資格情報と鍵管理のインターフェースを一般的な M2M クライアントとエージェントの両方で利用し、`Agent` はその上に所有権、目的、ライフサイクルの層だけを追加する。すべての Agent は所有者（`User` または所有する `Group`）を持たなければならず、所有者のない Agent は登録できない。所有者のオフボーディングは、孤立した非人間アイデンティティを残さないよう、その所有者が所有する Agent へ伝播させる。

ライフサイクルの状態は `active` / `disabled` / `killed` である。`disabled` は復元可能な運用停止、`killed` は一方向の緊急停止を表す。どちらも、各バインディングの `OAuth2Client` が通るトークン発行境界でフェイルクローズに強制する。ステータスが `active` ではない Agent には新しいトークンを発行せず、検査に曖昧さがあれば発行しない側へ倒す。これはキルスイッチに共通する意図的な方針である。`AgentRegistered` / `AgentUpdated` / `AgentDisabled` / `AgentEnabled` / `AgentDeleted` / `AgentOwnerChanged` は、既存の監査・アウトボックス経路へ発行する。Agent の CRUD とキルスイッチは一般的な管理者ロールを再利用せず、専用の `AdminAgentsManage` 権限で制御する。

### Design Decisions

- Context が単一の機能を超えて成長したら、`User`、`Group`、`Agent` は 1 個の平らな Context パッケージではなく、それぞれがドメイン、ポート、ユースケース、アダプターを持つ別々の垂直分割として構成する。
- User の削除は物理的な行の削除ではなく、再識別可能なフィールドを消した `Deleted` の Tombstone として、その場で匿名化する。`AdminAuditEvent` などの追記専用レコードが `sub` を参照しており、物理削除はその参照を壊すためである。
- `User` は最小限の型付き中核を保ち、それ以外のプロフィール属性を 1 個の疎な `attributes` マップに置く。ほとんどの User が使わない約 25 個の任意の OIDC / SCIM フィールドをすべて型として持たせない。
- 組み込みとテナント定義の attribute は 2 個の別の system ではなく、1 個の `UserAttributeDef` schema mechanism (builtin catalog ∪ tenant schema) で統治し、機微な値を安全側のデフォルトで扱うため `pii` のデフォルトを `true` とする。
- User CSV は可逆な部分アップサート形式を使い、サーバーが保持する成功済みのプレビューペイロードだけを適用し、現在の状態に対して変更を再計画する。
- `Group` はロールを一組として付与・取り消しできるテナント単位の Aggregate として導入し、実効ロールは優先順位や減算規則を持つ階層ではなく、`user.roles` と Group のロールの単純な和集合として計算する。
- `Group.attributes` は Keycloak の schema のない自由形式の key / value model ではなく、`User` と同じ schema 駆動の統治姿勢 (Okta / Entra ID 型) を再利用する。ただし `UserAttributeDef` / `TenantUserAttributeSchema` へ統合せず、別の `GroupAttributeDef` / `TenantGroupAttributeSchema` の仕組みを使う。`Group` には共有できる組み込み OIDC / SCIM catalog、self-service editor、claim exposure の層がないからである。
- `Agent` は `User` と `OAuth2Client` とは異なる第 3 の第一級プリンシパル型である。独自の資格情報や暗号機能のインターフェースは所有せず、既存の `OAuth2Client` 登録にバインドする。資格情報の攻撃面を倍増させず、自律型・監督型エージェントを一般的な M2M クライアントと区別できるようにするためである。
- `Agent` の登録、検索、変更はテナント単位とし、IdManagement の他の Aggregate と同じく、テナントを Aggregate の境界および永続化の境界とする。

## Scenarios

### REQ-IDMANAGEMENT-001: federated JITはpassword credentialを作らずactive Userを作成する
- ACTOR EndUser
- GIVEN Authentication Context が上流トークン / Assertion とテナントの JIT ポリシーを検証済みである
- WHEN Authentication Context が ProvisionFederatedUser を呼ぶ
- THEN mapped username、任意の name/email/attributes、テナント quota、一意性を検証する
  - ALT username/email が衝突する、quota 超過、または属性 schema が不正である → User を作成せずエラーを返す
- THEN password_hash が null の active User を作成して UserCreated を発行する

### REQ-IDMANAGEMENT-002: API トークン発行者はaccount スコープで自身のアイデンティティ情報だけを操作できる
- ACTOR SelfApiClient
- GIVEN クライアントは対象テナントの active User に固定された有効な API access トークンを提示している
- WHEN クライアントが summary、プロファイル、data export、または primary email change request の操作を要求する
  - ALT account:read だけで変更操作を要求する → 操作は AccessDeniedError で拒否される
  - ALT トークンのテナントまたは user_id が操作対象と一致しない → 操作は AccessDeniedError で拒否される
- THEN account:read scope は自身の summary、プロファイル、data エクスポートの参照だけを許可する
- THEN account:write scope は自身のプロファイルと primary email change request の変更だけを許可する

### REQ-IDMANAGEMENT-003: email verification画面は未認証でもCSRF境界を確立できる
- ACTOR EndUser
- WHEN EndUser がメール確認コンテキストを取得する
- THEN 応答に CSRF トークンと SameSite cookie が含まれる
- WHEN EndUser が CSRF トークンと SameSite cookie を使って email verification を送信する
  - ALT CSRF トークンと cookie が一致しない → email verification は InvalidRequestError で拒否される
- THEN email verification が受理される

### REQ-IDMANAGEMENT-004: 管理者は CSV を検証して有効な行だけをインポートできる
- ACTOR TenantAdministrator
- GIVEN ロール=["admin"] のユーザー "operator" が管理画面のユーザー一覧を開いている
- WHEN 管理者が machine-key header [id,email,ロール,custom:department] を任意順で含む CSV を事前検証へ投入する
  - ALT CSV が実効 UserCsvTransferPolicy の max_bytes、max_rows、max_field_bytes のいずれかを超える → インポート投入は拒否される → エラー "csv_too_large" または "too_many_rows" または "field_too_large"
  - ALT CSV のヘッダーに未知列、重複列、password または password_hash が含まれる → インポート投入は拒否される → エラー "invalid_header"
- THEN プレビュージョブは `created`、`updated`、`unchanged`、`rejected` の判定、行番号、安定したエラーコードを返し、`User` は変更されない
  - ALT 行の `id` と `preferred_username` が別の `User` を示す、識別子がない、同じ対象または同じ最終ユーザー名を複数行が示す → 対象行は `rejected` となり、安定したエラーコードを返す
- WHEN 管理者が同一テナントの成功済み プレビュー job id を指定して 適用 を開始する
  - ALT プレビュージョブが存在しない、`queued` または `failed` である、別テナントに属する、保存済みのペイロードとダイジェストが一致しない → 適用は `User` を変更せず `InvalidRequestError` または `AccessDeniedError` で拒否される
- THEN CSV は再送されず、保存済み プレビュー payload が使用される
- THEN 適用 は プレビュー payload と SHA-256 を検証し、現在の repository 状態に対して同じ planner で再計画する
  - ALT プレビュー後に対象 `User` の状態が別の操作で変更されている → 適用は古いプレビュー計画を実行せず、現在状態から `updated`、`unchanged`、`rejected` を再判定する
- THEN 有効行は create または update され、無効行は rejected として残り、各行のプロファイル・ロール・required actions・custom attributes は原子的に保存される
  - ALT 対象 `User` が外部ソース管理である → 対象行は安定したエラーコード `source_managed` で `rejected` となり、`User` は変更されない
  - ALT 1 行の validation、保存、または監査処理が途中で失敗する → その行のプロファイル・ロール・required actions・custom attributes は一部も保存されず、別の有効行は適用を継続する

### REQ-IDMANAGEMENT-005: 管理者はユーザー一覧をページングしながら安定して閲覧できる
- ACTOR TenantAdministrator
- GIVEN 所属テナントに limit を超えるユーザーが存在する
- WHEN 管理者が ListAdminUsers を limit のみで実行して先頭ページを取得する
  - ALT `query` または `status` を指定する → `ListAdminUsers` はテナント全体から条件に一致する `User` だけを返す → `pagination.total_items` は条件一致件数、`total_users` は削除済みを除くフィルター非依存の件数を返す → `query` または `status` を変更した管理者はカーソルを破棄して先頭ページから取得する
  - ALT 条件に一致する `User` が 0 件である → ユーザー一覧は空で、総項目数、総ページ数、現在のページ番号は `0 / 0 / 0` を返す → `first`、`prev`、`next`、`last` の `Link` は返さない
  - ALT 正確な件数の取得に失敗する → 0 件として成功させず、リクエスト全体をサーバーエラーで失敗させる
  - ALT 実行者が TenantAdministrator ロールを持たない → ListAdminUsers は AccessDeniedError で拒否される
- THEN 応答は filter に一致する exact total items / total pages / current page / page size と filter 非依存の total_users を返す
- THEN レスポンスの `Link` ヘッダー（`rel="next"`）にコンパクトなカーソルが含まれる
- WHEN 一覧の途中で他の管理者がユーザーを1件削除する
- THEN 削除されたユーザーは一覧対象から除外される
- WHEN 管理者が取得済みの cursor で次ページを取得する
  - ALT cursor が別テナントで発行された、改ざんされた、または query/status が発行時と異なる → ListAdminUsers は InvalidRequestError を返す → 管理者は先頭ページへ戻って再取得する
- THEN 削除された行を除き、既に返却済みの行との重複なく残りのユーザーが返る
- THEN レスポンスの `Link` ヘッダー（`rel="prev"`）に前ページのカーソルが含まれる
- WHEN 管理者が前ページの cursor で前ページを取得する
- THEN そのページのユーザーが canonical order で返る
- WHEN 管理者が rel="last" の end anchor cursor で最終ページを取得する
- THEN 端数を含む最終ページが返る
- WHEN 管理者が rel="first" の cursor を含まない URL で先頭ページを取得する
- THEN canonical な先頭ページが返る

### REQ-IDMANAGEMENT-006: 管理者はユーザー一覧を CSV に安全にエクスポートできる
- ACTOR TenantAdministrator
- GIVEN ロール=["admin"] のユーザー "operator" が管理画面のユーザー一覧を開いている
- GIVEN 一覧には自テナントのユーザーが存在する
- WHEN 管理者が列 [`preferred_username`, `email`] と `status` フィルターを指定して `/users/exports` へエクスポートを開始する
  - ALT 選択列に `User` の許可一覧にないキー（例: `password_hash`）が含まれる → エクスポート開始は `InvalidRequestError` で拒否される → エラー `invalid_columns`
- THEN エクスポートは 202 とエクスポート id を返し、ジョブは queued である
- WHEN 終端前に管理者がエクスポートを取り消す
- THEN ステータスは `canceled` となり、`DataExportCanceled` が発行される
- THEN `worker` プロセスが生成を開始し、`DataExportStarted` が発行される
- THEN 生成が完了してステータスは `succeeded`、`downloadable` は `true` となり、`total_rows` と `byte_size` が記録される
  - ALT 生成が失敗する → ステータスは `failed`、`downloadable` は `false` となり、`error_code` が記録される → `DataExportFailed` が発行される → 不完全なファイルはダウンロードできない
- THEN DataExportSucceeded が発行される
- WHEN 管理者がファイルをダウンロードする
  - ALT セル値が \"=\", \"+\", \"-\", \"@\", タブ, CR, LF のいずれかで始まる → 値は formula injection を避ける可逆 prefix で出力され、インポート decoder は規定どおり prefix 1 文字だけを戻す
  - ALT 保持期限を経過している → ステータスは `expired`、`downloadable` は `false` となる → ファイル本体は完全削除され、ダウンロードは `InvalidRequestError` で拒否される
  - ALT `User` エクスポートの ID を `/groups/exports` または別テナントで指定する → 種類とテナントの境界により、取得、ダウンロード、取り消しは `AccessDeniedError` または `InvalidRequestError` で拒否される
- THEN 選択した machine key と一致する header の RFC 4180 CSV が content-disposition attachment で返る
- THEN DataExportDownloaded が発行される

### REQ-IDMANAGEMENT-007: 管理者はエクスポートしたユーザー CSV を安全に再適用できる
- ACTOR TenantAdministrator
- GIVEN 実効 `TenantUserAttributeSchema` に `custom:department` があり、10,000 件の `User` を含む一覧が実効 `UserCsvTransferPolicy` の上限内に収まる
- WHEN 管理者がインポート-compatible な組み込み列、required_actions、custom:department を machine-key header でエクスポートする
  - ALT 値が危険な先頭文字、既存 apostrophe、comma、quote、または改行を含む → reversible formula-safe codec と RFC 4180 quoting により decode(encode(value)) は元の値と一致する
  - ALT 生成結果が実効 `UserCsvTransferPolicy` のいずれかの上限を超える → `User` エクスポートは `csv_transfer_limit_exceeded` で失敗し、再インポートできない成功済み成果物を作らない → 管理者はフィルターまたは列を絞って複数の成果物に分割できる
- THEN `worker` プロセスは CSV を不変の成果物ストアへストリーミング出力し、ジョブ結果にはテナント単位のペイロード参照、サーバーが算出した SHA-256、サイズ、行数を保持する
- WHEN 管理者が同じ 10,000 行 artifact を無編集で プレビュー する
- THEN 全行 unchanged となり、User は変更されない
- WHEN 管理者が 1 行の email と custom:department だけを編集して再度 プレビュー する
  - ALT 状態mfa_enrolled、created_at、updated_at、または id の値だけを編集する → 読み取り専用列は受理して無視し、writable 列に差分がなければ unchanged とする
  - ALT custom 属性の型、boolean、number、date、required custom 属性、または required_actions が不正である → 対象行は stable error code で rejected となり、値を job view、エラーaudit イベントに含めない
- THEN 変更行だけが updated、残りは unchanged と計画される
- WHEN 管理者が成功済み プレビュー job id を 適用 する
- THEN 指定した writable 列だけが更新され、未指定列は維持される

### REQ-IDMANAGEMENT-008: 管理者は特定グループのメンバー一覧を CSV にエクスポートできる
- ACTOR TenantAdministrator
- GIVEN ロール=["admin"] のユーザー "operator" がグループ "engineering" の詳細を開いている
- WHEN 管理者が /groups/{group_id}/members/exports へ列 [user_id, preferred_username] でエクスポートを開始する
  - ALT group_id を指定しない (per-group 必須) → エクスポート開始は InvalidRequestError で拒否される
- THEN エクスポートは group_id で scope され、そのグループのメンバーだけが対象になる
- WHEN 生成完了後、管理者がメンバー CSV をダウンロードする
  - ALT 別グループの path でそのエクスポート id を指定する → 取得・ダウンロードは InvalidRequestError で拒否される (per-group 分離)
- THEN 指定したグループのメンバーだけを含む CSV が返る

### REQ-IDMANAGEMENT-009: 管理者はエージェントを登録しクライアント資格情報をバインドできる
- ACTOR TenantAdministrator
- GIVEN ロール=["admin"] のユーザー "operator" が管理画面のエージェント一覧を開いている
- WHEN 管理者 "operator" がエージェント "batch-agent" を登録する
- THEN エージェント "batch-agent" が登録される
- WHEN 管理者 "operator" がエージェント "batch-agent" にクライアント資格情報をバインドする
  - ALT 別テナントのクライアント資格情報をバインドする → tenant_id "acme" の Agent に tenant_id "default" の client_id を指定する → エラー "InvalidRequestError"
- THEN クライアント資格情報がバインドされる
- WHEN 管理者 "operator" がエージェント "batch-agent" を無効化する
- THEN エージェントは無効状態になる
- WHEN 管理者 "operator" がエージェント "batch-agent" を再有効化する
- THEN エージェント一覧に "batch-agent" が表示される

### REQ-IDMANAGEMENT-010: 管理者は無効化したユーザーを再有効化できる
- ACTOR TenantAdministrator
- GIVEN 管理者がユーザー "alice" を無効化している
- WHEN 管理者がユーザー "alice" を再有効化する
- THEN ユーザー `alice` のステータスは `Active` である
- THEN "UserEnabled" が発行される

### REQ-IDMANAGEMENT-011: 管理者はユーザーを soft-delete し猶予期間内に復元できる
- ACTOR TenantAdministrator
- GIVEN ロール=["admin"] のユーザー "operator" が管理画面のユーザー一覧を開いている
- GIVEN ユーザー "alice" は Active である
- WHEN 管理者 "operator" がユーザー "alice" を削除する
- THEN ユーザー `alice` のステータスは `PendingDeletion` である
- THEN "UserSoftDeleted" が発行される
- WHEN 管理者 "operator" がユーザー "alice" を復元する
- THEN ユーザー `alice` のステータスは `Active` である
- THEN "UserRestored" が発行される

### REQ-IDMANAGEMENT-012: soft-delete されたユーザーはログインを拒否される
- ACTOR EndUser
- GIVEN ユーザー "alice" は PendingDeletion である
- WHEN ユーザー "alice" が正しいパスワードでログインを試みる
- THEN ログインは拒否される

### REQ-IDMANAGEMENT-013: 管理者はユーザーを完全削除できる
- ACTOR TenantAdministrator
- GIVEN ユーザー "alice" は PendingDeletion である
- WHEN 管理者がユーザー "alice" を完全削除する
  - ALT 対象が admin 自身である → soft-delete / 復元 / 完全削除のいずれも拒否される → エラー "self_delete_forbidden"
- THEN ユーザー `alice` のステータスは `Deleted` である
- THEN "UserDeleted" が発行される

### REQ-IDMANAGEMENT-014: ロールに応じて管理APIのアクセスが制御される
- ACTOR TenantAdministrator
- GIVEN ロールに "admin" を持つユーザー "operator" が認証済みである
- WHEN 管理者 "operator" が preferred_username "bob" のユーザーを作成する
- THEN "UserCreated" が発行される
- WHEN 管理者 "operator" がユーザー一覧を取得する
  - ALT admin ロールを持たないユーザーが管理 API を呼ぶ → ロールが空のユーザー "alice" が認証済みである → ユーザー "alice" がユーザー一覧を取得する → エラー "AccessDeniedError"
- THEN 応答にユーザー "bob" が含まれる

### REQ-IDMANAGEMENT-015: 管理者はグループを作成しユーザーを所属させると有効ロールにグループ由来ロールが乗る
- ACTOR TenantAdministrator
- GIVEN ロール=["admin"] のユーザー "operator" が認証済みである
- GIVEN ロールが空のユーザー "alice" が同一テナントに存在する
- WHEN 管理者 "operator" がロール=["catalog:read"] のグループ "engineering" を作成する
- THEN "GroupCreated" が発行される
- WHEN 管理者 "operator" がユーザー "alice" をグループ "engineering" に所属させる
  - ALT 同じユーザーを同じグループへ再度所属させる → 管理者 "operator" がユーザー "alice" をグループ "engineering" に再度所属させる → "GroupMemberAdded" は再発行されない
- THEN "GroupMemberAdded" が発行される
- WHEN 管理者がユーザー "alice" の所属グループを取得する
- THEN effective_roles に "catalog:read" が含まれる
- THEN group_roles は "catalog:read" を含み direct_roles は空である

### REQ-IDMANAGEMENT-016: ユーザーは自分のプロフィール表示名を更新できる
- ACTOR AuthenticatedSelf
- GIVEN ユーザー "alice" が認証済みでマイアカウントのプロフィールを開いている
- WHEN ユーザー "alice" が表示名を更新する
- THEN 更新後のプロフィールに新しい表示名が反映される
- THEN editable_by_user=false の属性は更新できない

### REQ-IDMANAGEMENT-017: ユーザーはメールアドレス変更を起票し確認リンクで確定できる
- ACTOR AuthenticatedSelf
- GIVEN ユーザー "alice" が認証済みでメールアドレス画面を開いている
- WHEN ユーザー "alice" が新しいメールアドレスへの変更を起票する
- THEN 新アドレスへ確認リンクが送られる
- WHEN ユーザー "alice" が確認リンクのトークンで変更を確定する
- THEN primary email が新しいアドレスへ更新される

### REQ-IDMANAGEMENT-018: ユーザーは自分のアカウントデータをエクスポートできる
- ACTOR AuthenticatedSelf
- GIVEN ユーザー "alice" が認証済みでデータとプライバシー画面を開いている
- WHEN ユーザー "alice" がアカウントデータをエクスポートする
- THEN 応答に自分のプロファイルと consents が含まれる

### REQ-IDMANAGEMENT-019: マイアカウントAPIは他人のリソースを返さない
- ACTOR AuthenticatedSelf
- GIVEN ユーザー "alice" が認証済みである
- WHEN ユーザー "alice" のマイアカウント概要を取得する
- THEN 応答は "alice" 自身のデータだけでロールを含まない

### REQ-IDMANAGEMENT-020: 管理者はCELルールで動的グループ所属を管理できる
- ACTOR TenantAdministrator
- GIVEN department 属性が定義され Engineering の User と Sales の User が存在する
- GIVEN membership_type=dynamic のグループが存在する
- WHEN 管理者が `user.department == "Engineering"` を保存して有効化する
- THEN 全件再評価後に Engineering の Active User だけが dynamic_rule source で所属する
- THEN effective_roles と application assignment はその所属を参照する

### REQ-IDMANAGEMENT-021: CELルールは保存前に選択ユーザーでプレビューできる
- ACTOR TenantAdministrator
- GIVEN 管理者が最大100 Userを選択している
- WHEN 未保存 CEL を評価する
- THEN 応答は matched と add/remove/unchanged を返し属性値を返さない

### REQ-IDMANAGEMENT-022: 不正なCELルールと動的グループの手動操作は拒否される
- ACTOR TenantAdministrator
- WHEN 管理者が未定義属性または許可外関数を参照する CEL を保存する
- THEN 保存は拒否される
- WHEN 管理者が dynamic group に対して手動 AddGroupMember または RemoveGroupMember を呼ぶ
- THEN membership の変更は拒否される

### REQ-IDMANAGEMENT-023: 評価不能なルールは権限を付与しない
- ACTOR TenantAdministrator
- GIVEN 有効な dynamic rule の version が更新された
- WHEN system が新 version の dynamic rule を再評価する
- THEN 旧 version の membership は直ちに effective ロールから除外される
- THEN 再評価が失敗した User は新 version の membership を取得しない

### REQ-IDMANAGEMENT-024: 管理者はグループの連絡先メールとカスタム属性を、テナント定義のスキーマに従って設定できる
- ACTOR TenantAdministrator
- GIVEN admin ロールを持つ "operator" が認証済みである
- GIVEN テナントの Group 属性スキーマに "cost_center" (string, required=false) が定義されている
- WHEN "operator" が email="sales@example.test" と attributes={cost_center: "CC-100"} を指定してグループ "sales" を作成する
  - ALT email がメールアドレスの形式を満たさない → 作成は InvalidEmailError で拒否される
  - ALT attributes に未定義の key を指定する、または定義済み key と型が一致しない → 作成は InvalidGroupAttributeError で拒否される
- THEN 作成されたグループの email と attributes が指定どおりに保存され "GroupCreated" が発行される
- WHEN "operator" が同グループの email と attributes を更新する
- THEN 更新後のグループに新しい email と attributes が反映され "GroupUpdated" の changed_fields に "email"/"attributes" が含まれる
