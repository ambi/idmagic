# ユーザー CSV

## 概要

この文書は、管理者が User を CSV で一括して作成、更新し、CSV へエクスポートする機能の仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | User の CSV の列の語彙、行の対象の決め方、インポートのプレビューと適用、エクスポートと再適用 |
| 行為者 | 管理者、インポートとエクスポートのジョブ |
| コードの機能スライス | `backend/idmanagement/user` |
| 扱わないもの | ファイルとセルの共通の読み書きは[CSV の転送](../csv-transfer/README.md)が、エクスポートのジョブの開始、一覧、取り消し、保持期限は[データエクスポート](../data-export/README.md)が扱う |

## モデル

User の CSV の列は、種別ごとに閉じた集合である。

| 列の種類 | 列 | 扱い |
| --- | --- | --- |
| 識別の列 | `id` | インポートの行の対象を決める |
| 組み込みの書き込める列 | `preferred_username`、`name`、`given_name`、`family_name`、`email`、`email_verified`、`roles`、`required_actions` | インポートで読み、エクスポートで書く |
| 組み込みの拡張属性 | `attr:<key>` | 同上 |
| テナント定義の属性 | `custom:<key>` | 実効的な属性スキーマを解決した後に加える |
| 読み取り専用の列 | `status`、`mfa_enrolled`、`created_at`、`updated_at` | エクスポートで書き、インポートでは値を変更として扱わない |
| 禁止する列 | `password`、`password_hash` | どちらの向きでも扱わない |

CSV は二つ目のプロビジョニングの権威ではなく、IdManagement が備える部分更新の窓口である。

## 操作

### インポート

| 項目 | 内容 |
| --- | --- |
| 行為者 | 管理者がプレビューと適用を要求し、ジョブが実行する |
| 入力 | プレビューでは CSV、適用では成功したプレビューのジョブの ID |
| 成功時の作用 | プレビューは行ごとの判定を返し、User を変えない。適用は有効な行を作成または更新し、行ごとに不可分に保存する |
| 拒否 | ファイル全体の拒否（上限の超過、不正な見出し）、行の拒否（識別子の矛盾、重複、不正なセル、外部の権威が管理する User）、使えないプレビューを指定した適用 |
| 冪等性 | 編集していないエクスポートの再適用は、全行が `unchanged` になる |

#### REQ-IDMANAGEMENT-004 User の CSV のインポートは、プレビューで全行を判定し、適用で有効な行だけを保存する

- 管理者は、機械可読な列名の見出しを任意の順で持つ CSV を、事前検証（プレビュー）に投入できる。
- プレビューのジョブは、行ごとの `created`、`updated`、`unchanged`、`rejected` の判定、行番号、安定したエラーコードを返し、User を変えない。
- 適用は、同じテナントの成功したプレビューのジョブの ID を指定して始める。CSV を受け取り直さず、保存したプレビューのペイロードを使う。
- 適用は、ペイロードを SHA-256 で検証し、現在の状態に対して同じ計画器で計画し直す。プレビューの後に対象の User が変わっていれば、現在の状態から `updated`、`unchanged`、`rejected` を判定し直す。
- 適用は有効な行を作成または更新し、無効な行を `rejected` として残す。各行のプロフィール、ロール、必須操作、カスタム属性は不可分に保存する。
- 一行の検証、保存、監査の途中で失敗した行は、一部も保存せず、ほかの有効な行の適用を続ける。
- 実効の転送ポリシーの `max_bytes`、`max_rows`、`max_field_bytes` のどれかを超える CSV の投入は、`csv_too_large`、`too_many_rows`、`field_too_large` で拒否する。
- 見出しに未知の列、重複した列、`password` または `password_hash` を含む CSV の投入は、`invalid_header` で拒否する。
- `id` と `preferred_username` が別の User を示す行、識別子のない行、同じ対象または同じ最終のユーザー名を示す複数の行は、安定したエラーコードで `rejected` とする。
- 存在しない、`queued` または `failed` の、別のテナントの、または保存したペイロードとダイジェストが一致しないプレビューのジョブを指定した適用は、User を変えずに InvalidRequestError または AccessDeniedError で拒否する。
- 外部の取り込み元が管理する User の行は、`source_managed` で `rejected` とし、User を変えない。
- **判断**：適用をプレビューのペイロードに束縛する理由は、[CSV の適用をプレビューで保存したペイロードに束縛する](../design/decisions.md#csv-の適用をプレビューで保存したペイロードに束縛する)。
- **判断**：CSV のインポートは、より強い上流の権威を上書きしない。取り込み元が管理する User を拒否するのはそのためである。
- **担保手段**：`usecases.StartUserImportPreview`、`usecases.StartUserImportApply`、`usecases.PlanUserImport`、`usecases.ApplyUserImport`
- **例**：EX-IDMANAGEMENT-004-01、EX-IDMANAGEMENT-004-08

#### REQ-IDMANAGEMENT-055 User の CSV の属性の列は、組み込みの属性を `attr:`、テナント定義の属性を `custom:` で表す

- 組み込みの拡張属性は `attr:<key>` の列で読み書きする。
- テナント定義の属性は `custom:<key>` の列で読み書きする。
- 組み込みの拡張属性を `custom:<key>` の列で指定したファイルは、`invalid_header` で拒否する。
- **担保手段**：`domain.NewUserCSVSchema`

#### REQ-IDMANAGEMENT-056 User の CSV の行は `id` を優先して対象を決め、先に現れた行を採る

- `id` を持つ行は `id` で対象を決める。テナントにない `id` の行は `target_not_found` で `rejected` とし、User を作らない。
- `id` を持たず `preferred_username` だけを持つ行は、そのユーザー名の User を対象とし、いなければ作成として計画する。
- 前の行と同じ `id` を持つ行は `duplicate_target`、前の行と同じ `preferred_username` を持つ行は `duplicate_username` で `rejected` とし、前の行を計画に残す。
- ユーザー名の重複は大文字と小文字を区別して判定する。
- **担保手段**：`usecases.PlanUserImport`

#### REQ-IDMANAGEMENT-057 User の CSV の組み込み列は、決まった字句形のセルだけを受け付ける

- `roles` と `required_actions` のセルは、`|` で区切った値の並びである。各値の前後の空白を除く。
- 空のセルは空の並びとし、空の値を含むセルは、`roles` では `invalid_roles`、`required_actions` では `invalid_required_actions` で拒否する。
- `required_actions` は重複を除いて昇順に保存する。
- `email_verified` のセルは `true` または `false` だけを受け付け、それ以外は `invalid_boolean` で拒否する。
- `name`、`given_name`、`family_name`、`email` の空のセルは、その項目を消す。
- `preferred_username` の空のセルは `required` で拒否する。
- **担保手段**：`usecases.PlanUserImport`

#### REQ-IDMANAGEMENT-058 CSV で作成する User には、無作為なパスワードと必須操作 `update_password` を付ける

- CSV で作成する User には、誰にも知らされない無作為なパスワードを設定する。
- CSV で作成する User には、必須操作 `update_password` を付ける。
- CSV で作成する User は `Active` であり、テナントの User の使用量を一つ増やす。
- **担保手段**：`usecases.ApplyUserImport`

#### REQ-IDMANAGEMENT-059 取り込み元の所有を判定できないとき、既存の User の行をすべて拒否する

- 取り込み元による所有を判定できないとき、既存の User を対象とする行はすべて `source_managed` で `rejected` とする。
- 同じとき、新しい User を作る行は受け付ける。
- **担保手段**：`usecases.PlanUserImport`

### エクスポート

| 項目 | 内容 |
| --- | --- |
| 行為者 | 管理者が開始し、`worker` が生成する |
| 入力 | 列と、`status` の絞り込み |
| 成功時の作用 | 選んだ列の RFC 4180 の CSV を生成し、ダウンロードで返す。開始、生成、ダウンロード、取り消しの流れは[データエクスポート](../data-export/README.md)が定める |
| 拒否 | 許可リストにない列（`invalid_columns`）。生成が上限を超えたときは成功の成果物を作らずに失敗する |

#### REQ-IDMANAGEMENT-006 User のエクスポートは、許可リストの列だけを受け付け、危険な先頭文字のセルに接頭辞を付けて書き出す

- User の許可リストにない列（例：`password_hash`）を含む開始は、InvalidRequestError と `invalid_columns` で拒否する。
- `=`、`+`、`-`、`@`、タブ、CR、LF で始まるセルは、数式の注入を避ける可逆な接頭辞を付けて書き出す。インポートは、その接頭辞の一文字だけを取り除く。
- **担保手段**：`usecases.StartDataExport`、`usecases.ExportUserCSV`
- **例**：EX-IDMANAGEMENT-006-02、EX-IDMANAGEMENT-006-04

#### REQ-IDMANAGEMENT-007 User のエクスポートは、そのまま再インポートすると変化なしになり、書き込める列の編集だけを反映する

- インポートできる組み込み列、`required_actions`、`custom:` の列でエクスポートした CSV を、編集せずにプレビューすると、全行が `unchanged` になり、User を変えない。
- 一部の行の書き込める列だけを編集してプレビューすると、その行だけが `updated`、残りの行は `unchanged` と計画される。適用は、指定した書き込める列だけを更新し、指定しなかった列を保つ。
- 危険な先頭文字、既存のアポストロフィー、カンマ、引用符、改行を含む値は、書き出して読み戻すと元の値と一致する。
- 生成結果が実効の転送ポリシーのどれかの上限を超える User のエクスポートは、`csv_transfer_limit_exceeded` で失敗し、再インポートできない成功の成果物を作らない。
- 読み取り専用の列（`status`、`mfa_enrolled`、`created_at`、`updated_at`、`id`）の値だけを編集した行は、列を受け付けたうえで無視し、書き込める列に差分がなければ `unchanged` とする。
- カスタム属性の型、真偽値、数値、日付、必須のカスタム属性、`required_actions` のどれかが不正な行は、安定したエラーコードで `rejected` とし、値をジョブの表示にも監査イベントにも含めない。
- **担保手段**：`usecases.ExportUserCSV`、`usecases.PlanUserImport`
- **例**：EX-IDMANAGEMENT-007-01、EX-IDMANAGEMENT-007-04

## セキュリティ上の考慮

| 観点 | 内容 |
| --- | --- |
| 秘密情報 | パスワードとそのハッシュは、どちらの向きの列にもならない。CSV で作る User には、誰にも知らされない無作為なパスワードを設定する |
| 上流の権威 | 外部の取り込み元が管理する User は、CSV で変更できない。所有を判定できないときも変更しない |
| 値の開示 | 拒否した行の値は、ジョブの表示にも監査イベントにも含めない |
| 数式の注入 | 表計算で開かれる前提で、数式として解釈される先頭文字に接頭辞を付ける |
