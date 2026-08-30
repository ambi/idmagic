# System Scenarios

### REQ-SYSTEM-001: Operator は分離された運用資産で SLO を検証する
- ACTOR Operator
- GIVEN API、UI ゲートウェイ、イベントリレーは個別の実行単位としてデプロイされる
- GIVEN `MetricsExposition` の公開範囲は管理ネットワークに制限される
- GIVEN OAuth2/OIDC のサービス目標、母集団、時間窓、除外条件は `docs/capacity.md` に定められている
- GIVEN 各サービス目標は `docs/observability.md` の HTTP RED メトリクスと Prometheus のスクレイプ状態に対応づけられている
- WHEN Operator が環境のオーバーレイを選んで運用マニフェストを適用する
  - ALT PostgreSQL へ到達できない → `ReadinessProbe` は `unavailable` を返し、API は新規トラフィックを受けない → `LivenessProbe` は `healthy` を維持し、依存障害だけでは再起動しない
  - ALT Prometheus Operator が導入されていない → `ServiceMonitor` は適用対象から外し、標準の Prometheus スクレイプ設定で `MetricsExposition` を収集する
- THEN API の生存、受付可否、起動完了の各プローブは、それぞれ `LivenessProbe`、`ReadinessProbe`、`StartupProbe` を呼ぶ
- THEN Prometheus が `MetricsExposition` をスクレイプし、定められた母集団と時間窓で OAuth2/OIDC の可用性、レイテンシー、非 5xx 比率を表示および評価する

### REQ-SYSTEM-002: オーケストレーション用プローブはプロセスのライフサイクルと依存先の状態を区別する
- ACTOR Operator
- WHEN Operator が初期化完了後に生存、受付可否、起動完了の各プローブを呼ぶ
  - ALT 初期化中またはグレースフルドレイン中である → 生存確認は `200 healthy` を維持する → 受付可否または起動完了の確認は 503 を返す
  - ALT 設定済みの永続化依存先へ到達できない → 受付可否の確認は `503 unavailable` を返す → 生存確認は `200 healthy` を維持する
- THEN すべて `200 healthy` を返す

### REQ-SYSTEM-003: 明示的に選択した表示言語でホスト認証画面が描画される
- ACTOR EndUser
- GIVEN 未認証セッションでログイン画面を表示している
- WHEN EndUser が表示言語 "en" を選択する
- THEN ログイン画面の文言が `en` 辞書で表示される
- THEN 選択したロケールがブラウザーに保存され、以後のアクセスで保存済み設定として優先される

### REQ-SYSTEM-004: 未対応のロケールはデフォルトのロケールへフォールバックする
- ACTOR EndUser
- GIVEN ブラウザーの言語設定が `fr` である
- GIVEN 表示言語の明示選択も保存済み設定も存在しない
- WHEN EndUser がログイン画面を表示する
- THEN 画面の文言はデフォルトロケール `en` の辞書で表示される

### REQ-SYSTEM-005: 起動時設定のデフォルトロケールがフォールバックに使われる
- ACTOR Operator
- GIVEN 表示言語の明示選択、`ui_locales` ヒント、保存済み設定、対応するブラウザー言語が存在しない
- WHEN Operator が `VITE_DEFAULT_LOCALE` を `ja` に設定してアプリケーションを起動する
  - ALT `VITE_DEFAULT_LOCALE` が未設定または未対応値である → 画面の文言は `FallbackLocale` の `en` 辞書で表示される
- WHEN EndUser が画面を表示する
- THEN 画面の文言は `ja` 辞書で表示される

### REQ-SYSTEM-006: 起動時設定により Vite 開発サーバー以外でも DemoLoginAffordance が表示される
- ACTOR Operator
- GIVEN Vite 開発サーバーではなくビルド済みのフロントエンドをデプロイしている
- WHEN Operator が `VITE_DEMO_LOGIN_ENABLED` を `true` に設定してビルドする
  - ALT `VITE_DEMO_LOGIN_ENABLED` が未設定または `true` 以外である → `HomePage` は `DemoLoginAffordance` を表示しない
- WHEN EndUser が HomePage を表示する
- THEN HomePage は DemoLoginAffordance を表示する
- WHEN EndUser が DemoLoginAffordance を選択する
  - ALT `development` プロファイルが適用されていない → 既知のデモ資格情報が存在しないため認可に失敗する
- THEN `development` プロファイルが投入したデモユーザーの資格情報で `authorization_code` フローが完了する

### REQ-SYSTEM-007: Vite 開発サーバーでの実行時は設定なしで DemoLoginAffordance が表示される
- ACTOR EndUser
- GIVEN Vite 開発サーバーでフロントエンドを実行している
- WHEN EndUser が HomePage を表示する
- THEN `VITE_DEMO_LOGIN_ENABLED` の設定にかかわらず `HomePage` は `DemoLoginAffordance` を表示する

### REQ-SYSTEM-008: OIDC の `ui_locales` ヒントにより表示言語が決まる
- ACTOR ResourceOwner
- GIVEN 未認証セッションで表示言語の明示選択も保存済み設定も存在しない
- WHEN "web-app" として ui_locales "en" で認可リクエストを送信する
  - ALT 表示言語がすでに明示選択済みである → EndUser が表示言語 `ja` を明示的に選択済みである → `web-app` として `ui_locales=en` で認可リクエストを送信する → ログイン画面の文言は `ja` 辞書で表示される
- THEN ログイン画面の文言は `en` 辞書で表示される

### REQ-SYSTEM-009: 管理者が選択した表示言語で管理画面が表示される
- ACTOR Administrator
- GIVEN ロールに "admin" を持つ Administrator が認証済みで AdminDashboard を表示している
- WHEN Administrator が表示言語 "en" を選択する
- THEN AdminDashboard の文言が en 辞書で表示される

### REQ-SYSTEM-010: 選択した表示言語ですべての UI 画面が描画される
- ACTOR EndUser
- GIVEN 対応する画面へ遷移できる認証状態である
- WHEN EndUser または Administrator が表示言語 "en" を選択する
  - ALT `ja` を選択する → 同じ要素が `ja` 辞書および `ja` の書式で表示される
- WHEN EndUser または Administrator が任意の UI 画面を表示する
- THEN 画面、共有シェル、ダイアログ、空状態の ARIA ラベル、状態ラベルが `en` 辞書で表示される
  - ALT 翻訳キーが欠落している → `FallbackLocale`（`en`）の対応するキーを表示する
- THEN 日時および数値が `en` の書式で表示される

### REQ-SYSTEM-011: 既知のバックエンドエラーコードは UI で翻訳される
- ACTOR EndUser
- GIVEN UI 操作に対しバックエンドがエラーレスポンスを返す
- WHEN バックエンドが既知の `stable` エラーコードを返す
  - ALT エラーコードが未知である、またはバックエンドが任意の `message` か Problem Details だけを返す → UI は `message`、`error_description`、`detail`、`title` のうち利用可能な人間可読文を英語のまま表示する → 有効なエラーレスポンスを受信した場合は通信障害用のフォールバックを表示しない
  - ALT RFC 9457 Problem Details の `type` が既知の `stable` エラーコードを表す → UI は `type` の `urn:idmagic:error:` 接尾辞をエラーコードとして解釈する → UI が選択済みの `DisplayLanguage` の辞書にあるエラー文を表示する
- THEN UI が選択済みの `DisplayLanguage` の辞書にあるエラー文を表示する

### REQ-SYSTEM-012: PostgreSQL クエリの期限は結果の読み取り完了まで維持される
- ACTOR Operator
- GIVEN PostgreSQL の永続化とクエリタイムアウトが設定されている
- WHEN System が共通の永続化アダプターで単一行または複数行のクエリを開始する
- THEN クエリが `Row` または `Rows` を返す
- WHEN 呼び出し側が期限内に `Scan` または反復処理を完了する
  - ALT 結果の読み取り中にクエリタイムアウトの期限へ到達する → 読み取りは `deadline exceeded` で中断される → 結果を閉じると接続とタイムアウトのリソースが解放される
  - ALT 単一行クエリに該当する行が存在しない → `Scan` は `no rows` を返す → `no rows` は正常なクエリ結果として扱われ、サーキットブレーカーの失敗率を増加させない
- THEN 結果が `context canceled` にならず返され、接続が解放される

### REQ-SYSTEM-013: バックエンド API のエラーは英語で返る
- ACTOR APIConsumer
- WHEN APIConsumer が不正な JSON を HTTP API に送信する
- THEN System は既存のエラーコードと HTTP ステータスを返す
- THEN System は英語の `message` を返す
- WHEN OAuth / OIDC のリダイレクトエンドポイントがリクエストを拒否する
- THEN System は既存の OAuth エラーコードと英語の `error_description` を返す
- WHEN 未知の内部エラーが発生する
- THEN System は既存のエラーコードと HTTP ステータスを維持し、英語のエラー本文を返す

### REQ-SYSTEM-014: 非推奨のインターフェースを呼ぶと Deprecation / Sunset ヘッダーが返る
- ACTOR APIConsumer
- GIVEN 安定版のインターフェースに `deprecated_since` が設定されている
- WHEN APIConsumer が非推奨とされたインターフェースを呼び出す
  - ALT インターフェースに `sunset_at` も設定されている → レスポンスに `Sunset` ヘッダーも付与される
  - ALT インターフェースに `deprecated_since` が設定されていない → レスポンスに `Deprecation` ヘッダーは付与されない
- THEN レスポンスに `Deprecation` ヘッダーが付与される

### REQ-SYSTEM-015: 管理コンソールとアカウントポータルは失効セッションから同一画面に復帰する
- ACTOR Administrator
- GIVEN Administrator がファーストパーティーの管理コンソールでアクセストークンを保持している
- GIVEN 保持しているアクセストークンが失効している
- WHEN Administrator が AdminDashboard で管理 API を呼び出す
- THEN API が 401 を返す
- THEN 保持していたアクセストークン、リフレッシュトークン、OIDC コールバックの `state` を破棄する
- THEN 直前の画面への同一オリジン相対の `return_to` を保ったまま再認可を 1 回だけ開始する
  - ALT 再認可から復旧できない → 再ログイン導線を提示する
- THEN 再ログイン完了後に元の AdminDashboard へ復帰する

### REQ-SYSTEM-016: 起動時設定の検証に失敗するとプロセスは部分起動せず集約エラーで停止する
- ACTOR Operator
- GIVEN Operator が環境変数でバックエンドプロセス（`idmagic`、`idmagic-worker`、`idmagic-batch`、`idmagic-seed`）の設定を与える
- GIVEN 製品ビルドが、実行時に選択可能な機能の識別子、版、成熟度、既定の有効化、依存機能、更新方針を閉じた `FeatureRegistry` として持つ
- WHEN プロセスが起動時に `Config` を集約および検証する
  - ALT 必須値が欠落している → 検証は該当キーを含む集約エラーを返す → プロセスはリスナーの待ち受け、永続化依存先への接続、seed の適用など副作用のある初期化を開始せず終了する
  - ALT 値の型または範囲が不正である（数値でない、負の期間など） → 検証は該当キーを含む集約エラーを返す → プロセスは副作用のある初期化を開始せず終了する
  - ALT 相互に矛盾する組み合わせである（`persistence=postgres` なのに DSN が空など） → 検証は該当する組み合わせを含む集約エラーを返す → プロセスは副作用のある初期化を開始せず終了する
  - ALT `FeatureRegistry` に識別子または未版名の重複、存在しない依存、依存循環、実験的機能の既定有効化、非推奨機能の新規既定有効化がある → 検証はすべての registry エラーを返す → プロセスは副作用のある初期化を開始せず終了する
  - ALT `FEATURES_ENABLE` または `FEATURES_DISABLE` が存在しない機能を指すか、同じ機能を両方で指定するか、明示的に無効化した依存を必要とする → 検証はすべての選択エラーを返す → プロセスは副作用のある初期化を開始せず終了する
- THEN 発生したすべての検証エラーが 1 回の起動試行で集約されて報告される
- THEN 検証エラーおよび起動ログは、シークレットに分類された値（DSN、SMTP 資格情報、API キーなど）を含まない
- WHEN すべての検証を通過する
- THEN プロセスは明示指定、既定値、依存閉包から決定した有効機能と検証済みの `Config` を用いて初期化を完了する
- THEN 明示的に有効化した `experimental` または `preview` の機能と、有効な `deprecated` の機能は、識別子と成熟度だけを秘密情報を含まない起動警告へ記録する

### REQ-SYSTEM-017: ConfigurationReference は起動時設定の定義から生成され乖離を検出できる
- ACTOR Operator
- GIVEN バックエンドプロセスの起動時設定が `Config` として一箇所で定義されている
- WHEN ConfigurationReference を生成する
- THEN 生成物は設定可能な各キーについて、キー名、値の型、デフォルト値、必須か、読むプロセス、説明を含む
- THEN 生成物は `FeatureRegistry` に登録された各機能について、識別子、版、成熟度、既定の有効化、依存機能、更新方針を含み、registry が空なら選択可能な機能が無いことを示す
- THEN 生成物はシークレットに分類されたキーの値を含まず、シークレットであることだけを示す
- WHEN 生成物と `Config` の定義を突き合わせる
  - ALT 生成物が定義と一致しない → 突き合わせは失敗し、乖離したキーを報告する
- THEN Operator は `Config` の実装を読まずに設定可能なすべてのキーを参照できる
- WHEN プロセスの `/health` を読む
- THEN レスポンスはメタデータ形式の版と、有効な各機能の識別子、版、成熟度、更新方針を含み、シークレットまたは無効な機能を含まない
