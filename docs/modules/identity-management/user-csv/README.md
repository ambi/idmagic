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

#### REQ-IDMANAGEMENT-004 User の CSV のインポートは、プレビューで全行を判定し、適用で有効な行だけを保存する

- 管理者が機械可読な列名の見出しを任意の順で持つ CSV をプレビューに投入したとき、IdManagement は、行ごとの `created`、`updated`、`unchanged`、`rejected` の判定、行番号、安定したエラーコードを返すプレビューのジョブを作り、User を変えない。
- 管理者が同じテナントの成功したプレビューのジョブの ID を指定して適用を始めたとき、IdManagement は、CSV を受け取り直さず、保存したプレビューのペイロードを SHA-256 で検証し、現在の状態に対して同じ計画器で計画し直す。
- プレビューの後に対象の User が変わっていた場合、IdManagement は、適用の時点の状態から `updated`、`unchanged`、`rejected` を判定し直す。
- 管理者が User の CSV を適用したとき、IdManagement は、有効な行の User を作成または更新し、無効な行を `rejected` として残す。
- 管理者が User の CSV を適用したとき、IdManagement は、各行のプロフィール、ロール、必須操作、カスタム属性を不可分に保存する。
- 管理者が User の CSV を適用したとき、IdManagement は、作成、更新、変更なしと判定した各行の User を、保存した後に有効な動的グループの規則で評価する。
- 動的グループの規則の評価に失敗した場合、IdManagement は、確定した行を残したまま適用を失敗として終える。
- 一行の検証、保存、監査の途中で失敗した場合、IdManagement は、その行を一部も保存せず、ほかの有効な行の適用を続ける。
- 実効の転送ポリシーの `max_bytes`、`max_rows`、`max_field_bytes` のどれかを超える CSV を投入された場合、IdManagement は、`csv_too_large`、`too_many_rows`、`field_too_large` で拒否する。
- 見出しに未知の列、重複した列、`password` または `password_hash` を含む CSV を投入された場合、IdManagement は、`invalid_header` で拒否する。
- `id` と `preferred_username` が別の User を示す行、識別子のない行、同じ対象または同じ最終のユーザー名を示す複数の行を読んだ場合、IdManagement は、その行を安定したエラーコードで `rejected` にする。
- 存在しないジョブ、または別のテナントのジョブを指定した適用とジョブの参照を要求された場合、IdManagement は、404 と `user_import_not_found` で拒否し、User を変えない。
- `queued` または `failed` のプレビューのジョブを指定した適用を要求された場合、IdManagement は、409 と `preview_not_ready` で拒否し、User を変えない。
- 保存したペイロードとダイジェストが一致しないプレビューのジョブを指定した適用を要求された場合、IdManagement は、409 と `preview_digest_mismatch` で拒否し、User を変えない。
- 外部の取り込み元が管理する User の行を読んだ場合、IdManagement は、`source_managed` で `rejected` とし、User を変えない。
- **上位の要件**：[REQ-IDMANAGEMENT-089](../user/README.md#作成の経路に共通する規則)
- **判断**：適用をプレビューのペイロードに束縛する理由は、[CSV の適用をプレビューで保存したペイロードに束縛する](../design/decisions.md#csv-の適用をプレビューで保存したペイロードに束縛する)。
- **判断**：変更なしの行も評価するのは、評価に失敗した適用を、同じ CSV の再適用で回収できるようにするためである。再適用では確定済みの行が変更なしになる。
- **判断**：CSV のインポートは、より強い上流の権威を上書きしない。取り込み元が管理する User を拒否するのはそのためである。
- **例**：EX-IDMANAGEMENT-004-01、EX-IDMANAGEMENT-004-08

#### REQ-IDMANAGEMENT-055 User の CSV の属性の列は、組み込みの属性を `attr:`、テナント定義の属性を `custom:` で表す

- User の CSV を読み書きするとき、IdManagement は、組み込みの拡張属性を `attr:<key>` の列で扱う。
- User の CSV を読み書きするとき、IdManagement は、テナント定義の属性を `custom:<key>` の列で扱う。
- 組み込みの拡張属性を `custom:<key>` の列で指定したファイルを受けた場合、IdManagement は、`invalid_header` で拒否する。

#### REQ-IDMANAGEMENT-056 User の CSV の行は `id` を優先して対象を決め、先に現れた行を採る

- `id` を持つ行を読んだとき、IdManagement は、`id` で対象を決める。
- `id` を持たず `preferred_username` だけを持つ行を読んだとき、IdManagement は、そのユーザー名の User を対象にする。
- `id` を持たず、`preferred_username` の User がテナントにない行を読んだとき、IdManagement は、作成として計画する。
- `preferred_username` で対象を照合し、行のユーザー名の重複を判定するとき、IdManagement は、[名前](../README.md#値オブジェクト)の定義で比較する。
- テナントにない `id` の行を読んだ場合、IdManagement は、`target_not_found` で `rejected` とし、User を作らない。
- 前の行と同じ `id` を持つ行を読んだ場合、IdManagement は、その行を `duplicate_target` で `rejected` にし、前の行を計画に残す。
- 前の行と同じ `preferred_username` を持つ行を読んだ場合、IdManagement は、その行を `duplicate_username` で `rejected` にし、前の行を計画に残す。
- 行の対象でない、同じテナントの削除されていない User と[メールアドレス](../README.md#値オブジェクト)が同じ `email` の行を読んだ場合、IdManagement は、その行を `email_taken` で `rejected` にする。
- 前の行とメールアドレスが同じ `email` を持つ行を読んだ場合、IdManagement は、その行を `duplicate_email` で `rejected` にし、前の行を計画に残す。

#### REQ-IDMANAGEMENT-057 User の CSV の組み込み列は、決まった字句形のセルだけを受け付ける

- `roles` と `required_actions` のセルを読んだとき、IdManagement は、`|` で区切った値の並びとして読み、各値の前後の空白を除き、空のセルを空の並びとして扱う。
- `required_actions` のセルを読んだとき、IdManagement は、重複を除いて昇順に保存する。
- `name`、`given_name`、`family_name`、`email` の空のセルを読んだとき、IdManagement は、その項目を消す。
- 空の値を含む `roles` のセルを読んだ場合、IdManagement は、`invalid_roles` で拒否する。
- 空の値を含む `required_actions` のセルを読んだ場合、IdManagement は、`invalid_required_actions` で拒否する。
- `true` と `false` のどちらでもない `email_verified` のセルを読んだ場合、IdManagement は、`invalid_boolean` で拒否する。
- 空の `preferred_username` のセルを読んだ場合、IdManagement は、`required` で拒否する。

#### REQ-IDMANAGEMENT-058 CSV で作成する User には、無作為なパスワードと必須操作 `update_password` を付ける

- CSV の適用で User を作成したとき、IdManagement は、誰にも知らされない無作為なパスワードと、必須操作 `update_password` を設定した `Active` の User を作り、テナントの User の使用量を一つ増やす。

#### REQ-IDMANAGEMENT-059 取り込み元の所有を判定できないとき、既存の User の行をすべて拒否する

- 取り込み元による所有を判定できない間、既存の User を対象とする行を読んだとき、IdManagement は、その行を `source_managed` で `rejected` にする。
- 取り込み元による所有を判定できない間、新しい User を作る行を読んだとき、IdManagement は、その行を受け付ける。

### エクスポート

#### REQ-IDMANAGEMENT-006 User のエクスポートは、許可リストの列だけを受け付け、危険な先頭文字のセルに接頭辞を付けて書き出す

- `worker` が User のエクスポートを生成したとき、IdManagement は、[データエクスポート](../data-export/README.md)の流れで、選んだ列の RFC 4180 の CSV を書き出す。
- `=`、`+`、`-`、`@`、タブ、CR、LF で始まるセルを書き出すとき、IdManagement は、数式の注入を避ける可逆な接頭辞を付ける。
- 接頭辞を付けたセルをインポートで読んだとき、IdManagement は、その接頭辞の一文字だけを取り除く。
- User の許可リストにない列（例：`password_hash`）を含む開始を要求された場合、IdManagement は、422 と `invalid_columns` で拒否し、ジョブを作らない。
- **例**：EX-IDMANAGEMENT-006-02、EX-IDMANAGEMENT-006-04

#### REQ-IDMANAGEMENT-007 User のエクスポートは、そのまま再インポートすると変化なしになり、書き込める列の編集だけを反映する

- インポートできる組み込み列、`required_actions`、`custom:` の列でエクスポートした CSV を編集せずにプレビューしたとき、IdManagement は、全行が `unchanged` になり、User を変えない。
- 一部の行の書き込める列だけを編集してプレビューしたとき、IdManagement は、その行だけを `updated`、残りの行を `unchanged` と計画する。
- 一部の行の書き込める列だけを編集して適用したとき、IdManagement は、指定した書き込める列だけを更新し、指定しなかった列を保つ。
- 危険な先頭文字、既存のアポストロフィー、カンマ、引用符、改行を含む値を書き出して読み戻したとき、IdManagement は、元の値と一致する値を読む。
- 読み取り専用の列（`status`、`mfa_enrolled`、`created_at`、`updated_at`、`id`）の値だけを編集した行を読んだとき、IdManagement は、列を受け付けたうえで無視し、その行を `unchanged` にする。
- 生成結果が実効の転送ポリシーのどれかの上限を超える場合、IdManagement は、`csv_transfer_limit_exceeded` で失敗し、再インポートできない成功の成果物を作らない。
- カスタム属性の型、真偽値、数値、日付、必須のカスタム属性、`required_actions` のどれかが不正な行を読んだ場合、IdManagement は、その行を安定したエラーコードで `rejected` にし、値をジョブの表示にも監査イベントにも含めない。
- **例**：EX-IDMANAGEMENT-007-01、EX-IDMANAGEMENT-007-04

## セキュリティ上の考慮

| 観点 | 内容 |
| --- | --- |
| 秘密情報 | パスワードとそのハッシュは、どちらの向きの列にもならない。CSV で作る User には、誰にも知らされない無作為なパスワードを設定する |
| 上流の権威 | 外部の取り込み元が管理する User は、CSV で変更できない。所有を判定できないときも変更しない |
| 値の開示 | 拒否した行の値は、ジョブの表示にも監査イベントにも含めない |
| 数式の注入 | 表計算で開かれる前提で、数式として解釈される先頭文字に接頭辞を付ける |
