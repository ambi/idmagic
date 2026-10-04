# Feature: ブランド設定の例

## Rule: REQ-TENANCY-032 ブランド設定の文字列は空白を除いて保存し、空文字列は未設定に戻す

### Example: EX-TENANCY-032-01 空白を含む値と空文字列

- Given admin ロールを持つ "operator" が認証済みである
- When "operator" が `product_name` に " Acme ID " を保存する
- Then 取得したブランド設定の `product_name` は "Acme ID" である
- When "operator" が `product_name` に空文字列を保存する
- Then 取得したブランド設定は `product_name` を持たない

### Example: EX-TENANCY-032-02 27 文字の日本語のラベル

- Given admin ロールを持つ "operator" が認証済みである
- When "operator" が 27 文字の日本語のラベルと HTTPS の URL を `footer_link_1` に保存する
- Then `invalid_branding` の 400 で拒否され、何も保存されない

### Example: EX-TENANCY-032-03 大文字のスキームで始まる URL

- Given admin ロールを持つ "operator" が認証済みである
- When "operator" が `footer_link_1` の URL に "HTTPS://help.example.test" を保存する
- Then `invalid_branding` の 400 で拒否され、何も保存されない

## Rule: REQ-TENANCY-033 ロゴとファビコンは 256 KiB 以下の PNG、JPEG、WebP、GIF だけを受け付ける

### Example: EX-TENANCY-033-01 ちょうど 262,144 バイトの PNG

- Given admin ロールを持つ "operator" が認証済みである
- When "operator" が 262,144 バイトの PNG をロゴとしてアップロードする
- Then アップロードは成功し、レスポンスに `logo_url` が含まれる

### Example: EX-TENANCY-033-02 262,145 バイトの PNG

- Given admin ロールを持つ "operator" が認証済みである
- When "operator" が 262,145 バイトの PNG をロゴとしてアップロードする
- Then `invalid_request` の 400 で拒否され、ブランド設定は `logo_url` を持たない

### Example: EX-TENANCY-033-03 ブランド設定が未設定のテナントで削除する

- Given admin ロールを持つ "operator" が認証済みである
- And テナントはブランド設定を一度も保存していない
- When "operator" がロゴを削除する
- Then 200 と空のブランド設定が返る

## Rule: REQ-TENANCY-004 管理者はテナントのロゴと配色をカスタマイズでき利用者のログイン画面に反映される

### Example: EX-TENANCY-004-01 通常経路

- Given admin ロールを持つ "operator" が認証済みである
- When "operator" が PNG ロゴをアップロードする
- Then アップロードレスポンスに logo_url が含まれる
- When "operator" が logo_url を GET する
- Then 同じ realm の検証済み PNG が返る
- Then 管理画面のロゴプレビューにアップロードした PNG が表示される
- When "operator" が primary_color / accent_color / footer_link_1={label: "ヘルプ", url: "https://help.example.test"} / footer_text を設定する
- Then 管理画面は各設定済み色に現在値と「デフォルトに戻す」操作を表示する
- When 管理者がプライマリカラーをデフォルトに戻して保存する
- Then UpdateTenantBranding には primary_color の空文字列が送られる
- When 未認証の利用者が login 画面を開く
- Then login / 同意 / account portal に設定したロゴが表示され、login 画面にはプライマリカラーのシステムデフォルト・設定済みアクセントカラー・指定ラベルの footer リンク・フッターテキストも表示される

### Example: EX-TENANCY-004-02 別テナントの id で同じ kind のアセット取得を試みる

- Given admin ロールを持つ "operator" が認証済みである
- When "operator" が PNG ロゴをアップロードする
- Then アップロードレスポンスに logo_url が含まれる
- When "operator" が logo_url を GET する
- But 別テナントの id で同じ kind のアセット取得を試みる
- Then アセットは存在しないものとして扱われ、応答は存在しない id を指定したときと同じ 404 not_found である
- Then 応答にアップロードした PNG の内容は含まれない

### Example: EX-TENANCY-004-03 realm 配下の logo_url をゲートウェイ越しに取得する

- Given admin ロールを持つ "operator" が認証済みである
- When "operator" が管理画面からゲートウェイ越しに PNG ロゴをアップロードする
- Then 管理画面のロゴプレビューは realm 配下の logo_url を参照する
- When logo_url をゲートウェイ越しに GET する
- Then ゲートウェイは要求を backend へ転送し、アップロードした PNG と同じバイト列が image/png で返る

## Rule: REQ-TENANCY-034 公開のブランド設定は、版を表す ETag とともに返す

### Example: EX-TENANCY-034-01 ブランド設定が未設定のテナント

- When 未認証の利用者がブランド設定を取得する
- Then 200 と空のブランド設定が返り、ETag は `"branding-default"`、Cache-Control は `public, max-age=60` である

### Example: EX-TENANCY-034-02 取得済みの ETag を If-None-Match に指定する

- When 未認証の利用者が、直前の応答の ETag を `If-None-Match` に指定してブランド設定を取得する
- Then 本文のない 304 が返る

### Example: EX-TENANCY-034-03 ブランド設定を更新した後

- Given 未認証の利用者がブランド設定の ETag を取得済みである
- When 管理者がブランド設定を更新する
- Then 次の取得の ETag は、取得済みの ETag と異なる

## Rule: REQ-TENANCY-035 ブランドアセットは、検証済みの形式と nosniff を付けて配信する

### Example: EX-TENANCY-035-01 アップロードしたロゴを取得する

- Given 管理者が PNG のロゴをアップロード済みである
- When 未認証の利用者が `logo_url` を取得する
- Then Content-Type は `image/png` で、応答は `X-Content-Type-Options: nosniff` と `Cache-Control: private, max-age=3600` を持つ

### Example: EX-TENANCY-035-02 未知の種別を指定する

- When 未認証の利用者が種別 "banner" のアセットを取得する
- Then `not_found` の 404 が返る

## Rule: REQ-TENANCY-005 不正な branding 入力は拒否されシステムデフォルトにフォールバックする

### Example: EX-TENANCY-005-01 通常経路

- Given admin ロールを持つ "operator" が認証済みである
- When "operator" が branding を一度も設定していないテナントで login 画面を開く
- Then login 画面はシステムデフォルト (IdMagic) のブランディングを表示する
- When "operator" が footer_link_1.url に javascript: スキームを指定して保存する
- Then `invalid_branding` の 400 で拒否され保存されない
- When "operator" が低コントラストの `#eeeeee` を primary_color に指定して保存する
- Then 保存に成功し、取得した branding と login 画面に `#eeeeee` が反映される
- When 管理者が SVG ファイルをロゴとしてアップロードする
- Then `invalid_request` の 400 で拒否され保存されない

### Example: EX-TENANCY-005-02 footer_link_1 に label だけを指定する

- Given admin ロールを持つ "operator" が認証済みである
- When "operator" が branding を一度も設定していないテナントで login 画面を開く
- Then login 画面はシステムデフォルト (IdMagic) のブランディングを表示する
- When "operator" が footer_link_1.url に javascript: スキームを指定して保存する
- But footer_link_1 に label だけを指定する
- Then `invalid_branding` の 400 で拒否され保存されない
