# Directory Connector による取り込み

## 概要

この文書は、閉域の Active Directory と LDAP を取り込み元とする Directory Connector を、テナントに登録し、その呼び出しを認証し、Connector が送る差分から User を取り込む仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | Directory Connector の作成、登録、無効化、Connector の呼び出しの認証、User の取り込みと取り込みの実行の記録 |
| 行為者 | テナント管理者、Directory Connector |
| 扱わないもの | グループの取り込みは提供しない。Connector の実行環境、LDAP の読み取り、LDAP の属性から正規の属性への対応付けは [DirectoryConnector](../../directory-connector/README.md) が担当する。二つのモジュールの契約は Connector API だけである |

Directory Connector は、AD と同じ閉域のネットワークに置く実行単位であり、IdMagic へ外向きの接続だけを行う。
IdMagic は AD への到達経路も、AD の資格情報も持たない。

## モデル

| 項目 | 内容 |
| --- | --- |
| ID | IdMagic が採番する Connector の ID。Connector の JWT の `iss` と `sub` になる |
| 表示名 | 管理者が付ける名前。前後の空白を除いて保存し、200 文字まで |
| 状態 | `AwaitingEnrollment`、`Active`、`Disabled` |
| 登録トークン | 一回だけ使える、有効期限付きのシークレット。IdMagic は SHA-256 のハッシュと有効期限だけを保存する |
| 公開鍵 | 登録で受け取った JWK。P-256 の EC 鍵か 2048 ビット以上の RSA 鍵であり、秘密鍵の成分を持たない |
| 鍵の ID | 公開鍵の RFC 7638 のサムプリント |
| 時刻 | 作成、登録、最終接続、無効化の時刻 |

一つの Connector が一つの取り込み元を表し、取り込み元（[用語集](../glossary.md)の IdentitySource）の関連付けと資格情報を兼ねる。

Connector の呼び出しは、Connector の秘密鍵で署名した JWT を `Authorization: Bearer` で提示して認証する。
JWT の `aud` には、テナントの発行者識別子に `/directory-connector/v1` を続けた値を使う。

- **判断**：mTLS を採らないのは、エッジに証明書の転送の設定を求めず、Connector の秘密鍵を閉域から出さずに、鍵の更新と失効を IdMagic 側の登録の差し替えだけで行うためである。

### 取り込み

Connector は、取り込みの実行（[用語集](../glossary.md)の IngestionRun）を始め、User の項目のバッチを送り、送った項目の数を添えて実行の完了を報告する。
差分の位置（SourceCursor）は Connector の手元だけに置き、IdMagic は保存しない。
Connector を再導入して位置を失ったときは、完全な再同期で回復する。

| 項目 | 内容 |
| --- | --- |
| 実行の種類 | `full` は取り込み元の全件を送る完全な再同期、`incremental` は前回からの差分である |
| 実行の状態 | `running`、`completed`、`superseded` |
| User の項目 | 外部の不変 ID、削除かどうか、有効かどうか、`preferred_username`、`name`、`given_name`、`family_name`、`email`、`employee_number`、`department`、`on_premises_immutable_id` |
| 外部の不変 ID | AD の `objectGUID` や LDAP の `entryUUID` のような、名前の変更で変わらない ID。1 から 256 文字の文字列であり、正規化せずに比べる |
| 対応 | Connector と外部の不変 ID の組から内部の User への対応（[用語集](../glossary.md)の SourceCorrelation）と、その対応が最後に現れた実行 |
| バッチ | Connector が付けたバッチの ID、項目のダイジェスト、項目ごとの結果 |
| 項目の結果 | `created`、`updated`、`unchanged`、`deleted`、`failed` のどれか。`failed` はエラーコードを持つ |

User の属性の対応は SCIM と同じであり、`employee_number` と `department` は User の同名の属性に保存する。
`on_premises_immutable_id` は、Entra のドメインフェデレーションの sourceAnchor にする値であり、User の同名の組み込みの属性に保存する。SCIM はこの属性を変えない。
取り込んだ User は、パスワードを持たず、メールアドレスを確認済みとし、CSV のインポートが変更しない取り込み元の管理下の User になる。

- **判断**：User を外部の不変 ID で相関し、ユーザー名で既存の User に結び付けない。名前の一致だけで結び付けると、取り込み元の別人が既存のアカウントを乗っ取れるからである。

## 状態遷移

### DirectoryConnectorLifecycle

作成直後の Connector は登録待ちであり、登録トークンで公開鍵を登録すると呼び出せるようになる。
再発行は登録済みの公開鍵を破棄して登録待ちへ戻す。
無効化は終端であり、置き換えるときは新しい Connector を作る。

<!-- spec:state-machine backend/sourcing/testdata/DirectoryConnectorLifecycle.statemachine.json -->

| State | Kind | Meaning |
| --- | --- | --- |
| AwaitingEnrollment | initial | 登録トークンを発行し、公開鍵の登録を待っている |
| Active | — | 公開鍵を登録し、その鍵で署名した JWT で呼び出せる |
| Disabled | terminal | 管理者が無効化した。登録も呼び出しも受け付けない |

| From | Event | Guard | To | Effects |
| --- | --- | --- | --- | --- |
| AwaitingEnrollment | Connector による登録 | 登録トークンが一致し、有効期限内であり、公開鍵が条件を満たす | Active | DirectoryConnectorEnrolled |
| AwaitingEnrollment | 管理者による登録トークンの再発行 | — | AwaitingEnrollment | DirectoryConnectorEnrollmentReissued |
| Active | 管理者による登録トークンの再発行 | — | AwaitingEnrollment | DirectoryConnectorEnrollmentReissued |
| AwaitingEnrollment | 管理者による Connector の無効化 | — | Disabled | DirectoryConnectorDisabled |
| Active | 管理者による Connector の無効化 | — | Disabled | DirectoryConnectorDisabled |

| State | Connector による登録 | Connector による接続の報告 | 管理者による登録トークンの再発行 | 管理者による Connector の無効化 |
| --- | --- | --- | --- | --- |
| AwaitingEnrollment | → Active（登録トークンが一致し、有効期限内であり、公開鍵が条件を満たす）<br>拒否：401 invalid_token（登録トークンが一致しないか、有効期限を過ぎている）<br>拒否：400 invalid_request（公開鍵が条件を満たさない） | 拒否：401 invalid_token | → AwaitingEnrollment | → Disabled |
| Active | 拒否：401 invalid_token | 何もしない | → AwaitingEnrollment | → Disabled |
| Disabled | 拒否：401 invalid_token | 拒否：401 invalid_token | 拒否：409 directory_connector_disabled | 何もしない |

<!-- /spec:state-machine -->

### IngestionRunLifecycle

取り込みの実行は `running` で始まる。
完全再同期の無効化がしきい値を超えると `awaiting_approval` になり、承認後は `applying` を経て `completed` になる。
同じ Connector の新しい実行は、`running` と `awaiting_approval` を `superseded` にする。

<!-- spec:state-machine backend/sourcing/testdata/IngestionRunLifecycle.statemachine.json -->

| State | Kind | Meaning |
| --- | --- | --- |
| Running | initial | バッチを受け付けている |
| AwaitingApproval | — | 候補を保存し、管理者の承認を待つ |
| Applying | — | 保存した候補を適用している。失敗時は承認の再送で再開する |
| Completed | terminal | Connector が完了を報告した。記録したバッチの再送だけに答える |
| Superseded | terminal | 同じ Connector の新しい実行が始まり、打ち切られた。記録したバッチの再送だけに答える |
| Discarded | terminal | 管理者が候補の無効化を破棄した |

| From | Event | Guard | To | Effects |
| --- | --- | --- | --- | --- |
| Running | Connector による取り込みの実行の完了 | 項目数が一致し、増分かしきい値以内の完全再同期である | Completed | — |
| Running | Connector による取り込みの実行の完了 | 項目数が一致し、完全再同期の候補がしきい値を超える | AwaitingApproval | DirectoryIngestionRunHeld |
| Running | Connector による取り込みの実行の開始 | 同じ Connector の新しい実行である | Superseded | — |
| AwaitingApproval | Connector による取り込みの実行の開始 | 同じ Connector の新しい実行である | Superseded | — |
| AwaitingApproval | 管理者による保留した無効化の承認 | — | Applying | DirectoryIngestionRunApproved |
| Applying | 管理者による保留した無効化の承認 | 保存した候補の適用が終わった | Completed | — |
| AwaitingApproval | 管理者による保留した無効化の破棄 | — | Discarded | DirectoryIngestionRunDiscarded |

| State | Connector による取り込みの実行の開始 | Connector による User のバッチの送信 | Connector による取り込みの実行の完了 | 管理者による保留した無効化の承認 | 管理者による保留した無効化の破棄 |
| --- | --- | --- | --- | --- | --- |
| Running | → Superseded | 何もしない | → Completed（項目数一致、増分かしきい値以内）<br>→ AwaitingApproval（項目数一致、しきい値超過）<br>拒否：409 ingestion_run_incomplete（項目数不一致） | 拒否：409 ingestion_run_closed | 拒否：409 ingestion_run_closed |
| AwaitingApproval | → Superseded | 何もしない（記録済みの同一バッチ）<br>拒否：409 ingestion_run_closed（新しいバッチ） | 何もしない | → Applying | → Discarded |
| Applying | 拒否：409 ingestion_run_busy | 何もしない（記録済みの同一バッチ）<br>拒否：409 ingestion_run_closed（新しいバッチ） | 拒否：409 ingestion_run_closed | → Completed（全候補を処理）<br>何もしない（未完了なら再開） | 拒否：409 ingestion_run_closed |
| Completed | 何もしない | 何もしない（記録済みの同一バッチ）<br>拒否：409 ingestion_run_closed（新しいバッチ） | 何もしない | 何もしない | 拒否：409 ingestion_run_closed |
| Superseded | 何もしない | 何もしない（記録済みの同一バッチ）<br>拒否：409 ingestion_run_closed（新しいバッチ） | 拒否：409 ingestion_run_closed | 拒否：409 ingestion_run_closed | 拒否：409 ingestion_run_closed |
| Discarded | 何もしない | 何もしない（記録済みの同一バッチ）<br>拒否：409 ingestion_run_closed（新しいバッチ） | 拒否：409 ingestion_run_closed | 拒否：409 ingestion_run_closed | 何もしない |

<!-- /spec:state-machine -->

## 操作

### 管理者による Connector の作成

#### REQ-SOURCING-008 Directory Connector の作成は、登録待ちの Connector と一回だけ使える登録トークンを返す

- 管理者が表示名を指定して Directory Connector を作成したとき、Sourcing は、`AwaitingEnrollment` の Connector を作り、201 で Connector と登録トークンを返し、`DirectoryConnectorCreated` を発行する。
- 登録トークンを発行したとき、Sourcing は、`idmagic_dce_` に 32 バイトの乱数の base64url を続けた値を作り、発行時刻の 1 時間後を有効期限として記録する。
- 登録トークンを発行したとき、Sourcing は、トークンを応答で一度だけ返し、SHA-256 のハッシュだけを保存する。
- 表示名が空か空白だけの場合、Sourcing は、400 と `invalid_request` で拒否し、Connector を作らない。
- 表示名が 200 文字を超える場合、Sourcing は、422 と `field_length_exceeded` で拒否し、Connector を作らない。
- `admin` または `system_admin` のロールを持たない利用者が作成を要求した場合、Sourcing は、403 と `access_denied` で拒否し、Connector を作らない。
- **判断**：再発行がいつでもできるので、有効期間を長くする利点より、漏えいしたトークンで偽の Connector を登録される期間を短くすることを選ぶ。

### 管理者による Connector の一覧

#### REQ-SOURCING-013 Directory Connector の一覧は、テナントの Connector を作成順に、シークレットを含めずに返す

- 管理者が Directory Connector を一覧したとき、Sourcing は、テナントのすべての Connector を作成時刻の昇順に返す。
- 管理者が Directory Connector を一覧または取得したとき、Sourcing は、各 Connector の ID、表示名、状態、作成時刻と、記録した登録時刻、最終接続時刻、無効化時刻を返す。
- `Active` の Connector を一覧または取得したとき、Sourcing は、鍵の ID を返す。
- `AwaitingEnrollment` の Connector を一覧または取得したとき、Sourcing は、登録トークンの有効期限を返す。
- 管理者が Directory Connector を一覧または取得したとき、Sourcing は、登録トークンとそのハッシュを返さない。
- `admin` または `system_admin` のロールを持たない利用者が一覧を要求した場合、Sourcing は、403 と `access_denied` で拒否する。

### 管理者による Connector の取得

#### REQ-SOURCING-014 Directory Connector の取得は、要求したテナントの Connector だけを返す

- **上位の要件**：[REQ-SOURCING-013](directory.md#管理者による-connector-の一覧)
- 管理者が Directory Connector を取得したとき、Sourcing は、200 でその Connector を返す。
- 存在しないか別のテナントの Connector の ID を指定された場合、Sourcing は、404 と `directory_connector_not_found` で拒否する。
- `admin` または `system_admin` のロールを持たない利用者が取得を要求した場合、Sourcing は、403 と `access_denied` で拒否する。

### Connector による登録

#### REQ-SOURCING-009 Directory Connector の登録は、有効な登録トークンと公開鍵で Connector を `Active` にする

- `AwaitingEnrollment` の Connector の有効期限内の登録トークンと公開鍵の JWK を受けたとき、Sourcing は、Connector を `Active` にし、公開鍵と鍵の ID と登録時刻を保存し、登録トークンのハッシュを消し、`DirectoryConnectorEnrolled` を発行する。
- Connector を登録したとき、Sourcing は、200 で Connector の ID、鍵の ID、JWT の `aud` に使う値を返す。
- 登録トークンが要求したテナントの `AwaitingEnrollment` の Connector のものでないか、有効期限を過ぎている場合、Sourcing は、401 と `invalid_token` で拒否し、どの Connector も変えない。
- 公開鍵が P-256 の EC 鍵でも 2048 ビット以上の RSA 鍵でもないか、秘密鍵の成分を含む場合、Sourcing は、400 と `invalid_request` で拒否し、登録トークンを使用済みにしない。
- **判断**：使用済み、期限切れ、無効化済みのトークンを同じ応答で拒否し、トークンを持たない呼び出し元に Connector の有無と状態を明かさない。

### Connector による接続の報告

#### REQ-SOURCING-010 Directory Connector の呼び出しは、登録した鍵で署名した短命の JWT だけを認証する

- `Active` の Connector が登録した鍵で署名した JWT を提示して接続を報告したとき、Sourcing は、Connector の最終接続時刻を記録し、204 を返す。
- Sourcing は、`iss` と `sub` が Connector の ID であり、`aud` が登録の応答の値であり、`alg` が `ES256` か `PS256` であり、`exp` が検証の時刻の 5 分後までにあり、`jti` が未使用である JWT だけを、時計のずれを前後 60 秒まで許して認証する。
- JWT を認証したとき、Sourcing は、`jti` を使用済みとして 6 分間記録する。
- JWT がないか、形式、署名、`aud`、`exp`、`alg` のいずれかが条件を満たさない場合、Sourcing は、401 と `invalid_token` で拒否し、最終接続時刻を変えない。
- 使用済みの `jti` を持つ JWT を提示された場合、Sourcing は、401 と `invalid_token` で拒否し、最終接続時刻を変えない。
- JWT の `iss` が要求したテナントの `Active` の Connector でない場合、Sourcing は、401 と `invalid_token` で拒否し、最終接続時刻を変えない。

### 管理者による登録トークンの再発行

#### REQ-SOURCING-011 登録トークンの再発行は、登録済みの鍵を破棄して Connector を `AwaitingEnrollment` に戻す

- **上位の要件**：[REQ-SOURCING-008](directory.md#管理者による-connector-の作成)
- `AwaitingEnrollment` か `Active` の Connector の登録トークンを管理者が再発行したとき、Sourcing は、Connector を `AwaitingEnrollment` にし、以前の登録トークンと公開鍵を破棄し、201 で Connector と新しい登録トークンを返し、`DirectoryConnectorEnrollmentReissued` を発行する。
- `Disabled` の Connector の再発行を要求された場合、Sourcing は、409 と `directory_connector_disabled` で拒否し、Connector を変えない。
- 存在しないか別のテナントの Connector の ID を指定された場合、Sourcing は、404 と `directory_connector_not_found` で拒否する。
- `admin` または `system_admin` のロールを持たない利用者が再発行を要求した場合、Sourcing は、403 と `access_denied` で拒否し、Connector を変えない。
- **判断**：鍵の更新と、漏えいが疑われる鍵の失効を、同じ一つの操作で行えるようにする。

### 管理者による Connector の無効化

#### REQ-SOURCING-012 Directory Connector の無効化は、以後の登録と呼び出しを拒否する

- `AwaitingEnrollment` か `Active` の Connector を管理者が無効化したとき、Sourcing は、Connector を `Disabled` にし、登録トークンのハッシュを消し、無効化時刻を記録し、200 で Connector を返し、`DirectoryConnectorDisabled` を発行する。
- `Disabled` の Connector を管理者が無効化したとき、Sourcing は、200 で Connector を返し、Connector を変えず、イベントを発行しない。
- 存在しないか別のテナントの Connector の ID を指定された場合、Sourcing は、404 と `directory_connector_not_found` で拒否する。
- `admin` または `system_admin` のロールを持たない利用者が無効化を要求した場合、Sourcing は、403 と `access_denied` で拒否し、Connector を変えない。

### Connector による取り込みの実行の開始

#### REQ-SOURCING-015 取り込みの実行の開始は、同じ Connector の進行中の実行を打ち切り、新しい実行を返す

- **上位の要件**：[REQ-SOURCING-010](directory.md#connector-による接続の報告)
- `Active` の Connector が `full` か `incremental` を指定して取り込みの実行を始めたとき、Sourcing は、`running` の実行を作り、201 で実行の ID、種類、状態、開始時刻を返す。
- Connector が取り込みの実行を始めたとき、Sourcing は、同じ Connector の `running` と `awaiting_approval` の実行を `superseded` にする。
- 同じ Connector の `applying` の実行があるか、開始、バッチ、完了、承認、破棄の処理が競合している場合、Sourcing は、409 と `ingestion_run_busy` で拒否し、要求の作用を行わない。
- 種類が `full` でも `incremental` でもない場合、Sourcing は、400 と `invalid_request` で拒否し、実行を作らない。
- **判断**：打ち切るのは、古い完全な再同期の完了が、新しい実行にだけ現れた User を無効にしないためである。

### Connector による User のバッチの送信

#### REQ-SOURCING-016 User のバッチは、Connector と外部の不変 ID で対応する User へ、作成、変更、無効化、削除の予約を反映する

- **上位の要件**：[REQ-SOURCING-010](directory.md#connector-による接続の報告)
- Connector が `running` の実行へ 1 件から 100 件の User の項目のバッチを送ったとき、Sourcing は、項目を送られた順に一件ずつ適用して確定し、200 でバッチの ID と、項目ごとの外部の不変 ID、結果、`failed` のエラーコードを返す。
- 対応のない外部の不変 ID の、削除でない項目を適用するとき、Sourcing は、IdManagement の作成を通し、項目の有効かどうかに合わせた `Active` か `Disabled` の User を作り、User と対応を一緒に確定し、`created` を返す。
- 対応のある外部の不変 ID の、削除でない項目を適用するとき、Sourcing は、項目にない任意の属性を空として User の属性を IdManagement の変更を通して置き換え、項目の有効かどうかに合わせて IdManagement の無効化か再有効化を通し、`updated` を返す。
- 対応のある User の属性と有効かどうかが項目と同じとき、Sourcing は、User を書かず、`unchanged` を返す。
- 対応のある外部の不変 ID の削除の項目を適用するとき、Sourcing は、IdManagement の削除の予約を通して User を `PendingDeletion` にし、`deleted` を返す。
- 対応のない外部の不変 ID の削除の項目か、対応の User が `PendingDeletion` か削除済みの削除の項目を適用するとき、Sourcing は、User を変えず、`unchanged` を返す。
- Sourcing が User を作成または変更するとき、Sourcing は、管理 API と同じ使用量の確認、作成または変更のイベント、ライフサイクルワークフローの捕捉と下流への通知を伴わせ、イベントの actor を `directory_connector` にする。
- 対応のある外部の不変 ID の項目を適用したとき、Sourcing は、結果が `failed` でもその対応にこの実行に現れた印を付ける。
- 対応のない既存の User か別の対応の User とユーザー名が衝突する項目の場合、Sourcing は、User を結び付けも作成も変更もせず、`failed` と `username_conflict` を返す。
- 同じテナントのほかの User とメールアドレスが衝突する項目の場合、Sourcing は、User を作成も変更もせず、`failed` と `email_conflict` を返す。
- 同じテナントのほかの User と `unique` の属性の値が衝突する項目の場合、Sourcing は、User を作成も変更もせず、`failed` と `attribute_conflict` を返す。
- 属性がテナントの属性スキーマに違反する項目の場合、Sourcing は、User を作成も変更もせず、`failed` と `invalid_attribute` を返す。
- User の Hard Quota を超える作成の項目の場合、Sourcing は、User と対応を作らず、使用量を変えず、`failed` と `quota_exceeded` を返す。
- 対応の User が `PendingDeletion` か削除済みである、削除でない項目の場合、Sourcing は、User を変えず、`failed` と `user_deleted` を返す。
- 作成で対応の保存に失敗した場合、Sourcing は、User を残さず、要求を失敗させる。
- JSON として読めない本文、空か 128 文字を超える `batch_id`、0 件か 101 件以上の項目、空か 256 文字を超えるかバッチの中で重複する外部の不変 ID、`preferred_username` のない削除でない項目のどれかを受けた場合、Sourcing は、400 と `invalid_request` で拒否し、どの項目も適用しない。
- 存在しないか、別の Connector か別のテナントの実行を指定された場合、Sourcing は、404 と `ingestion_run_not_found` で拒否する。
- **判断**：項目ごとに確定し、バッチを不可分にしない。途中で失敗したバッチは、同じバッチの再送が適用済みの項目を `unchanged` として回収する。
- **判断**：変わらない項目で User を書かないのは、完全な再同期で全件を送っても、変わらない User にイベントと下流への通知を出さないためである。

#### REQ-SOURCING-017 同じバッチの再送は、適用をやり直さずに最初の結果を返す

- **上位の要件**：[REQ-SOURCING-016](directory.md#connector-による-user-のバッチの送信)
- Sourcing がバッチのすべての項目を適用したとき、Sourcing は、バッチの ID、項目のダイジェスト、項目ごとの結果を実行に記録する。
- 実行に記録したバッチと同じ ID と同じ項目のバッチを受けたとき、Sourcing は、実行の状態を問わず、項目を適用し直さず、記録した結果を 200 で返す。
- 実行に記録したバッチと同じ ID で、項目の異なるバッチを受けた場合、Sourcing は、409 と `batch_conflict` で拒否し、どの項目も適用しない。
- `completed` か `superseded` の実行へ、記録していないバッチを送られた場合、Sourcing は、409 と `ingestion_run_closed` で拒否し、どの項目も適用しない。

### Connector による取り込みの実行の完了

#### REQ-SOURCING-018 実行の完了は、完全な再同期に現れなかった対応済みの User を無効にする

- **上位の要件**：[REQ-SOURCING-010](directory.md#connector-による接続の報告)
- Connector が、`running` の実行の完了を、記録したバッチの項目数の和と同じ項目数で報告したとき、Sourcing は、以下のしきい値判定を行い、200 で実行を返す。
- 実行を `completed` にしたとき、Sourcing は、完了時刻を記録する。
- `full` の完了で、同じ Connector の対応を持つ `Active` の User のうち、この実行に現れなかった候補の件数が `accidental_deletion_count_threshold` を超えるか、その Active の User の総数に対する割合が `accidental_deletion_percent_threshold` を超える場合、Sourcing は、実行を `awaiting_approval` にし、候補、総数、判定したしきい値と保留時刻を保存し、候補の User を変更せず、200 で保留状態の実行を返し、`DirectoryIngestionRunHeld` を発行する。
- 候補が 0 件か両方のしきい値以下の場合、Sourcing は、候補だけを IdManagement の無効化を通して `Disabled` にし、無効にした件数を実行に記録して `completed` にする。
- `awaiting_approval` の完了を再送されたとき、Sourcing は、候補と User とイベントを変えずに、200 で保留状態の実行を返す。
- `awaiting_approval` の実行では、Sourcing は、完了時刻を記録せず、後から送られた新しいバッチを 409 と `ingestion_run_closed` で拒否する。
- `incremental` の実行を完了したとき、Sourcing は、どの User も無効にしない。
- `completed` の実行の完了を報告されたとき、Sourcing は、実行も User も変えず、200 で実行を返す。
- 報告した項目数が、記録したバッチの項目数の和と一致しない場合、Sourcing は、409 と `ingestion_run_incomplete` で拒否し、実行も User も変えない。
- `superseded`、`applying`、`discarded` の実行の完了を報告された場合、Sourcing は、409 と `ingestion_run_closed` で拒否し、実行も User も変えない。
- 存在しないか、別の Connector か別のテナントの実行を指定された場合、Sourcing は、404 と `ingestion_run_not_found` で拒否する。
- **判断**：項目数を照合するのは、取りこぼしたバッチの User を、現れなかったものとして無効にしないためである。
- **判断**：削除ではなく無効にとどめるのは、取り込み元の検索範囲の設定の誤りで多数の User が現れなかったとき、取り込み元を直した次の同期で回復できるようにするためである。

### 管理者による取り込みの実行の照会

#### REQ-SOURCING-019 管理者は Connector の取り込みの実行の件数と失敗を照会できる

- 管理者が Connector の取り込みの実行を一覧したとき、Sourcing は、開始時刻の新しい順に最大 50 件の実行を、ID、種類、状態、開始と完了の時刻、バッチの数、項目の数、結果ごとの項目の数、無効にした User の数とともに返す。
- 管理者が取り込みの実行を取得したとき、Sourcing は、一覧と同じ項目に加え、`failed` の項目の外部の不変 ID、エラーコード、バッチの ID を、バッチを記録した順に最大 100 件返す。
- 保留を記録した実行を照会したとき、Sourcing は、候補件数、総数、判定時のしきい値、保留時刻、承認者、承認時刻、破棄時刻を返す。
- 管理者が候補を照会したとき、Sourcing は、その実行に保存した候補の User ID と外部の不変 ID を、保存した順に、0 以上の `offset` と 1 以上 100 以下の `limit`（省略時 100）で返し、総件数を返す。
- 不正なページ指定を受けた場合、Sourcing は、400 と `invalid_request` で拒否する。
- 存在しないか別のテナントの Connector の ID を指定された場合、Sourcing は、404 と `directory_connector_not_found` で拒否する。
- 存在しないか別の Connector の実行の ID を指定された場合、Sourcing は、404 と `ingestion_run_not_found` で拒否する。
- `admin` または `system_admin` のロールを持たない利用者が照会を要求した場合、Sourcing は、403 と `access_denied` で拒否する。

### 管理者による無効化のしきい値の更新

#### REQ-SOURCING-020 管理者は Connector ごとに件数と割合のしきい値を変更できる

- Connector を作成したとき、Sourcing は、`accidental_deletion_count_threshold` を 500、`accidental_deletion_percent_threshold` を 20 にする。
- 管理者が Connector を取得または一覧したとき、Sourcing は、両方の設定値を返す。
- 管理者が両方のしきい値を更新したとき、Sourcing は、Connector の設定を保存して 200 で Connector を返し、`DirectoryConnectorThresholdsUpdated` を発行する。
- 件数が 0 以上 2,147,483,647 以下の整数でないか、割合が 1 以上 100 以下の整数でない場合、Sourcing は、400 と `invalid_request` で拒否し、設定を変えない。
- 設定を変更したとき、Sourcing は、保存済みの保留とその候補を変えず、自動で承認しない。
- 存在しないか別テナントの Connector を指定された場合、Sourcing は、404 と `directory_connector_not_found` で拒否する。
- 管理者でない要求の場合、Sourcing は、403 と `access_denied` で拒否する。

### 管理者による保留した無効化の承認

#### REQ-SOURCING-021 承認は保存した候補だけを無効化する

- 管理者が `awaiting_approval` の実行を承認したとき、Sourcing は、承認者と承認時刻を保存して `applying` にし、`DirectoryIngestionRunApproved` を発行し、保存した候補だけを IdManagement の無効化操作へ渡し、全候補の処理後に `completed` と完了時刻を記録して 200 で実行を返す。
- 承認時に候補の User が不在、別テナント、対応を失っている、または `Active` でない場合、Sourcing は、その候補を無効化せずに処理済みにする。
- 適用が失敗した場合、Sourcing は、`applying` と候補を保持し、承認の再送で未処理の候補から再開する。
- `completed` の承認を再送されたとき、Sourcing は、User と実行を変えず 200 で実行を返す。
- `running`、`superseded`、`discarded` の承認の場合、Sourcing は、409 と `ingestion_run_closed` で拒否し、User を変えない。
- 操作が競合している場合、Sourcing は、409 と `ingestion_run_busy` で拒否し、User を変えない。
- 存在しないか別テナントの Connector を指定された場合、Sourcing は、404 と `directory_connector_not_found` で拒否する。
- 別 Connector か存在しない実行を指定された場合、Sourcing は、404 と `ingestion_run_not_found` で拒否する。
- 管理者でない要求の場合、Sourcing は、403 と `access_denied` で拒否する。

### 管理者による保留した無効化の破棄

#### REQ-SOURCING-022 破棄は候補の User を変更しない

- 管理者が `awaiting_approval` の実行を破棄したとき、Sourcing は、実行を `discarded` にし、破棄時刻を保存して 200 で実行を返し、`DirectoryIngestionRunDiscarded` を発行し、User を変更しない。
- `discarded` の破棄を再送されたとき、Sourcing は、実行と User とイベントを変えずに 200 で実行を返す。
- その他の状態では、Sourcing は、409 と `ingestion_run_closed` で拒否し、実行と User を変えない。
- 操作が競合している場合、Sourcing は、409 と `ingestion_run_busy` で拒否し、実行と User を変えない。
- 対象の不在か権限不足の場合、Sourcing は、承認と同じ条件で拒否する。

## セキュリティ上の考慮

Connector の管理 API は、`admin` または `system_admin` のロールを持つ、有効かつ認証済みの利用者の対話のセッションに限り、API アクセストークンでは到達できない。
作成と再発行は Connector の資格情報の元になるシークレットを返すので、API アクセストークンの発行と同じ扱いにする。

Connector の呼び出しのテナントは、要求の経路が解決したテナントで決まる。
JWT の `iss` は、そのテナントの Connector の ID として引き、別のテナントの Connector は存在しないものとして扱う。
`aud` を発行者識別子そのものにしないので、Connector の JWT は、同じ発行者の別の経路の受け手に通用しない。

IdMagic が保存するのは、登録トークンのハッシュと公開鍵だけである。
Connector の秘密鍵、AD の資格情報、利用者のパスワードは、IdMagic のデータベース、API の応答、イベントのどれにも含まれない。

取り込みの実行と対応は、それを作った Connector に属する。
Connector は、別の Connector の実行へバッチを送れず、別の Connector の対応の User を変えられない。
外部の不変 ID が同じでも Connector が違えば別の対応であり、別の Connector の User にはユーザー名の衝突として失敗する。

## 実現方式

### 無効化の保留と再開

しきい値はテナント一括ではなく、同期設定である Connector に保存する。
件数と割合の項目名、厳密な超過と論理和の比較は下流の Provisioning の設定とそろえる。
今回の保護は完全再同期の不在を理由とする無効化に適用し、バッチの明示的な無効化と削除、SCIM、下流の既存ガードの適用範囲は変えない。

同じ Connector の開始、バッチ、完了、承認、破棄は PostgreSQL のトランザクション単位の advisory lock で排他する。
排他用のトランザクションは作用の実行中保持し、競合は待たずに拒否する。
User のライフサイクル操作は既存の所有者が確定するため、この排他は全 User と実行の変更を一つのトランザクションにはしない。
メモリ構成では同じ単位の mutex で排他する。

保留では判定時の候補としきい値を保存し、承認では候補を再計算しない。
`applying` は新しい同期による失効を防ぎ、候補ごとの進捗によって承認の再送で再開できる。
無効化を試みた印を先に記録し、User の保存後に作用が失敗した場合も、再送は所有者の無効化操作へ戻る。
部分適用と無効化件数の制限は [Sourcing のリスク](../risks.md) が扱う。

保留の通知は監査イベントを使い、監査画面と実行の照会で確認できる。
メール通知は今回見送るが、恒久的な対象外にはせず、メール通知全般の追加、拡充時には対応対象に含める。
共有の通知基盤は既にあり、後続の対応では通知先の設定とテンプレートをその基盤へ接続する。
