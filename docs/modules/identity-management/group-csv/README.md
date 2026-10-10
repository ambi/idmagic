# グループ CSV

## 概要

この文書は、管理者が Group と、一つの Group のメンバーシップを、CSV で一括して変更し、CSV へエクスポートする機能の仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | Group の CSV のインポートとエクスポート、CSV の行による Group の削除、一つの Group のメンバーシップの CSV のインポートとエクスポート |
| 行為者 | 管理者、インポートとエクスポートのジョブ |
| コードの機能スライス | `backend/idmanagement/group` |
| 扱わないもの | ファイルとセルの共通の読み書きは[CSV の転送](../csv-transfer/README.md)が、エクスポートのジョブの開始、一覧、取り消し、保持期限は[データエクスポート](../data-export/README.md)が扱う |

## モデル

| CSV | 対象 | 意図を表す列 |
| --- | --- | --- |
| Group の CSV | テナントの Group | 書き込める列の値（目標の状態）と、削除だけを表す書き込み専用の列 `lifecycle_action` |
| メンバーシップの CSV | URL の `group_id` が示す一つの Group | 行ごとの `membership_state`（`present` または `absent`） |

どちらの CSV も、ファイルに現れない Group とメンバーシップを変えない。
意図は行ごとに宣言したものだけを実行する。

- **判断**：[CSV に現れない Group とメンバーシップを変えない](../design/decisions.md#csv-に現れない-group-とメンバーシップを変えない)。

## 操作

### Group のインポート

#### REQ-IDMANAGEMENT-026 Group の CSV のインポートは、プレビューで全行を判定し、適用で有効な行だけを保存する

- 管理者が機械可読な列名の見出しを任意の順で持つ CSV をプレビューに投入したとき、IdManagement は、行ごとの `created`、`updated`、`unchanged`、`deleted`、`rejected` の判定、行番号、安定したエラーコードを返すプレビューのジョブを作り、Group を変えない。
- 管理者が同じテナントの成功したプレビューのジョブの ID を指定して適用を始めたとき、IdManagement は、CSV を受け取り直さず、保存したプレビューのペイロードを SHA-256 で検証し、現在の状態に対して同じ計画器で計画し直す。
- プレビューの後に対象の Group が変わっていた場合、IdManagement は、適用の時点の状態から `updated`、`unchanged`、`rejected` を判定し直す。
- 管理者が Group の CSV を適用したとき、IdManagement は、有効な行の Group を作成または更新し、無効な行を `rejected` として残す。
- 管理者が Group の CSV を適用したとき、IdManagement は、各行の名前、説明、連絡先、ロール、カスタム属性、動的規則を不可分に保存する。
- 一行の検証、保存、監査の途中で失敗した場合、IdManagement は、その行を一部も保存せず、ほかの有効な行の適用を続ける。
- 実効の転送ポリシーの `max_bytes`、`max_rows`、`max_field_bytes` のどれかを超える CSV を投入された場合、IdManagement は、`csv_too_large`、`too_many_rows`、`field_too_large` で拒否する。
- 見出しに未知の列、重複した列、`password` または `password_hash` を含む CSV を投入された場合、IdManagement は、`invalid_header` で拒否する。
- `id` も `name` もない行、`id` と `name` が別の Group を示す行、同じ対象または同じ最終の `name` を示す複数の行を読んだ場合、IdManagement は、その行を安定したエラーコードで `rejected` にする。
- 存在しないプレビューのジョブ、または別のテナントのプレビューのジョブを指定した適用とジョブの参照を要求された場合、IdManagement は、404 と `group_import_not_found` で拒否し、Group を変えない。
- `queued` または `failed` のプレビューのジョブを指定した適用を要求された場合、IdManagement は、409 と `preview_not_ready` で拒否し、Group を変えない。
- 保存したペイロードとダイジェストが一致しないプレビューのジョブを指定した適用を要求された場合、IdManagement は、409 と `preview_digest_mismatch` で拒否し、Group を変えない。
- 既存の Group の `membership_type` を現在と異なる値にする行を読んだ場合、IdManagement は、`immutable_membership_type` で `rejected` とし、その Group の `membership_type`、ロール、メンバーシップを変えない。
- 外部の取り込み元が管理する Group と、所有を判定できない Group の行を読んだ場合、IdManagement は、`source_managed` で `rejected` とし、Group を変えない。
- `membership_type=manual` の Group に動的規則の式を与える行、最終の状態の式が空のまま規則を有効にする行、式が定義していない属性か許可していない関数を参照する行を読んだ場合、IdManagement は、`invalid_dynamic_rule` で `rejected` とし、その Group の動的規則を保存しない。
- メールアドレスの形式を満たさない `email`、宣言した型の字句形に合わない `custom:<key>` の値、テナントのスキーマにない `custom:<key>` の列を与える行を読んだ場合、IdManagement は、安定したエラーコードで `rejected` とし、その Group の連絡先も属性も変えず、値をジョブの表示にも監査イベントにも含めない。
- **判断**：動的規則の式と有効化は、行が与えた列と、維持するもう一方の列を合わせた最終の状態で検証する。列ごとに検証すると、どちらの列だけでは不正に見えない組み合わせ（式を消して有効のまま、手動の Group に式だけを与える）を通してしまう。
- **判断**：適用をプレビューのペイロードに束縛する理由は、[CSV の適用をプレビューで保存したペイロードに束縛する](../design/decisions.md#csv-の適用をプレビューで保存したペイロードに束縛する)。
- **例**：EX-IDMANAGEMENT-026-01、EX-IDMANAGEMENT-026-11

#### REQ-IDMANAGEMENT-072 Group の CSV は、名前を値オブジェクトの定義で照合し、連絡先をアドレスだけで受け付ける

- `name` で対象を照合し、行の重複を判定するとき、IdManagement は、[名前](../README.md#値オブジェクト)の定義で比較する。
- `email` のセルを読んだとき、IdManagement は、アドレスを小文字にして保存する。
- `dynamic_rule_expression` のセルを読んだとき、IdManagement は、前後の空白を除いて読む。
- 空の `dynamic_rule_enabled` のセルを読んだとき、IdManagement は、現在の有効か無効かを変えない。
- 表示名付きの形式の `email` のセルを読んだ場合、IdManagement は、`invalid_email` で拒否する。

#### REQ-IDMANAGEMENT-028 Group の CSV のインポートは、`lifecycle_action=delete` の行の Group だけを削除する

- 管理者が Group の CSV を適用したとき、IdManagement は、`lifecycle_action` に `delete` を書いた行の Group だけを削除し、CSV に現れない Group を削除しない。
- 管理者が削除の行を含む CSV をプレビューしたとき、IdManagement は、削除する Group の件数と、巻き込むメンバーシップの件数を、ほかの操作と分けて返し、Group もメンバーシップも変えない。
- 削除の行を適用したとき、IdManagement は、Group を削除し、そのメンバーシップを同じトランザクションで解除し、`GroupMemberRemoved` と `GroupDeleted` を発行し、所属していた User の実効ロールからその Group のロールを外す。
- 削除の行を含む CSV を適用したとき、IdManagement は、同じファイルのほかの行の作成と更新を、削除の影響を受けずに適用する。
- `lifecycle_action` に `delete` 以外の値を書いた行を読んだ場合、IdManagement は、`invalid_lifecycle_action` で `rejected` とし、既知の値へ丸めない。
- `lifecycle_action=delete` の行が現在の状態と差分のある書き込める列も持つ場合、IdManagement は、`conflicting_lifecycle_action` で `rejected` とし、その Group を更新も削除もしない。
- `lifecycle_action=delete` の行が既存の Group を示さない場合、IdManagement は、`target_not_found` で `rejected` とし、Group を作らない。
- 外部の取り込み元が管理する Group と、所有を判定できない Group の削除の行を読んだ場合、IdManagement は、`source_managed` で `rejected` とし、Group を削除しない。
- **判断**：削除を、エクスポートが書き出す目標の状態の列ではなく、書き込み専用の `lifecycle_action` で表す。エクスポートが書き出す列を破壊的な意図に流用すると、編集していないエクスポートの再適用が全行 `unchanged` になる性質が失われ、絞り込みや分割で一部の行だけを含むファイルが大量の削除の引き金になる。
- **判断**：削除と更新を一つの行に同居させない。Group は削除で消えるので、直前の更新を監査に残すと、存在しなかった状態の記録になる。
- **例**：EX-IDMANAGEMENT-028-01、EX-IDMANAGEMENT-028-05

### メンバーシップのインポート

#### REQ-IDMANAGEMENT-029 メンバーシップの CSV のインポートは、行ごとに宣言した所属の状態だけを適用する

- 管理者が機械可読な列名の見出しを任意の順で持つ CSV を一つの Group のプレビューに投入したとき、IdManagement は、行ごとの `added`、`removed`、`unchanged`、`rejected` の判定、行番号、安定したエラーコードと、ほかの操作と分けた解除の件数を返すプレビューのジョブを作り、メンバーシップを変えない。
- 管理者が同じテナントかつ同じ Group の成功したプレビューのジョブの ID を指定して適用を始めたとき、IdManagement は、CSV を受け取り直さず、保存したプレビューのペイロードとその SHA-256 を検証する。
- 管理者がメンバーシップの CSV を適用したとき、IdManagement は、`membership_state=present` の行の User のメンバーシップを追加して `GroupMemberAdded` を発行し、User の実効ロールに Group のロールを加える。
- 管理者がメンバーシップの CSV を適用したとき、IdManagement は、`membership_state=absent` の行の User の手動のメンバーシップを外して `GroupMemberRemoved` を発行し、User の実効ロールから Group のロールを外す。
- すでに望みどおりの状態の行を読んだとき、IdManagement は、その行を `unchanged` にし、メンバーシップを変えない。
- プレビューの後にメンバーシップが変わっていた場合、IdManagement は、適用の時点のメンバーシップから `added`、`removed`、`unchanged`、`rejected` を判定し直す。
- 一行のメンバーシップの確定または監査の記録の途中で失敗した場合、IdManagement は、その行のメンバーシップも監査の記録も一部も残さず、ほかの有効な行の適用を続ける。
- 実効の転送ポリシーの `max_bytes`、`max_rows`、`max_field_bytes` のどれかを超える CSV を投入された場合、IdManagement は、`csv_too_large`、`too_many_rows`、`field_too_large` で拒否する。
- 見出しに未知の列、重複した列、`password` または `password_hash` を含む CSV と、見出しに `membership_state` のない CSV を投入された場合、IdManagement は、ファイル全体を `invalid_header` で拒否し、メンバーシップを一件も変えない。
- `membership_state` のセルが空の行と、`present` と `absent` のどちらでもない行を読んだ場合、IdManagement は、`invalid_membership_state` で `rejected` とし、既知の値へ丸めず、その User のメンバーシップを変えない。
- `user_id` も `preferred_username` もない行、両者が別の User を示す行、テナントにない User の行、同じ User を示す複数の行を読んだ場合、IdManagement は、`missing_identifier`、`identifier_mismatch`、`target_not_found`、`duplicate_target` で `rejected` とし、メンバーシップを変えない。
- 行の `group_id` または `group_name` が URL の Group 以外を示す行を読んだ場合、IdManagement は、`group_mismatch` で `rejected` とし、示された別の Group のメンバーシップも変えない。
- 存在しないプレビューのジョブ、別のテナントのプレビューのジョブ、または別の Group のプレビューのジョブを指定した適用とジョブの参照を要求された場合、IdManagement は、404 と `group_membership_import_not_found` で拒否し、メンバーシップを変えない。
- `queued` または `failed` のプレビューのジョブを指定した適用を要求された場合、IdManagement は、409 と `preview_not_ready` で拒否し、メンバーシップを変えない。
- 保存したペイロードとダイジェストが一致しないプレビューのジョブを指定した適用を要求された場合、IdManagement は、409 と `preview_digest_mismatch` で拒否し、メンバーシップを変えない。
- **判断**：対象の Group を URL の `group_id` だけで決める。ファイルの中身が対象を選べると、一つの Group を編集する権限が、ファイルに書いた任意の Group を編集する権限になる。
- **判断**：`membership_state` のない CSV をファイルごと拒否する。この列のないファイルは何の意図も表せず、それを変更 0 件として受け付けると、管理者は自分の編集が読まれたうえで差分がなかったと受け取る。
- **判断**：同じ User を示す複数の行を拒否する。どの行の意図が勝つかを、ファイルの並び順に委ねないためである。
- **例**：EX-IDMANAGEMENT-029-01、EX-IDMANAGEMENT-029-10

#### REQ-IDMANAGEMENT-031 メンバーシップ CSV は動的規則と外部の取り込み元が所有する所属を上書きしない

- 対象の Group の `membership_type` が `dynamic` の場合、IdManagement は、ファイル全体を `dynamic_group` で拒否し、メンバーシップを一件も変えない。
- 対象の User の現在のメンバーシップの `source` が `dynamic_rule` の行を読んだ場合、IdManagement は、`present` でも `absent` でも `dynamic_membership` で `rejected` とし、そのメンバーシップを変えない。
- 対象の Group が外部の取り込み元に管理されているか、所有を判定できない場合、IdManagement は、ファイル全体を `source_managed` で拒否し、メンバーシップを一件も変えない。
- 対象の User が外部の取り込み元に管理されているか、所有を判定できない行を読んだ場合、IdManagement は、`source_managed` で `rejected` とし、その User のメンバーシップを変えない。
- 対象の Group がテナントにないか、適用の直前に削除された場合、IdManagement は、ファイル全体を `target_not_found` で拒否し、Group もメンバーシップも作らない。
- **判断**：動的な所属の `present` の行も拒否する。何も書き込まないので受け付けてもよさそうに見えるが、規則が次に一致しなくなればその所属は消える。変化なしと答えると、管理者が受け取る保証と、所属が実際に続く期間が食い違う。
- **判断**：メンバーシップは Group と User を結ぶ関係なので、どちらか一方でも外部が所有していれば、その関係を決める権限も外部にあるとみなす。
- **例**：EX-IDMANAGEMENT-031-01、EX-IDMANAGEMENT-031-05

### Group のエクスポート

#### REQ-IDMANAGEMENT-027 Group のエクスポートは、そのまま再インポートすると変化なしになり、書き込める列の編集だけを反映する

- `worker` が Group のエクスポートを生成したとき、IdManagement は、選んだ列の CSV を不変の成果物ストアへ書き出し、ジョブの結果にテナント単位のペイロードの参照、サーバーが計算した SHA-256、サイズ、行数を記録する。
- `worker` が Group のエクスポートを生成したとき、IdManagement は、`lifecycle_action` の列を全行で空として書き出す。
- インポートできる組み込み列と `custom:` の列でエクスポートした CSV を編集せずにプレビューしたとき、IdManagement は、全行が `unchanged` になり、Group を一件も作成、更新、削除しない。
- 一部の行の書き込める列だけを編集してプレビューしたとき、IdManagement は、その行だけを `updated`、残りの行を `unchanged` と計画する。
- 一部の行の書き込める列だけを編集して適用したとき、IdManagement は、指定した書き込める列だけを更新し、指定しなかった列を保つ。
- 危険な先頭文字、既存のアポストロフィー、カンマ、引用符、改行を含む値を書き出して読み戻したとき、IdManagement は、元の値と一致する値を読む。
- 読み取り専用の列（`id`、`created_at`、`updated_at`）の値だけを編集した行を読んだとき、IdManagement は、列を受け付けたうえで無視し、その行を `unchanged` にする。
- 生成結果が実効の転送ポリシーのどれかの上限を超える場合、IdManagement は、`csv_transfer_limit_exceeded` で失敗し、再インポートできない成功の成果物を作らない。
- `roles`、`membership_type`、`email`、`custom:<key>` のどれかの値が不正な行を読んだ場合、IdManagement は、安定したエラーコードで `rejected` とし、値をジョブの表示にも監査イベントにも含めない。
- **例**：EX-IDMANAGEMENT-027-01、EX-IDMANAGEMENT-027-04

### メンバーシップのエクスポート

#### REQ-IDMANAGEMENT-008 メンバーシップのエクスポートは、パスの Group のメンバーだけを書き出す

- `worker` がメンバーシップのエクスポートを生成したとき、IdManagement は、パスの `group_id` の Group のメンバーだけを書き出す。
- `group_id` を指定しない開始を要求された場合、IdManagement は、InvalidRequestError で拒否する。
- 別の Group のパスでエクスポートの ID を指定した取得とダウンロードを要求された場合、IdManagement は、[データエクスポート](../data-export/README.md)の REQ-IDMANAGEMENT-088 の応答で拒否する。
- **例**：EX-IDMANAGEMENT-008-01、EX-IDMANAGEMENT-008-03

#### REQ-IDMANAGEMENT-030 メンバーシップのエクスポートは、分けて再適用しても、ほかのファイルにしかないメンバーを外さない

- `worker` がメンバーシップのエクスポートを生成したとき、IdManagement は、CSV を不変の成果物ストアへ書き出し、ジョブの結果にテナント単位のペイロードの参照、サーバーが計算した SHA-256、サイズ、行数を記録する。
- `worker` がメンバーシップのエクスポートを生成したとき、IdManagement は、`membership_state` の列を全行で `present` として書き出す。
- 編集していないエクスポートをプレビューしたとき、IdManagement は、全行が `unchanged` になり、メンバーシップを一件も追加も解除もしない。
- エクスポートを二つのファイルに分けて片方だけを適用したとき、IdManagement は、もう一方のファイルにしか現れないメンバーシップを解除しない。
- メンバーシップの CSV を適用したとき、IdManagement は、CSV に現れない User のその Group への所属を変えない。
- 危険な先頭文字、既存のアポストロフィー、カンマ、引用符、改行を含む値を書き出して読み戻したとき、IdManagement は、元の値と一致する値を読む。
- 読み取り専用の列（`source`、`created_at`、`group_name`）の値だけを編集した行を読んだとき、IdManagement は、列を受け付けたうえで無視する。
- `group_name` が別の Group を示す行を読んだ場合、IdManagement は、行を拒否する。
- 生成結果が実効の転送ポリシーのどれかの上限を超える場合、IdManagement は、`csv_transfer_limit_exceeded` で失敗し、再インポートできない成功の成果物を作らない。
- **例**：EX-IDMANAGEMENT-030-01、EX-IDMANAGEMENT-030-04

## セキュリティ上の考慮

| 観点 | 内容 |
| --- | --- |
| 破壊的な操作 | 削除と所属の解除は不可逆で、所属していた User の実効ロールを一度に変える。プレビューは削除と解除の件数を分けて返し、管理 UI は削除や解除を含む適用に明示の確認を求める |
| 対象の範囲 | メンバーシップの CSV が変えられる Group は、URL の一つだけである |
| 上流の権威 | 外部の取り込み元が管理する Group、User、メンバーシップは、CSV で変更しない。所有を判定できないときも変更しない |
| 値の開示 | 拒否した行の値は、ジョブの表示にも監査イベントにも含めない |
