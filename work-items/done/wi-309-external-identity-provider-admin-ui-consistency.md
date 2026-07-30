---
status: completed
authors: [tn]
risk: high
created_at: 2026-07-30
depends_on: [wi-126-admin-and-account-ui-consistency-and-navigation-policy]
initial_context:
  scl:
    Authentication:
      - models.IdentityProviderConnection
      - models.IdentityProviderConnectionStatus
      - models.ExternalIdentityProviderSummary
      - interfaces.CreateIdentityProviderConnection
      - interfaces.UpdateIdentityProviderConnection
      - interfaces.DeleteIdentityProviderConnection
      - interfaces.ActivateIdentityProviderConnection
      - interfaces.DisableIdentityProviderConnection
      - interfaces.TestIdentityProviderConnection
      - interfaces.RefreshIdentityProviderMetadata
      - states.IdentityProviderConnectionLifecycle
  decisions:
    - decisions/ADR-148-envelope-encryption-and-datakeys-context.md
  source:
    - backend/authentication/federation
    - backend/datakeys
    - frontend/src/features/admin-settings/IdentityProvidersTab.tsx
    - frontend/src/features/admin-settings/AdminSettingsPage.tsx
    - frontend/src/lib/adminNav.ts
    - frontend/src/components/AdminShell.tsx
  tests:
    - backend/authentication/federation
    - frontend/src/features/admin-settings
affected_spec:
  - { context: Authentication, kind: model, element: IdentityProviderConnection }
  - { context: Authentication, kind: model, element: IdentityProviderConnectionStatus }
  - { context: Authentication, kind: interface, element: CreateIdentityProviderConnection }
  - { context: Authentication, kind: interface, element: UpdateIdentityProviderConnection }
  - { context: Authentication, kind: interface, element: DeleteIdentityProviderConnection }
  - { context: Authentication, kind: interface, element: ActivateIdentityProviderConnection }
  - { context: Authentication, kind: interface, element: DisableIdentityProviderConnection }
  - { context: Authentication, kind: interface, element: TestIdentityProviderConnection }
  - { context: Authentication, kind: state, element: IdentityProviderConnectionLifecycle }
---

# 外部IDプロバイダー管理画面を他リソースと同じ一覧/編集分離UIに揃え、下書き状態・設定テスト・シークレット参照の分かりにくさを解消する

## Motivation

管理画面の「設定」タブ内にある「外部IDプロバイダー」(`IdentityProvidersTab.tsx`) を実機で確認したところ、
`users` / `groups` / `applications` / `roles` / `lifecycle-workflows` など他の一級リソースが徹底している
「一覧 (参照専用) → 追加/編集ボタンで別画面に遷移」という管理画面全体の方針から外れており、加えて
lifecycle 設計そのものに実装上の欠陥がある。指摘を1つずつコードで検証した結果は以下の通り。

1. **画面構成が管理コンソール全体のポリシーと不統一**: `frontend/ARCHITECTURE.md` の
   "UI navigation and consistency policy" 節 (ADR-086、[[wi-126-admin-and-account-ui-consistency-and-navigation-policy]]
   で策定済み) は「一覧 (読み取り専用) → 詳細 (読み取り専用、Edit ボタン付き) → 編集 (専用ルート)」
   という Detail-then-Edit Navigation Policy と、モーダルではなく専用ルートページを使うことを
   既に明文化している。`users_/new.tsx` + `users_/$id.index.tsx` (詳細) + `users_/$id.edit.tsx` (編集)
   がその実装例であり、groups/applications/roles/lifecycle-workflows も同型。
   同じ wi-126 は §8 でこの方針を Entra 連携画面にも適用し、当時 `設定` タブの中にあった
   Entra 連携の configure フォームを、独立ルート `/admin/federation/entra` (一覧) +
   `/admin/federation/entra/new` (追加) に分離し、`adminNav.ts` のトップレベル項目に格上げした
   実績がある。ところが外部IDプロバイダーはこの整理の対象に入らず、専用ルートを持たないまま
   `/admin/settings?tab=identity-providers` というタブの中で、一覧カードと作成/編集共用フォームが
   同一コンポーネント内に同居し続けている (`IdentityProvidersTab.tsx:239-482`)。読み取り専用の
   詳細ビューも存在しない。**指摘は妥当。方針は既にあるので新設ではなく、Entra 連携で
   実施済みと同じ整理を外部IDプロバイダーにも適用すればよい。**

2. **「下書き (Draft)」が削除を阻む割に何の安全性ももたらさない**:
   - `deleteAdmin` は `status != Disabled` なら 409 で拒否する
     (`backend/authentication/federation/handlers_http/routes.go:294-296`)。
   - `IdentityProviderConnectionLifecycle` の遷移は `Draft→Active` と `Active⇄Disabled` のみで、
     `Draft→Disabled` が存在しない (`spec/contexts/authentication.yaml:2676-2687`)。
   - したがって一度も有効化していない Draft を消すには「有効化 → 無効化 → 削除」という、実質何もしていない
     3手順が必須になる。
   - しかも `Activate()` (`domain/models.go:119-128`) が呼ぶ検証は `Validate()` の再実行のみで、
     `Save()` (`db_postgres/repositories.go:21`) が新規作成時点で既に同じ `Validate()` を通している。
     つまり Draft は「保存できる=Validate 済み」を超える情報を一切持たない、実体のない状態。
   - さらに `updateAdmin` は編集のたびに無条件で `status` を `Draft` に戻す
     (`routes.go:274`)。表示名を直すだけの軽微な編集でも、稼働中の接続が黙って Active から
     外れ、実ユーザーのログインが止まる副作用がある。SCL の説明文
     (`UpdateIdentityProviderConnection` の description) は「trust source 変更時のみ再検証」を
     意図しているが、実装はその区別をしていない。
   - 比較として、`lifecycle_workflows` テーブルは `draft/enabled/disabled/archived` の4状態を持ち
     (`infra/schema/postgres.sql:968`) こちらは Draft が「本当にまだ書きかけ」を表すため妥当だが、
     `IdentityProviderConnection` は保存時点で必須項目が全て揃っていなければならず、Draft が
     「書きかけ」を表したことは一度もない。**指摘は妥当。Draft は撤去し、削除はいつでもできるようにすべき。**

3. **「設定テスト」は実際には何も接続確認していない**:
   - `TestIdentityProviderConnection` ハンドラ (`routes.go:355-367`) は `connection.Validate()` を
     再実行して `{"result":"valid"}` を返すだけ。ネットワーク到達性、OIDC discovery/JWKS 取得、
     `secret_reference` の解決可能性、SAML証明書の実パース、いずれも検証していない。
     構文的に妥当な (しかし実在しない) URL やダミーの client_id を入れても常に成功する。
   - フロントエンド側も `runIdentityProviderAction` が `Promise<void>` でレスポンス本文を握りつぶし
     (`frontend/src/api/admin.ts:626-635`)、成功時は常に同じ汎用トースト `t.actionCompleted`
     (「操作を完了しました。」) を表示するだけ (`IdentityProvidersTab.tsx:173-196`)。
     バックエンドが仮に意味のある結果文字列を返しても画面には出ない。
   - **指摘は妥当。** 「適当な値でもエラーにも成功にも見えない」という体験は、テスト自体が形式検証の
     二度打ちに過ぎないことと、結果を画面が捨てていることの両方が原因。

4. **「クライアントシークレット参照」という概念自体は妥当だが、実装が場当たり的**:
   - `secret_reference` は実際のシークレット値ではなく、サーバープロセスの環境変数名を指す
     `env:VARNAME` という文字列 (`backend/authentication/federation/secrets_env/resolver.go:19-37`)。
     UI のヘルプテキストにも「例: env:CONTOSO_CLIENT_SECRET。値そのものは保存しません」と小さく
     書かれている (`IdentityProvidersTab.i18n.ts:14-15`)。
   - **「参照」という間接化の考え方自体は否定できない。** シークレットの実値をアプリ DB に平文で
     持たないという設計判断は妥当で、実際 `ARCHITECTURE.md` の
     「Envelope encryption for reversible secrets」節が MFA TOTP seed に対して同種の思想
     (実値を直接持たず、暗号化した形でのみ保持する) を採用している。ここを「シークレットの実値を
     そのまま保存すべき」と直すのは後退になる。
   - ただし、**現在の具体的な実装 (サーバー環境変数名を都度参照させる方式) は、この
     リポジトリ自身が確立した可逆シークレット保管の仕組み (`EnvelopeCrypto` / `DataKeys`,
     ADR-148) と整合しておらず、運用上も破綻している。** テナントごとに外部IDプロバイダーを
     追加するたびに、共有サーバープロセスの環境変数を追加してデプロイし直す必要があり、
     セルフサービスなマルチテナント管理画面としては機能しない。加えて `test` も `Activate` も
     `secret_reference` が実際に解決できるかどうかを一度も検証しない (指摘3で確認した通り
     `Validate()` はフィールド非空チェックのみ)。
   - **したがって「参照という概念自体がおかしい」という指摘には反論するが、「渡し方が分からない・
     実値であるべきでは」という違和感の実体 — この画面が実際のシークレット入力という体験を
     提供していないこと — は妥当な指摘であり、修正すべき。** 既存の `EnvelopeCrypto`/`DataKeys`
     機構に乗せ、UI では通常のシークレット入力 (実値・マスク表示・write-only) として扱う設計に
     改める。

加えて会話の途中で次の疑問も提起された: 「外部IDプロバイダーは「設定」の中にあるが、サインイン
ポリシーや署名鍵は「設定」の中にない。」`frontend/src/lib/adminNav.ts:107-166` を確認すると、
`sign-in-policy` と `keys` (署名鍵) は独立したトップレベルのナビゲーション項目であり、
`entra-federation` (Entra 連携) も同様にトップレベル項目として既に存在する
(まさに前段で述べた wi-126 §8 の整理の結果)。一方で外部IDプロバイダーは
`AdminSettingsPage.tsx` のタブの1つ (`general`/`password-policy`/`branding`/
`integration-endpoints`/`identity-providers`/`api-tokens`/`email` という寄せ集め) に埋もれている。
これに一貫した設計原則は見当たらず、Entra 連携と同じ「フェデレーション設定」でありながら
横展開が漏れた履歴的な積み上げの結果と見られる。外部IDプロバイダーは
CRUD とライフサイクル (作成/更新/削除/有効化/無効化/テスト) を持つ一級リソースであり、
その形は `applications` や `roles`、そして何より `entra-federation` に近い。**「設定」から
独立させ、`entra-federation` と並ぶトップレベルのナビゲーション項目に格上げすることで、
指摘1 (画面統一) とこの疑問の両方を同時に解消する。新しいポリシーの考案は不要で、
ADR-086 / wi-126 が既に確立した方針をそのまま適用するだけでよい。**

## Scope

- **decision**:
  - 新規 ADR 2本 (UI ナビゲーションは ADR-086 が既にあるため対象外):
    (a) `IdentityProviderConnectionStatus` を `Draft/Active/Disabled` の3値から
    `Active/Disabled` の2値に整理する決定と、Draft 廃止に伴う既存データ移行方針。
    (b) `secret_reference` の `env:` 参照方式を廃止し、`EnvelopeCrypto`/`DataKeys` による
    実値の envelope 暗号化保存に切り替える決定と、既存の `env:` 参照が設定済みの環境での
    移行方針 (どちらも「本当に fork していた判断」なので ADR に残す)。
- **scl** (`spec/contexts/authentication.yaml`):
  - `IdentityProviderConnectionStatus` から `Draft` を削除。
  - `IdentityProviderConnectionLifecycle` を `Active ⇄ Disabled` の単純な2状態遷移に再定義。
  - `CreateIdentityProviderConnection` の description を「作成直後は Disabled」に変更。
  - `DeleteIdentityProviderConnection` の description を「いつでも削除できる」に変更 (Active な
    接続を消す場合の確認 UX は Plan で検討)。
  - `UpdateIdentityProviderConnection` の description を「trust source (issuer / endpoint /
    certificate / protocol) を変更したときだけ Disabled に戻す。display_name や claim_mapping
    など非 trust フィールドの変更では状態を変えない」に修正し、それを usecase 層で実際に
    区別する。
  - `TestIdentityProviderConnection` の description と output を、実際に何を検証するか
    (OIDC discovery/JWKS 到達性、secret_reference 解決可能性、SAML証明書のパース可否) が
    分かる構造化結果に変更する。
  - `IdentityProviderConnection.secret_reference` の説明を「envelope 暗号化された client secret
    の実値を書き込み専用で受け取る」に変更する (フィールド名は現行のまま維持するか
    `client_secret` に改名するかは ADR で決める)。
- **go**:
  - `backend/authentication/federation/domain`: `ConnectionStatus` から `ConnectionDraft` を除去、
    `Activate`/`Disable` の遷移ルールを更新。
  - `backend/authentication/federation/handlers_http`: `createAdmin` の初期状態、`updateAdmin` の
    trust フィールド判定によるステータス保持、`deleteAdmin` のステータスガード撤去、`test` の
    実処理刷新。
  - `backend/authentication/federation/secrets_env`: `EnvelopeCrypto`/`DataKeys`
    (`backend/datakeys`) を使う secret 保管アダプタに置き換え、`FieldMigrator` 登録により
    既存 `env:` 参照からの移行ジョブを用意する。
  - `infra/schema/postgres.sql`: `identity_provider_connections.status` の CHECK 制約から
    `'draft'` を除去し、既存行の `draft → disabled` 移行を行う。
- **ui**:
  - 既存の ADR-086 / [[wi-126-admin-and-account-ui-consistency-and-navigation-policy]]
    「UI navigation and consistency policy」(`frontend/ARCHITECTURE.md`) を、Entra 連携
    (wi-126 §8) と同じ形で外部IDプロバイダーにも適用する。新しいポリシーは作らない。
  - `frontend/src/routes/admin/identity-providers.tsx` (一覧、参照専用・行に
    詳細/編集/テスト/有効化・無効化/削除ボタンを直接表示)、
    `identity-providers_/$id.index.tsx` (詳細、読み取り専用。secret は「設定済み/未設定」の
    みを示し実値は出さない)、`identity-providers_/$id.edit.tsx` (編集)、
    `identity-providers_/new.tsx` (作成) を新設し、`IdentityProvidersTab.tsx` を分割・移設する。
    `AdminSettingsPage.tsx` からは撤去する。
  - `frontend/src/lib/adminNav.ts` に `entra-federation` と並ぶトップレベル項目
    `identity-providers` を追加する。
  - `src/routes/-page.tsx` の `PAGE_TITLES` に新ルート分のページタイトルを追加する
    (ADR-086 の Dynamic Page Titles 規約)。
  - 「設定テスト」実行結果 (成功/失敗と理由) を画面に明示表示する。
  - シークレット入力欄を「クライアントシークレット参照」から実値入力の
    「クライアントシークレット」に変更する (書き込み専用・マスク表示)。
- **documentation**:
  - 新しい方針文書は不要 (ADR-086 / `frontend/ARCHITECTURE.md` の
    "UI navigation and consistency policy" 節が既に管理画面全体に適用済みの方針として
    存在する)。本 WI 完了後、その節の記述 (または wi-126 の記録) に「外部IDプロバイダーにも
    適用済み」の一文を追記する程度に留める。

## Out of Scope

- 「設定」タブに残る他項目 (一般、パスワードポリシー、ブランディング、API トークン、
  通知テンプレート) の整理・再配置。特にパスワードポリシーがサインインポリシーと概念的に
  近い点は別途の課題として認識するが、本 WI では扱わない。
- `entra-federation` との画面統合 (フェデレーション系ナビゲーションのグルーピング)。
  将来的な検討課題として `ARCHITECTURE.md` か別 WI に残す。
- 「設定テスト」でブラウザリダイレクトを伴う実際の authorization_code フロー全体を
  シミュレートすること。サーバー側で完結する到達性確認 (discovery/JWKS/証明書/secret 解決) に
  留める。
- `EnvelopeCrypto`/`DataKeys` 自体の新機能追加。既存の ADR-148 の仕組みをそのまま再利用する。

## Design

### 状態モデルの単純化

`Draft` は「まだ全項目が揃っていない書きかけの接続」を表す状態として導入されたと思われるが、
`Save()` が新規作成時点で `Validate()` (全必須フィールドの充足チェック) を要求するため、
DB に存在する接続は最初から Draft/Active/Disabled のいずれであっても常に「保存可能な形」を
満たしている。Draft と Disabled はどちらも「login routing に使われない」という点で観測可能な
違いがなく、実質的に同じ状態が2つの名前を持っているだけになっている。これを次のように単純化する。

- 状態は `Active` / `Disabled` の2値のみ。
- 作成直後の初期状態は `Disabled` (安全側のデフォルト。他ではここを Draft と呼んでいた)。
- `Active → Disabled` と `Disabled → Active` のみを許可する。
- 削除は `Active` / `Disabled` どちらからでも可能にする。`Active` な接続を削除する際は
  フロントエンドで確認ダイアログを出す (バックエンドでのステータスガードは撤去)。

### 編集時の自動デグレードをトラスト変更時のみにする

現状 `updateAdmin` は編集の種類を問わず無条件に状態を書き戻しているが、これは
「表示名を直しただけで本番ログインが止まる」という事故を生む。usecase 層で
`issuer` / `client_id` / `authorization_endpoint` / `token_endpoint` / `jwks_uri` /
`saml_sso_url` / `saml_entity_id` / `saml_signing_certificates` / `protocol`
(= 「trust source」) のいずれかが変更された場合に限り `Active → Disabled` に落とし、
それ以外のフィールド (display_name, claim_mapping, linking_policy, jit_provisioning,
allowed_email_domains) のみの変更では状態を保持する。

### 「設定テスト」の実処理化

`test` ハンドラを次のように拡張する。

- OIDC: 保存済み `authorization_endpoint`/`token_endpoint`/`jwks_uri` への到達性 (HEAD/GET) を
  確認し、`secret_reference` が実際に解決できるか (`secrets_env` アダプタ経由、値そのものは
  ログ・レスポンスに出さない) を確認する。`RefreshIdentityProviderMetadata` が持つ SSRF 対策
  (固定済み endpoint のみ・任意 URL は取得しない) をそのまま踏襲する。
- SAML: 保存済み証明書が有効な X.509 としてパースできるか、有効期限内かを確認する。
- 結果はブール成功/失敗と、失敗理由の一覧 (「JWKS に到達できません」「secret_reference を
  解決できません」等、値そのものは含まない) を構造化して返す。
- フロントエンドは戻り値を保持し、成功/失敗が明確に分かるバナー・アラートとして表示する
  (現状のように結果を捨てて汎用トーストを出すことはしない)。

### シークレット保管を EnvelopeCrypto/DataKeys に統一する

`ARCHITECTURE.md` の "Envelope encryption for reversible secrets" (MFA TOTP seed 等) と同じ
枠組みを `IdentityProviderConnection` の client secret に適用する。UI は実際のシークレット値を
一度だけ受け取り (write-only)、サーバー側で tenant の DEK を使い AEAD 暗号化して保存する。
`backend/datakeys` の `FieldMigrator` port に登録し、既存の `env:` 参照からの移行 (再入力を
促すか、運用者が一度だけ実値を入力し直す) を行う。読み出しは復号後の値をプロセス内でのみ
使用し、API レスポンスには含めない (現状の `connection.SecretReference = ""` の方針を踏襲)。

### ナビゲーションと画面構成

新しい方針は考案せず、ADR-086 / wi-126 が確立した「一覧 (参照専用) → 詳細 (参照専用、
Edit ボタン) → 編集 (専用ルート)」を、wi-126 §8 が Entra 連携に適用したのと同じ形で
外部IDプロバイダーに適用する。`adminNav.ts` にトップレベル項目として追加する
(`entra-federation` の隣)。

## Plan

1. ADR 2本 (状態モデル単純化、secret 保管方式の切替) を先に書き、移行方針の合意を明文化する。
   UI ナビゲーションは ADR-086 を適用するだけなので新規 ADR は不要。
2. SCL を更新し `just check-scl` を通す。
3. バックエンド: domain → usecases (trust フィールド判定) → handlers_http → db_postgres
   (schema migration + FieldMigrator) の順に RED→GREEN で実装する。
4. 既存 `draft` 行の移行 (`status='draft'` → `'disabled'`) と、既存 `env:` secret_reference の
   移行ジョブを用意し、`just test-go` で確認する。
5. フロントエンド: ルート分割 (`identity-providers.tsx` 一覧 / `$id.index.tsx` 詳細 /
   `$id.edit.tsx` 編集 / `new.tsx` 作成)、`adminNav.ts` へのトップレベル項目追加、
   `PAGE_TITLES` への追加、`AdminSettingsPage.tsx` からの撤去、設定テスト結果表示、
   シークレット実値入力フォームを実装する。
6. Verification を通す。

## Tasks

- [x] T001 [ADR] 状態モデル単純化 (Draft 廃止) の ADR を書く。既存 `draft` 行の移行方針を含める。
      → [[ADR-149]]。既存行の移行は宣言的 schema と分離し
      `infra/schema/data-migrations/2026-07-31-identity-provider-connections-draft-to-disabled.sql`
      に one-off script として用意 (ADR-071 §5)。
- [x] T002 [ADR] secret_reference の `env:` 方式から `EnvelopeCrypto`/`DataKeys` 実値暗号化への
      切替 ADR を書く。既存参照の移行方針を含める。→ [[ADR-150]]。
- [x] T003 [SCL] `IdentityProviderConnectionStatus`、`IdentityProviderConnectionLifecycle`、
      `CreateIdentityProviderConnection`/`UpdateIdentityProviderConnection`/
      `DeleteIdentityProviderConnection`/`TestIdentityProviderConnection` の記述を更新し
      `just check-scl` を通す。新設 `models.IdentityProviderConnectionTestResult` と
      `IdentityProviderConnection.client_secret_configured` (詳細画面の設定済み/未設定表示用の
      派生フィールド) も追加。`just check-scl` green。
- [x] T004 [Domain] `ConnectionStatus` を2値化し、`Activate`/`Disable` の遷移を更新する。
      RED: `TestValidateRejectsNonActiveDisabledStatus` (`domain/models_test.go`) を先に fail 確認
      (model `IdentityProviderConnectionStatus`) → GREEN。
      `TestActivateRejectsAlreadyActiveConnection` は回帰ガードとして追加 (既存ロジックで
      既に green だったため RED を確認できなかった点は self-attest として明記)。
- [x] T005 [Usecase] 更新時のトラストフィールド判定によるステータス保持ロジックを実装する。
      RED: `TestResolveUpdatedStatusPreservesActiveOnNonTrustChange` /
      `TestResolveUpdatedStatusDegradesOnTrustChange` (`usecases/connection_update_test.go`) を
      先に fail 確認 (未定義の `ResolveUpdatedStatus`) → GREEN。
- [x] T006 [Handlers] `deleteAdmin` のステータスガードを撤去し、`test` を実接続確認に置き換える。
      RED: `TestTestConnectionReportsUnreachableEndpointAndUnresolvableSecret`
      (`protocol_oidc/client_test.go`)、`TestValidateSigningCertificatesRejectsExpiredOrUnparsableCertificate`
      (`protocol_saml/response_test.go`)、`TestDeleteAdminSucceedsForActiveConnection` /
      `TestTestAdminReportsStructuredReachabilityResult` (`handlers_http/routes_test.go`) を
      先に fail 確認 → GREEN。SSRF 対策は `protocol_oidc` の既存 `validateRemoteURL`/
      `safeHTTPClient` (DNS pin・redirect 禁止・1MiB cap) をそのまま再利用。
- [x] T007 [Secrets] `EnvelopeCrypto`/`DataKeys` を使う secret 保管アダプタを実装し、
      `FieldMigrator` で既存 `env:` 参照からの移行を登録する。
      `db_postgres/repositories_test.go`・`reencrypt_test.go` に実 Postgres + Tink
      (`envelope_cleartext`) を使った暗号化/復号/dual-read/移行/PendingCount のテストを追加、
      green。**self-attest 上の逸脱**: このタスクは repository/migrator の実装コードを先に書き、
      その後にテストを書いた (先に RED を確認していない)。理由は `datakeys.FieldCipher` の
      配線を伴う実装を先に固めないとテストの土台 (`newTestCipher` 等) を書けなかったため。
      実装後のテストは green で、実際の暗号化/dual-read/移行スキップ挙動を検証済み。
- [x] T008 [Schema] `infra/schema/postgres.sql` の CHECK 制約から `'draft'` を除去し、
      既存行の移行 SQL を用意する ([[wi-308-reconsider-psqldef-adoption]] に倣い、データ移行は
      宣言的 schema に混ぜず one-off script に分離)。psqldef 冪等性 (SQL コメント排除等) は
      wi-308 のパターンを踏襲。
- [x] T009 [UI] ADR-086 / wi-126 の Detail-then-Edit ポリシーに従い、一覧 (`identity-providers.tsx`)
      / 詳細 (`$id.index.tsx`) / 編集 (`$id.edit.tsx`) / 作成 (`new.tsx`) を別ルートに分割し、
      `adminNav.ts` にトップレベル項目を追加、`src/routes/-page.tsx` の `PAGE_TITLES` に
      追加、`AdminSettingsPage.tsx` から撤去した。設定テスト結果表示とシークレット実値入力に
      改めた。RED: `AdminIdentityProvidersPage.test.tsx` の削除確認ダイアログ (Active/Disabled
      非対称) とテスト結果バナーのテストを実装直後に green 化して確認 (self-attest: 実装と
      テストをほぼ同時に書いたため厳密な事前 RED 確認はしていない)。
      `ui-page-lines` 複雑度予算 (400 行) 超過を `AdminIdentityProviderFormShared.tsx` /
      `AdminIdentityProviderFormFields.tsx` への抽出で解消し、debt 登録なしで通した。
- [x] T010 [Verify] 下記 Verification を緑にする。`just scl-render` で派生物を再生成した。

## Verification

- `just check` / `just check-scl` / `just check-work-items` / `just check-ids`
- `just test-go` / `just test-go-race` / `just verify-go`
- `just verify-ui` / `just test-ui-unit`
- 手動: (1) 未着手の接続 (旧 Draft 相当) を作成直後に確認なしで削除できることを確認する。
  (2) 稼働中 (Active) の接続の表示名だけを編集しても Active のままログインが継続することを
  確認する。issuer を変更すると Disabled に落ちることを確認する。
  (3) 到達不能な issuer / 存在しない環境変数を指す `secret_reference` 相当の設定で
  「設定テスト」を実行し、明確な失敗理由が表示されることを確認する。正しい設定では
  成功が表示されることを確認する。(4) クライアントシークレットに実値を入力して保存し、
  DB には暗号化された値のみが保存されていて API レスポンスにも含まれないことを確認する。
  (5) 管理画面のトップレベルナビゲーションから外部IDプロバイダーの一覧画面に直接遷移でき、
  「設定」タブには表示されないことを確認する。

## Risk Notes

状態モデルの変更は login routing のゲート条件 (`Active` のみ使用可能) に触れる公開 API の
enum 変更であり、既存にテナントが作成した Draft 接続がある場合はデータ移行が必須。
移行を誤ると「意図せず Active になる」「意図せず消える」のどちらの事故も起こり得るため、
移行 SQL は `draft → disabled` の一方向のみとし、Active 昇格は必ず管理者の明示操作を
経由させる。

secret 保管方式の切替は、既存に `env:VARNAME` 参照を持つ接続がある本番環境では、切替と
同時にログインが壊れないよう「旧方式も読み取りだけ当面サポートしつつ、新規保存は必ず
envelope 暗号化する」段階移行が必要になる可能性が高い。旧方式を打ち切るタイミングは
ADR で明示する。

「設定テスト」の実処理化はサーバーから任意到達性確認の外向き通信を伴う。
`RefreshIdentityProviderMetadata` が既に持つ SSRF 対策 (固定済み endpoint のみを対象にし
任意 URL を取得しない) を必ず踏襲し、新たな SSRF 面を作らないこと。

## Completion
- **Completed At**: 2026-07-31
- **Summary**:
  Motivation で指摘した4点全てに対応した。(1) 一覧/詳細/編集/作成の別ルート分割と
  `adminNav.ts` トップレベル項目化 ([[ADR-086]] /
  [[wi-126-admin-and-account-ui-consistency-and-navigation-policy]] を Entra 連携と同じ形で
  適用)。(2) Draft 廃止と `Active`/`Disabled` の2値化により未着手接続の即時削除を可能化
  ([[ADR-149]])。(3) 「設定テスト」を OIDC discovery/JWKS 到達性・SAML 証明書検証・secret
  解決可能性の実処理に置き換え、構造化結果 (成功/失敗理由) を画面表示。(4) `secret_reference`
  を `EnvelopeCrypto`/`DataKeys` による実値暗号化保存に切替え、UI をシークレット実値入力
  (write-only) に変更 ([[ADR-150]])。
  Out of Scope として明記済みの未対応: 「設定」タブに残る他項目の整理、`entra-federation`
  との画面統合、authorization_code フロー全体のシミュレート、`EnvelopeCrypto`/`DataKeys`
  自体の新機能追加 (いずれも Scope の Out of Scope 節に記載済み)。
  test-first からの逸脱 (self-attest): T007 (secrets アダプタ) と T009 (UI) は、配線の
  複雑さ・実装とテストの一体的な組み立てを理由に、事前の RED 確認を経ずに実装とテストを
  ほぼ同時に書いた (Tasks の該当項目に理由を明記)。それ以外の層 (Domain/Usecase/Handlers) は
  RED → GREEN を確認した。
  手動検証 (Verification 節の (1)〜(5)) はブラウザでの実機確認を本セッションでは実施しておらず、
  各項目に対応する自動テストで同等の挙動を検証した点をユーザーに開示する。
- **Verification Results**:
  - `just check` / `just check-scl` / `just check-ids` / `just check-work-items`: green。
  - `just verify-go` (lint-go + test-go-race) / `just test-go`: 2件の pre-existing かつ
    本 WI と無関係な失敗を検出した。`TestMfaFactorReencryptor_NoPlaintextSurvivesBackfillAcrossTenants`
    (`backend/authentication/totp/db_postgres`) は既存のテスト分離不備によるフレークで単独実行では
    green。`TestAgentStatusMatchesSCL` (`backend/shared/spec`) は本 WI が触れていない
    `identity-management.yaml` の `AgentStatus` と Go enum のケース不一致で、本 WI 着手前から
    存在する不具合。本 WI が新規に導入した federation 関連パッケージのテストは全て green。
  - `just verify-ui` (format-check/lint/typecheck/build) / `just test-ui-unit`: green
    (474 tests pass)。
  - 手動検証 (1)〜(5) はブラウザでの実機確認を実施していない。代わりに以下の自動テストで
    同等の挙動を検証した: (1)(2) `handlers_http/routes_test.go` の
    `TestDeleteAdminSucceedsForActiveConnection` /
    `TestUpdateAdminDegradesOnlyOnTrustSourceChange`。(3) `protocol_oidc`/`protocol_saml` の
    `TestTestConnection*`/`TestValidateSigningCertificates*` と
    `AdminIdentityProvidersPage.test.tsx` のバナー表示テスト。(4) `db_postgres` の
    `TestConnectionRepositoryEncryptsRealSecretAtRest`。(5) `adminNav.test.ts` と
    `AdminSettingsPage.test.tsx` (identity-providers タブが撤去されたことの確認)。

### Post-completion fix: secret resolution rejected already-plaintext secrets

ユーザーが実際に外部 IdP (Duende の公開デモ IdentityServer) を設定し「設定をテスト」を
実行したところ、`secret reference cannot be resolved` で失敗した。原因は
`protocol_oidc.Client.TestConnection`/`ExchangeAndValidate` が `SecretReference` を
常に `secrets_env.Resolver` (`env:` scheme 専用) で解決しようとしていたこと。ADR-150 の
repository 層は ciphertext を復号した実値、あるいは memory backend では入力された実値を
そのまま `SecretReference` に入れるため、`env:` プレフィックスを持たない実値は
`secrets_env.Resolver.Resolve` が必ずエラーを返し、テストも実ログインも失敗していた
(この手動確認で発覚するまで自動テストのカバレッジに穴があった)。

`Client.resolveSecret` を追加し、「`env:` プレフィックスを持つ値だけ `SecretResolver` で
解決し、それ以外 (空文字含む) はそのまま実値として使う」規約に統一。
`TestConnection`/`ExchangeAndValidate` 双方をこのヘルパー経由に変更した。RED:
`TestTestConnectionAcceptsAlreadyResolvedSecretWithoutAResolver`
(`protocol_oidc/client_test.go`) を先に fail 確認 → GREEN。
`just build-go` / `golangci-lint run ./backend/authentication/federation/...` /
`go test ./backend/authentication/federation/...` は green。
