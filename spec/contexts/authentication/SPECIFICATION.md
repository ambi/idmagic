---
context: authentication
updated_at: 2026-08-15
---

# Authentication Specification

## Overview

エンドユーザーの資格情報の検証、MFA、ログインセッション、ステップアップ認証、パスワードの変更とリセット、アカウントの復旧、ログイン時のフェデレーション、認証イベントを所有する。

`User` / `Group` / `Agent` のライフサイクルそのものは `IdManagement` が持つ。この Context が扱うのは、そのプリンシパルが本人であることをどう確かめ、確かめた結果をどうセッションとして保つかである。

## Glossary

| Term | Definition | Aliases |
|---|---|---|
| IdentityBroker | 外部の IdP による認証結果を検証し、テナント内のローカル User と安全に相関させて、LoginSession の発行へ引き渡す機能。 |  |
| ExternalIdentityProvider | IdMagic にとって上流の認証権威となる OIDC Provider または SAML Identity Provider。 | 上流 IdP, 外部 IdP |
| FederatedIdentity | テナント、プロバイダー、外部の不変な subject の組を、ローカルの User へ一意に結び付ける関連付け。 |  |
| JitProvisioning | 検証済みの外部クレームと、テナントが明示したポリシーおよびクレームの対応付けに基づき、初回のフェデレーションログインの途中でローカルの User を作成すること。 | JIT プロビジョニング |
| Totp | RFC 6238 に基づく時刻同期型のワンタイムパスワード。 | totp, otp |
| Webauthn | WebAuthn の公開鍵クレデンシャルによる認証。 | webauthn |
| RecoveryCode | TOTP や WebAuthn の認証要素を失ったときに使う、単回限りの控えの復旧コード。 | recovery_code |
| EndUser | 認証済みまたは認証を試みる一般利用者。ログイン・MFA継続・パスワードリセットなど、認証が未完了の操作の主体を指す。 |  |
| ResourceOwner | OAuth2/OIDC 認可フローでリソースの所有者として認可判断を行う利用者。EndUser と同一人物を OAuth2 文脈で指す呼称。 |  |

## Standards

### OpenID Connect Core 1.0

1.0 incorporating errata set 2 — https://openid.net/specs/openid-connect-core-1_0.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| OIDC-CORE-CODE-FLOW | required | MUST | 外部 OIDC 認証は authorization code flow を使い、ID Token の署名、issuer、audience、有効時間、nonce を検証する。 |
| OIDC-CORE-CSRF | required | SHOULD | callback は login attempt に束縛された単発 state を照合し、不一致または再利用を拒否する。 |

### OpenID Connect Discovery 1.0

1.0 incorporating errata set 2 — https://openid.net/specs/openid-connect-discovery-1_0.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| OIDC-DISCOVERY-ISSUER | required | MUST | Discovery Metadata の `issuer` は設定した発行者と完全一致し、エンドポイントと JWKS URI は事前に許可された HTTPS オーソリティに限定する。 |

### TOTP Time-Based One-Time Password Algorithm

RFC 6238 — https://www.rfc-editor.org/rfc/rfc6238.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC6238-TOTP | optional | MUST | TOTP の認証要素を使うときは、共有シークレットと時間ステップから OTP を生成し検証する。 |

### Digital Identity Guidelines — Authentication and Authenticator Management

NIST SP 800-63B-4 — https://pages.nist.gov/800-63-4/sp800-63b.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| NIST63B4-PASSWORD-MINIMUM | excluded | MUST | 単一要素の認証に使うパスワードに 15 文字以上の最小長を要求する。 |
| NIST63B4-NO-COMPOSITION | required | MUST NOT | 文字種の混在のような、パスワードの構成規則を課さない。 |
| NIST63B4-PASSWORD-STORAGE | required | MUST | パスワードは salt とコストパラメーターを備え、オフライン攻撃に耐えるハッシュとして保存する。 |

### Web Authentication — An API for accessing Public Key Credentials Level 3

Candidate Recommendation Snapshot — https://www.w3.org/TR/webauthn-3/

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| WEBAUTHN3-AUTHENTICATION | required | MAY | WebAuthn の認証要素を使うときは、オリジンと Relying Party の範囲に限定された公開鍵クレデンシャルを検証する。 |
| WEBAUTHN3-REGISTRATION | required | MUST | WebAuthn の credential を登録するときは、attestation の challenge / RP ID / origin を検証し、COSE の公開鍵と sign count を保存する。 |

### Authentication Method Reference Values

RFC 8176 — https://www.rfc-editor.org/rfc/rfc8176.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC8176-AMR-VOCABULARY | required | MUST | LoginSession.amr は RFC 8176 登録値 (pwd, otp, webauthn, hwk, swk) のサブセットに、本アプリ固有の非 IANA 拡張値 rc (recovery code) を加えた語彙のみを許可する。 |

## State Transitions

### IdentityProviderConnectionLifecycle

上流との接続は、利用できる `Active` と経路を止めた `Disabled` の 2 状態だけを行き来する。作成直後は `Disabled` である。メタデータの再取得の失敗や、信頼の根拠にあたらない項目の更新は状態を変えず、最後に成功した内容を保持する。

Initial: `Disabled` Terminal: none

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Active | IdentityProviderConnectionDisabled | — | Disabled |  |
| Disabled | IdentityProviderConnectionActivated | — | Active |  |

## Authorization Boundary

この Context の大半は認証完了前のリクエストを扱うため、ロールではなく、そのセッションで確認済みの認証要素と認証時刻が境界になる。

セルフサービス API (`/api/account/*`) は、認証済みセッション自身の `actor.sub` に対してのみ作用する。URL、本文、クエリ文字列で与えられた `sub` や `tenant_id` を信頼することは決してないため、ユーザーをまたぐアクセスとテナントをまたぐアクセスは構造的に生じ得ない。本人が変更できるのは自分の表示名、`editable_by_user=true` の属性、パスワードに限る。ロール、状態、組織の属性、`required_actions` は管理者専用であり、ユーザーからは閲覧しかできない。例外は、パスワード変更の成功で `update_password` が解除されるように、本人の操作の副作用として解除される場合だけである。

機密性の高いセルフサービス操作 — `ChangePassword`、`RemoveTotpFactor`、`RequestEmailChange`、`RevokeMyOtherSessions` — には、CSRF と同一オリジンの検査に加えて直近の再認証 (ステップアップ認証) を要求する。セッション Cookie を盗んだ攻撃者が、そのままアカウントを乗っ取れないようにするためである。満たさない場合は `401` ではなく `403 step_up_required` を返す。セッションは認証済みだが、この操作に必要な直近の認証を示していないからである。

MFA の強制開始後に未登録のユーザーが到達できるのは登録専用のフローだけであり、そこへ入るには管理者が発行した未消費・未失効・期限内の `MfaEnrollmentBypass` を要する。`pending_purpose=Enrollment` の保留中セッションは、登録専用 API と元の認可トランザクション以外のすべての場所で未認証として扱う。アカウント、管理、アプリケーションのいずれのリソースにも到達できない。

上流の IdP から受け取ったものはすべて信頼しない。ログインリクエストが使うのは保存済みのプロバイダーとエンドポイントの設定だけであり、ブラウザーから任意の Discovery URL やトークン URL を指定することはできない。クライアントシークレットはデータベースに保存せず `secret_reference` だけを保持し、外部トークンと SAML Assertion はログイン時に検証したうえで保持しない。

この Context が持つ管理 API のうち、セッションと認証情報に対する操作は API アクセストークンから到達できる。`sessions:read` が利用者のセッションとサインイン履歴の参照だけを、`sessions:write` がセッションの失効を許可する。MFA 登録の一時免除と認証器のリセットは対象 `User` の資格情報を変えるので `users:write` に属する。認証イベントのバケットは監査の派生的な参照なので `audit:read` に対応させる。

外部 IdP 接続の管理 API はこれと異なり、**API アクセストークンからは到達できない対話セッション限定の操作** とする。攻撃者が管理する IdP を登録できることは、任意のユーザーとしてサインインできることと同じであり、認証の迂回そのものだからである。この能力を既存の設定変更スコープへ畳み込むと、設定を更新するために発行したトークンが黙って認証の迂回能力を得る。したがって外部 IdP 接続の参照と変更は、ブラウザーのログインセッションまたは管理ポータルのアクセストークンからのみ行える。

## Design

### Inbound identity federation

`federation/` はアイデンティティブローカーの機能単位である。テナント単位の上流 OIDC / SAML 接続、外部 subject との関連付け、ログイン試行、プロトコル検証、関連付けポリシー、IdMagic のログインセッションへの引き渡しを所有する。下流向けの SAML IdP と WS-Federation の発行は、各プロトコルの Context が所有する。IdManagement は JIT 用に資格情報を持たないユーザーを作成するが、ログイン時の相関ポリシーは持たない。相関と JIT の判断はセッション発行と不可分なので、Authentication が直接所有し、Sourcing には委ねない。

ブローカーは、まず `FederatedIdentity` を通じて不変な外部 subject を解決する。検証済みメールアドレスによるリンクを許可するのは、接続に明示的なポリシーがあり、上流のクレームが検証済みで、テナント内で一意に一致する場合だけである。JIT は接続ごとに個別に有効化し、メールドメインの許可リストでさらに絞り込める。明示的なリンクとリンク解除の操作には直近のステップアップ認証を要求し、最後に残った利用可能なサインイン手段は取り除けない。

プロトコルのアダプターは上流の文書をすべて信頼できないものとして扱う。OIDC は保存済みの HTTPS の discovery エンドポイント、PKCE 付きの Authorization Code、`state`、`nonce`、issuer と audience と時刻のチェック、そして制限された JWK のアルゴリズムを使う。SAML は相関の取れた AuthnRequest を使い、XML の署名、issuer、destination、audience、subject confirmation、時刻、再送を検証する。要求していない IdP 起点のレスポンスと暗号化された Assertion は、最初の SAML アダプターの範囲外とする。

クライアントシークレットは `secret_reference` だけで表す。実行時の解決器は `env:NAME` の参照を受け付ける。生のシークレットも上流のトークンやアサーションも、保存も返却もしない。公開するプロバイダーのディスカバリーに含めるのは、有効なプロバイダーの識別子、表示名、プロトコルだけである。

コールバックを検証した後、ブローカーは AMR に `federated` を持つ通常の Authentication セッションを作成する。OAuth の認可は `/authorize/resume` を通じて再開するため、アプリケーションポリシー、必須操作、同意、認可コードの発行はブローカーに複製せず、引き続き OAuth2 が所有する。

### Persistence

`authentication_sessions` を `LoginSession` の単一の正とする。通常の [`tenant_id` retention classes](../../SPECIFICATION.md#2-tenant_id-retention-classes) の例外として、ユーザーから導出できる `tenant_id` も保持する。不透明な Cookie 値であるセッション ID をすべてのリクエストで照合する際に、テナント境界をフェイルクローズで確認するためである。失効では行を削除せず、`revoked_at` と `revoke_reason` を設定する。これにより、失効の再実行は安全な no-op となり、物理削除は保持期間に従う独立した処理になる。インデックスは、ユーザーごとの有効なセッションを `auth_time DESC` でページングする処理と、`expires_at` による一括削除に使う。

セッションの有効期限は作成時に設定する固定 1 時間 (`SessionTTLSeconds`) であり、利用によって `expires_at` を延長しない。更新するのは `last_seen_at` だけで、書き込み増幅、VACUUM の負荷、ロック競合を抑えるため、最短でも 5 分間隔とする。90 日の保持期間は、期限切れの行を調査用に残す期間であり、セッションの有効期間ではない。

`mfa_factors.secret` は以前からある平文の TOTP の種の列であり、既存の行が読めるようにするためだけに残している (二重読み)。新しい書き込みは `secret_key_version` と `secret_ciphertext` を埋め、`secret` は `NULL` のままにする。残りの平文の行は保留中の埋め戻しで移行し、その後 `secret` を削除する。

`webauthn_credentials` は `credential_id` を鍵とする。1 人のユーザーが複数登録できるからであり、同じ理由で `mfa_factors` とは別のテーブルのままにしている。`public_key` は COSE の公開鍵 (base64url) を保持する。`recovery_codes` は平文のコードを決して保存せず、`code_hash` (SHA-256 の 16 進) だけを持つ。`consumed_at` が `NULL` でなければそのコードは使用済みで再送できず、再生成はユーザーの一式をまとめて置き換える。`webauthn_sessions` は WebAuthn の手続きの challenge のストアであり、`GetDel` は `DELETE ... WHERE expires_at > now() RETURNING data` である。

`tenant_correlation_salts` はテナントごとのシークレットで、利用者名や IP の相関用ハッシュ（`SaltedHash`）と、スロットルや集計の `keyHash` の算出に使う。これにより相関がテナントをまたいで集約されることはない。あらかじめ用意するのではなく、最初に使うときに生成する。

`login_throttle_counters` は `LOGGED` である。切り替え時に失うことが多層防御の後退にあたるからである。また `fillfactor = 80` を使い、カウンターが頻繁に受ける同一行への `UPDATE` (HOT 更新) のために余地を残す。`identifier_hash` は SHA-256 の 16 進の要約なので、平文の利用者名や IP は保持されない。

### Password lifecycle

パスワードポリシーは NIST SP 800-63B-4 §3.1.1.2 に従う。長さ (`min_length=12`、`max_length=128`)、ユーザー名、メールアドレス、その local part との大文字小文字を無視した照合、同梱する一般的なパスワードの辞書を適用する。誤検知を避けるため、4 文字未満の識別子は照合しない。文字種の構成規則と強制的な定期変更は要求しない。

「記号を含める」「大文字を含める」などの構成規則を設けないのは、NIST が SHALL NOT としているためである。この種の規則は `password` を `Password1!` に変えるような予測可能な対応を促しやすく、問い合わせ負荷やパスワードの使い回しを増やす。代わりに、長さ、漏洩・辞書との照合、ログインスロットル、MFA を適用する。

長さ、履歴の深さ、有効期限はテナントごとに上書きできる (`PasswordPolicyOverride`)。省略した項目は全体のデフォルトを継承し、パスワードの変更、リセット、管理者によるユーザー作成のすべてで、解決後の同じポリシーを評価する。上書きは厳しくする方向に限り、全体のデフォルトより短い `min_length`、長い `max_length`、浅い `history_depth` は拒否する。

パスワードの有効期限 (`max_age_days`) は、デフォルトで無効なテナント単位の選択項目であり、規制上の理由で定期変更が必要な場合にだけ使う。有効期限は `max(password_changed_at, tenant のポリシー更新時刻)` から測り、設定を有効にした直後に既存ユーザーのパスワードが一斉に期限切れになることを防ぐ。期限切れでも認証自体は成功させ、`update_password` の必須操作を付与する。変更またはリセットが成功すると解除する。パスワード資格情報を持たないユーザーは対象外とする。`max_age_days` は 30〜3,650 日に制限する。

パスワードの変更は直近 5 件のパスワードのハッシュの再利用も拒否する (`history_depth=5`)。これらは `password_hash` と同じ Argon2id の PHC 文字列として `password_histories` に保存する。別の符号化にしても攻撃のコストは上がらず、二重の保守が増えるだけだからである。履歴は登録時とパスワード変更時の両方で書き込むが (初回と同じ値への変更も検出できるようにするため)、照合はパスワードの変更時のみ行う。初回の登録には比較する相手がないからである。

`BreachedPasswordChecker` は、同梱の辞書に加えて HIBP Range API を利用できる。k-匿名性を保つため、外部へ送るのは SHA-1 の接頭辞だけである。この追加検査はフェイルオープンとし、タイムアウトや障害では資格情報の変更を止めず `breached=false` を返すが、障害自体は監査用に記録する。デフォルトは何もしないアダプターで、`breached_password_check_enabled=false` とする。

パスワードを忘れた場合の処理は、単回限りの 32 バイトの乱数のトークンを発行し、`password_reset_tokens` には SHA-256 のハッシュとしてのみ保存する (`ttl=1800s`)。引き換えではパスワードの変更と同じ検証、履歴、漏洩の一連の処理を通す。この経路の応答はすべて同一である (メールアドレスが存在するか、未確認か、打ち間違いかによらず `204`) ので、復旧の流れが利用者名の存在を暴く手段になることはない。メールの配送は最大限努力であり、送信の失敗が呼び出し元に現れることはなく記録のみに留まるので、認証されていない側から SMTP の障害を探ることもできない。

メールは `EmailSender` ポートを通じて配送する。本番アダプターは送信サービスごとの HTTP SDK ではなく SMTP を使い、デフォルトで STARTTLS を要求し、PLAIN 認証は TLS 上でだけ許可する。送信前に CRLF と NUL を除去し、HTML 本文へ差し込む値を無害化し、件名を RFC 2047 で符号化して、SMTP ヘッダーや HTML への注入を防ぐ。

### Login throttling

ログイン試行は、アカウント単位と IP 単位で独立して抑制する。アカウント単位では 900 秒間に 10 回、IP 単位では 900 秒間に 30 回失敗すると、900 秒間拒否する。前者は特定アカウントへの辞書攻撃、後者は 1 つの発信元から多数のアカウントを狙う credential stuffing を抑止する。どちらかの閾値を超えれば 429 を返す。カウンターは PostgreSQL の共有テーブル `login_throttle_counters` に保存し、すべてのレプリカで共有する。

カウンターは平文ではなく識別子の SHA-256 のハッシュを鍵とする。またアカウント単位のカウンターは、アカウントが存在するかを確かめる *前* に加算する。存在しない場合はパスワードの検証の段で固定の番人のハッシュを使う。そうしないと、処理時間 (Argon2 の検証が起きるかどうか) や 429 の応答の形から、どの利用者名が存在するかが漏れるからである。ログインの成功はアカウントのカウンターを消すが、IP 単位のカウンターは意図的に残す。共有のオフィスや NAT の IP から 1 回ログインが成功しても、その IP の残りの通信が信頼できることにはならず、IP のカウンターは時間枠によって自然に消えるからである。共有のスロットルのストアへ到達できない場合、ログインは抑制なしに通すのではなく fail-closed で失敗する。クライアントの IP はデフォルトでは直接の接続相手から読む。`X-Forwarded-For` を尊重するのは `TRUSTED_FORWARDED_HOPS` を明示的に有効にした場合だけである。無条件に信頼すると、攻撃者が IP 単位の軸を偽装で回避できるからである。締め出しのたびに `LoginThrottled` を発行し、そこには平文ではなく、下の集計と同じテナントの salt を効かせた `keyHash` を載せる。

### Authentication event logging

認証イベントは 2 系統で保持し、攻撃による急増で監査ストアが過負荷になったり、重要な兆候が埋もれたりすることを防ぐ。`authentication_events` は個々の操作（成功、失敗、MFA、フェデレーション、セッション開始）を 1 行ずつ記録する。`authentication_event_buckets` は、同じ `(tenant, kind, keyHash)` から続く集中を 5 分単位で集約し、行を増やさず `count` を更新する。アカウントまたは IP がログイン試行の制限値に達した後は、失敗ごとの `AuthenticationFailed` を発行せず集計側へ回す。スロットルのカウンターと、テナント固有のソルトを使った同じ `keyHash` を共有するため、平文を公開せず両者を照合できる。

集計の閾値 (5 分の時間枠。デフォルトでアカウント 10 回、IP 50 回、テナント 1000 回の失敗。テナントごとに上書き可能) は、ふつうの打ち間違いを個別のイベントとして見えるように保つことと、本物の氾濫が延々と行を書き続けるのを許さないことの釣り合いを取る。低すぎれば通常の間違いが集計に消え、高すぎれば氾濫がいつまでもたたまれない。各時間枠はちょうど 1 つの `AuthenticationEventAggregated` の管理用のイベントを発行し、同じ時間枠の以後の発生は `count` を増やすだけである。管理者はその行から、同じ鍵についての個別の失敗の見本を最大 10 件までたどれる。なりすましのイベント (`SessionImpersonationStarted` と `Ended`) は、集計へのたたみ込みからも保持期間の短縮からも除外する。管理者がユーザーとして振る舞ったことを記録するものであり、そのユーザーを守るために手を加えずに残すからである。

保持期間は種別ごとに異なり、`occurred_at` に対する冪等な毎時の一括スイープで強制する。成功イベントは 365 日、個別の失敗行は 30 日、集計行とセッションや MFA のチャレンジ行は 90 日保持する。テナントは全体の上限 `max_retention_days` の範囲で期間を調整できるが、なりすましイベントを上限より短くすることはできない。検索性能は、スイープによる行数の抑制、`(tenant_id, occurred_at)` インデックス、クエリの `limit` によって確保する。

確定したアカウントに結び付くイベント（`UserAuthenticated` や OAuth2 フローのイベント）は `user_id` で相関し、管理者がユーザー名で検索するときは、その場で `user_id` に解決する。ユーザー名、IP、User-Agent、端末の指紋は各イベントの通常の保持期間中は平文で保存し、`AuthenticationFailed` のユーザー名も 30 日間保持する。所在地は国コードだけを保存する。`LoginThrottled` と集計では、テナントごとのソルトを使った `keyHash` をスロットルと集計のキーとして使用するが、監査上の PII 項目としては扱わない。

### Account portal trust boundary and step-up authentication

アカウントのポータルの API (`/api/account/*`) は、認証されたセッション自身の `actor.sub` に対してのみ作用する。URL、本文、問い合わせ文字列で与えられた `sub` や `tenant_id` を信頼することは決してないので、ユーザーをまたぐアクセスやテナントをまたぐアクセスは構造的に生じ得ない。これは管理 API (`/api/auth/account`。ロールを含む) とは別の契約である。ポータル自身の要約のエンドポイント (`/api/account/summary`) は意図的にロールを省くので、利用者自身が操作する面が管理用の情報を誤って漏らすこともない。利用者自身が変更できるのは、自分の表示名、`editable_by_user=true` の属性、パスワードである。ロール、状態、組織の属性、`editable_by_user=false` の属性は管理者専用のままであり、`required_actions` はユーザーからは閲覧のみで、付与も取り消しもできない。ただしユーザー自身の操作の副作用として解除されるもの (パスワードの変更が成功したときの `update_password` など) は除く。ポータルの UI は独立した外枠であり、たまたま管理者のロールを持つユーザーに対しても管理用の案内を一切出さない。

CSRF と同一オリジンの検査は自己管理の変更操作を保護するが、セッション Cookie 自体が盗まれた後には役立たない。Cookie を持つ攻撃者は、パスワード変更、MFA の解除、通知先の変更などによってアカウントを乗っ取れる。そのため機密性の高い自己管理操作には、CSRF に加えて直近の再認証（ステップアップ認証）を要求する。ステップアップ認証が必要な操作は `ChangePassword`、`RemoveTotpFactor`、`RequestEmailChange`、`RevokeMyOtherSessions` であり、テスト (`TestStepUpAnnotatedInterfacesMatchGatedHandlers`) が注釈と実際に制御されるハンドラーの不一致を防ぐ。`max(session.auth_time, session.step_up_at)` が `StepUpRecencySeconds`（5 分）以内であれば、セッションをステップアップ認証済みと見なす。したがってログイン直後のセッションもステップアップ認証済みであり、Google や Okta で利用者が慣れている再認証の形と一致する。制御に失敗した場合は `401` ではなく `403 step_up_required` を返す。セッションは認証済みだが、この操作に必要な直近の認証を示していないためである。`POST /api/account/step_up/complete` が成功すると、UI は元のリクエストを再送する。認証の新しさは `LoginSession` の行自体に `step_up_at` として保存するため、Cookie とともに別の端末へ引き継がれることはない。

### WebAuthn/passkey MFA and recovery codes

TOTP は RFC 6238 の標準的なパラメーター（SHA1、30 秒のステップ、6 桁、前後 1 ステップの許容、160 ビットの seed）を使う。WebAuthn の資格情報は `mfa_factors` へ押し込めず、`credential_id` をキーとする専用テーブル `webauthn_credentials` に置く。`mfa_factors` の `(user_id, type)` という同一性では、ユーザーごと・種別ごとに 1 つの要素しか持てないが、WebAuthn の価値は 1 つのアカウントへ複数の認証器を登録できることにあるためである。CBOR と COSE の解析、署名検証といったセレモニーのロジックは自作せず、`go-webauthn/webauthn` に全面的に委ねる。自作のアテステーションとアサーションの検証器では、わずかな誤りがそのままセキュリティ回避につながるためである。登録と認証のチャレンジには新しいストアを設けず、既存の一時的な `SessionStore` を再利用する。登録では `sub`、認証では保留中のログインセッション ID をキーとする。チャレンジは他のセッションデータと同じライフサイクルを持つ、短命なサーバー側の値だからである。

RP ID と許可するオリジンはデプロイ時の設定（`WEBAUTHN_RP_ID` と `WEBAUTHN_RP_ORIGINS`）から取得し、起動時に検証したうえで、セレモニーごとに再確認する。アテステーションは `none`（端末機種の強制よりプライバシーを優先）、ユーザー検証は `preferred`、常駐鍵は `discouraged` とする（`challenge_bytes=32`、`timeout_seconds=120`）。この段階では WebAuthn をパスワードと組み合わせるフィッシング耐性の高い第 2 要素として追加し、パスワードレス認証や Discoverable Credential のフローは明確に対象外とする。返された `sign_count` が保存値以下の場合（0 から 0 は除く）、認証器が複製された証拠と見なしてアサーションをその場で拒否する。真正な認証器のカウンターは増加する一方だからである。

復旧コード（SHA-256 ハッシュだけを保存し、`consumed_at` によって単回利用を保証する。再生成時は一式をまとめて置き換え、紛らわしい文字を除いた文字集合から 10 文字のコードを 10 個生成する）は、TOTP や WebAuthn の認証要素を失ったときの控えとしてのみ存在し、`User.mfa_enrolled` には意図的に **数えない**。復旧コードを単独の第二認証要素として扱うと、ユーザーがそれを唯一の MFA として利用でき、控えを持つ意味が失われるからである。`mfa_enrolled` は「TOTP 認証要素または WebAuthn クレデンシャルが 1 つ以上存在すること」から導出し、どちらかを削除するたびに再計算する。復旧コードの生成、再生成、失効にはステップアップ認証が必要である。第二認証要素の検証に成功すると `acr` は `urn:idmagic:acr:mfa` へ上がり、`amr` には `webauthn`、復旧コードを使った場合は `rc` が加わる。`webauthn` は RFC 8176 の登録値である一方、`rc` はこのアプリケーション独自の IANA 未登録値である。

### MFA enrollment bypass

ある時点から MFA を強制すると鶏と卵の問題が生じる。未登録のユーザーをすべて拒むと、正当な新規ユーザーと認証要素を失った利用者の復旧が止まる。一方、パスワードの検証に成功しただけで誰でも認証要素を登録できるようにすると、パスワードを知る攻撃者が自身の認証要素を登録し、MFA 要件を無効にできてしまう。強制開始前は、未登録のユーザーも通常のパスワードセッションを得て、ステップアップ認証で保護されたアカウントのセキュリティ設定画面から認証要素を先に登録するよう促される。強制開始後は、管理者が発行した `MfaEnrollmentBypass` が未消費、未失効、期限内である場合に限り、未登録のユーザーは登録専用フローに到達できる。これは短命で単回限りのサーバー側の許可であり、配布されるシークレットではない。強制開始日と猶予期間は運用上の時刻にすぎず、誰が登録しているかを信頼する根拠にはならない。

パスワードによるログインが成功すると、その許可を不可分に消費し、同じ `LoginSession` を `pending_purpose=Enrollment` へ移す。この保留中のセッションは、登録専用 API と、元々処理していた認可トランザクションを除くすべての場所で未認証として扱われる。アカウント、管理、アプリケーションのリソースには到達できない。ユーザーが新しい認証要素の所持を証明して初めて、セッションは第二認証要素の AMR を得て、MFA 済みとして再開する。期限切れ、発行不可、失効済み、消費済みの許可は、より弱い経路へフォールバックせずフェイルクローズで失敗する。最初に登録できる認証要素は TOTP である。WebAuthn は同じ保留と許可の契約を使うアダプターとして追加でき、別のポリシーやセッション状態を必要としない。

### Design Decisions

- ログイン時のアイデンティティブローカー（上流の OIDC / SAML 接続、外部 subject へのリンク、リンクと JIT のポリシー）は、独立した Sourcing Context へ分けず Authentication が直接所有する。
- PostgreSQL の `authentication_sessions` を `LoginSession` の唯一の情報源とする。失効時も行を削除せず、失効済みであることを記録する。
- `users.id` を正式かつ全体で一意なユーザー識別子とし、プロトコルの `sub` クレームはここから導出する。その逆ではない。
- 認証のイベントの保持期間は種別ごとに非対称 (365 / 30 / 90 日) で、全体の上限の範囲でテナントが調整でき、テーブルの分割や低速な保存先ではなく冪等な毎時の掃除が強制する。
- MFA の TOTP seed を含め、アプリケーションのデータベースに残る可逆なシークレットは平文で保存せず、`DataKeys` Context と `EnvelopeCrypto` ポートによるエンベロープ暗号化へ移す。
- WebAuthn のクレデンシャルと復旧コードは `mfa_factors` へ押し込めず、独自のテーブルとして表す。セレモニーの処理は `go-webauthn/webauthn` へ委ね、復旧コードの所持だけでは `mfa_enrolled` に数えない。
- ログインのスロットルをはじめとする共有の一時的な状態は、ストアへ到達できないとき抑制なしの試行を通さず fail-closed で失敗する。
- パスワードのポリシーは NIST SP 800-63B-4 に従う。長さと識別子との類似の照合に加えてよくあるパスワードの辞書を用い、構成規則も強制的な定期変更も要求しない。
- 文字種の構成規則は意図的に実装しない。SP 800-63B-4 がこれを SHALL NOT として述べており、推測への耐性ではなく予測しやすさを上げるからである。ただし PCI DSS v4.0 のような適合の確認項目のために、デフォルトで無効なテナントごとの選択制として後から加える可能性は残る。その場合、当該テナントにとって `NIST63B4-NO-COMPOSITION` の採否は明示的な逸脱になる。
- テナントによる長さ、履歴の深さ、有効期限の上書きは、制約を厳しくする方向に限る。パスワードを設定するすべての経路で、解決後の同じポリシーを評価する。
- パスワードの有効期限はテナントごとの選択制であり、直近のパスワードの変更とテナントのポリシーの更新時刻のうち遅いほうから測り、認証の失敗ではなくログイン後の `update_password` の必須の操作として強制する。
- 認証とアイデンティティ管理の設定値 (パスワードの履歴の深さ、漏洩の照合のデフォルト、TOTP / WebAuthn / 復旧コードのパラメーター、リセットのトークンの有効期間、ログインのスロットルの閾値) は、製品の目標に散らさずこのポリシーの節にまとめる。
- パスワードの変更は直近 5 件のパスワードのハッシュの再利用を拒否する。照合はパスワードの変更時のみで、初回の登録では行わない。比較する相手がないからである。
- `BreachedPasswordChecker` は同梱の辞書の上に HIBP の k-匿名性による照合を重ね、障害時は fail-open とする。デフォルトは何もしないアダプターなので、外部への依存を持ち込まない。
- パスワードを忘れた場合の処理は、単回限りでハッシュ化されたリセットのトークンを発行し、応答を一様にし、メールの配送を最大限努力とする。これによりこの流れが利用者名の存在を暴く手段にも、SMTP の障害を探る手段にもならない。
- `EmailSender` の本番のアダプターは、送信サービスごとの HTTP SDK ではなく SMTP だけを話す。SMTP だけで主要な送信サービスにはすべて届くからである。
- ログインのスロットルはアカウント単位と IP 単位の失敗を独立に数え、ハッシュ化した識別子を鍵とし、恒久的な締め出しは意図的に使わない。
- 認証のイベントは個別の行と 5 分の集計に分け、抑制された行為者の氾濫が際限なく増えるのではなく 1 行にたたまれるようにする。
- 確定したアカウントの認証イベントは `user_id` で相関し、ユーザー名による管理者検索は入力されたユーザー名をその場で `user_id` へ解決する。
- 利用者自身が操作するアカウントのポータルと管理用のアカウント API は別の契約である。ポータル自身の要約のエンドポイントは意図的にロールを省くので、管理用の情報を漏らすことはない。
- 機微の高い自己操作 (パスワードの変更、TOTP の削除、メールアドレスの変更、他のセッションの失効) は、CSRF の防御に加えてステップアップ認証による再認証を要求する。
- MFA の強制が始まった後、未登録のユーザーが登録専用の流れに到達できるのは、管理者が発行した単回限りの `MfaEnrollmentBypass` を通じてのみであり、素のパスワードの成功だけでは決して到達できない。

## Scenarios

### REQ-AUTHENTICATION-001: 外部 OIDC 認証は検証済みの subject を常に同じローカル User へ相関する
- ACTOR EndUser
- GIVEN リクエスト先のテナントで OIDC 接続が `Active` である
- GIVEN issuer、認可エンドポイント、トークンエンドポイント、JWKS は登録時に検証済みである
- WHEN EndUser が StartFederatedLogin を開始する
- THEN `state`、`nonce`、PKCE を単回限りのログイン試行として保存し、上流へ遷移する
- WHEN 上流のコールバックが認可コードと ID Token を返す
  - ALT 同じ `state` またはトークンレスポンスを再利用する → 単回限りの試行と再送防止によって拒否する
- THEN CompleteFederatedLogin は code と ID Token の署名、issuer、audience、時刻、nonce を検証する
  - ALT `state`、`nonce`、issuer、audience、署名、時刻のいずれかが一致しない → コールバックを拒否し、LoginSession も関連付けも作成しない → FederatedLoginRejected を発行する
- THEN 初回は、明示した JIT ポリシーとクレームの対応付けに従ってローカルの User と FederatedIdentity を作成する
- THEN 2 回目以降は、同じテナント・プロバイダー・外部 subject の既存の関連付けから同じローカル User を解決する
- THEN AMR に `federated` を持つ LoginSession を発行する

### REQ-AUTHENTICATION-002: 検証済みメールアドレスによる自動リンクは明示ポリシーと一意な一致を要求する
- ACTOR EndUser
- GIVEN その外部 subject に対する既存の関連付けはない
- GIVEN 同じメールアドレスを持つローカル User がテナント内に存在する
- GIVEN 接続の `linking_policy` が `VerifiedEmail` である
- GIVEN 上流の `email_verified` クレームが true で、メールアドレスがテナント内で一意に一致する
- WHEN EndUser が未連携の外部 subject でフェデレーションログインを完了する
  - ALT ポリシーが `None`、メールアドレスが未検証、または一致が一意でない → 自動リンクと LoginSession の発行を拒否する
- THEN 既存の User に対して FederatedIdentity を作成する

### REQ-AUTHENTICATION-003: 外部アイデンティティの明示的なリンクと解除はステップアップ認証を要求する
- ACTOR AuthenticatedSelf
- GIVEN ResourceOwner は対象テナントの有効な User である
- WHEN 直近 5 分以内にステップアップ認証を済ませたセッションで、外部プロバイダーの認証を完了する
  - ALT ステップアップ認証が古い、または行われていない → リンクと解除を AccessDeniedError で拒否する
- THEN その外部 subject が未使用であれば、自身へリンクする
- WHEN 直近 5 分以内にステップアップ認証を済ませたセッションでリンクの解除を要求する
  - ALT パスワード資格情報も他の外部アイデンティティのリンクも残らなくなる → 締め出しを防ぐため解除を拒否する
- THEN 対象の外部アイデンティティのリンクを解除する

### REQ-AUTHENTICATION-004: API トークンの発行者は機密操作のスコープで自身の認証情報だけを操作できる
- ACTOR SelfApiClient
- GIVEN クライアントは対象テナントの有効な User に固定された、有効な API アクセストークンを提示している
- WHEN クライアントがアカウントのセキュリティ設定、サインイン履歴、セッション、MFA 認証要素、復旧コード、またはパスワードの操作を要求する
  - ALT 対応しないスコープで機密操作の変更を要求する → 操作は AccessDeniedError で拒否される
  - ALT トークンのテナントまたは `user_id` が操作対象と一致しない → 操作は AccessDeniedError で拒否される
  - ALT API トークンでステップアップ認証のエンドポイントを要求する → 操作は AccessDeniedError で拒否される
- THEN `account:read` スコープは、自身のアカウント情報、セキュリティ設定、サインイン履歴、セッションの参照だけを許可する
- THEN `account:mfa:write` スコープは、自身の MFA 認証要素と復旧コードの変更だけを許可する
- THEN `account:sessions:write` スコープは、自身のセッションの失効だけを許可する
- THEN `account:password:write` スコープと現在のパスワードの提示は、自身のパスワードの変更だけを許可する

### REQ-AUTHENTICATION-005: ブラウザーの初期化情報は認証状態と CSRF 境界を保持する
- ACTOR AuthenticatedSelf
- GIVEN ユーザー "alice" が認証済みセッション、またはファーストパーティーのポータルのアクセストークンを持つ
- WHEN ブラウザーまたは API クライアントがアカウントコンテキストをリクエストする
  - ALT セッションが未認証または認証途中である → アカウントコンテキストの取得を AccessDeniedError で拒否する
  - ALT Bearer トークンが許可されたポータルスコープまたは `account:read` スコープを 1 つも持たない → アカウントコンテキストの取得を AccessDeniedError で拒否する
- THEN 管理ポータルは `idmagic.admin`、アカウントポータルは `idmagic.account`、自己管理 API クライアントは `account:read` スコープで同じアカウントコンテキストを取得できる
- THEN レスポンスは subject、realm、実効ロール、CSRF トークンを含む
- WHEN 未認証のパスワードリセット画面がパスワードリセットコンテキストをリクエストする
- THEN CSRF トークンを含むコンテキストが返る

### REQ-AUTHENTICATION-006: ユーザーは WebAuthn でステップアップ認証のチャレンジを開始できる
- ACTOR AuthenticatedSelf
- GIVEN ユーザー "alice" が WebAuthn のクレデンシャルを登録済みで、認証済みセッションを持つ
- WHEN ユーザー "alice" が正しい CSRF トークンでステップアップ認証の WebAuthn チャレンジを要求する
  - ALT CSRF トークンが一致しない、または WebAuthn を利用できない → チャレンジは発行されず、要求を拒否する
- THEN レスポンスの `PublicKeyCredentialRequestOptions` は現在のセッションに束縛される

### REQ-AUTHENTICATION-007: ResourceOwner はブラウザーでパスワード認証し、認可を継続する
- ACTOR ResourceOwner
- GIVEN 未認証セッションで "web-app" として認可リクエストを送信済みである
- WHEN ブラウザーのログイン API にユーザー名 "alice" と正しいパスワードを送信する
  - ALT SameSite の Cookie とリクエストのトークンが一致しない → CSRF の値を改ざんしてログイン API を送信する → エラー "InvalidRequestError"
  - ALT 直近 900 秒の時間枠で、アカウント単位の失敗回数が 10 回に達している → 正しいパスワードでログイン API を送信する → エラー "RateLimitedError" → "LoginThrottled" が発行される
  - ALT 失敗回数によらず、同一 IP からのログイン API リクエストが `EndpointRateLimitPolicy` の時間枠内で上限に達している → 正しいパスワードでログイン API を送信する → エラー "RateLimitedError"
- THEN セッション Cookie が発行される
- THEN 認可コードが redirect_uri に返る
- THEN "UserAuthenticated" が発行される

### REQ-AUTHENTICATION-008: パスワードリセットの要求は識別子と IP の組で流量制限される
- ACTOR EndUser
- GIVEN 未認証である
- WHEN "alice" 宛のパスワードリセットを要求する
  - ALT 同じ識別子と IP の組で、`EndpointRateLimitPolicy` の時間枠内の上限に達している → "alice" 宛のパスワードリセットを再度要求する → エラー "RateLimitedError"
- THEN ユーザーの存在にかかわらず 204 を返す
- THEN "PasswordResetRequested" が発行される

### REQ-AUTHENTICATION-009: 無効化されたユーザーは新規ログインも既存セッションも拒否される
- ACTOR TenantAdministrator
- GIVEN ユーザー "alice" が認証済みセッションを持つ
- WHEN 管理者がユーザー "alice" を無効化する
- THEN ユーザー "alice" は無効状態になる
- WHEN ユーザー "alice" が既存セッションで認証必須 API を呼ぶ
- THEN エラー "AccessDeniedError"
- WHEN ユーザー "alice" が正しいパスワードで新規ログインを試みる
- THEN エラー "AccessDeniedError"

### REQ-AUTHENTICATION-010: ユーザーは現在のパスワードを確認して新しいパスワードへ変更できる
- ACTOR AuthenticatedSelf
- GIVEN ユーザー "alice" が認証済みでパスワード変更画面を開いている
- WHEN ユーザー "alice" が正しい現在のパスワードと新しいパスワードを送信する
  - ALT 新しいパスワードが 12 文字未満である → ユーザー "alice" が 12 文字未満のパスワードを送信する → エラー "InvalidRequestError"
  - ALT 新しいパスワードが直近 5 件の履歴に一致する → ユーザー "alice" が直近使用した過去のパスワードを新パスワードとして送信する → エラー "InvalidRequestError"
- THEN パスワードが変更され、`password_changed_at` が更新される
- THEN "PasswordChanged" が発行される

### REQ-AUTHENTICATION-011: ユーザーは TOTP 認証要素を登録して有効化できる
- ACTOR AuthenticatedSelf
- GIVEN ユーザー "alice" が認証済みでセキュリティ画面を開いている
- WHEN ユーザー "alice" が TOTP 登録を開始する
- THEN レスポンスにシークレットとアカウント名が含まれる
- WHEN ユーザー "alice" がそのシークレットに対する正しいコードで登録を確認する
- THEN セキュリティ概要の MFA 状態が登録済みになる
- THEN "MfaFactorEnrolled" が発行される

### REQ-AUTHENTICATION-012: ユーザーはステップアップ再認証のうえで TOTP 認証要素を解除する
- ACTOR AuthenticatedSelf
- GIVEN ユーザー "alice" が登録済みの TOTP 認証要素を持ち認証済みである
- WHEN ユーザー "alice" がステップアップ認証を成立させ現在の TOTP コードで解除する
  - ALT ステップアップ認証なしで解除を試みる → ユーザー "alice" がステップアップ認証なしで TOTP 認証要素の解除を試みる → ステップアップ認証による再認証が要求される
- THEN TOTP 認証要素が解除される
- THEN "MfaFactorRemoved" が発行される

### REQ-AUTHENTICATION-013: ユーザーは自分の有効なセッションを一覧して失効できる
- ACTOR AuthenticatedSelf
- GIVEN ユーザー "alice" が複数の有効なセッションを持ち認証済みである
- WHEN ユーザー "alice" がアクティビティ画面でセッション一覧を取得する
  - ALT プロセスの再起動を挟んでセッション一覧を取得する → サーバープロセスを再起動する → ユーザー "alice" が同じセッション Cookie でアクティビティ画面を開く → セッションは再起動前と同じ内容で解決できる
- THEN 自分の有効なセッションが返る
- WHEN ユーザー "alice" が現在以外のセッションを 1 件失効させる
  - ALT 既に失効済みのセッションへ同じ失効操作を再送する → ユーザー "alice" が直前に失効させた同じセッション ID へ再度失効を要求する → 要求は成功として扱われ、最初の失効時刻を保持する
- THEN 失効したセッションは一覧から消える
- WHEN ユーザー "alice" が現在以外のすべてのセッションを一括失効させる
- THEN 現在のセッションだけが残る

### REQ-AUTHENTICATION-014: ユーザーは自分のサインイン履歴を確認できる
- ACTOR AuthenticatedSelf
- GIVEN ユーザー "alice" が認証済みでアクティビティ画面を開いている
- WHEN ユーザー "alice" が自分のサインイン履歴を取得する
  - ALT 認証手段に WebAuthn が含まれる → UI は `webauthn` という技術名ではなく「パスキー」と表示する
- THEN レスポンスに自分のサインインイベントだけが含まれる
- THEN 第二要素を使ったサインインは、`pwd` と第二要素の `amr` を持つ完了後の `UserAuthenticated` として表示される

### REQ-AUTHENTICATION-015: MFA 登録済みでも、ポリシーが要求しない限り第二要素は求めない
- ACTOR EndUser
- GIVEN ユーザー "alice" は TOTP または WebAuthn のクレデンシャルを登録済みである
- GIVEN 対象 Application の実効サインインポリシーは `Password` である
- WHEN ユーザー "alice" がユーザー名とパスワードを送信する
  - ALT 対象 Application の実効サインインポリシーが `Mfa` である → LoginSession は `authentication_pending=true` へ切り替わる → 利用できる第二要素 (TOTP / パスキー / 復旧コード) の選択画面へ進む
- THEN LoginSession は `authentication_pending=false` で作られる
- THEN 認可フローは第二要素画面に進まず、同意または認可コード発行へ進む

### REQ-AUTHENTICATION-016: ユーザーはメールのリセットリンクでパスワードを再設定する
- ACTOR EndUser
- GIVEN ユーザー "alice" 宛に有効なパスワードリセットトークンが発行されている
- WHEN ユーザー "alice" がそのトークンと新しいパスワードを送信する
  - ALT トークンが期限切れまたは不正である → 無効なパスワードリセットトークンで新しいパスワードを送信する → エラー "InvalidRequestError"
- THEN パスワードが更新される
- WHEN ユーザー "alice" が新しいパスワードをブラウザーのログイン API へ送信する
- THEN ログインに成功する
- WHEN EndUser が未登録のメールアドレスでパスワードリセットを要求する
- THEN レスポンスは登録済みアドレスに対するものと区別できない
- WHEN EndUser が登録済みのメールアドレスでパスワードリセットを要求する
- THEN 登録済みアドレスへリセットリンクが送られる

### REQ-AUTHENTICATION-017: TOTP が必須のユーザーは正しいコードで認証を継続できる
- ACTOR EndUser
- GIVEN TOTP 認証要素が登録された `authentication_pending` の LoginSession が存在する
- WHEN ブラウザーの TOTP API に正しいコードを送信する
  - ALT 誤った TOTP コードを送信する → ブラウザーの TOTP API に誤ったコードを送信する → エラー "InvalidRequestError" → LoginSession は `authentication_pending` のままである
- THEN 認証が成立し認可フローが継続する
- THEN "UserAuthenticated" が発行される

### REQ-AUTHENTICATION-018: MFA 未登録のユーザーは管理者が承認した登録を終えて同じ認可処理を継続できる
- ACTOR EndUser
- GIVEN 対象 Application の実効ポリシーは MFA 必須かつ強制開始済みで、登録バイパスを許可し猶予期限内である
- GIVEN ユーザーは TOTP と WebAuthn のいずれの認証要素も持たない
- GIVEN 管理者が対象ユーザーへ有効な単回限りの登録バイパスを発行済みである
- WHEN ユーザーが正しいパスワードを送信する
  - ALT 登録バイパスがない、取り消し済み、消費済み、または期限切れである → パスワードが正しくてもログインを完了せずアクセスを拒否する → 認証要素の登録 API も利用できない
- THEN バイパスを消費し、同じ LoginSession は `pending_purpose=Enrollment` の未完了状態になる
- THEN `MfaEnrollmentRequired` と `MfaEnrollmentBypassConsumed` が発行され、登録専用画面へ進む
- WHEN ユーザーが TOTP のシークレットに対する正しいコードで登録を確定する
  - ALT 登録期限を過ぎている → 認証要素を保存せずアクセスを拒否する → LoginSession を認証完了へ昇格させない
  - ALT TOTP コードが不正である → 認証要素を保存せず InvalidRequestError を返す → LoginSession は `Enrollment` の保留状態のままである
- THEN 認証要素が保存され、同じ LoginSession の `amr` に `otp` が追加されて保留状態が解除される
- THEN `MfaEnrollmentCompleted` と `UserAuthenticated` が発行され、元の認可トランザクションが継続する

### REQ-AUTHENTICATION-019: MFA の強制開始前は、未登録のユーザーもログインできるが登録を促される
- ACTOR EndUser
- GIVEN テナントデフォルトポリシーは将来時刻から MFA 必須になる
- GIVEN ユーザーは MFA 認証要素を持たない
- WHEN ユーザーが正しいパスワードでログインする
- THEN 強制開始前なので、パスワードだけのセッションが成立する
- THEN UI は強制開始日時と事前登録を促す警告を表示する
- THEN ユーザーは通常のステップアップ認証を経たアカウントのセキュリティ設定画面から認証要素を事前登録できる

### REQ-AUTHENTICATION-020: 登録待ちのセッションは通常のリソースへアクセスできない
- ACTOR EndUser
- GIVEN `pending_purpose=Enrollment` の LoginSession が存在する
- WHEN ユーザーがアカウント、管理、Application のいずれかのリソースを要求する
- THEN 未認証として拒否する
- THEN 登録の開始と確定の API、および元の認可トランザクションだけを許可する

### REQ-AUTHENTICATION-021: 管理者は対象ユーザーのセッションを一覧・個別失効・全失効できる
- ACTOR TenantAdministrator
- GIVEN ユーザー "alice" が複数の有効な LoginSession を持つ
- WHEN 管理者がユーザー "alice" の ListSessions を呼ぶ
  - ALT 他テナントの管理者が呼び出す → エラー "AccessDeniedError"
- THEN 開始時刻の降順で有効なセッション一覧が返る
- WHEN 管理者がそのうち 1 件の `RevokeSession` を呼ぶ
  - ALT 既に失効済みのセッションへ再度 `RevokeSession` を呼ぶ → 204 が返り、`revoked_at` は初回の値を保持する
- THEN 対象セッションは `revoke_reason=admin_revoke` で失効し、"SessionEnded" が発行される
- WHEN 管理者がユーザー "alice" の RevokeUserSessions を呼ぶ
- THEN 残り全セッションが失効する

### REQ-AUTHENTICATION-022: 管理者は認証器を全リセットしたユーザーに次回ログインで再登録を強制できる
- ACTOR TenantAdministrator
- GIVEN ユーザー "alice" は TOTP 認証要素を持ち、復旧コードも生成済みである
- WHEN 管理者がユーザー "alice" の ResetUserAuthenticators を targets=[Totp, RecoveryCode] で呼ぶ
  - ALT 他テナントの管理者、または `admin` ロールを持たない操作者が呼び出す → エラー "AccessDeniedError" → 対象ユーザーの認証器は変更されない
- THEN "AuthenticatorResetRequested" が発行される
- THEN TOTP 認証要素と復旧コードが削除され、他に WebAuthn クレデンシャルもないため `mfa_enrolled` が `false` になる
- THEN `reenrollment_required=true` のレスポンスが返り、単回限りの登録バイパスを自動発行する
- THEN "AuthenticatorResetCompleted" と "MfaEnrollmentBypassIssued" が発行される
- WHEN "alice" が正しいパスワードで次にログインする
- THEN 有効なバイパスにより、同じ LoginSession が `pending_purpose=Enrollment` になる
- WHEN "alice" が新しい TOTP 認証要素の登録を確定する
- THEN 同じ LoginSession が MFA 済みへ昇格し、元の認可トランザクションが継続する

### REQ-AUTHENTICATION-023: 管理者が一部の認証器のみリセットした場合は残存要素でログインを継続できる
- ACTOR TenantAdministrator
- GIVEN ユーザー "bob" は TOTP 認証要素と WebAuthn クレデンシャルを両方持つ
- WHEN 管理者がユーザー "bob" の ResetUserAuthenticators を targets=[Webauthn] で呼ぶ
- THEN WebAuthn クレデンシャルだけが削除され、TOTP 認証要素は残るため `mfa_enrolled` は `true` のままである
- THEN `reenrollment_required=false` のレスポンスが返り、登録バイパスは発行されない
- WHEN "bob" が次回のログインで TOTP コードによる第二要素の検証を完了する
- THEN ログインを完了できる

### REQ-AUTHENTICATION-024: 有効期限を過ぎたパスワードのユーザーは次回ログイン後にパスワード変更を強制される
- ACTOR EndUser
- GIVEN テナントのパスワードポリシーは `max_age_days=90` で、ポリシーの更新から 90 日以上が経過している
- GIVEN ユーザー "alice" の `password_changed_at` は 91 日前である
- WHEN ユーザー "alice" が正しいパスワードでログインする
  - ALT `password_changed_at` が 89 日前である → ログインはそのまま完了し、`update_password` は付与されない
  - ALT `max_age_days` が未設定である → 経過日数によらず `update_password` は付与されない
  - ALT ポリシーの更新から 90 日が経過していない → 猶予期間内なので `update_password` は付与されない
  - ALT ユーザーがパスワード資格情報を持たない (フェデレーションまたはパスワードレス) → `update_password` は付与されない
- THEN ログイン自体は成功する
- THEN ユーザー "alice" に必須操作 `update_password` が付与される
- THEN ユーザー "alice" はパスワード変更画面へ誘導され、変更完了までフローを継続できない
- WHEN ユーザー "alice" がポリシーを満たす新しいパスワードへ変更する
- THEN `update_password` が解除され、"PasswordChanged" が発行される

### REQ-AUTHENTICATION-025: 外部 IdP 接続の管理は対話セッションに限る
- ACTOR ManagementApiClient
- GIVEN クライアントは対象テナントの有効な API アクセストークンを提示している
- GIVEN トークンの発行者は `admin` ロールを持つ
- WHEN クライアントが外部 IdP 接続の参照または変更をリクエストする
  - ALT トークンが `ApiTokenScope` のどのスコープを持っていても → 操作は `insufficient_scope` で拒否され、必要な資格として対話セッションを提示する
- THEN 外部 IdP 接続の管理は、ブラウザーのログインセッションまたは管理ポータルのアクセストークンからのみ行える
- WHEN 同じクライアントがセッションと認証情報の管理 API をリクエストする
- THEN `sessions:read` は利用者のセッションとサインイン履歴の参照を、`sessions:write` はセッションの失効を、`users:write` は MFA 登録の一時免除と認証器のリセットを許可する
