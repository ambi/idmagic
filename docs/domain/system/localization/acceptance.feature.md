# Feature: 表示言語の例

## Rule: REQ-SYSTEM-003 明示的に選択した表示言語でホスト認証画面が描画される

### Example: EX-SYSTEM-003-01 通常経路

- Given 未認証セッションでログイン画面を表示している
- When EndUser が表示言語 "en" を選択する
- Then ログイン画面の文言が `en` 辞書で表示される
- Then 選択したロケールがブラウザーに保存され、以後のアクセスで保存済み設定として優先される

## Rule: REQ-SYSTEM-004 未対応のロケールはデフォルトのロケールへフォールバックする

### Example: EX-SYSTEM-004-01 通常経路

- Given ブラウザーの言語設定が `fr` である
- And 表示言語の明示選択も保存済み設定も存在しない
- When EndUser がログイン画面を表示する
- Then 画面の文言はデフォルトロケール `en` の辞書で表示される

## Rule: REQ-SYSTEM-005 起動時設定のデフォルトロケールがフォールバックに使われる

### Example: EX-SYSTEM-005-01 通常経路

- Given 表示言語の明示選択、`ui_locales` ヒント、保存済み設定、対応するブラウザー言語が存在しない
- When Operator が `VITE_DEFAULT_LOCALE` を `ja` に設定してアプリケーションを起動する
- When EndUser が画面を表示する
- Then 画面の文言は `ja` 辞書で表示される

### Example: EX-SYSTEM-005-02 `VITE_DEFAULT_LOCALE` が未設定または未対応値である

- Given 表示言語の明示選択、`ui_locales` ヒント、保存済み設定、対応するブラウザー言語が存在しない
- When Operator が `VITE_DEFAULT_LOCALE` を `ja` に設定してアプリケーションを起動する
- But `VITE_DEFAULT_LOCALE` が未設定または未対応値である
- Then 画面の文言は `FallbackLocale` の `en` 辞書で表示される

## Rule: REQ-SYSTEM-008 OIDC の `ui_locales` ヒントにより表示言語が決まる

### Example: EX-SYSTEM-008-01 通常経路

- Given 未認証セッションで表示言語の明示選択も保存済み設定も存在しない
- When "web-app" として ui_locales "en" で認可リクエストを送信する
- Then ログイン画面の文言は `en` 辞書で表示される

### Example: EX-SYSTEM-008-02 表示言語がすでに明示選択済みである

- Given 未認証セッションで表示言語の明示選択も保存済み設定も存在しない
- When "web-app" として ui_locales "en" で認可リクエストを送信する
- But 表示言語がすでに明示選択済みである
- Then EndUser が表示言語 `ja` を明示的に選択済みである
- And `web-app` として `ui_locales=en` で認可リクエストを送信する
- And ログイン画面の文言は `ja` 辞書で表示される

## Rule: REQ-SYSTEM-009 管理者が選択した表示言語で管理画面が表示される

### Example: EX-SYSTEM-009-01 通常経路

- Given ロールに "admin" を持つ Administrator が認証済みで AdminDashboard を表示している
- When Administrator が表示言語 "en" を選択する
- Then AdminDashboard の文言が en 辞書で表示される

## Rule: REQ-SYSTEM-010 選択した表示言語ですべての UI 画面が描画される

### Example: EX-SYSTEM-010-01 通常経路

- Given 対応する画面へ遷移できる認証状態である
- When EndUser または Administrator が表示言語 "en" を選択する
- When EndUser または Administrator が任意の UI 画面を表示する
- Then 画面、共有シェル、ダイアログ、空状態の ARIA ラベル、状態ラベルが `en` 辞書で表示される
- Then 日時および数値が `en` のフォーマットで表示される

### Example: EX-SYSTEM-010-02 `ja` を選択する

- Given 対応する画面へ遷移できる認証状態である
- When EndUser または Administrator が表示言語 "en" を選択する
- But `ja` を選択する
- Then 同じ要素が `ja` 辞書および `ja` のフォーマットで表示される

### Example: EX-SYSTEM-010-03 翻訳キーが欠落している

- Given 対応する画面へ遷移できる認証状態である
- When EndUser または Administrator が表示言語 "en" を選択する
- When EndUser または Administrator が任意の UI 画面を表示する
- Then 翻訳キーが欠落している
- Then `FallbackLocale`（`en`）の対応するキーを表示する

## Rule: REQ-SYSTEM-011 既知のバックエンドエラーコードは UI で翻訳される

### Example: EX-SYSTEM-011-01 通常経路

- Given UI 操作に対しバックエンドがエラーレスポンスを返す
- When バックエンドが既知の `stable` エラーコードを返す
- Then UI が選択済みの `DisplayLanguage` の辞書にあるエラー文を表示する

### Example: EX-SYSTEM-011-02 エラーコードが未知である、またはバックエンドが任意の `message` か Problem Details だけを返す

- Given UI 操作に対しバックエンドがエラーレスポンスを返す
- When バックエンドが既知の `stable` エラーコードを返す
- But エラーコードが未知である、またはバックエンドが任意の `message` か Problem Details だけを返す
- Then UI は `message`、`error_description`、`detail`、`title` のうち利用可能な人間可読文を英語のまま表示する
- And 有効なエラーレスポンスを受信した場合は通信障害用のフォールバックを表示しない

### Example: EX-SYSTEM-011-03 RFC 9457 Problem Details の `type` が既知の `stable` エラーコードを表す

- Given UI 操作に対しバックエンドがエラーレスポンスを返す
- When バックエンドが既知の `stable` エラーコードを返す
- But RFC 9457 Problem Details の `type` が既知の `stable` エラーコードを表す
- Then UI は `type` の `urn:idmagic:error:` 接尾辞をエラーコードとして解釈する
- And UI が選択済みの `DisplayLanguage` の辞書にあるエラー文を表示する

## Rule: REQ-SYSTEM-013 バックエンド API のエラーは英語で返る

### Example: EX-SYSTEM-013-01 通常経路

- When APIConsumer が不正な JSON を HTTP API に送信する
- Then System は既存のエラーコードと HTTP ステータスを返す
- Then System は英語の `message` を返す
- When OAuth / OIDC のリダイレクトエンドポイントがリクエストを拒否する
- Then System は既存の OAuth エラーコードと英語の `error_description` を返す
- When 未知の内部エラーが発生する
- Then System は既存のエラーコードと HTTP ステータスを維持し、英語のエラー本文を返す
