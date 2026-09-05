# Feature: System Scenarios

## Rule: REQ-SYSTEM-001 Operator は分離された運用資産で SLO を検証する

Primary actor: `Operator`

### Example: EX-SYSTEM-001-01 通常経路

- Given API、UI ゲートウェイ、イベントリレーは個別の実行単位としてデプロイされる
- And `MetricsExposition` の公開範囲は管理ネットワークに制限される
- And OAuth2/OIDC のサービス目標、母集団、時間窓、除外条件は `docs/capacity.md` に定められている
- And 各サービス目標は `docs/observability.md` の HTTP RED メトリクスと Prometheus のスクレイプ状態に対応づけられている
- When Operator が環境のオーバーレイを選んで運用マニフェストを適用する
- Then API の生存、受付可否、起動完了の各プローブは、それぞれ `LivenessProbe`、`ReadinessProbe`、`StartupProbe` を呼ぶ
- Then Prometheus が `MetricsExposition` をスクレイプし、定められた母集団と時間窓で OAuth2/OIDC の可用性、レイテンシー、非 5xx 比率を表示および評価する

### Example: EX-SYSTEM-001-02 PostgreSQL へ到達できない

- Given API、UI ゲートウェイ、イベントリレーは個別の実行単位としてデプロイされる
- And `MetricsExposition` の公開範囲は管理ネットワークに制限される
- And OAuth2/OIDC のサービス目標、母集団、時間窓、除外条件は `docs/capacity.md` に定められている
- And 各サービス目標は `docs/observability.md` の HTTP RED メトリクスと Prometheus のスクレイプ状態に対応づけられている
- When Operator が環境のオーバーレイを選んで運用マニフェストを適用する
- But PostgreSQL へ到達できない
- Then `ReadinessProbe` は `unavailable` を返し、API は新規トラフィックを受けない
- And `LivenessProbe` は `healthy` を維持し、依存障害だけでは再起動しない

### Example: EX-SYSTEM-001-03 Prometheus Operator が導入されていない

- Given API、UI ゲートウェイ、イベントリレーは個別の実行単位としてデプロイされる
- And `MetricsExposition` の公開範囲は管理ネットワークに制限される
- And OAuth2/OIDC のサービス目標、母集団、時間窓、除外条件は `docs/capacity.md` に定められている
- And 各サービス目標は `docs/observability.md` の HTTP RED メトリクスと Prometheus のスクレイプ状態に対応づけられている
- When Operator が環境のオーバーレイを選んで運用マニフェストを適用する
- But Prometheus Operator が導入されていない
- Then `ServiceMonitor` は適用対象から外し、標準の Prometheus スクレイプ設定で `MetricsExposition` を収集する

## Rule: REQ-SYSTEM-002 オーケストレーション用プローブはプロセスのライフサイクルと依存先の状態を区別する

Primary actor: `Operator`

### Example: EX-SYSTEM-002-01 通常経路

- When Operator が初期化完了後に生存、受付可否、起動完了の各プローブを呼ぶ
- Then すべて `200 healthy` を返す

### Example: EX-SYSTEM-002-02 初期化中またはグレースフルドレイン中である

- When Operator が初期化完了後に生存、受付可否、起動完了の各プローブを呼ぶ
- But 初期化中またはグレースフルドレイン中である
- Then 生存確認は `200 healthy` を維持する
- And 受付可否または起動完了の確認は 503 を返す

### Example: EX-SYSTEM-002-03 設定済みの永続化依存先へ到達できない

- When Operator が初期化完了後に生存、受付可否、起動完了の各プローブを呼ぶ
- But 設定済みの永続化依存先へ到達できない
- Then 受付可否の確認は `503 unavailable` を返す
- And 生存確認は `200 healthy` を維持する

## Rule: REQ-SYSTEM-003 明示的に選択した表示言語でホスト認証画面が描画される

Primary actor: `EndUser`

### Example: EX-SYSTEM-003-01 通常経路

- Given 未認証セッションでログイン画面を表示している
- When EndUser が表示言語 "en" を選択する
- Then ログイン画面の文言が `en` 辞書で表示される
- Then 選択したロケールがブラウザーに保存され、以後のアクセスで保存済み設定として優先される

## Rule: REQ-SYSTEM-004 未対応のロケールはデフォルトのロケールへフォールバックする

Primary actor: `EndUser`

### Example: EX-SYSTEM-004-01 通常経路

- Given ブラウザーの言語設定が `fr` である
- And 表示言語の明示選択も保存済み設定も存在しない
- When EndUser がログイン画面を表示する
- Then 画面の文言はデフォルトロケール `en` の辞書で表示される

## Rule: REQ-SYSTEM-005 起動時設定のデフォルトロケールがフォールバックに使われる

Primary actor: `Operator`

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

## Rule: REQ-SYSTEM-006 起動時設定により Vite 開発サーバー以外でも DemoLoginAffordance が表示される

Primary actor: `Operator`

### Example: EX-SYSTEM-006-01 通常経路

- Given Vite 開発サーバーではなくビルド済みのフロントエンドをデプロイしている
- When Operator が `VITE_DEMO_LOGIN_ENABLED` を `true` に設定してビルドする
- When EndUser が HomePage を表示する
- Then HomePage は DemoLoginAffordance を表示する
- When EndUser が DemoLoginAffordance を選択する
- Then `development` プロファイルが投入したデモユーザーの資格情報で `authorization_code` フローが完了する

### Example: EX-SYSTEM-006-02 `VITE_DEMO_LOGIN_ENABLED` が未設定または `true` 以外である

- Given Vite 開発サーバーではなくビルド済みのフロントエンドをデプロイしている
- When Operator が `VITE_DEMO_LOGIN_ENABLED` を `true` に設定してビルドする
- But `VITE_DEMO_LOGIN_ENABLED` が未設定または `true` 以外である
- Then `HomePage` は `DemoLoginAffordance` を表示しない

### Example: EX-SYSTEM-006-03 `development` プロファイルが適用されていない

- Given Vite 開発サーバーではなくビルド済みのフロントエンドをデプロイしている
- When Operator が `VITE_DEMO_LOGIN_ENABLED` を `true` に設定してビルドする
- When EndUser が HomePage を表示する
- Then HomePage は DemoLoginAffordance を表示する
- When EndUser が DemoLoginAffordance を選択する
- But `development` プロファイルが適用されていない
- Then 既知のデモ資格情報が存在しないため認可に失敗する

## Rule: REQ-SYSTEM-007 Vite 開発サーバーでの実行時は設定なしで DemoLoginAffordance が表示される

Primary actor: `EndUser`

### Example: EX-SYSTEM-007-01 通常経路

- Given Vite 開発サーバーでフロントエンドを実行している
- When EndUser が HomePage を表示する
- Then `VITE_DEMO_LOGIN_ENABLED` の設定にかかわらず `HomePage` は `DemoLoginAffordance` を表示する

## Rule: REQ-SYSTEM-008 OIDC の `ui_locales` ヒントにより表示言語が決まる

Primary actor: `ResourceOwner`

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

Primary actor: `Administrator`

### Example: EX-SYSTEM-009-01 通常経路

- Given ロールに "admin" を持つ Administrator が認証済みで AdminDashboard を表示している
- When Administrator が表示言語 "en" を選択する
- Then AdminDashboard の文言が en 辞書で表示される

## Rule: REQ-SYSTEM-010 選択した表示言語ですべての UI 画面が描画される

Primary actor: `EndUser`

### Example: EX-SYSTEM-010-01 通常経路

- Given 対応する画面へ遷移できる認証状態である
- When EndUser または Administrator が表示言語 "en" を選択する
- When EndUser または Administrator が任意の UI 画面を表示する
- Then 画面、共有シェル、ダイアログ、空状態の ARIA ラベル、状態ラベルが `en` 辞書で表示される
- Then 日時および数値が `en` の書式で表示される

### Example: EX-SYSTEM-010-02 `ja` を選択する

- Given 対応する画面へ遷移できる認証状態である
- When EndUser または Administrator が表示言語 "en" を選択する
- But `ja` を選択する
- Then 同じ要素が `ja` 辞書および `ja` の書式で表示される

### Example: EX-SYSTEM-010-03 翻訳キーが欠落している

- Given 対応する画面へ遷移できる認証状態である
- When EndUser または Administrator が表示言語 "en" を選択する
- When EndUser または Administrator が任意の UI 画面を表示する
- Then 翻訳キーが欠落している
- Then `FallbackLocale`（`en`）の対応するキーを表示する

## Rule: REQ-SYSTEM-011 既知のバックエンドエラーコードは UI で翻訳される

Primary actor: `EndUser`

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

## Rule: REQ-SYSTEM-012 PostgreSQL クエリの期限は結果の読み取り完了まで維持される

Primary actor: `Operator`

### Example: EX-SYSTEM-012-01 通常経路

- Given PostgreSQL の永続化とクエリタイムアウトが設定されている
- When System が共通の永続化アダプターで単一行または複数行のクエリを開始する
- Then クエリが `Row` または `Rows` を返す
- When 呼び出し側が期限内に `Scan` または反復処理を完了する
- Then 結果が `context canceled` にならず返され、接続が解放される

### Example: EX-SYSTEM-012-02 結果の読み取り中にクエリタイムアウトの期限へ到達する

- Given PostgreSQL の永続化とクエリタイムアウトが設定されている
- When System が共通の永続化アダプターで単一行または複数行のクエリを開始する
- Then クエリが `Row` または `Rows` を返す
- When 呼び出し側が期限内に `Scan` または反復処理を完了する
- But 結果の読み取り中にクエリタイムアウトの期限へ到達する
- Then 読み取りは `deadline exceeded` で中断される
- And 結果を閉じると接続とタイムアウトのリソースが解放される

### Example: EX-SYSTEM-012-03 単一行クエリに該当する行が存在しない

- Given PostgreSQL の永続化とクエリタイムアウトが設定されている
- When System が共通の永続化アダプターで単一行または複数行のクエリを開始する
- Then クエリが `Row` または `Rows` を返す
- When 呼び出し側が期限内に `Scan` または反復処理を完了する
- But 単一行クエリに該当する行が存在しない
- Then `Scan` は `no rows` を返す
- And `no rows` は正常なクエリ結果として扱われ、サーキットブレーカーの失敗率を増加させない

## Rule: REQ-SYSTEM-013 バックエンド API のエラーは英語で返る

Primary actor: `APIConsumer`

### Example: EX-SYSTEM-013-01 通常経路

- When APIConsumer が不正な JSON を HTTP API に送信する
- Then System は既存のエラーコードと HTTP ステータスを返す
- Then System は英語の `message` を返す
- When OAuth / OIDC のリダイレクトエンドポイントがリクエストを拒否する
- Then System は既存の OAuth エラーコードと英語の `error_description` を返す
- When 未知の内部エラーが発生する
- Then System は既存のエラーコードと HTTP ステータスを維持し、英語のエラー本文を返す

## Rule: REQ-SYSTEM-014 非推奨のインターフェースを呼ぶと Deprecation / Sunset ヘッダーが返る

Primary actor: `APIConsumer`

### Example: EX-SYSTEM-014-01 通常経路

- Given 安定版のインターフェースに `deprecated_since` が設定されている
- When APIConsumer が非推奨とされたインターフェースを呼び出す
- Then レスポンスに `Deprecation` ヘッダーが付与される

### Example: EX-SYSTEM-014-02 インターフェースに `sunset_at` も設定されている

- Given 安定版のインターフェースに `deprecated_since` が設定されている
- When APIConsumer が非推奨とされたインターフェースを呼び出す
- But インターフェースに `sunset_at` も設定されている
- Then レスポンスに `Sunset` ヘッダーも付与される

### Example: EX-SYSTEM-014-03 インターフェースに `deprecated_since` が設定されていない

- Given 安定版のインターフェースに `deprecated_since` が設定されている
- When APIConsumer が非推奨とされたインターフェースを呼び出す
- But インターフェースに `deprecated_since` が設定されていない
- Then レスポンスに `Deprecation` ヘッダーは付与されない

## Rule: REQ-SYSTEM-015 管理コンソールとアカウントポータルは失効セッションから同一画面に復帰する

Primary actor: `Administrator`

### Example: EX-SYSTEM-015-01 通常経路

- Given Administrator がファーストパーティーの管理コンソールでアクセストークンを保持している
- And 保持しているアクセストークンが失効している
- When Administrator が AdminDashboard で管理 API を呼び出す
- Then API が 401 を返す
- Then 保持していたアクセストークン、リフレッシュトークン、OIDC コールバックの `state` を破棄する
- Then 直前の画面への同一オリジン相対の `return_to` を保ったまま再認可を 1 回だけ開始する
- Then 再ログイン完了後に元の AdminDashboard へ復帰する

### Example: EX-SYSTEM-015-02 再認可から復旧できない

- Given Administrator がファーストパーティーの管理コンソールでアクセストークンを保持している
- And 保持しているアクセストークンが失効している
- When Administrator が AdminDashboard で管理 API を呼び出す
- Then API が 401 を返す
- Then 保持していたアクセストークン、リフレッシュトークン、OIDC コールバックの `state` を破棄する
- Then 再認可から復旧できない
- Then 再ログイン導線を提示する

## Rule: REQ-SYSTEM-016 起動時設定の検証に失敗するとプロセスは部分起動せず集約エラーで停止する

Primary actor: `Operator`

### Example: EX-SYSTEM-016-01 通常経路

- Given Operator が環境変数でバックエンドプロセス（`idmagic`、`idmagic-worker`、`idmagic-batch`、`idmagic-seed`）の設定を与える
- And 製品ビルドが、実行時に選択可能な機能の識別子、版、成熟度、既定の有効化、依存機能、更新方針を閉じた `FeatureRegistry` として持つ
- When プロセスが起動時に `Config` を集約および検証する
- Then 発生したすべての検証エラーが 1 回の起動試行で集約されて報告される
- Then 検証エラーおよび起動ログは、シークレットに分類された値（DSN、SMTP 資格情報、API キーなど）を含まない
- When すべての検証を通過する
- Then プロセスは明示指定、既定値、依存閉包から決定した有効機能と検証済みの `Config` を用いて初期化を完了する
- Then 明示的に有効化した `experimental` または `preview` の機能と、有効な `deprecated` の機能は、識別子と成熟度だけを秘密情報を含まない起動警告へ記録する

### Example: EX-SYSTEM-016-02 必須値が欠落している

- Given Operator が環境変数でバックエンドプロセス（`idmagic`、`idmagic-worker`、`idmagic-batch`、`idmagic-seed`）の設定を与える
- And 製品ビルドが、実行時に選択可能な機能の識別子、版、成熟度、既定の有効化、依存機能、更新方針を閉じた `FeatureRegistry` として持つ
- When プロセスが起動時に `Config` を集約および検証する
- But 必須値が欠落している
- Then 検証は該当キーを含む集約エラーを返す
- And プロセスはリスナーの待ち受け、永続化依存先への接続、seed の適用など副作用のある初期化を開始せず終了する

### Example: EX-SYSTEM-016-03 値の型または範囲が不正である（数値でない、負の期間など）

- Given Operator が環境変数でバックエンドプロセス（`idmagic`、`idmagic-worker`、`idmagic-batch`、`idmagic-seed`）の設定を与える
- And 製品ビルドが、実行時に選択可能な機能の識別子、版、成熟度、既定の有効化、依存機能、更新方針を閉じた `FeatureRegistry` として持つ
- When プロセスが起動時に `Config` を集約および検証する
- But 値の型または範囲が不正である（数値でない、負の期間など）
- Then 検証は該当キーを含む集約エラーを返す
- And プロセスは副作用のある初期化を開始せず終了する

### Example: EX-SYSTEM-016-04 相互に矛盾する組み合わせである（`persistence=postgres` なのに DSN が空など）

- Given Operator が環境変数でバックエンドプロセス（`idmagic`、`idmagic-worker`、`idmagic-batch`、`idmagic-seed`）の設定を与える
- And 製品ビルドが、実行時に選択可能な機能の識別子、版、成熟度、既定の有効化、依存機能、更新方針を閉じた `FeatureRegistry` として持つ
- When プロセスが起動時に `Config` を集約および検証する
- But 相互に矛盾する組み合わせである（`persistence=postgres` なのに DSN が空など）
- Then 検証は該当する組み合わせを含む集約エラーを返す
- And プロセスは副作用のある初期化を開始せず終了する

### Example: EX-SYSTEM-016-05 `FeatureRegistry` に識別子または未版名の重複、存在しない依存、依存循環、実験的機能の既定有効化、非推奨機能の新規既定有効化がある

- Given Operator が環境変数でバックエンドプロセス（`idmagic`、`idmagic-worker`、`idmagic-batch`、`idmagic-seed`）の設定を与える
- And 製品ビルドが、実行時に選択可能な機能の識別子、版、成熟度、既定の有効化、依存機能、更新方針を閉じた `FeatureRegistry` として持つ
- When プロセスが起動時に `Config` を集約および検証する
- But `FeatureRegistry` に識別子または未版名の重複、存在しない依存、依存循環、実験的機能の既定有効化、非推奨機能の新規既定有効化がある
- Then 検証はすべての registry エラーを返す
- And プロセスは副作用のある初期化を開始せず終了する

### Example: EX-SYSTEM-016-06 `FEATURES_ENABLE` または `FEATURES_DISABLE` が存在しない機能を指すか、同じ機能を両方で指定するか、明示的に無効化した依存を必要とする

- Given Operator が環境変数でバックエンドプロセス（`idmagic`、`idmagic-worker`、`idmagic-batch`、`idmagic-seed`）の設定を与える
- And 製品ビルドが、実行時に選択可能な機能の識別子、版、成熟度、既定の有効化、依存機能、更新方針を閉じた `FeatureRegistry` として持つ
- When プロセスが起動時に `Config` を集約および検証する
- But `FEATURES_ENABLE` または `FEATURES_DISABLE` が存在しない機能を指すか、同じ機能を両方で指定するか、明示的に無効化した依存を必要とする
- Then 検証はすべての選択エラーを返す
- And プロセスは副作用のある初期化を開始せず終了する

## Rule: REQ-SYSTEM-017 ConfigurationReference は起動時設定の定義から生成され乖離を検出できる

Primary actor: `Operator`

### Example: EX-SYSTEM-017-01 通常経路

- Given バックエンドプロセスの起動時設定が `Config` として一箇所で定義されている
- When ConfigurationReference を生成する
- Then 生成物は設定可能な各キーについて、キー名、値の型、デフォルト値、必須か、読むプロセス、説明を含む
- Then 生成物は `FeatureRegistry` に登録された各機能について、識別子、版、成熟度、既定の有効化、依存機能、更新方針を含み、registry が空なら選択可能な機能が無いことを示す
- Then 生成物はシークレットに分類されたキーの値を含まず、シークレットであることだけを示す
- When 生成物と `Config` の定義を突き合わせる
- Then Operator は `Config` の実装を読まずに設定可能なすべてのキーを参照できる
- When プロセスの `/health` を読む
- Then レスポンスはメタデータ形式の版と、有効な各機能の識別子、版、成熟度、更新方針を含み、シークレットまたは無効な機能を含まない

### Example: EX-SYSTEM-017-02 生成物が定義と一致しない

- Given バックエンドプロセスの起動時設定が `Config` として一箇所で定義されている
- When ConfigurationReference を生成する
- Then 生成物は設定可能な各キーについて、キー名、値の型、デフォルト値、必須か、読むプロセス、説明を含む
- Then 生成物は `FeatureRegistry` に登録された各機能について、識別子、版、成熟度、既定の有効化、依存機能、更新方針を含み、registry が空なら選択可能な機能が無いことを示す
- Then 生成物はシークレットに分類されたキーの値を含まず、シークレットであることだけを示す
- When 生成物と `Config` の定義を突き合わせる
- But 生成物が定義と一致しない
- Then 突き合わせは失敗し、乖離したキーを報告する

## Rule: REQ-SYSTEM-018 飽和した API プロセスは優先度の低い要求から拒否する

Primary actor: `APIConsumer`

### Example: EX-SYSTEM-018-01 通常経路

- Given 登録済みのすべての経路が `interactive_auth`、`management`、`management_bulk`、`infrastructure` のいずれか 1 つの優先度クラスに分類されている
- And 起動時設定が優先度クラスごとの同時実行の入場上限を持ち、`management_bulk` の上限は `management` の上限以下、`management` の上限はプロセス全体の上限以下である
- When 実行中の要求数が `management_bulk` の上限に達している状態で、APIConsumer が `management_bulk` の経路へ要求を送る
- Then System は `Retry-After` と `urn:idmagic:error:service_overloaded` の Problem Details を伴う 503 を返す
- Then 要求はハンドラーへ到達せず、永続状態もドメインイベントも変化しない
- When 同じ状態で APIConsumer が `interactive_auth` の経路へ要求を送る
- Then 要求はハンドラーへ渡り、通常どおり処理される
- When APIConsumer が `infrastructure` に分類された経路へ要求を送る
- Then System は実行中の要求数にかかわらず拒否せず、要求はハンドラーへ渡る
- When 実行中の要求数がどの入場上限にも達していない
- Then System はどの優先度クラスの要求も拒否しない

### Example: EX-SYSTEM-018-02 実行中の要求数が `management_bulk` の上限に達していない

- Given 登録済みのすべての経路が `interactive_auth`、`management`、`management_bulk`、`infrastructure` のいずれか 1 つの優先度クラスに分類されている
- And 起動時設定が優先度クラスごとの同時実行の入場上限を持ち、`management_bulk` の上限は `management` の上限以下、`management` の上限はプロセス全体の上限以下である
- When 実行中の要求数が `management_bulk` の上限に達している状態で、APIConsumer が `management_bulk` の経路へ要求を送る
- But 実行中の要求数が `management_bulk` の上限に達していない
- Then 要求はハンドラーへ渡り、通常どおり処理される

### Example: EX-SYSTEM-018-03 実行中の要求数がプロセス全体の上限にも達している

- Given 登録済みのすべての経路が `interactive_auth`、`management`、`management_bulk`、`infrastructure` のいずれか 1 つの優先度クラスに分類されている
- And 起動時設定が優先度クラスごとの同時実行の入場上限を持ち、`management_bulk` の上限は `management` の上限以下、`management` の上限はプロセス全体の上限以下である
- When 実行中の要求数が `management_bulk` の上限に達している状態で、APIConsumer が `management_bulk` の経路へ要求を送る
- Then System は `Retry-After` と `urn:idmagic:error:service_overloaded` の Problem Details を伴う 503 を返す
- Then 要求はハンドラーへ到達せず、永続状態もドメインイベントも変化しない
- When 同じ状態で APIConsumer が `interactive_auth` の経路へ要求を送る
- But 実行中の要求数がプロセス全体の上限にも達している
- Then System は同じ 503 で拒否し、要求はハンドラーへ到達しないので状態を部分的に更新しない

## Rule: REQ-SYSTEM-019 RoutePriorityReference は分類の定義から生成され乖離を検出できる

Primary actor: `Operator`

### Example: EX-SYSTEM-019-01 通常経路

- Given 登録済みの各経路の優先度クラスが、経路の登録と同じ場所に一箇所で定義されている
- When RoutePriorityReference を生成する
- Then 生成物は組み立て済みの経路それぞれについて、経路パターン、メソッド、属する優先度クラスを示す
- Then 生成物は優先度クラスごとに、対応する縮退のステージと、その上限を与える起動時設定のキーを示す
- When 生成物と分類の定義を突き合わせる
- Then Operator は分類の実装を読まずに、どの経路がどの優先度クラスに属するかを参照できる

### Example: EX-SYSTEM-019-02 生成物が定義と一致しない

- Given 登録済みの各経路の優先度クラスが、経路の登録と同じ場所に一箇所で定義されている
- When RoutePriorityReference を生成する
- Then 生成物は組み立て済みの経路それぞれについて、経路パターン、メソッド、属する優先度クラスを示す
- Then 生成物は優先度クラスごとに、対応する縮退のステージと、その上限を与える起動時設定のキーを示す
- When 生成物と分類の定義を突き合わせる
- But 生成物が定義と一致しない
- Then 突き合わせは失敗し、再生成すべきことを報告する
