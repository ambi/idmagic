# ユーザーの内部設計

## フェデレーションによる Just-in-Time プロビジョニング

フェデレーションのログイン時に User を作る経路は、通常の作成経路と同じ不変条件を通る。テナントのクォータ、ユーザー名とメールアドレスの一意性、属性スキーマ、`UserCreated` イベントは、上流からの作成であっても緩まない。この経路のためだけの近道は存在しない。

作られる User にはパスワード資格情報を設定しない。上流が認証の権威である以上、ローカルの資格情報を同時に設定すると、上流を無効にした後もローカルのパスワードでサインインできる経路が残るからである。資格情報を後から追加するかどうかは、テナントの明示的な設定に委ねる。

## 利用者のライフサイクル：削除と匿名化

削除は物理的な除去ではなく匿名化である。`User.lifecycle.status` は `Deleted` へ遷移する。これはどの状態からも到達でき、戻る遷移がない終端状態である。Aggregate は破棄せず、その場で書き換える。`AdminAuditEvent` をはじめとする追記専用の記録が `sub` を参照しており、物理削除はその参照を壊すうえ、「削除済み」と単なる「停止中」の運用上の区別も消してしまうからである。`sub` は永久に保持し、再利用しない。

墓標への置き換えは、ユーザーを再識別または再認証しうるすべての項目を不可分に消す。`preferred_username` は `deleted:<sub>` になり、`name` / `given_name` / `family_name` / `email` は空になり、`email_verified` と `mfa_enrolled` は `false` に戻り、`password_hash` は空になり、`roles` は空になり、疎な `attributes` の対応表は丸ごと消え、`lifecycle.status` は `Deleted` になる。`preferred_username` は墓標を立てた時点で再利用のために解放される。削除されていないレコードに範囲を限った部分的な一意インデックスにより、墓標の値は将来のユーザーと衝突せず、解放された名前は再び使える。

削除は、削除したユーザーから以後たどれてはならないすべての Aggregate へ同期的にカスケードする。その `sub` に対する `Consent`、`RefreshTokenRecord`、`LoginSession`、`PasswordHistory`、`MfaFactor`、有効な `DeviceAuthorization` の記録をすべて取り除く。PostgreSQL を使うカスケード処理は 1 つのトランザクション内で行う。Valkey に置くセッションやデバイスコードの状態はストアごとに削除する。これらの状態は本来揮発的であり、短い不整合期間はトランザクションを複雑にしないこととのトレードオフとして許容できるためである。

削除は冪等である。すでに Tombstone 化したユーザーに対して再び呼び出しても、監査イベントを再発行せず成功を返す `no_op` になる。そのため、再試行や管理者の並行操作が失敗として現れたり、監査記録を重複させたりしない。自己破壊を防ぐため、操作者と対象が同じプリンシパルで、対象が `admin` または `system_admin` を持つ場合は削除を拒否する。管理者が自身の特権アカウントを削除する経路は、どの対話フローにも必要ないためである。削除のたびに `actorSub` / `targetSub` / `reason` / `occurredAt` を載せた `UserDeleted` 監査イベントを発行する。`sub` と Tombstone が残るため、匿名化後も「誰が何をいつ削除したか」を再構成できる。

## 利用者プロフィール：最小限の中核と属性バッグ

`User` は型として持つ中核を、アイデンティティ、認証、RBAC に必要なものだけに限る。`sub`、`tenant_id`、`preferred_username`、`password_hash`、`email`、`email_verified`、`mfa_enrolled`、`roles`、`name` / `given_name` / `family_name`、`lifecycle`、各種の時刻である。滅多に使わない OIDC や SCIM の任意項目 25 個ほどを、すべてのユーザーに型と保存の水準で持たせると、それらを使わないテナントにとってモデルが肥大するだけである。それ以外のプロフィール属性 — 残りの OIDC §5.1 の任意クレーム (`middle_name`、`nickname`、`picture`、`phone_number`、`address_*` など)、SCIM 相当の組織属性 (`title`、`department`、`manager_sub` など)、テナントが定義する独自項目 — は、単一の疎な `attributes: Map<String, AttributeValue>` に置き、実際に値を持つキーだけが領域を消費する。OIDC の `address` クレームは入れ子の構造ではなく平坦なキー (`address_formatted`、`address_locality` など) として保存し、`AttributeValue` を素直な直和型 (文字列、数値、真偽値、日付、文字列の配列) に保つ。入れ子の `address` へ組み直すのは、UserInfo や ID Token のクレームを作るときだけである。

ライフサイクルの正は `User.lifecycle.status`（`Active` / `Disabled` / `Locked` / `Staged` / `Suspended` / `Deleted`）と `status_changed_at` の 1 組だけである。遷移時刻の監査記録は、時刻を持つ `UserDisabled` / `UserDeleted` イベントに残す。認証を許可するのは `status == Active` だけであり、それ以外の状態は、デフォルトで `Active` に解決されるゼロ値も含めて認証を拒否する。

#### 属性定義（`UserAttributeDef`）

OIDC と SCIM の組み込みの属性も、テナントが定義する独自の属性も、同じ `UserAttributeDef` の仕組みが統べるので、管理者が設定するスキーマの形は 2 つではなく 1 つで済む。定義は 2 つの段から来て、1 つの実効的なスキーマに合わさる。

- **組み込みのカタログ** `BuiltinUserAttributeDefs`。コードで定義しすべてのテナントで共有する。OIDC §5.1 の任意の claim と、SCIM の `enterprise:User` に相当する組織の属性である。
- **テナントのスキーマ** `TenantUserAttributeSchema`。`Tenant` Aggregate に埋め込まず、`tenant_id` をキーとする独立した Aggregate とする。スキーマがテナント設定より速く変化すること、将来独自のテーブルへ分ける候補であること、テナント削除時に明示的なカスケード経路が必要なことが理由である。

実効的な定義は組み込みとテナントの和である。組み込みの鍵を再定義するテナントのスキーマはその場で拒否する。各 `UserAttributeDef` は `key` (snake_case、先頭は英字)、`type` (`string` / `number` / `boolean` / `date` / `string_array`)、`required`、`editable_by_user`、`visibility == claim_exposed` のときにのみ効く任意の `claim_name` と `oidc_scope` の組、そして `visibility` 自体を持つ。`visibility` は `private` / `self_readable` / `admin_readable` / `claim_exposed` のいずれかで、relying party へ開示されるのは `claim_exposed` だけである。`pii` のデフォルトは `true` である。定義が明示的に外さない限り、保存され監査される値は平文ではなく SHA-256 で要約される。テナントの利便よりも開示の上限を優先する、安全側のデフォルトである。

`ValidateAttributes` は `User.attributes` の対応表を、保存する前に実効的なスキーマと照合する。定義のない鍵、欠けている必須の値、型の不一致を拒否し、各 `AttributeValue` が宣言された `type` の選ぶ項目だけを埋めていることを強制する。利用者自身の経路 (`UpdateUserProfile` と `/api/account/profile`) はさらに、書き込みを `editable_by_user == true` の属性に限り、対応表全体を置き換えるのではなく鍵ごとに併合する。これにより利用者自身の編集が、触れる理由のない管理者管理の属性を上書きすることはない。またユーザーへ開示するのも `self_readable` と `claim_exposed` の属性だけである。削除の際は型として持つ中核とともに `attributes` の対応表も丸ごと消えるので、疎な入れ物が墓標より長生きすることはない。
