# System Internals

## BackendErrorResponse
バックエンドの HTTP API とプロトコルエンドポイントが返す、共通のエラー契約である。`message`、`error_description`、`detail`、プレーンテキスト本文はいずれも `BackendErrorText` であり、`DisplayLanguage` にかかわらず英語で固定する。RFC 9457 Problem Details では、`type` の `urn:idmagic:error:` に続く部分を安定したエラーコードとして解釈できる。UI は `detail` または `title` を、人間が読めるフォールバックとして利用できる。この契約は、個別エンドポイントのエラーコードや HTTP ステータスを変更しない。
- Result invariant: text_is_english(output.message)
- Result invariant: text_is_english(output.error_description)
- Result invariant: text_is_english(output.detail)

## Deployment boundary

React と Go は別々のビルド成果物であり、別々のサービスである。

```text
Browser
  |
  | same origin
  v
Gateway / static server (Caddy, Nginx, CDN + proxy, etc.)
  |-- /login, /consent, /device, /status, /admin/* -> React SPA
  `-- /api/* and OAuth/OIDC endpoints                -> Go
```

Caddy は参照用の設定であり、必須のランタイムではない。同一オリジンの境界、TLS、ヘッダー、経路制御の契約を保つゲートウェイなら置き換えられる。

## Authorization transaction

Go サービスは OAuth の認可要求全体をサーバー側に保持する。ブラウザーには、短命な内部 UUID だけを `HttpOnly`、`SameSite=Lax`、HTTPS では `Secure` のトランザクション Cookie に保存する。認可要求の内容は HTML、URL、JavaScript から読める状態に含めない。

SPA は `GET /api/auth/transaction` を呼び、画面の種類、クライアント名、要求されたスコープなど表示用のデータだけを取得する。ログインと同意のコマンドは cookie からトランザクションを解決する。

## Browser protections

- セッションと認可トランザクションの Cookie は `HttpOnly` とする。
- 状態を変更する UI API は、二重送信方式の CSRF Cookie と `X-CSRF-Token` ヘッダーを要求する。
- 状態を変更するブラウザー API は、設定済みの公開発行者と一致する `Origin` ヘッダーを要求する。
- 同意処理では、現在のログインセッションの subject が認可トランザクションの subject と一致することを検証する。
- 認可リクエストは 10 分で期限切れとなり、完了したリクエストは再利用できない。
- OAuth のリダイレクト URI、PKCE 値、スコープ、クライアント識別子はサーバー側の状態から読み取る。
- UI API のレスポンスには `Cache-Control: no-store` を付け、資格情報や内部のリクエスト ID は返さない。

## API boundary

ブラウザー向け認証 API は `/api/auth/*` に置き、OAuth / OIDC のプロトコルエンドポイントは標準パスを維持する。管理 API は `/api/admin/*`、セルフサービス API は `/api/account/*` に置き、どちらもログイントランザクション API とは独立した明示的な認可ポリシーを使う。

## Admin console and account portal as OIDC RPs

管理コンソール (`/admin/*`) とアカウントポータル (`/account/*`) は IdP 自身の OIDC RP であり、IdP の `/authorize` と `/token` に対する `authorization_code` + PKCE で認証する。管理用の `…0022` とアカウント用の `…0023` という固定 UUID の `client_id` を持つファーストパーティーのパブリッククライアントとして登録し、`src/api/oidc.ts` と bootstrap seed に反映する。リソース所有者は IdP 自身のユーザーなので、同意画面は省略する。

純粋な SPA RP なので、アクセストークンはブラウザーの `sessionStorage` に保持し、`Authorization: Bearer` として `/api/{admin,account}/*` へ送る。バックエンドは RFC 9068 のリソースサーバーとして検証する。JavaScript からトークンへアクセスできる設計上のリスクは、600 秒の短い有効期間、`Cache-Control: no-store`、URL、ログ、DOM にトークンを置かないことで抑える。OIDC クライアントや鍵の設定不備で管理者が復旧不能にならないよう、ファーストパーティーのセッションログイン (`POST /api/auth/login`) は緊急経路として残す。

## Client-side routing

SPA は TanStack Router を使い、`src/routes/` 配下のファイルベースルーティングでクライアント側のナビゲーションを行う。Vite のルータープラグインが `src/routeTree.gen.ts` を生成し、ルートローダーとコンポーネントを自動的にコード分割する。ルートファイルはリクエストパスの構造に従う。`admin/route.tsx` と `account/route.tsx` は `<Outlet>` を描画する薄いレイアウトルートとし、`admin/index.tsx`、`account/index.tsx`、末端のルートファイルは自身の `loader`、API リクエスト、ページコンポーネント、パスパラメーターを所有する。一覧ページを介さず描画する詳細ページには TanStack Router の末尾アンダースコア規約を使い、`/admin/users/$sub` は `admin/users_/$sub.tsx` で表す。`-` で始まるファイルはルート固有の補助ファイルとし、ルート生成から除外する。管理画面とアカウント画面内のナビゲーションには `<Link>` を使い、ページ移動のたびに文書全体や全ページのデータを再読み込みせず、対象ルートのローダーだけを実行する。OIDC のログインガード（`ensureLoggedIn`）はローダー内で動き、初回読み込みとアプリケーション内のナビゲーションの両方に適用する。ログイン、同意、コールバック、OIDC リダイレクトからなる認証フローの遷移は、ページ全体のナビゲーションとする。E2E テスト向けに、描画済みページの種類を `<meta name="idmagic:page">` で DOM へ示す。

## Design Guidelines

- **信頼できる情報を示す**: 落ち着いた配色、明確なサービス識別、現在の操作とセキュリティ状態の説明により、利用者が認証の可否を判断できるようにする。
- **重要な情報を先に示す**: 明確な情報階層により、ページの題名、要求元、共有する情報、次の操作、取り消し方を最初に示す。
- **重要な操作を単純にする**: 画面ごとの主な操作を 1 個に絞り、拒否や取り消しとは視覚的に区別する。UI 固有の理由で OAuth / OIDC フォームの名前、送信値、遷移の契約を変えない。
- **アクセシビリティを標準とする**: キーボード操作、見えるフォーカス表示、十分な色のコントラスト、明示的なラベル、適切な `aria-*` 属性、動きを減らす設定を尊重するアニメーションに対応する。
- **業務利用に適した情報密度を保つ**: 過剰なアニメーションや装飾を避け、一貫した余白、タイポグラフィ、ボーダー、状態色で情報を構造化する。
- **情報を失わずレスポンシブにする**: デスクトップでは補足情報を表示し、モバイルではサービスの識別や安全上の警告を省かず認証操作を優先する。
- **共有コンポーネントで一貫させる**: Tailwind CSS、Radix UI、shadcn/ui に沿ったローカルコンポーネントを使う。色、角丸、フォーカスリング、無効状態をその場ごとに実装しない。

## Admin Console Policy

管理コンソールの情報設計は、Keycloak、Okta、Google Cloud IAM のようなディレクトリ中心のシステムを参考にする。左側のナビゲーションサイドバーで管理対象を識別し、検索、状態、主な権限を情報密度の高い表で表示し、同じ文脈で詳細と変更手段を示す。削除や無効化などの破壊的な操作は、通常の読み取り専用画面から視覚的に分離する。

- **表を操作の中心にする**: 一覧ビューを主な作業場所とし、検索、絞り込み、状態、MFA 設定、ロールを一目で確認できるようにする。
- **変更前に確認する**: 変更を確定する前に、詳細ペインでプリンシパル ID、認証状態、割り当て済みの権限を確認できるようにする。
- **権限を明示的に変更する**: 機微なロールの変更ではインライン編集を避け、確定前に追加と削除の差分を表示する専用の設定画面を使う。
- **危険な操作を見えるようにする**: 誤操作を防ぐため、危険な操作を明確な説明と適切な警告色で強調する。
- **資格情報を安全に扱う**: クライアントシークレットは作成時に 1 回だけ表示する。クライアントを削除するときは、影響するシステムを確認した後に再確認する。
- **拡張できる構造にする**: グループ、アプリケーション、監査ログなど将来のモジュールを収容できるようナビゲーションを構成する。未実装の機能を操作できるように見せない。
- **レイアウトを一貫させる**: `AdminShell` を使い、ヘッダー、サイドバー、パンくずリスト、コンテンツ幅、操作の配置をコンソール全体で統一する。
- **未認証の直接リンクを扱う**: `/admin/*` への未認証の直接リクエストは `/login` へ送り、ログイン成功後に元の対象へ戻す。リダイレクト先は現在の realm の `/admin` パスに制限する。

*References:*
- [Keycloak Server Administration Guide](https://www.keycloak.org/docs/latest/server_admin/)
- [Okta Manage users](https://help.okta.com/en-us/content/topics/users-groups-profiles/usgp-people.htm)
- [Google Cloud IAM access management](https://cloud.google.com/iam/docs/granting-changing-revoking-access)

## UI Library Selection

UI の基盤は、複雑な組み込み済みテーマに依存せず、アクセシビリティと設計の一貫性を両立する。

| Library | Role | Selection Rationale |
| --- | --- | --- |
| React + TypeScript | UI と型安全なビュー | 単純なログイン画面から管理コンソールまで、明確なコンポーネント境界と状態管理を保つ。 |
| Vite | 開発サーバーと本番ビルド | API ゲートウェイや CDN から配信できる静的バンドルを高速かつ単純に生成する。 |
| Tailwind CSS | デザイントークンとスタイル | 企業のブランディング制御を保ちながら、状態、レスポンシブレイアウト、アクセシビリティのスタイルを一貫させる。 |
| Radix UI | アクセシビリティを備えたヘッドレス部品 | 見た目から独立したキーボード操作と ARIA 準拠を提供する。 |
| ローカルコンポーネント（shadcn/ui のレイアウト） | ボタン、入力、ラベル、カード、アラート | 監査とカスタマイズを容易にし、実行時依存の負荷を減らすため、リポジトリ内で保守する。 |
| TanStack Router | 型安全なルーティング | Go バックエンドのページメタデータを対象の UI ビューへ安全に変換する。 |
| TanStack Table | 管理用データグリッド | 並び替え、絞り込み、ページネーションのロジックを UI の表示から分離する。ユーザーとクライアントの表に使用する。 |
| Tabler Icons | ベクターアイコン | 単なる装飾ではなく、状態と操作の視覚的な補助として、一貫した線の太さと豊富なアイコンを提供する。 |
| Class Variance Authority / Clsx / Tailwind Merge | クラスの統合 | 型安全なスタイルのバリエーションと、競合する Tailwind クラスの実行時統合を提供する。 |
| Biome | リンターとフォーマッター | 構文、スタイル、コード品質の指針を高速に自動適用する。 |

優先順位はアクセシビリティ、バンドルサイズ、保守性、設計の所有権、API 契約の維持である。既存ツールで具体的な要件を満たせない場合にだけ、新しいライブラリを導入する。

## UI navigation and consistency policy

管理コンソールとアカウントポータルには、次の UI とナビゲーションの指針を共通して適用する。Entra フェデレーションと外部アイデンティティプロバイダー (`/admin/identity-providers`) も対象とする。

1. **詳細を確認してから編集する**
   - リソースの作成や編集では、読み取り専用の詳細ビューと書き込み用の編集ビューを分ける。
   - 最初にリソース設定の読み取り専用の詳細ビューを示し、明示的な「編集」ボタンから専用の編集ルート（`/admin/users/$id/edit` や `/account/profile/edit`）へ移動する。
   - 主要リソースの作成や編集にはモーダルを使わず、ブラウザーの「戻る」ボタンの予測可能な動作とディープリンクを保証する専用ページを使う。
2. **一覧の操作を統一する**
   - 表形式の一覧ビューにある詳細、編集、削除などの操作ボタンは、ドロップダウンやケバブメニューに隠さず、各行へ直接表示する。
   - 削除などの破壊的な操作には赤系のボタン（`variant="outline" tone="danger"`）を使う。
3. **動的なページタイトルを使う**
   - すべてのページは、`src/routes/-page.tsx` の `PAGE_TITLES` マップで定義し `PageMarker` コンポーネントが評価する、文脈に応じた動的なブラウザータブのタイトルを持つ（例: 「ユーザー | IdMagic 管理コンソール」）。
4. **用語を統一する**
   - 元の仕様 (`AuditEvent` / `audit_events`) と一貫させるため、UI では「監査ログ」ではなく「監査イベント」を使う。

## Container / Presentation component split

新しい `*Page.tsx` ファイルの作成時と既存ファイルのリファクタリング時は、コンテナと表示用コンポーネントを分割し、データ取得や副作用から切り離した UI 描画を単体テストできるようにする。

1. **ファイル数ではなく責務で分ける。** 公開する `XxxPage` 関数は薄いコンテナとし、`useState`、API 呼び出し、副作用を所有して、ページの `*Shell` を直接配置する。コンテナの全状態を props として受け取る単一の `XxxPresentation` でページ全体を包まない。その形では複雑さを別の層へ移すだけだからである。
2. **セクションの境界で抽出する。** 独立してテストする価値がある自己完結した単位ごとに、表示用コンポーネントを抽出する。自身の検証を持つフォーム (`AdminSignInPolicyPage.tsx` の `DefaultPolicyFormPresentation` など)、項目一覧 (`PasskeyList`)、対話的な状態を持つカード (`AccountSecurityPage.tsx` の `TotpEnrollmentForm`) が該当する。静的で読み取り専用のマークアップはコンテナ内に置いてよく、個別のコンポーネントへ分ける必要はない。
3. **表示用のプロパティを小さく保つ。** コンポーネントが受け取るのは自身のセクションに必要なプロパティ（通常は 10 個より十分少ない）と、実行する操作のコールバックだけとし、コンテナの状態オブジェクト全体は渡さない。独立したセクションが複数あってプロパティが増える場合は、プロパティを広げず、セクションごとのコンポーネントへさらに分ける。
4. **表示用コンポーネントに副作用を置かない。** データとコールバックを受け取って描画するだけとし、`fetch` や `api.*` の呼び出し、`useEffect`、ナビゲーションはコンテナに置く。セクションが自身の状態を管理して純粋なフォームに委譲する場合は、`DefaultPolicyCard` のような小さいセクション専用コンテナに置いてよい。
5. **抽出した単位をテストする。** 抽出した各表示用コンポーネントと純粋な補助関数（日付の整形、検証、派生値の計算）に Vitest / Testing Library の単体テストを付ける。`AccountShell`、`AdminShell`、`AuthShell` を包むコンポーネントは、これらのシェルが TanStack Router の `Link` を使うため、描画にルーターコンテキストを必要とする。テストを省略せず、`src/test/renderWithRouter.tsx` の `renderWithRouter` テスト用補助関数を使う。

## Startup configuration

起動時設定は `backend/cmd/internal/bootstrap` が所有する単一の `Config` 型へ集約し、そこで解析と検証を行う。すべてのバックエンドプロセス（`idmagic`、`idmagic-worker`、`idmagic-batch`、`idmagic-seed`）はこの `Config` を通して環境を読み、`bootstrap` の外で環境変数を直接読まない。

設定検証は、副作用のある初期化を始める前に失敗させる。必須値の欠落、型や範囲の不正、`persistence=postgres` なのに DSN が空であるなどの矛盾を 1 回の起動試行ですべて集約し、リスナーの待ち受け、依存先への接続、seed の適用前に報告して終了する。設定を 1 つ直すたびに次のエラーが現れる往復を避けるため、最初のエラーだけでは停止しない。

シークレットに分類したフィールド（DSN、SMTP 資格情報、API キーなど）の値は、検証エラー、起動ログ、`ConfigurationReference` のいずれにも出力しない。`ConfigurationReference` は `Config` の定義から生成し、生成物として追跡する。定義と生成物の乖離はリポジトリ検証で失敗させ、運用者向けの設定表を手書きで二重管理しない。
