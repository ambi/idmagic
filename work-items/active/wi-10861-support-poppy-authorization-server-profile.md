---
status: pending
authors: [tn]
risk: high
reversibility: irreversible
created_at: 2026-11-15
priority: p3
group: federation-protocols
depends_on: [wi-28138-support-confidential-cimd-for-personal-agents]
change_kind: feature
affected_spec:
  - { path: docs/modules/oauth2/features/client.md, requirement: REQ-OAUTH2-017 }
  - { path: docs/modules/oauth2/features/client.md, requirement: REQ-OAUTH2-028 }
  - { path: docs/modules/oauth2/features/authorization.md, requirement: REQ-OAUTH2-005 }
  - { path: docs/modules/oauth2/features/authorization.md, requirement: REQ-OAUTH2-009 }
  - { path: docs/modules/oauth2/features/authorization.md, requirement: REQ-OAUTH2-054 }
  - { path: docs/modules/oauth2/features/consent.md, requirement: REQ-OAUTH2-008 }
  - { path: docs/modules/oauth2/features/token.md, requirement: REQ-OAUTH2-006 }
  - { path: docs/modules/oauth2/features/token.md, requirement: REQ-OAUTH2-019 }
  - { path: docs/modules/oauth2/features/token.md, requirement: REQ-OAUTH2-021 }
  - { path: docs/modules/oauth2/features/protocol-endpoints.md, requirement: REQ-OAUTH2-030 }
  - { path: docs/modules/oauth2/features/protocol-endpoints.md, requirement: REQ-OAUTH2-033 }
  - { path: docs/modules/oauth2/features/admin-access.md, requirement: REQ-OAUTH2-003 }
  - { path: docs/modules/oauth2/standards.md, requirement: RFC8414-METADATA }
  - { path: docs/modules/oauth2/standards.md, requirement: RFC8707-MCP-RESOURCE-BINDING }
  - { path: docs/modules/oauth2/standards.md, requirement: RFC9068-DEFAULT-AUDIENCE }
  - { path: docs/modules/oauth2/standards.md, requirement: RFC9207-ISS }
  - { path: spec/modules/oauth2/models.tsp, symbol: IdMagic.Contract.DiscoveryDocument }
  - { path: spec/modules/oauth2/models.tsp, symbol: IdMagic.Contract.AuthorizationCodeRecord }
  - { path: spec/modules/oauth2/models.tsp, symbol: IdMagic.Contract.RefreshTokenRecord }
---

# Poppy 1.0 の認可サーバーとして企業の発見と Direct Sign-In を提供する

## 動機

[[wi-28138-support-confidential-cimd-for-personal-agents]] は、Poppy にも使える CIMD の秘密鍵認証と PS384 を追加する汎用基盤である。
その完了だけでは、Poppy が定める企業と issuer の対応、クライアントの制約、継続的な付与、サービスごとのトークン取得は成立しない。
IdMagic を企業サービスの認可サーバーとして利用するため、これらを明示的なプロファイルとして実装する。

根拠は [Poppy 1.0 仕様 §3.2・§4](https://personalagentprotocol.org/docs/1.0/spec) と [企業向け実装ガイド](https://personalagentprotocol.org/docs/1.0/guides) である。
公式の必須条件、推奨、任意機能を区別し、Direct Sign-In を提供する認可サーバーの範囲を適合対象にする。
企業の全チャネルを IdMagic が実装するという意味での「Poppy 全体への対応」とは表示しない。

優先度は p3 とする。
既存 OAuth の不足を補う先行項目と異なり、企業単位の付与という新しい意味を導入するプロトコル拡張である。
優先度とセキュリティ上のリスクは分け、認可とテナント境界に影響するため risk は high とする。

## 対象範囲

- テナント管理者が明示的に有効化する、realm 単位の Poppy 1.0 認可サーバープロファイルと管理 API。
- 一企業一 issuer の構成、企業ドメインと AS メタデータの対応、および企業側へ配置する discovery 文書の接続例。
- 許可した CIMD クライアントへの Poppy 固有の鍵、assertion、redirect、グラントの制約。
- Direct Sign-In、本人の同意、refresh token の発行と更新、失効、必要なら既存 PAR を利用する一連の経路。
- 初回に `resource` を省略した付与から、企業が現在提供する個別サービスのトークンを取得できる仕組み。
- 設定、付与、資源、クライアントの停止・削除後の拒否と、既存 OAuth の互換性の検証。
- 企業と Personal Agent を模した HTTP 相手を用いた、発見から失効までの相互運用試験。

## 対象外

- Poppy の Session JWT と委譲鍵、ブラウザー参加、会話、Operations の承認、Web Bot Auth の検証。
- Device Sign-In、Mediated Sign-In、UCP などの任意機能。この版は Direct Sign-In を提供し、Poppy クライアントのほかの sign-in 方式は拒否する。
- 企業の事業 API、Web サイト、会話 Agent の実装と、そのコンテンツのホスティング。
- `poppy:browser-sign-in` の付与。接続例ではブラウザーのアカウントアクセスを提供しない。
- 公開済み issuer の変更、異なる企業間の付与の移行、企業をまたぐアカウントの統合。
- 管理コンソールの新しい設定画面。初期設定は管理 API と接続例で提供する。
- 特定ベンダーとの本番接続の保証。外部製品の試験は実施できた相手と版だけを別に報告する。

## 設計

### 既存基盤との分担と適合範囲

| 領域 | この項目で加えるもの、または検証して使うもの |
| --- | --- |
| CIMD と署名 | 先行項目の取得器と ES256・PS384 を使う。Poppy 向けの同一ドメイン、issuer 限定の `aud`、許可するクライアントの規則を追加する |
| 認可と同意 | 既存の S256、`prompt=consent`、安全な認可エラーにも付く `iss` を使う。スコープ必須、成功時の refresh token、Poppy クライアントへの第一者扱いの禁止を追加する |
| 資源と付与 | 既存の登録済み資源、スコープ上限、単一 audience のトークン発行を使う。付与が企業全体を対象にする場合を、従来の単一資源に束縛した付与から区別する |
| Discovery | realm ごとの issuer を使い、AS の `protected_resources` に企業ドメインを掲載する。企業ドメインの PRM と `poppy.json` は企業側に配置する |
| DPoP と失効 | 既存の DPoP、refresh token の鍵への束縛、ローテーション、失効を維持する。Poppy Session を認可サーバーへ要求しない |

`iss`、PKCE、同意画面、失効がすべて未実装であるとは扱わない。
対応する規範とテストを調べ、Poppy の経路で不足する分だけ実装する。
着手時に §3.2・§4 の各条件を必須、推奨、任意に分け、既存の証拠、追加する証拠、非提供の機能を対応付ける。

### プロファイルと適用対象

初期版は一つの realm を一企業の issuer に対応させる。
共有サーバーでも realm ごとの issuer を使い、二企業へ同じ issuer を割り当てない。
一企業が複数ドメインを持つ場合は、同じプロファイルの企業ドメインとして列挙する。

OAuth2 が `PoppyCompanyProfile` を所有し、テナント、状態、企業ドメイン、既定の資源、提供資源、許可するクライアント、スコープ上限、改訂番号を保持する。
状態は未設定、有効、停止を区別し、デフォルトは未設定とする。
管理 API は所属テナントの管理者と OAuth2 の管理スコープで保護し、ほかのテナントの設定と資源を参照させない。
変更のイベントには操作者、テナント、改訂番号と非機密の差分を残す。

プロファイルが設定された realm の URL 形式の `client_id` は、登録済みか CIMD から解決したかによらず、この規則の対象にする。
許可リストの URL は完全一致で照合し、未許可は `invalid_client`、許可したクライアントの非提供グラントは `unauthorized_client` として拒否する。
許可リストの除去やプロファイル停止で、汎用 CIMD の解決へ戻って同じ要求を受け入れる経路を作らない。
非 Poppy の公開 CIMD も使う構成は別 realm に分ける。
URL 形式でない既存の登録クライアントと、プロファイル未設定の realm は従来の規則を使う。

プロファイルは停止時も残し、新しい認可と発行を拒否する。
有効な付与が残る間は、設定の物理削除や企業の識別子の差し替えを拒否する。
認証できるクライアントが自分のトークンを失効させる経路は、停止後も利用できるようにする。

### 企業の発見とトークンの宛先

企業ドメインは HTTPS の origin とし、パスや末尾のスラッシュを付けずに AS の `protected_resources` へ掲載する。
企業が自分のドメインで公開する PRM は `resource` をその origin、`authorization_servers` を realm の issuer とし、`poppy.json` の `organization.domain` と対応させる。
設定の表示だけでドメイン所有を証明したとは扱わず、接続確認で企業側の文書と AS 側の文書の相互参照を検証する。
本番では企業の管理する HTTPS の入口に配置し、試験では企業側の入口を TLS の HTTP サーバーで再現する。

既定の資源は企業ドメインのうち一つの origin とし、提供資源は同じテナントの Active な資源登録から明示的に選ぶ。
既定トークンの `aud` はその既定資源にする。
既定トークンを使う企業 API はこの audience を検証し、専用トークンを使う MCP などは自分の `resource` の PRM を公開して、その audience だけを受け入れる。
接続例には両方の資源サーバーを含め、IdMagic の管理 API や別企業で同じトークンが使えないことを確認する。
既定トークンに企業外の audience や全テナントの資源一覧を載せる案は採らない。

### クライアント認証と Direct Sign-In

Poppy クライアントは `private_key_jwt` と公開鍵だけの `jwks_uri` を持つ CIMD で解決する。
`jwks_uri` は `client_id` と同じドメインの HTTPS URL に限り、redirect は同じドメインの HTTPS または RFC 8252 のループバック例外だけを許す。
一般の CIMD で受け入れるインライン鍵だけの文書は、このプロファイルでは使わない。
ドメインの比較を eTLD+1 の一致へ広げず、loopback の例外とポートの扱いを規範と具体例で固定する。

assertion は ES256 と PS384 を受理し、`iss == sub == client_id`、署名、`iat`、`exp`、推測困難な `jti` と再送拒否を検証する。
`aud` はその realm の issuer と完全一致する単一文字列だけを認め、配列、token endpoint URL、別 realm を拒否する。
有効期間の上限を `exp - iat` で検査し、5 分以下の正常な assertion を長さだけを理由に拒否しない。
汎用クライアントの既存の audience の受理規則は変更しない。

認可と PAR のスコープ省略は `openid` に補わず `invalid_scope` にし、未知のスコープと、企業またはクライアントの上限外も拒否する。
要求以上のスコープを付与せず、本人が一つも許可しなければ拒否として終える。
初期版の同意画面は既存の全件許可か拒否を使い、部分選択 UI は必須にしない。
`prompt=consent` を既存同意や第一者扱いで省略しないこと、成功と安全な拒否の両方で `iss` が返ることを検証する。
認可 URL が他の端末で開かれる場合を含め、§4.2 が参照する RFC 10027 の本人確認と同意の要件を仕様・テストへ対応付ける。

Direct Sign-In 成功時は、`offline_access` の有無によらず access token と refresh token を返す。
認可サーバー側で適用したプロファイルを付与へ記録し、文書の `grant_types` のデフォルト値により refresh を利用不能にしない。
既存の REQ-OAUTH2-021 に、この明示的なプロファイルの場合の例外を定める。
リフレッシュと失効でも同じクライアント認証を使い、DPoP に束縛したファミリーには同じ鍵の証明を要求する。

### 付与とサービスごとのトークン

認可要求、PAR、認可コード、refresh token に、通常の付与、Poppy の企業全体の付与、Poppy の資源に限った付与を区別できる情報を保持する。
プロファイルと企業の issuer、元のスコープ、元の資源制限をトークン交換のたびに引き継ぎ、要求のヘッダーや現在の設定だけから推測し直さない。
旧レコードの未設定値は通常の付与として読む。

- 初回の認可で `resource` を省略した場合は、企業全体の付与にする。コード交換時と refresh 時に、現在そのクライアントへ提供する資源のうち一つを指定でき、省略時は既定トークンを発行する。
- 企業全体の付与は、元のスコープを保ったまま、後から追加した提供資源にも使える。初回の資源一覧を固定のスナップショットとして保存しない。
- 初回に `resource` を指定した場合は、その資源に限る。企業ドメインの origin を明示した付与だけが既定トークンも取得でき、ほかの資源へ広げない。
- 資源が同じテナントで Active、現在も提供対象、クライアントも許可済みであることを発行時に確かめる。未登録、無効、提供対象外、複数指定は `invalid_target` で拒否する。
- 発行するスコープは、要求、元の付与、現在の企業とクライアントの上限、対象資源の上限の共通部分に限る。refresh で狭めた access token のスコープと宛先を、新しい refresh token の付与全体へ上書きしない。
- 通常の付与は、従来どおり未束縛付与への `resource` の後付けと、別資源への変更を拒否する。Poppy の情報がない付与を自動で企業全体の付与へ昇格させない。

### 型、作用と境界の判断

中核の計算は `CheckPoppyClient(profile, clientID, metadata, grantType)` と `SelectPoppyTokenTarget(grant, profile, resource, scopes)` に相当する操作に閉じる。
入力には検証済みのメタデータと現在の資源・許可を渡し、計算自体は HTTP 取得、DB 更新、時刻取得、トークン署名を行わない。
設定取得、認証、認可コードの消費、refresh のローテーション、署名、イベントの確定は既存の OAuth2 ユースケースが編成する。
失敗時にコードだけを消費したり、親 refresh を変更してトークンを返さなかったりする状態を残さない。

境界の入力は OAuth2 の責務表、`data.md`、上記の規範と、企業サービスが別の配布物であるという事実である。
D9 により企業サービスとは HTTP 契約と文書で接続し、その内部型を共有しない。
D1 により認可プロファイルと付与は OAuth2 に置き、D2 により Tenant、User、Agent の状態所有者は変更しない。
Poppy 専用モジュール、Tenancy の汎用設定だけで判断する案、OAuth2 の内部機能として扱う案を比較し、最後の案を採る。
クライアント、同意、資源、付与の規則を同時に検査でき、企業サービスの変更が新しいモジュール間依存を生まないためである。
共有 JWT ライブラリには暗号と引数で与える検証規則だけを残し、企業の許可やスコープの意味は持ち込まない。
追加する設定表と付与フィールドは OAuth2 が所有し、新しい公開 Go パッケージや組み立て地点は追加しない。

### 仕様変更と互換性

OAuth2 の新しい機能仕様にプロファイルの管理、適用、企業の発見、認証、付与、停止を規範化し、標準表に Poppy 1.0 の採用範囲を追加する。
REQ-OAUTH2-005、REQ-OAUTH2-006、REQ-OAUTH2-021、REQ-OAUTH2-054 と標準表の既定 audience・資源束縛に、適用条件を限定した例外を追加する。
既存の `iss`、PKCE、同意、失効については、規則を複製せず参照と必要な追加証拠で扱う。
管理 API、設定モデル、AS メタデータ、付与の識別情報は TypeSpec に定める。
必要な新規要件 ID は実装前の仕様変更時に割り当て、`affected_spec` に追記する。

永続化は新しい設定表と既存レコードへの追加フィールドで導入し、旧レコードの付与範囲を広げない。
プロファイル停止は新規の認可と発行を止め、既発行 access token は従来の期限と失効規則に従う。
停止を「全資源で即時にログアウトした」と表示しない。
新形式の付与を旧サーバーが通常の付与として処理するロールバックは許さず、停止と対象ファミリーの失効、対応版への復旧を手順にする。

## 計画

1. 基準リビジョンを固定し、先行項目の完了と、§3.2・§4 の適合表を確認する。
   検証表の中心経路を `primary_use_cases` に具体化する。
2. 上記の仕様差分、TypeSpec、永続化と管理 API を先に定める。
   企業ドメイン、既定資源、専用 MCP 資源、二つの realm を持つ試験構成を固定する。
3. プロファイルの管理・適用、認可と同意、コード交換・refresh の資源選択、停止と失効の順に、各境界の RED から実装する。
4. 正式なルーターと外部相手の試験で発見から失効までを通し、設定・配布・移行・停止の手順を文書化する。

初期版の提供機能と適用範囲は上記で固定する。
実装時に外部仕様との不整合が見つかった場合は、適合表と仕様へ戻って解決し、認可の規則を黙って緩めない。

## タスク

- [ ] T001 [Spec] 適合表、新しい機能仕様と Poppy の標準要件、既存要件の限定的な例外、TypeSpec を更新する。
- [ ] T002 [Acceptance] 二企業・二 realm の発見、Direct Sign-In、複数資源の取得、失効の中心経路を定義し、正式な HTTP 入口で失敗を観測する。
- [ ] T003 [Domain] 新しい規範と REQ-OAUTH2-017、REQ-OAUTH2-028 に対応する、プロファイルとクライアントの適用・拒否の計算とテストを実装する。
- [ ] T004 [Adapters] 管理 API、メモリと PostgreSQL の保存、AS メタデータの生成を接続し、新しい管理規範と REQ-OAUTH2-030、REQ-OAUTH2-033 のテストで観測する。
- [ ] T005 [App] 新しい認可規範と REQ-OAUTH2-005、REQ-OAUTH2-008、REQ-OAUTH2-021 に対応する、スコープ必須、同意、refresh token の発行を実装・検証する。
- [ ] T006 [App] 新しい付与規範と REQ-OAUTH2-006、REQ-OAUTH2-054 に対応する、企業全体と単一資源の付与、宛先選択、縮小、拒否後の無作用を実装・検証する。
- [ ] T007 [Acceptance] プロファイル停止、許可の削除、DPoP、失効、通常の OAuth の互換性と、構成からの全体経路を検証する。
- [ ] T008 [Docs] 企業側の discovery 文書、資源サーバーの検証、管理 API、配置、停止・復旧の接続例を担当する開発・運用文書へ追加する。
- [ ] T009 [Verify] 局所検査と、差分に応じた最終検査を通す。

## 検証

| 中心経路・故障 | 観測する結果と境界 |
| --- | --- |
| 発見と本人の許可 | 企業の `poppy.json`・PRM から realm の AS メタデータを発見し、許可した CIMD、ログイン、同意、コード交換を通る。`offline_access` を要求せず refresh token が得られ、拒否応答にも安全な `iss` が返る。正式な HTTP ルーターで配線を検証する |
| 一度の付与から別サービスへアクセス | `resource` なしで許可した付与から、既定 API と専用 MCP のトークンを取得する。個別の audience とスコープを資源側で検証し、追加資源への refresh と、削除・無効化した資源の拒否を確認する |
| 許可の拡大と企業間の混同 | 別 realm、別企業、未許可クライアント、上限外スコープ、配列・endpoint URL の `aud`、同一ドメイン外の鍵・redirect を拒否する。資源束縛済みの付与と既存の通常の付与を企業全体へ昇格させない |
| 停止と認証失敗 | プロファイル停止・許可削除後に汎用 CIMD へ迂回せず拒否し、自分のトークンの失効は利用できる。不正な assertion、DPoP、資源選択でコード・親 refresh・同意を変更せずトークンも発行しない |
| 永続化と既存利用者 | 新しい付与の区別が再起動とローテーション後も保たれ、旧レコードは通常の付与のまま処理される。メモリと PostgreSQL の共有契約で保存と読み戻しを確認する |

利用者、資源、プロファイルの準備は公開管理 API と既存ログイン経路を使う。
企業と Personal Agent の HTTP 相手だけを記録可能な TLS サーバーで置換し、送信先、方式、パラメーターと、資源側の独立した検証結果を確認する。
DB の状態は読み戻して表明し、HTTP 取得や鍵検証を成功固定のスタブで置き換えた経路を相互運用の証拠にはしない。

- `mise run check-work-items`、`mise run check-links`：起票時と実装時の形式・参照を確かめる。
- `mise run check-spec`、`mise run check-schema`：実装前の契約と、追加フィールド・設定表の収束を確かめる。
- `mise run test-go-package -- ./backend/shared/security/tokens_jose`：assertion の検証規則と汎用クライアントの互換性を確かめる。
- `mise run test-go-package -- ./backend/oauth2/internal/handlers_http`：設定、Discovery、認証の HTTP 契約を確かめる。
- `mise run test-go-package -- ./backend/shared/http/server_http`：公開 API から準備して、企業の発見から複数資源の取得、停止、失効までの正式な配線を確かめる。
- プロファイル、認可、トークン、保存の実装には、追加したパッケージと既存の対象パッケージの `test-go-package` を選び、実装前にここへ追記する。
- 最終検査は `mise run verify-changed -- <着手時の基準リビジョン>` とし、同意画面の実装まで変更した場合は対象の `test-ui-e2e-file` も追加する。

HTTP・TLS・DB を使う検査と `check-schema` は初回から待受けを許す実行環境で実行する。
ローカルの模擬相手による適合と実製品との接続結果を区別する。

## リスク

企業全体の付与を通常の付与へ混ぜると、既存の refresh token から想定外のサービスへアクセスできる。
永続的な区別、停止後の迂回拒否、企業間の拒否を独立に観測し、認可サーバーのトークン発行と資源側の受理の両方を検証する。

新しい資源の追加は、過去の同意済みスコープで到達できるサービスを増やす。
これは Poppy §4.5 の意味として管理者と本人へ説明し、サービス側がスコープの意味を変えた場合に以前の同意を流用しないことを運用条件にする。
Poppy 専用の issuer と付与は外部クライアントが保持するため、公開後の契約は取り消し可能な内部設定とは扱わない。
