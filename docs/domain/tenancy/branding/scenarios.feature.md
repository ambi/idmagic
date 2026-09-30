# Feature: ブランド設定のシナリオ

## 結果

### Rule: REQ-TENANCY-004 管理者はテナントのロゴと配色をカスタマイズでき利用者のログイン画面に反映される

Primary actor: `TenantAdministrator`

#### Example: EX-TENANCY-004-01 通常経路

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

#### Example: EX-TENANCY-004-02 別テナントの id で同じ kind のアセット取得を試みる

- Given admin ロールを持つ "operator" が認証済みである
- When "operator" が PNG ロゴをアップロードする
- Then アップロードレスポンスに logo_url が含まれる
- When "operator" が logo_url を GET する
- But 別テナントの id で同じ kind のアセット取得を試みる
- Then アセットは存在しないものとして扱われ、応答は存在しない id を指定したときと同じ 404 not_found である
- Then 応答にアップロードした PNG の内容は含まれない

#### Example: EX-TENANCY-004-03 realm 配下の logo_url をゲートウェイ越しに取得する

- Given admin ロールを持つ "operator" が認証済みである
- When "operator" が管理画面からゲートウェイ越しに PNG ロゴをアップロードする
- Then 管理画面のロゴプレビューは realm 配下の logo_url を参照する
- When logo_url をゲートウェイ越しに GET する
- Then ゲートウェイは要求を backend へ転送し、アップロードした PNG と同じバイト列が image/png で返る

## 拒否

### Rule: REQ-TENANCY-005 不正な branding 入力は拒否されシステムデフォルトにフォールバックする

Primary actor: `TenantAdministrator`

#### Example: EX-TENANCY-005-01 通常経路

- Given admin ロールを持つ "operator" が認証済みである
- When "operator" が branding を一度も設定していないテナントで login 画面を開く
- Then login 画面はシステムデフォルト (IdMagic) のブランディングを表示する
- When "operator" が footer_link_1.url に javascript: スキームを指定して保存する
- Then InvalidRequestError で拒否され保存されない
- When "operator" が低コントラストの `#eeeeee` を primary_color に指定して保存する
- Then 保存に成功し、取得した branding と login 画面に `#eeeeee` が反映される
- When 管理者が SVG ファイルをロゴとしてアップロードする
- Then InvalidRequestError で拒否され保存されない

#### Example: EX-TENANCY-005-02 footer_link_1 に label だけを指定する

- Given admin ロールを持つ "operator" が認証済みである
- When "operator" が branding を一度も設定していないテナントで login 画面を開く
- Then login 画面はシステムデフォルト (IdMagic) のブランディングを表示する
- When "operator" が footer_link_1.url に javascript: スキームを指定して保存する
- But footer_link_1 に label だけを指定する
- Then InvalidRequestError で拒否され保存されない
