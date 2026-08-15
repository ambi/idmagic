---
context: system
updated_at: 2026-08-15
---

# System Specification

## Overview

外部標準、共有語彙、横断的なユーザー体験、複数の Context にまたがるシナリオを所有する。

React UI と Go API は別々にビルドし、ゲートウェイを通じて同一オリジンで公開する。組み込みの認証画面 (ログイン、同意、デバイス認証)、管理コンソール、アカウントポータルはこの Context に属する。

実行手順と検証コマンドは所有せず、`README.md` に置く。

## Glossary

| Term | Definition | Aliases |
|---|---|---|
| Locale | UI の表示言語を一意に決める BCP 47 言語タグ。IdMagic は `ja` と `en` だけに対応し、それ以外は未対応のロケールとして扱う。 | locale tag, 表示言語コード |
| DisplayLanguage | EndUser または Administrator が言語切り替え UI で明示的に選択したロケール。選択はブラウザーに保存し、以後のアクセスでは保存済みの設定を優先する。 | 表示言語, 言語設定 |
| FallbackLocale | 要求されたロケールが未対応である場合、または対応する辞書に翻訳キーがない場合に使うロケール。IdMagic では `en` とする。 | デフォルトロケール |
| ConfiguredDefaultLocale | 起動時設定 `VITE_DEFAULT_LOCALE` で指定するデフォルトのロケール。`ja` または `en` だけを受け付け、未設定または未対応の値であれば `FallbackLocale` を使う。 | 起動時のデフォルトロケール |
| DemoLoginAffordance | HomePage に表示する、ローカルデモ用資格情報による `authorization_code` フローへのショートカット。資格情報は Seeding の `development` プロファイルが作成する。Vite 開発サーバーではデフォルトで表示し、それ以外のビルドでは起動時設定 `VITE_DEMO_LOGIN_ENABLED=true` の場合だけ表示する。表示時に `development` プロファイルの適用状態は検査しない。 | デモログインのショートカット |
| BackendErrorText | バックエンドが HTTP、OAuth / OIDC リダイレクト、SAML、SCIM などの外部 API レスポンスで返す利用者向けのエラー本文。`message`、`error_description`、`detail`、プレーンテキストのエラー本文を含む。常に英語であり、表示言語によって変化しない。 | API エラーメッセージ, エラーの説明 |
| PersistedStateModel | `created_at` を持ち、作成後に現在状態を更新する場合は `updated_at` も持つ永続状態モデルの規約。作成後は不変で、消費または削除だけを行う記録モデルは `updated_at` を持たない。`issued_at`、`granted_at`、`occurred_at`、`expires_at`、`revoked_at` などのドメイン時刻は `created_at` を置き換えない。各 Context のモデル定義はこの規約に従う。 |  |
| EndUser | 認証済みまたは認証を試みる一般利用者。 |  |
| Operator | IdP をデプロイ・起動時設定を行う運用者。 |  |
| ResourceOwner | OAuth2/OIDC 認可フローでリソースの所有者として認可判断を行う利用者。EndUser と同一人物を OAuth2 文脈で指す呼称。 |  |
| Administrator | テナント内または横断のリソースを管理する権限を持つ利用者。 |  |
| APIConsumer | HTTP API を直接呼び出す外部クライアント。 |  |
| InterfaceStability | インターフェースの外部契約としての安定性を表す区分。`stable` は互換性を保証する外部契約、`beta` は互換性を保証する前の外部契約、`internal` はブラウザーセッション専用またはドメイン内部で外部契約に含めないインターフェースを表す。`stable` と `beta` は同時に 2 版までパスの版として提供し、非推奨の表明から最低 12 か月は維持する。 | 安定性区分 |
| Deprecation | `stable` または `beta` のインターフェースを将来削除することの予告。`deprecated_since` 以降はレスポンスに `Deprecation` ヘッダーを付与し、`sunset_at` が定まれば `Sunset` ヘッダーも付与する。`sunset_at` は `deprecated_since` の最低 12 か月後でなければならない。 | 非推奨化 |
| ConfigurationReference | バックエンドプロセスが起動時に読む設定キーの網羅的な一覧。キー名、値の型、デフォルト、必須かどうか、読み取るプロセス、説明を持ち、シークレットに分類したキーの値は持たない。Config の定義から生成する。 | 設定リファレンス |

## Standards

### Web Content Accessibility Guidelines 2.2

W3C Recommendation — https://www.w3.org/TR/WCAG22/

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| WCAG22-KEYBOARD | required | MUST | すべての認証操作をキーボードだけで完了可能にする。 |
| WCAG22-FOCUS | required | MUST | フォーカスを視認可能にし重要な要素が完全に隠れないようにする。 |
| WCAG22-LABELS-ERRORS | required | MUST | 入力にラベルを付け、エラーをテキストで識別して修正方法を示す。 |
| WCAG22-STATUS | required | MUST | 認証結果や送信エラーをフォーカス移動なしに支援技術へ通知する。 |

### General Data Protection Regulation

Regulation (EU) 2016/679 — https://eur-lex.europa.eu/eli/reg/2016/679/oj

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| GDPR-CONSENT-WITHDRAWAL | required | MUST | ResourceOwner が同意を撤回でき、撤回後の新規発行には利用しない。`Consent` と `ConsentLifecycle` は OAuth2 Context が所有する。 |
| GDPR-ERASURE | required | MUST | 削除要求後は法的保存義務を除く PII を定義済み期間内に消去する。消去は IdManagement の UserLifecycle Purge 遷移と Authentication の資格情報破棄が個別に担う。 |
| GDPR-PROCESSING-RECORDS | required | MUST | セキュリティおよび認可イベントの監査記録を定義済みの期間保持する。保持期間は Audit Context が所有する。 |

## Design

### Authorization boundary

この Context 自身は業務データの認可を判断しない。どの経路へどの資格情報で到達できるかという、システム入口の境界を所有する。

- ブラウザーの認証フロー (`/api/auth/*`) は認可トランザクションの Cookie で解決する。トランザクションの内容は常にサーバー側に保持し、HTML、URL、JavaScript から読めるアプリケーション状態には含めない。
- 管理 API (`/api/admin/*`) とセルフサービス API (`/api/account/*`) は、ログイントランザクションとは独立した認可を通る。ポータル境界のスコープ (`idmagic.admin` / `idmagic.account`) を要求し、アカウントポータルのトークンで管理 API を呼ぶ経路をフェイルクローズで塞ぐ。実際に何ができるかは、記録を所有する各 Context のロールとスコープが決める。
- 状態を変更するブラウザー API は、二重送信方式の CSRF トークンと、設定済みの公開発行者と一致する `Origin` ヘッダーを要求する。
- 生存確認、受付可否、起動完了の各プローブは認証を要さないが、返すのは状態だけで、設定値や依存先の詳細は含めない。`/metrics` も認証を持たないため、公開先は折り返しアドレス、管理用ネットワーク、認証付きプロキシの背後に限る。

### Internal Interfaces

#### BackendErrorResponse
バックエンドの HTTP API とプロトコルエンドポイントが返す、共通のエラー契約である。`message`、`error_description`、`detail`、プレーンテキスト本文はいずれも `BackendErrorText` であり、`DisplayLanguage` にかかわらず英語で固定する。RFC 9457 Problem Details では、`type` の `urn:idmagic:error:` に続く部分を安定したエラーコードとして解釈できる。UI は `detail` または `title` を、人間が読めるフォールバックとして利用できる。この契約は、個別エンドポイントのエラーコードや HTTP ステータスを変更しない。
- Result invariant: text_is_english(output.message)
- Result invariant: text_is_english(output.error_description)
- Result invariant: text_is_english(output.detail)

### Deployment boundary

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

### Authorization transaction

Go サービスは OAuth の認可要求全体をサーバー側に保持する。ブラウザーには、短命な内部 UUID だけを `HttpOnly`、`SameSite=Lax`、HTTPS では `Secure` のトランザクション Cookie に保存する。認可要求の内容は HTML、URL、JavaScript から読める状態に含めない。

SPA は `GET /api/auth/transaction` を呼び、画面の種類、クライアント名、要求されたスコープなど表示用のデータだけを取得する。ログインと同意のコマンドは cookie からトランザクションを解決する。

### Browser protections

- セッションと認可トランザクションの Cookie は `HttpOnly` とする。
- 状態を変更する UI API は、二重送信方式の CSRF Cookie と `X-CSRF-Token` ヘッダーを要求する。
- 状態を変更するブラウザー API は、設定済みの公開発行者と一致する `Origin` ヘッダーを要求する。
- 同意処理では、現在のログインセッションの subject が認可トランザクションの subject と一致することを検証する。
- 認可リクエストは 10 分で期限切れとなり、完了したリクエストは再利用できない。
- OAuth のリダイレクト URI、PKCE 値、スコープ、クライアント識別子はサーバー側の状態から読み取る。
- UI API のレスポンスには `Cache-Control: no-store` を付け、資格情報や内部のリクエスト ID は返さない。

### API boundary

ブラウザー向け認証 API は `/api/auth/*` に置き、OAuth / OIDC のプロトコルエンドポイントは標準パスを維持する。管理 API は `/api/admin/*`、セルフサービス API は `/api/account/*` に置き、どちらもログイントランザクション API とは独立した明示的な認可ポリシーを使う。

### Admin console and account portal as OIDC RPs

管理コンソール (`/admin/*`) とアカウントポータル (`/account/*`) は IdP 自身の OIDC RP であり、IdP の `/authorize` と `/token` に対する `authorization_code` + PKCE で認証する。管理用の `…0022` とアカウント用の `…0023` という固定 UUID の `client_id` を持つファーストパーティーのパブリッククライアントとして登録し、`src/api/oidc.ts` と bootstrap seed に反映する。リソース所有者は IdP 自身のユーザーなので、同意画面は省略する。

純粋な SPA RP なので、アクセストークンはブラウザーの `sessionStorage` に保持し、`Authorization: Bearer` として `/api/{admin,account}/*` へ送る。バックエンドは RFC 9068 のリソースサーバーとして検証する。JavaScript からトークンへアクセスできる設計上のリスクは、600 秒の短い有効期間、`Cache-Control: no-store`、URL、ログ、DOM にトークンを置かないことで抑える。OIDC クライアントや鍵の設定不備で管理者が復旧不能にならないよう、ファーストパーティーのセッションログイン (`POST /api/auth/login`) は緊急経路として残す。

### Client-side routing

SPA は TanStack Router を使い、`src/routes/` 配下のファイルベースルーティングでクライアント側のナビゲーションを行う。Vite のルータープラグインが `src/routeTree.gen.ts` を生成し、ルートローダーとコンポーネントを自動的にコード分割する。ルートファイルはリクエストパスの構造に従う。`admin/route.tsx` と `account/route.tsx` は `<Outlet>` を描画する薄いレイアウトルートとし、`admin/index.tsx`、`account/index.tsx`、末端のルートファイルは自身の `loader`、API リクエスト、ページコンポーネント、パスパラメーターを所有する。一覧ページを介さず描画する詳細ページには TanStack Router の末尾アンダースコア規約を使い、`/admin/users/$sub` は `admin/users_/$sub.tsx` で表す。`-` で始まるファイルはルート固有の補助ファイルとし、ルート生成から除外する。管理画面とアカウント画面内のナビゲーションには `<Link>` を使い、ページ移動のたびに文書全体や全ページのデータを再読み込みせず、対象ルートのローダーだけを実行する。OIDC のログインガード（`ensureLoggedIn`）はローダー内で動き、初回読み込みとアプリケーション内のナビゲーションの両方に適用する。ログイン、同意、コールバック、OIDC リダイレクトからなる認証フローの遷移は、ページ全体のナビゲーションとする。E2E テスト向けに、描画済みページの種類を `<meta name="idmagic:page">` で DOM へ示す。

### Design Guidelines

- **信頼できる情報を示す**: 落ち着いた配色、明確なサービス識別、現在の操作とセキュリティ状態の説明により、利用者が認証の可否を判断できるようにする。
- **重要な情報を先に示す**: 明確な情報階層により、ページの題名、要求元、共有する情報、次の操作、取り消し方を最初に示す。
- **重要な操作を単純にする**: 画面ごとの主な操作を 1 個に絞り、拒否や取り消しとは視覚的に区別する。UI 固有の理由で OAuth / OIDC フォームの名前、送信値、遷移の契約を変えない。
- **アクセシビリティを標準とする**: キーボード操作、見えるフォーカス表示、十分な色のコントラスト、明示的なラベル、適切な `aria-*` 属性、動きを減らす設定を尊重するアニメーションに対応する。
- **業務利用に適した情報密度を保つ**: 過剰なアニメーションや装飾を避け、一貫した余白、タイポグラフィ、ボーダー、状態色で情報を構造化する。
- **情報を失わずレスポンシブにする**: デスクトップでは補足情報を表示し、モバイルではサービスの識別や安全上の警告を省かず認証操作を優先する。
- **共有コンポーネントで一貫させる**: Tailwind CSS、Radix UI、shadcn/ui に沿ったローカルコンポーネントを使う。色、角丸、フォーカスリング、無効状態をその場ごとに実装しない。

### Admin Console Policy

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

### UI Library Selection

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

### UI navigation and consistency policy

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

### Container / Presentation component split

新しい `*Page.tsx` ファイルの作成時と既存ファイルのリファクタリング時は、コンテナと表示用コンポーネントを分割し、データ取得や副作用から切り離した UI 描画を単体テストできるようにする。

1. **ファイル数ではなく責務で分ける。** 公開する `XxxPage` 関数は薄いコンテナとし、`useState`、API 呼び出し、副作用を所有して、ページの `*Shell` を直接配置する。コンテナの全状態を props として受け取る単一の `XxxPresentation` でページ全体を包まない。その形では複雑さを別の層へ移すだけだからである。
2. **セクションの境界で抽出する。** 独立してテストする価値がある自己完結した単位ごとに、表示用コンポーネントを抽出する。自身の検証を持つフォーム (`AdminSignInPolicyPage.tsx` の `DefaultPolicyFormPresentation` など)、項目一覧 (`PasskeyList`)、対話的な状態を持つカード (`AccountSecurityPage.tsx` の `TotpEnrollmentForm`) が該当する。静的で読み取り専用のマークアップはコンテナ内に置いてよく、個別のコンポーネントへ分ける必要はない。
3. **表示用のプロパティを小さく保つ。** コンポーネントが受け取るのは自身のセクションに必要なプロパティ（通常は 10 個より十分少ない）と、実行する操作のコールバックだけとし、コンテナの状態オブジェクト全体は渡さない。独立したセクションが複数あってプロパティが増える場合は、プロパティを広げず、セクションごとのコンポーネントへさらに分ける。
4. **表示用コンポーネントに副作用を置かない。** データとコールバックを受け取って描画するだけとし、`fetch` や `api.*` の呼び出し、`useEffect`、ナビゲーションはコンテナに置く。セクションが自身の状態を管理して純粋なフォームに委譲する場合は、`DefaultPolicyCard` のような小さいセクション専用コンテナに置いてよい。
5. **抽出した単位をテストする。** 抽出した各表示用コンポーネントと純粋な補助関数（日付の整形、検証、派生値の計算）に Vitest / Testing Library の単体テストを付ける。`AccountShell`、`AdminShell`、`AuthShell` を包むコンポーネントは、これらのシェルが TanStack Router の `Link` を使うため、描画にルーターコンテキストを必要とする。テストを省略せず、`src/test/renderWithRouter.tsx` の `renderWithRouter` テスト用補助関数を使う。

### Startup configuration

起動時設定は `backend/cmd/internal/bootstrap` が所有する単一の `Config` 型へ集約し、そこで解析と検証を行う。すべてのバックエンドプロセス（`idmagic`、`idmagic-worker`、`idmagic-batch`、`idmagic-seed`）はこの `Config` を通して環境を読み、`bootstrap` の外で環境変数を直接読まない。

設定検証は、副作用のある初期化を始める前に失敗させる。必須値の欠落、型や範囲の不正、`persistence=postgres` なのに DSN が空であるなどの矛盾を 1 回の起動試行ですべて集約し、リスナーの待ち受け、依存先への接続、seed の適用前に報告して終了する。設定を 1 つ直すたびに次のエラーが現れる往復を避けるため、最初のエラーだけでは停止しない。

シークレットに分類したフィールド（DSN、SMTP 資格情報、API キーなど）の値は、検証エラー、起動ログ、`ConfigurationReference` のいずれにも出力しない。`ConfigurationReference` は `Config` の定義から生成し、生成物として追跡する。定義と生成物の乖離はリポジトリ検証で失敗させ、運用者向けの設定表を手書きで二重管理しない。

### Design Decisions

- 管理コンソールとアカウントポータルは、BFF の背後に置かずファーストパーティーの OIDC RP として扱う。IdP 自身が持つ認可とトークン発行の経路を、管理画面のためだけに二重化しないためである。
- 純粋な SPA RP としてアクセストークンをブラウザーに保持することを受け入れ、短命なトークンと no-store で影響範囲を限定する。BFF を挟むとセッション状態を持つ層が増え、IdP の可用性と復旧経路が複雑になるからである。
- ファーストパーティーのセッションログインを緊急経路として残す。OIDC クライアントや鍵の設定を壊すと、直す手段そのものが失われるからである。
- 管理・アカウントポータルの固定 `client_id` を含む内部生成の ID 列は、`TEXT` ではなく `UUID` 型とする。
- 主要リソースの作成と編集はモーダルではなく専用ページとする。ブラウザーの「戻る」の挙動とディープリンクを保つためである。
- 一覧の操作はケバブメニューに隠さず行内に表示する。管理作業では、その行に何ができるかが一覧の時点で分かることのほうが、見た目の整理より重要だからである。

## Scenarios

### REQ-SYSTEM-001: Operator は分離された運用資産で SLO を検証する
- ACTOR Operator
- GIVEN API、UI ゲートウェイ、イベントリレーは個別の実行単位としてデプロイされる
- GIVEN `MetricsExposition` の公開範囲は管理ネットワークに制限される
- WHEN Operator が環境のオーバーレイを選んで運用マニフェストを適用する
  - ALT PostgreSQL へ到達できない → `ReadinessProbe` は `unavailable` を返し、API は新規トラフィックを受けない → `LivenessProbe` は `healthy` を維持し、依存障害だけでは再起動しない
  - ALT Prometheus Operator が導入されていない → `ServiceMonitor` は適用対象から外し、標準の Prometheus スクレイプ設定で `MetricsExposition` を収集する
- THEN API の生存、受付可否、起動完了の各プローブは、それぞれ `LivenessProbe`、`ReadinessProbe`、`StartupProbe` を呼ぶ
- THEN Prometheus が `MetricsExposition` をスクレイプし、OAuth2 の可用性、レイテンシー、エラー率の目標を表示および評価する

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
- WHEN プロセスが起動時に `Config` を集約および検証する
  - ALT 必須値が欠落している → 検証は該当キーを含む集約エラーを返す → プロセスはリスナーの待ち受け、永続化依存先への接続、seed の適用など副作用のある初期化を開始せず終了する
  - ALT 値の型または範囲が不正である（数値でない、負の期間など） → 検証は該当キーを含む集約エラーを返す → プロセスは副作用のある初期化を開始せず終了する
  - ALT 相互に矛盾する組み合わせである（`persistence=postgres` なのに DSN が空など） → 検証は該当する組み合わせを含む集約エラーを返す → プロセスは副作用のある初期化を開始せず終了する
- THEN 発生したすべての検証エラーが 1 回の起動試行で集約されて報告される
- THEN 検証エラーおよび起動ログは、シークレットに分類された値（DSN、SMTP 資格情報、API キーなど）を含まない
- WHEN すべての検証を通過する
- THEN プロセスは検証済みの `Config` を用いて初期化を完了する

### REQ-SYSTEM-017: ConfigurationReference は起動時設定の定義から生成され乖離を検出できる
- ACTOR Operator
- GIVEN バックエンドプロセスの起動時設定が `Config` として一箇所で定義されている
- WHEN ConfigurationReference を生成する
- THEN 生成物は設定可能な各キーについて、キー名、値の型、デフォルト値、必須か、読むプロセス、説明を含む
- THEN 生成物はシークレットに分類されたキーの値を含まず、シークレットであることだけを示す
- WHEN 生成物と `Config` の定義を突き合わせる
  - ALT 生成物が定義と一致しない → 突き合わせは失敗し、乖離したキーを報告する
- THEN Operator は `Config` の実装を読まずに設定可能なすべてのキーを参照できる
