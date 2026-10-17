# Feature: グループ CSV の例

## Rule: REQ-IDMANAGEMENT-026 Group の CSV のインポートは、プレビューで全行を判定し、適用で有効な行だけを保存する

### Background:

- Given ロール=["admin"] のユーザー "operator" が管理画面のグループ一覧を開いている
- And テナントの Group 属性スキーマに "cost_center" (string) が定義されている

### Example: EX-IDMANAGEMENT-026-01 通常経路

- When 管理者が機械可読なヘッダー [id, name, email, roles, membership_type, custom:cost_center] を任意の順で含む CSV を事前検証へ投入する
- Then プレビュージョブは `created`、`updated`、`unchanged`、`deleted`、`rejected` の判定、行番号、安定したエラーコードを返し、`Group` は変更されない
- When 管理者が同じテナントの成功済みプレビュージョブの ID を指定して適用を開始する
- Then CSV は再送されず、保存済みのプレビューペイロードが使われる
- Then 適用はプレビューペイロードと SHA-256 を検証し、現在の Repository の状態に対して同じ計画器で再計画する
- Then 有効な行は作成または更新され、無効な行は `rejected` として残る。各行の名前、説明、連絡先、ロール、カスタム属性、動的規則は不可分に保存される

### Scenario Outline: ファイル全体の投入の拒否

- When 管理者が機械可読なヘッダー [id, name, email, roles, membership_type, custom:cost_center] を任意の順で含む CSV を事前検証へ投入する
- But <condition>
- Then インポートの投入は拒否される
- And エラー <error>

#### Examples:

  | example_id | condition | error |
  | --- | --- | --- |
  | EX-IDMANAGEMENT-026-02 | CSV が実効 `CsvTransferPolicy` の `max_bytes`、`max_rows`、`max_field_bytes` のいずれかを超える | "csv_too_large" / "too_many_rows" / "field_too_large" |
  | EX-IDMANAGEMENT-026-03 | CSV のヘッダーに未知の列、重複した列、`password` または `password_hash` が含まれる | "invalid_header" |

### Example: EX-IDMANAGEMENT-026-04 行に `id` も `name` も無い、`id` と `name` が別の `Group` を示す、同じ対象または同じ最終 `name` を複数行が示す

- When 管理者が機械可読なヘッダー [id, name, email, roles, membership_type, custom:cost_center] を任意の順で含む CSV を事前検証へ投入する
- Then 行に `id` も `name` も無い、`id` と `name` が別の `Group` を示す、同じ対象または同じ最終 `name` を複数行が示す
- Then 対象行は `rejected` となり、安定したエラーコードを返す

### Example: EX-IDMANAGEMENT-026-05 プレビュージョブが存在しない、`queued` または `failed` である、別テナントに属する、保存済みのペイロードとダイジェストが一致しない

- When 管理者が機械可読なヘッダー [id, name, email, roles, membership_type, custom:cost_center] を任意の順で含む CSV を事前検証へ投入する
- Then プレビュージョブは `created`、`updated`、`unchanged`、`deleted`、`rejected` の判定、行番号、安定したエラーコードを返し、`Group` は変更されない
- When 管理者が同じテナントの成功済みプレビュージョブの ID を指定して適用を開始する
- But プレビュージョブが存在しない、`queued` または `failed` である、別テナントに属する、保存済みのペイロードとダイジェストが一致しない
- Then 適用は `Group` を変更せず、存在しないジョブと別テナントのジョブは 404 と `group_import_not_found`、`queued` または `failed` のジョブは 409 と `preview_not_ready`、ダイジェストが一致しないジョブは 409 と `preview_digest_mismatch` で拒否される

### Example: EX-IDMANAGEMENT-026-06 プレビュー後に対象 `Group` の状態が別の操作で変更されている

- When 管理者が機械可読なヘッダー [id, name, email, roles, membership_type, custom:cost_center] を任意の順で含む CSV を事前検証へ投入する
- Then プレビュージョブは `created`、`updated`、`unchanged`、`deleted`、`rejected` の判定、行番号、安定したエラーコードを返し、`Group` は変更されない
- When 管理者が同じテナントの成功済みプレビュージョブの ID を指定して適用を開始する
- Then CSV は再送されず、保存済みのプレビューペイロードが使われる
- Then プレビュー後に対象 `Group` の状態が別の操作で変更されている
- Then 適用は古いプレビュー計画を実行せず、現在状態から `updated`、`unchanged`、`rejected` を再判定する

### Scenario Outline: 適用の再計画で行を拒否する

- When 管理者が機械可読なヘッダー [id, name, email, roles, membership_type, custom:cost_center] を任意の順で含む CSV を事前検証へ投入する
- Then プレビュージョブは `created`、`updated`、`unchanged`、`deleted`、`rejected` の判定、行番号、安定したエラーコードを返し、`Group` は変更されない
- When 管理者が同じテナントの成功済みプレビュージョブの ID を指定して適用を開始する
- Then CSV は再送されず、保存済みのプレビューペイロードが使われる
- Then 適用はプレビューペイロードと SHA-256 を検証し、現在の Repository の状態に対して同じ計画器で再計画する
- Then <condition>
- Then <result>

#### Examples:

  | example_id | condition | result |
  | --- | --- | --- |
  | EX-IDMANAGEMENT-026-07 | 既存 `Group` の `membership_type` を現在値と異なる値へ変更する | 対象行は安定したエラーコード `immutable_membership_type` で `rejected` となり、その `Group` の `membership_type`、ロール、メンバーシップはいずれも変更されない |
  | EX-IDMANAGEMENT-026-08 | 対象 `Group` が外部の取り込み元に管理されている、または所有権を判定できない | 対象行は安定したエラーコード `source_managed` で `rejected` となり、`Group` は変更されない |
  | EX-IDMANAGEMENT-026-09 | `membership_type=manual` の `Group` に動的規則の式を与える、最終状態の式が空のまま規則を有効化する、または式が未定義の属性か許可外の関数を参照する | 対象行は安定したエラーコード `invalid_dynamic_rule` で `rejected` となり、その `Group` の動的規則は保存されない |
  | EX-IDMANAGEMENT-026-10 | `email` がメールアドレスの形式を満たさない、`custom:<key>` の値が宣言された型の字句形に合わない、またはテナントスキーマに無い `custom:<key>` 列を与える | 対象行は安定したエラーコードで `rejected` となり、その `Group` の連絡先も属性も変更されず、値はジョブの表示にも監査イベントにも含めない |
  | EX-IDMANAGEMENT-026-11 | 1 行の検証、保存、監査処理が途中で失敗する | その行の名前、説明、連絡先、ロール、カスタム属性、動的規則は一部も保存されず、他の有効な行は適用を続ける |

## Rule: REQ-IDMANAGEMENT-072 Group の CSV は、名前を値オブジェクトの定義で照合し、連絡先をアドレスだけで受け付ける

### Example: EX-IDMANAGEMENT-072-01 大文字の名前による照合

- Given テナントに名前 "engineering" の Group がある
- When 管理者が `name` に "ENGINEERING" だけを書いた行を事前検証へ投入する
- Then 行は既存の "engineering" を対象とする

### Example: EX-IDMANAGEMENT-072-02 表示名付きの連絡先

- When 管理者が `email` のセルに "Sales <sales@example.test>" を書いて事前検証へ投入する
- Then 行は `invalid_email` で `rejected` となる

## Rule: REQ-IDMANAGEMENT-028 Group の CSV のインポートは、`lifecycle_action=delete` の行の Group だけを削除する

### Background:

- Given グループ "engineering" が存在し、ユーザー "alice" が所属している
- And 管理者はエクスポートした CSV の "engineering" の行だけに `lifecycle_action=delete` を書き込んでいる

### Example: EX-IDMANAGEMENT-028-01 通常経路

- When 管理者がその CSV を事前検証へ投入する
- Then プレビューは削除される `Group` の件数と、巻き込まれるメンバーシップの件数を他の操作と分けて返し、`Group` もメンバーシップも変更されない
- Then CSV に現れない `Group` は削除対象に含まれない
- When 管理者が削除を含むことを確認したうえで適用する
- Then "engineering" は削除され、"alice" のメンバーシップは同じトランザクションで解除され、"GroupMemberRemoved" と "GroupDeleted" が発行される
- Then "alice" の実効ロールから "engineering" のロールが外れる
- Then 同じファイルの他の行の作成と更新は削除の影響を受けずに適用される

### Scenario Outline: 削除の行を拒否する

- When 管理者がその CSV を事前検証へ投入する
- But <condition>
- Then 対象行は安定したエラーコード <error_code> で `rejected` となり、<effect>

#### Examples:

  | example_id | condition | error_code | effect |
  | --- | --- | --- | --- |
  | EX-IDMANAGEMENT-028-02 | `lifecycle_action` に `delete` 以外の値を書く | `invalid_lifecycle_action` | 既知の値へ丸めない |
  | EX-IDMANAGEMENT-028-03 | `lifecycle_action=delete` の行が、現在状態に対して差分のある書き込み可能な列も同時に持つ | `conflicting_lifecycle_action` | その `Group` は更新も削除もされない |
  | EX-IDMANAGEMENT-028-04 | `lifecycle_action=delete` の行が既存の `Group` を指さない | `target_not_found` | `Group` は作成されない |
  | EX-IDMANAGEMENT-028-05 | 対象 `Group` が外部の取り込み元に管理されている、または所有権を判定できない | `source_managed` | `Group` は削除されない |

## Rule: REQ-IDMANAGEMENT-029 メンバーシップの CSV のインポートは、行ごとに宣言した所属の状態だけを適用する

### Background:

- Given ロール=["admin"] のユーザー "operator" が手動グループ "engineering" の詳細画面を開いている
- And "engineering" にはユーザー "alice" が所属し、"bob" は所属していない

### Example: EX-IDMANAGEMENT-029-01 通常経路

- When 管理者が機械可読なヘッダー [user_id, preferred_username, membership_state] を任意の順で含む CSV を、そのグループの事前検証へ投入する
- Then プレビュージョブは `added`、`removed`、`unchanged`、`rejected` の判定、行番号、安定したエラーコードを返し、メンバーシップは変更されない
- Then 解除される件数は他の操作と分けて返される
- When 管理者が解除を含むことを確認したうえで、同じテナントかつ同じグループの成功済みプレビュージョブの ID を指定して適用する
- Then CSV は再送されず、保存済みのプレビューペイロードとその SHA-256 が検証される
- Then `membership_state=present` の "bob" の行はメンバーシップを追加し、"GroupMemberAdded" を発行する
- Then `membership_state=absent` の "alice" の行は手動メンバーシップを解除し、"GroupMemberRemoved" を発行する
- Then "alice" の実効ロールから "engineering" のロールが外れ、"bob" の実効ロールにそれが加わる

### Scenario Outline: ファイル全体の投入の拒否

- When 管理者が機械可読なヘッダー [user_id, preferred_username, membership_state] を任意の順で含む CSV を、そのグループの事前検証へ投入する
- But <condition>
- Then インポートの投入は拒否される
- And エラー <error>

#### Examples:

  | example_id | condition | error |
  | --- | --- | --- |
  | EX-IDMANAGEMENT-029-02 | CSV が実効 `CsvTransferPolicy` の `max_bytes`、`max_rows`、`max_field_bytes` のいずれかを超える | "csv_too_large" / "too_many_rows" / "field_too_large" |
  | EX-IDMANAGEMENT-029-03 | CSV のヘッダーに未知の列、重複した列、`password` または `password_hash` が含まれる | "invalid_header" |

### Example: EX-IDMANAGEMENT-029-04 CSV のヘッダーに `membership_state` が無い

- When 管理者が機械可読なヘッダー [user_id, preferred_username, membership_state] を任意の順で含む CSV を、そのグループの事前検証へ投入する
- But CSV のヘッダーに `membership_state` が無い
- Then ファイル全体が安定したエラーコード "invalid_header" で拒否され、メンバーシップは 1 件も追加も解除もされない

### Scenario Outline: 行の拒否

- When 管理者が機械可読なヘッダー [user_id, preferred_username, membership_state] を任意の順で含む CSV を、そのグループの事前検証へ投入する
- But <condition>
- Then 対象行は安定したエラーコード <error_code> で `rejected` となり、<effect>

#### Examples:

  | example_id | condition | error_code | effect |
  | --- | --- | --- | --- |
  | EX-IDMANAGEMENT-029-05 | `membership_state` のセルが空である、または `present` と `absent` のどちらでもない | `invalid_membership_state` | 既知の値へ丸めず、その User のメンバーシップは変更されない |
  | EX-IDMANAGEMENT-029-06 | 行に `user_id` も `preferred_username` も無い、両者が別の User を示す、対象 User がテナントに存在しない、同じ User を複数行が示す | `missing_identifier` / `identifier_mismatch` / `target_not_found` / `duplicate_target` | メンバーシップは変更されない |
  | EX-IDMANAGEMENT-029-07 | 行の `group_id` または `group_name` がパスのグループ以外を示す | `group_mismatch` | 示された別のグループのメンバーシップも変更されない |

### Example: EX-IDMANAGEMENT-029-08 プレビュージョブが存在しない、`queued` または `failed` である、別テナントまたは別グループに属する、保存済みのペイロードとダイジェストが一致しない

- When 管理者が機械可読なヘッダー [user_id, preferred_username, membership_state] を任意の順で含む CSV を、そのグループの事前検証へ投入する
- Then プレビュージョブは `added`、`removed`、`unchanged`、`rejected` の判定、行番号、安定したエラーコードを返し、メンバーシップは変更されない
- When 管理者が解除を含むことを確認したうえで、同じテナントかつ同じグループの成功済みプレビュージョブの ID を指定して適用する
- But プレビュージョブが存在しない、`queued` または `failed` である、別テナントまたは別グループに属する、保存済みのペイロードとダイジェストが一致しない
- Then 適用はメンバーシップを変更せず、存在しないジョブと別テナントまたは別グループのジョブは 404 と `group_membership_import_not_found`、`queued` または `failed` のジョブは 409 と `preview_not_ready`、ダイジェストが一致しないジョブは 409 と `preview_digest_mismatch` で拒否される

### Scenario Outline: 適用中の状態変化と部分的な失敗

- When 管理者が機械可読なヘッダー [user_id, preferred_username, membership_state] を任意の順で含む CSV を、そのグループの事前検証へ投入する
- Then プレビュージョブは `added`、`removed`、`unchanged`、`rejected` の判定、行番号、安定したエラーコードを返し、メンバーシップは変更されない
- When 管理者が解除を含むことを確認したうえで、同じテナントかつ同じグループの成功済みプレビュージョブの ID を指定して適用する
- Then <condition>
- Then <result>

#### Examples:

  | example_id | condition | result |
  | --- | --- | --- |
  | EX-IDMANAGEMENT-029-09 | プレビュー後にメンバーシップが別の操作で変更されている | 適用は古いプレビュー計画を実行せず、現在のメンバーシップから `added`、`removed`、`unchanged`、`rejected` を再判定する |
  | EX-IDMANAGEMENT-029-10 | 1 行のメンバーシップ確定または監査記録が途中で失敗する | その行のメンバーシップも監査記録も一部すら残らず、他の有効な行は適用を続ける |

## Rule: REQ-IDMANAGEMENT-031 メンバーシップ CSV は動的規則と外部の取り込み元が所有する所属を上書きしない

### Background:

- Given ロール=["admin"] のユーザー "operator" がメンバーシップ CSV を用意している

### Scenario Outline: 所有者の異なる所属を上書きしない

- When 管理者がそのグループの事前検証へ CSV を投入する
- But <condition>
- Then <result>

#### Examples:

  | example_id | condition | result |
  | --- | --- | --- |
  | EX-IDMANAGEMENT-031-01 | 対象グループの `membership_type` が `dynamic` である | ファイル全体が安定したエラーコード `dynamic_group` で拒否され、メンバーシップは 1 件も追加も解除もされない |
  | EX-IDMANAGEMENT-031-02 | 対象 User の現在のメンバーシップの `source` が `dynamic_rule` である | 対象行は安定したエラーコード `dynamic_membership` で `rejected` となり、`present` でも `absent` でもそのメンバーシップは変更されない |
  | EX-IDMANAGEMENT-031-03 | 対象グループが外部の取り込み元に管理されている、または所有権を判定できない | ファイル全体が安定したエラーコード `source_managed` で拒否され、メンバーシップは 1 件も追加も解除もされない |
  | EX-IDMANAGEMENT-031-04 | 対象 User が外部の取り込み元に管理されている、または所有権を判定できない | 対象行は安定したエラーコード `source_managed` で `rejected` となり、その User のメンバーシップは変更されない |
  | EX-IDMANAGEMENT-031-05 | 対象グループがテナントに存在しない、または適用の直前に削除されている | ファイル全体が安定したエラーコード `target_not_found` で拒否され、グループは作成されず、メンバーシップも作られない |

## Rule: REQ-IDMANAGEMENT-027 Group のエクスポートは、そのまま再インポートすると変化なしになり、書き込める列の編集だけを反映する

### Background:

- Given 実効 `TenantGroupAttributeSchema` に `custom:cost_center` があり、10,000 件の `Group` を含む一覧が実効 `CsvTransferPolicy` の上限内に収まる

### Example: EX-IDMANAGEMENT-027-01 通常経路

- When 管理者がインポート可能な組み込み列と `custom:cost_center` を機械可読ヘッダーでエクスポートする
- Then `worker` プロセスは CSV を不変の成果物ストアへストリーミング出力し、ジョブ結果にはテナント単位のペイロード参照、サーバーが算出した SHA-256、サイズ、行数を保持する
- Then `lifecycle_action` 列は全行で空として出力される
- When 管理者が同じ 10,000 行の成果物を編集せずプレビューする
- Then 全行が `unchanged` となり、`Group` は 1 件も作成、更新、削除されない
- When 管理者が 1 行の `roles`、`email`、`custom:cost_center` だけを編集して再びプレビューする
- Then 変更行だけが `updated`、残りは `unchanged` と計画される
- When 管理者が成功済みプレビュージョブの ID を指定して適用する
- Then 指定した書き込み可能な列だけが更新され、指定しなかった列は維持される

### Example: EX-IDMANAGEMENT-027-02 値が危険な先頭文字、既存のアポストロフィー、カンマ、引用符、改行を含む

- When 管理者がインポート可能な組み込み列と `custom:cost_center` を機械可読ヘッダーでエクスポートする
- But 値が危険な先頭文字、既存のアポストロフィー、カンマ、引用符、改行を含む
- Then 可逆な数式安全変換と RFC 4180 の引用により `decode(encode(value))` は元の値と一致する

### Example: EX-IDMANAGEMENT-027-03 生成結果が実効 `CsvTransferPolicy` のいずれかの上限を超える

- When 管理者がインポート可能な組み込み列と `custom:cost_center` を機械可読ヘッダーでエクスポートする
- But 生成結果が実効 `CsvTransferPolicy` のいずれかの上限を超える
- Then `Group` エクスポートは `csv_transfer_limit_exceeded` で失敗し、再インポートできない成功済み成果物を作らない
- And 管理者は列を絞って複数の成果物に分割できる

### Scenario Outline: 編集した成果物の再プレビュー

- When 管理者がインポート可能な組み込み列と `custom:cost_center` を機械可読ヘッダーでエクスポートする
- Then `worker` プロセスは CSV を不変の成果物ストアへストリーミング出力し、ジョブ結果にはテナント単位のペイロード参照、サーバーが算出した SHA-256、サイズ、行数を保持する
- Then `lifecycle_action` 列は全行で空として出力される
- When 管理者が同じ 10,000 行の成果物を編集せずプレビューする
- Then 全行が `unchanged` となり、`Group` は 1 件も作成、更新、削除されない
- When 管理者が 1 行の `roles`、`email`、`custom:cost_center` だけを編集して再びプレビューする
- But <condition>
- Then <result>

#### Examples:

  | example_id | condition | result |
  | --- | --- | --- |
  | EX-IDMANAGEMENT-027-04 | `id`、`created_at`、`updated_at` の値だけを編集する | 読み取り専用列は受理したうえで無視し、書き込み可能な列に差分がなければ `unchanged` とする |
  | EX-IDMANAGEMENT-027-05 | `roles`、`membership_type`、`email`、または `custom:<key>` の値が不正である | 対象行は安定したエラーコードで `rejected` となり、値はジョブの表示にも監査イベントにも含めない |

## Rule: REQ-IDMANAGEMENT-008 メンバーシップのエクスポートは、パスの Group のメンバーだけを書き出す

### Background:

- Given ロール=["admin"] のユーザー "operator" がグループ "engineering" の詳細を開いている

### Example: EX-IDMANAGEMENT-008-01 通常経路

- When 管理者が `/groups/{group_id}/members/exports` へ列 [user_id, preferred_username] を指定してエクスポートを開始する
- Then エクスポートの対象は `group_id` に閉じ、そのグループのメンバーだけを含む
- When 生成完了後、管理者がメンバーの CSV をダウンロードする
- Then 指定したグループのメンバーだけを含む CSV が返る

### Example: EX-IDMANAGEMENT-008-02 `group_id` を指定しない

- When 管理者が `/groups/{group_id}/members/exports` へ列 [user_id, preferred_username] を指定してエクスポートを開始する
- But `group_id` を指定しない
- Then グループ単位の指定は必須であり、エクスポートの開始は InvalidRequestError で拒否される

### Example: EX-IDMANAGEMENT-008-03 別グループのパスでそのエクスポート ID を指定する

- When 管理者が `/groups/{group_id}/members/exports` へ列 [user_id, preferred_username] を指定してエクスポートを開始する
- Then エクスポートの対象は `group_id` に閉じ、そのグループのメンバーだけを含む
- When 生成完了後、管理者がメンバーの CSV をダウンロードする
- But 別グループのパスでそのエクスポート ID を指定する
- Then グループごとに分離しているため、取得とダウンロードは 404 と `data_export_not_found` で拒否される

## Rule: REQ-IDMANAGEMENT-030 メンバーシップのエクスポートは、分けて再適用しても、ほかのファイルにしかないメンバーを外さない

### Background:

- Given 手動グループ "engineering" が 10,000 件の手動メンバーシップを持ち、実効 `CsvTransferPolicy` の上限内に収まる

### Example: EX-IDMANAGEMENT-030-01 通常経路

- When 管理者がインポート可能な組み込み列を機械可読ヘッダーでエクスポートする
- Then `worker` プロセスは CSV を不変の成果物ストアへストリーミング出力し、ジョブ結果にはテナント単位のペイロード参照、サーバーが算出した SHA-256、サイズ、行数を保持する
- Then `membership_state` 列は全行で `present` として出力される
- When 管理者が同じ 10,000 行の成果物を編集せずプレビューする
- Then 全行が `unchanged` となり、メンバーシップは 1 件も追加も解除もされない
- When 管理者がその成果物を 2 つのファイルへ分け、片方だけをプレビューして適用する
- Then もう一方のファイルにしか現れないメンバーシップは解除されない
- Then CSV に現れない User は、そのグループの所属についても一切変更されない

### Scenario Outline: 成果物の書き出しの境界

- When 管理者がインポート可能な組み込み列を機械可読ヘッダーでエクスポートする
- But <condition>
- Then <result>

#### Examples:

  | example_id | condition | result |
  | --- | --- | --- |
  | EX-IDMANAGEMENT-030-02 | 値が危険な先頭文字、既存のアポストロフィー、カンマ、引用符、改行を含む | 可逆な数式安全変換と RFC 4180 の引用により `decode(encode(value))` は元の値と一致する |
  | EX-IDMANAGEMENT-030-03 | 生成結果が実効 `CsvTransferPolicy` のいずれかの上限を超える | メンバーシップのエクスポートは `csv_transfer_limit_exceeded` で失敗し、再インポートできない成功済み成果物を作らない |

### Example: EX-IDMANAGEMENT-030-04 `source`、`created_at`、`group_name` の値だけを編集する

- When 管理者がインポート可能な組み込み列を機械可読ヘッダーでエクスポートする
- Then `membership_state` 列は全行で `present` として出力される
- When 管理者が同じ 10,000 行の成果物を編集せずプレビューする
- But `source`、`created_at`、`group_name` の値だけを編集する
- Then 読み取り専用列は受理したうえで無視し、`group_name` が別のグループを指す場合にだけ行を拒否する
