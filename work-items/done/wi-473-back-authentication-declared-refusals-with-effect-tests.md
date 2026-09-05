---
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-05
change_kind: maintenance
priority: p1
depends_on: []
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: 宣言済みの拒否に検証を与えるだけで、製品の振る舞い、公開契約、運用手順のいずれも変わらない。メモリアダプターのテナント境界の修正も、PostgreSQL 実装が既に持つ契約へ揃えるものであり、宣言は変わらない。
  references: []
initial_context:
  specification:
    - docs/contexts/authentication/scenarios.feature.md#REQ-AUTHENTICATION-002
    - docs/contexts/authentication/scenarios.feature.md#REQ-AUTHENTICATION-003
    - docs/contexts/authentication/scenarios.feature.md#REQ-AUTHENTICATION-004
    - docs/contexts/authentication/scenarios.feature.md#REQ-AUTHENTICATION-005
    - docs/contexts/authentication/scenarios.feature.md#REQ-AUTHENTICATION-006
    - docs/contexts/authentication/scenarios.feature.md#REQ-AUTHENTICATION-008
    - docs/contexts/authentication/scenarios.feature.md#REQ-AUTHENTICATION-009
    - docs/contexts/authentication/scenarios.feature.md#REQ-AUTHENTICATION-017
    - docs/contexts/authentication/scenarios.feature.md#REQ-AUTHENTICATION-020
    - docs/contexts/authentication/scenarios.feature.md#REQ-AUTHENTICATION-021
    - docs/contexts/authentication/scenarios.feature.md#REQ-AUTHENTICATION-022
    - docs/contexts/authentication/scenarios.feature.md#REQ-AUTHENTICATION-025
  typespec:
    - IdMagic.Authentication.Operations.SubmitBrowserTotp
    - IdMagic.Authentication.Operations.ListIdentityProviderConnections
    - IdMagic.Authentication.Operations.UpdateIdentityProviderConnection
  source:
    - backend/authentication/handlers_http
    - backend/authentication/federation/handlers_http
    - backend/authentication/federation/usecases
    - backend/authentication/mfa/handlers_http
    - backend/authentication/password/handlers_http
    - backend/authentication/session/db_memory
    - backend/authentication/session/db_postgres
    - backend/authentication/session/handlers_http
    - backend/authentication/session/usecases
    - backend/authentication/webauthn/usecases
    - backend/oauth2/handlers_http
    - backend/shared/http/server_http
    - backend/shared/http/support_http
    - tools/check/src
affected_spec:
  - { path: docs/contexts/authentication/scenarios.feature.md, requirement: REQ-AUTHENTICATION-002 }
  - { path: docs/contexts/authentication/scenarios.feature.md, requirement: REQ-AUTHENTICATION-003 }
  - { path: docs/contexts/authentication/scenarios.feature.md, requirement: REQ-AUTHENTICATION-004 }
  - { path: docs/contexts/authentication/scenarios.feature.md, requirement: REQ-AUTHENTICATION-005 }
  - { path: docs/contexts/authentication/scenarios.feature.md, requirement: REQ-AUTHENTICATION-006 }
  - { path: docs/contexts/authentication/scenarios.feature.md, requirement: REQ-AUTHENTICATION-008 }
  - { path: docs/contexts/authentication/scenarios.feature.md, requirement: REQ-AUTHENTICATION-009 }
  - { path: docs/contexts/authentication/scenarios.feature.md, requirement: REQ-AUTHENTICATION-017 }
  - { path: docs/contexts/authentication/scenarios.feature.md, requirement: REQ-AUTHENTICATION-020 }
  - { path: docs/contexts/authentication/scenarios.feature.md, requirement: REQ-AUTHENTICATION-021 }
  - { path: docs/contexts/authentication/scenarios.feature.md, requirement: REQ-AUTHENTICATION-022 }
  - { path: docs/contexts/authentication/scenarios.feature.md, requirement: REQ-AUTHENTICATION-025 }
---

# Authentication が宣言する未検証の拒否に効果まで確かめるテストを与え、台帳から外す

## Motivation

`tools/check/example-coverage-debt.json` は、どのテストからも id を引用されていない規範 id を保持する、縮むだけの台帳である。

対象は自動リンク、ステップアップ認証、CSRF、流量制限、無効化ユーザーの締め出しという、ログインの入口そのものを守る防護である。

ここが素通りする欠陥は、他の Context がどれだけ正しく認可していても手前で無効化する。

台帳が言えるのは「id を引用したテストが無い」ことだけで、「拒否を確かめるテストが無い」こととは違う。

`mise run report-coverage-debt` はこの差を named / nearby / none に分けるが、named は「同じエラー型名を含む拒否テストが同じパッケージへ到達する」ことしか示さない。

この分類が別の Context で `REQ-APPLICATION-005` に対して指したのは、SAML 署名証明書とは無関係な配信一覧のカーソル検証テストだった。

したがって named を「注記が抜けているだけ」と扱うことはできない。

そして注記だけを足す解消は、この項目では認めない。

[[wi-390-security-control-test-standard-and-gate]] が見つけた欠陥は、テストがあり、カバレッジもあり、それでも防護が素通りしていた形だった。

拒否の応答だけを確かめるテストに id を書き足せば、検査は通り、台帳は縮み、防護が効いていることは何ひとつ確かめられないまま「検証済み」の見た目だけが残る。

台帳が縮むこと自体には価値がなく、価値があるのは拒否が実際に副作用を止めていると分かることである。

## Scope

- 下記「この項目が持つ id」の 16 件それぞれについて、宣言された拒否に production と同じ入口から到達するテストを用意する。
- 各テストで 2 つを assert する。呼び出し元が観測する応答 (ステータスとエラー種別) と、その拒否が変えなかった状態。
- 各テストのソースに対応する `EX-AUTHENTICATION-NNN-MM` を書き、具体例とテストを突き合わせて読めるようにする。
- 対応が取れた id を `tools/check/example-coverage-debt.json` から削除する。
- 拒否を実装が持っていないと判明した場合、その防護をこの項目で実装する。
- 防護の追加に新しい設計判断が要る場合は、その id だけを台帳に残し、引き取る work item を切って `depends_on` でつなぐ。
- 完了時点で、この項目が持つ id が台帳から 1 件残らず消えていることを確認する。

## Out of Scope

- 他 Context の拒否。
  Context ごとに別の work item が持つ。
- 12 の Rule が持つ正常系の具体例。
  この項目は拒否だけを引き取る。同じ Rule の `通常経路` が台帳に残っていても、それは正常系の網羅を持つ別の項目の対象である。
- 同じ台帳に載る、この項目が持たない id。
  規則は同じだが、引き取る範囲は下記の 16 件に限る。
- R1、R2、R4 の検査そのものの変更。
  [[wi-390-security-control-test-standard-and-gate]] と [[wi-391-refusal-declaration-floor-and-reinventory]] が持つ。
- 既存テストへ id を注記するだけで台帳から外すこと。
  この項目ではこれを解消として扱わない。
- 具体例の Then ステップへ「変わらなかったもの」を規範として書き足すこと。
  下記「拒否ごとに何を読み直すか」を参照。
- 認証方式そのものの追加、ステップアップ認証の要件変更、流量制限の閾値の見直し。
- `SubmitBrowserTotp` が 401 に載せる本体の型。
  下記「持ち越した観測」を参照。[[wi-454-declared-403-body-vs-guard-error-codes]] と同じ形の不一致であり、
  拒否そのものは効いている。

## Design

### 台帳の単位が Rule から具体例へ移ったこと

この項目は `REQ-AUTHENTICATION-NNN` 12 件を持つものとして起票された。

その後 [[wi-491-track-tests-per-example]] が台帳を Rule 単位から具体例単位へ移し、`scenarios.feature.md` は Markdown with Gherkin になった。

台帳の現在の単位は `EX-AUTHENTICATION-NNN-MM` であり、Rule の id はもう台帳に載らない。

Rule の採番は移行を越えて保たれているため、起票時に挙げた 12 件は今も同じ拒否を指す。

そこで持ち分を、その 12 Rule が宣言する拒否の具体例へ読み替える。

正常系の具体例は持たない。

Rule 009 と 020 は Rule 自体が拒否の宣言であり、その `通常経路` が拒否そのものなので、これらの `-01` は持ち分に入る。

### この項目が持つ id

16 件。

| Rule | 具体例 | 宣言している拒否 | 読み直して確かめる対象 |
|---|---|---|---|
| REQ-AUTHENTICATION-002 | EX-AUTHENTICATION-002-02 | ポリシー `None`、未検証メール、一意でない一致では自動リンクを拒否 | 外部アイデンティティのリンクが作られておらず、セッション Cookie も発行されていない |
| REQ-AUTHENTICATION-003 | EX-AUTHENTICATION-003-02 | ステップアップが古い、または行われていないリンクと解除を拒否 | リンクの本数が変わらず、上流への遷移も返らない |
| REQ-AUTHENTICATION-003 | EX-AUTHENTICATION-003-03 | 締め出しになる解除を拒否 | 最後の 1 本のリンクが残る |
| REQ-AUTHENTICATION-004 | EX-AUTHENTICATION-004-02 | 対応しないスコープでの機密操作の変更を拒否 | 復旧コードの集合が入れ替わっていない |
| REQ-AUTHENTICATION-004 | EX-AUTHENTICATION-004-03 | 不一致の `user_id` とテナントを拒否 | 対象のセッションが有効なまま |
| REQ-AUTHENTICATION-004 | EX-AUTHENTICATION-004-04 | API トークンでのステップアップ要求を拒否 | どのセッションの `step_up_at` も進まず、チャレンジも保存されない |
| REQ-AUTHENTICATION-005 | EX-AUTHENTICATION-005-02 | 未認証・認証途中のアカウントコンテキスト取得を拒否 | 応答にアカウント情報と CSRF トークンが含まれない |
| REQ-AUTHENTICATION-005 | EX-AUTHENTICATION-005-03 | スコープを持たない Bearer を拒否 | 応答にアカウント情報と CSRF トークンが含まれない |
| REQ-AUTHENTICATION-006 | EX-AUTHENTICATION-006-02 | CSRF 不一致と WebAuthn 不可時のチャレンジ発行を拒否 | チャレンジが 1 件も保存されていない |
| REQ-AUTHENTICATION-008 | EX-AUTHENTICATION-008-02 | 識別子と IP の組で上限に達した再要求を `RateLimitedError` で拒否 | リセットトークンが 1 本も発行されず、メールも送信されていない |
| REQ-AUTHENTICATION-009 | EX-AUTHENTICATION-009-01 | 無効なユーザーの新規ログインと既存セッションを拒否 | セッション Cookie が発行されず、保護リソースの取得も成功しない |
| REQ-AUTHENTICATION-017 | EX-AUTHENTICATION-017-02 | 誤った TOTP コードを拒否 | 同じセッションが第二要素を待ったまま残り、正しいコードで継続できる |
| REQ-AUTHENTICATION-020 | EX-AUTHENTICATION-020-01 | 登録待ちセッションからの通常リソースへのアクセスを未認証として拒否 | 対象リソースの内容が応答に含まれない |
| REQ-AUTHENTICATION-021 | EX-AUTHENTICATION-021-02 | 他テナントの管理者によるセッション操作を拒否 | 対象ユーザーのセッションが失効していない |
| REQ-AUTHENTICATION-022 | EX-AUTHENTICATION-022-02 | 他テナントおよび非 admin の認証器リセットを拒否 | 対象ユーザーの TOTP と復旧コードが残っている |
| REQ-AUTHENTICATION-025 | EX-AUTHENTICATION-025-02 | API トークンによる外部 IdP 接続の管理を `insufficient_scope` で拒否 | 接続が作成も更新も削除もされていない |

### 解消の条件

1 件の id を台帳から外してよいのは、次の 3 つがすべて成り立つときだけとする。

第 1 に、テストが production の使う入口から入り、宣言された拒否の判断を実際に通ること。

ブラウザーの経路で宣言された拒否は、Cookie とヘッダーを伴う HTTP の境界から入る。

use case を直接呼ぶテストは、CSRF や Origin の検証を通らないため、この条件を満たさない。

第 2 に、拒否が変えなかったものをテスト自身が読み直して確かめること。

第 3 に、具体例の id をテストのソースが引用していること。

「変わっていないこと」を保存層まで読むか、後続の操作で確かめるかは、拒否ごとに決める。

`EX-AUTHENTICATION-017-02` の「`authentication_pending` のまま」は、後続の要求で第二要素をまだ待っていること、
通常のリソースへ到達できないこと、正しいコードで同じセッションが継続することの 3 つとして読む。

保存層を直接覗くテストは、経路が変わると壊れるうえ、素通りした実装を捕まえる力も強くならないためである。

### 対照を必ず 1 つ置く

どの拒否のテストにも、同じ入口で同じ操作が通る対照を 1 つ置く。

「拒否されたので何も起きていない」と「そもそも何も起こせない構成だった」は、効果の側だけを読むと同じに見える。

対照が無ければ、配線を落としただけのテストがそのまま緑になる。

### 拒否ごとに何を読み直すかを仕様へ書き足さない理由

上の表は各拒否の 2 つ目の assert を決めるが、これを具体例の Then ステップへ書き足すことはしない。

「拒否は副作用を止めたことまで assert する」は
[docs/development/specification-first-workflow.md](../../docs/development/specification-first-workflow.md) の検証ラダーが全 Context に課している検証の規範であって、Authentication が外部に約束する振る舞いの追加ではない。

同じ内容を 16 の具体例へ書き写せば、規範の記述が 1 か所から 17 か所へ増え、検証方法の変更が仕様の変更に見えるようになる。

### 却下した進め方

**named を注記だけで閉じる案。**

分類の根拠は同じパッケージへ到達する拒否テストにエラー型名が現れることだけであり、上に挙げた誤検出の実例がその弱さを示している。

**ステップアップと CSRF をモックで置き換えて単体で確かめる案。**

置き換えた時点で、配線が外れた実装を検出できなくなる。

これらの防護はミドルウェアとハンドラーの結線に宿るため、境界を通らないテストでは意味を持たない。

**16 件を 1 つのテーブル駆動テストにまとめる案。**

入口がアカウント API、管理 API、ブラウザーのログイン API、フェデレーションの callback にまたがり、
資格もセッション Cookie と API アクセストークンの 2 種類あるため、共通のテーブルに載せるには入口を抽象化するほかない。
抽象化した時点で「production と同じ入口から入る」条件を失う。

## Plan

1. 16 件の入口と「変わっていないこと」を確定する。 (完了、上表)
2. アカウントコンテキストとスコープの 5 件 (004-02、004-03、004-04、005-02、005-03) を進める。
3. CSRF と流量制限の 2 件 (006-02、008-02) を進める。
4. セッションと締め出しの 3 件 (009-01、017-02、020-01) を進める。
5. 管理操作のテナント境界の 3 件 (021-02、022-02、025-02) を進める。
6. 自動リンクとステップアップの 3 件 (002-02、003-02、003-03) を進める。
7. 各テストについて防護を外すと落ちることを確かめる。落ちないものは入口が誤っているので入口からやり直す。
8. 対応の取れた id を台帳から削除し、この項目が持つ id が 1 件も残っていないことを確認する。

## Tasks

- [x] T001 [Inventory] 16 件の入口と「変わっていないこと」を確定し、既存テストの流用可否を判断する。
- [x] T002 [Acceptance] アカウントコンテキストとスコープの拒否 5 件を効果まで確かめる。
- [x] T003 [Acceptance] CSRF と流量制限の拒否 2 件を効果まで確かめる。
- [x] T004 [Acceptance] セッションと締め出しの拒否 3 件を効果まで確かめる。
- [x] T005 [Acceptance] 管理操作のテナント境界の拒否 3 件を効果まで確かめる。
- [x] T006 [Acceptance] 自動リンクとステップアップの拒否 3 件を効果まで確かめる。
- [x] T007 [App] 拒否の実装が欠けていた経路を修正する。メモリ実装のセッションストアにテナント境界を入れた。
- [x] T008 [Ledger] 対応の取れた id を `example-coverage-debt.json` から削除する。
- [x] T009 [Verify] 防護を意図的に外すと各テストが落ちることを確かめ、検査を通す。

## Verification

- `mise run test-go-race`
- `mise run check-spec`
- `mise run check-security-controls`
- `mise run report-coverage-debt`
- `mise run check-work-items`
- `mise run check-ids`
- `mise run verify`

## Intended RED checks

この項目は宣言済みの拒否へ検証を足すもので、新しい製品要求を作らない。

そのため通常の Acceptance RED — 「まだ無い振る舞いを求めるテストが落ちる」 — は成立しない。

代わりに、各テストについて防護を外した状態でテストが落ちることを観測する。

これは実装済みの防護に対して取れる唯一の RED であり、同時に medium risk が求める変更耐性の証拠でもある。

Unit RED は、防護が欠けていた経路が見つかった場合にだけ発生する。

## Risk Notes

拒否のテストは、防護に到達する前の入力検証で落ちていても同じ応答を返すため、書いた本人にも成立して見える。

これを避けるため、各テストについて防護を外した状態でテストが落ちることを確かめ、落ちなかったものは入口が誤っていると判断してやり直す。

流量制限の拒否は共有カウンタの状態に依存するため、テストが互いの計数を汚さないよう、識別子と IP の組をテストごとに分ける。

台帳は id 順に 1 エントリー 1 id なので、Context ごとの work item が並行しても衝突はエントリー単位に収まる。

各項目は自分が持つ id のエントリーだけを削除し、コメント配列と他の id には触れない。

## Completion

- **Completed At**: 2026-09-05
- **Summary**:
  `mise run spec-diff` は `no normative specification change against main` を返す。
  規範の宣言は 1 件も変わっていない。変わったのは、宣言済みの拒否 16 件が
  「応答だけ確かめられている」状態から「副作用を止めたことまで確かめられている」状態へ
  移ったことと、その 16 件が網羅台帳から消えたことである。
  `check-spec` の網羅は「テストが名指しする id」59 件から 75 件へ増え、
  `example-coverage-debt.json` は 671 件から 655 件へ、Authentication 分は 89 件から 73 件へ減った。
  実装の変更は 1 か所だけで、`session/db_memory.SessionStore` が
  `Find` / `FindOwned` / `Revoke` / `Touch` / `ListBySub` / `DeleteAllForSub` のいずれでも
  テナントを見ていなかったのを、PostgreSQL 実装と同じくリクエストのテナントで絞るようにした。

- **Acceptance RED Evidence**:
  - **Test**: `TestAdminSessionRevokeFromOwnRealmDoesNotReachAnotherTenant`
    (`backend/authentication/handlers_http/admin_refusal_effects_test.go`)。
    ほかの 15 例も同じ形で観測した。対象の要求は `affected_spec` の 12 件すべてに及ぶ。
  - **Requirement**: REQ-AUTHENTICATION-021
  - **Observed Failure**: `acme の管理者が default の利用者のセッションを失効させた: status=204 body=`。
    acme のレルムから他テナントの利用者を名指した全失効が 204 で通り、
    対象のセッションに tombstone が付いていた。
    障害注入のたびに、対応するテストだけが落ちた。落ちなかったものは無い。
  - **Detection Reason**: どのテストも応答と「拒否が変えなかった状態」の 2 つを assert する。
    このテストは応答を読まず、対象のセッションが失効していないことだけを条件にしているので、
    拒否の応答を返しつつ失効させる実装も、応答なしで失効させる実装も、どちらも捕まえる。

- **Unit RED Evidence**:
  - **Test**: `TestSessionStoreIsolatesTenants`
    (`backend/authentication/session/db_memory/sessions_test.go`)
  - **Requirement**: REQ-AUTHENTICATION-021
  - **Observed Failure**: `別テナントの Find が行を返した` を先頭に、
    `FindOwned`、`ListBySub`、`Revoke`、`DeleteAllForSub` のすべてが別テナントの行へ届いた。
    メモリ実装は id だけを鍵にしており、`tenancy.TenantID(ctx)` を一度も見ていなかった。
    PostgreSQL 実装は同じ 6 つの操作すべてを `tenant_id` 述語で絞っており、
    この store の doc comment も「PostgreSQL 実装と同じ contract」を名乗っていた。
  - **Detection Reason**: 参照 3 つと更新 2 つを別テナントの文脈から呼び、
    行が返らないことと tombstone が書かれないことの両方を見る。
    id だけを鍵にする実装はどれも通らない。

- **Change-Resistance Results**:
  防護を 1 つずつ外し、対応するテストだけが落ちることを確かめた。
  結果は 2 つに分かれ、2 つ目はこの項目の想定より重要だった。

  1 つ目は、外した防護に対して素直に落ちたもの。

  - ポータルとアカウントのスコープ境界 (`resolveAuthnContext`) を素通しにすると
    `005-03`、`004-02`、`004-04`、`025-02` が落ちた。
  - `StepUpSatisfied` を常に真にすると `003-02` が落ちた。
  - 締め出し防止の判定 (`user.PasswordHash == "" && len(links) <= 1`) を外すと `003-03` が落ちた。
  - 自動リンクのポリシーと `email_verified` の判定を反転させると `002-02` が落ちた。
    一意判定 (`uniqueVerifiedEmailUser`) だけを外しても同じテストが落ちた。
  - チャレンジ発行の CSRF 検証と WebAuthn 可用性の判定を外すと `006-02` が落ちた。
  - パスワードリセットの流量制限を配線から外すと `008-02` が落ちた。
  - 認証途中の判定 (`authn.AuthenticationPending`) を外すと `005-02` と `020-01` が落ちた。
  - 自分のセッションの所有者判定を `FindOwned` から `Find` へ替えると `004-03` が落ちた。
  - 誤った TOTP を受理させると `017-02` が落ちた。
  - 無効ユーザーの締め出し (`!user.IsActive()`) を外すと `009-01` が落ちた。

  2 つ目は、防護が四重になっていて 1 本外しただけでは落ちなかったもの。
  これは想定していなかったので、外す範囲を広げて確かめ直した。

  他テナントの管理者による操作 (`021-02`、`022-02`) は、次の 4 本で守られていた。

  1. `SessionManager.Resolve` の `sess.TenantID != tenancy.TenantID(ctx)`
  2. `Authenticator.ResolveAuthentication` の `user.TenantID != RequestTenantID(c)`
  3. `Authenticator.RequireAdmin` の同じ判定
  4. 本項目で足した `session/db_memory.SessionStore` のテナント境界

  4 本すべてを外すと `TestAdminSessionOperationsAcrossTenantsRevokeNothing`、
  `TestAdminSessionRevokeFromOwnRealmDoesNotReachAnotherTenant`、
  `TestAdminAuthenticatorResetRefusalLeavesAuthenticatorsUnchanged` の 3 つが落ちた。

  四重の防護は弱点ではないが、「1 本外して落ちること」を変更耐性の基準にすると、
  この 3 例は「テストが効いていない」と誤読されうる。実際には防護一式を外せば落ちる。

- **持ち越した観測**:
  `SubmitBrowserTotp` は契約上 401 の本体を `AuthenticationRequiredError` と宣言しているが、
  誤ったコードに対して実装が書くのは `invalid_totp` である。
  `EX-AUTHENTICATION-017-02` が宣言する `InvalidRequestError` (400) とも一致しない。
  拒否そのものは効いており、副作用も止まっていることを確かめたうえで `017-02` は解消した。
  本体型と宣言の不一致は [[wi-454-declared-403-body-vs-guard-error-codes]] が 403 について持つのと
  同じ形であり、401 へ広げる判断はその項目に属する。

- **Verification Results**:
  - `mise run test-go-race` - passed
  - `mise run check-spec` - passed (711 例中 75 id をテストが名指し。実施前は 59)
  - `mise run check-security-controls` - passed
  - `mise run report-coverage-debt` - Authentication は 89 件から 73 件へ
  - `mise run check-work-items` - passed
  - `mise run check-ids` - passed
  - `mise run verify` - passed
