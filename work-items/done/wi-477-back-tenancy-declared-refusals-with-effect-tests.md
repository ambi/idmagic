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
  reason: 宣言済みの拒否に検証を与えるだけで、製品の振る舞い、公開契約、運用手順のいずれも変わらない。
  references: []
initial_context:
  specification:
    - docs/contexts/tenancy/scenarios.feature.md#REQ-TENANCY-001
    - docs/contexts/tenancy/scenarios.feature.md#REQ-TENANCY-005
    - docs/contexts/tenancy/scenarios.feature.md#REQ-TENANCY-013
    - docs/contexts/tenancy/scenarios.feature.md#REQ-TENANCY-014
    - docs/contexts/tenancy/scenarios.feature.md#REQ-TENANCY-017
    - docs/contexts/tenancy/scenarios.feature.md#REQ-TENANCY-018
  typespec:
    - IdMagic.Tenancy.Operations.GetAdminIntegrationEndpoints
    - IdMagic.Tenancy.Operations.ListTenants
    - IdMagic.Tenancy.Operations.UpdateTenantBranding
    - IdMagic.Tenancy.Operations.UpdateNotificationTemplate
    - IdMagic.Tenancy.Operations.SendTestNotification
  source:
    - backend/tenancy/handlers_http
    - backend/tenancy/usecases
    - backend/tenancy/db_memory
    - backend/idmanagement/group/usecases
    - backend/idmanagement/usecases/helpers.go
    - backend/shared/http/support_http/auth.go
    - tools/check/example-coverage-debt.json
  tests:
    - backend/tenancy/handlers_http
affected_spec:
  - { path: docs/contexts/tenancy/scenarios.feature.md, requirement: REQ-TENANCY-001 }
  - { path: docs/contexts/tenancy/scenarios.feature.md, requirement: REQ-TENANCY-005 }
  - { path: docs/contexts/tenancy/scenarios.feature.md, requirement: REQ-TENANCY-013 }
  - { path: docs/contexts/tenancy/scenarios.feature.md, requirement: REQ-TENANCY-014 }
  - { path: docs/contexts/tenancy/scenarios.feature.md, requirement: REQ-TENANCY-017 }
  - { path: docs/contexts/tenancy/scenarios.feature.md, requirement: REQ-TENANCY-018 }
---

# Tenancy が宣言する未検証の拒否 6 件に効果まで確かめるテストを与え、台帳から外す

## Motivation

`tools/check/example-coverage-debt.json` は、どのテストからも id を引用されていない規範 id を保持する、縮むだけの台帳である。

このうち 102 件は [[wi-390-security-control-test-standard-and-gate]] が拒否専用の台帳へ据え置いた分で、[[wi-490-fold-refusal-coverage-into-one-normative-coverage-rule]] が網羅台帳へ統合した。

その 102 件のうち 6 件が Tenancy の拒否である。

対象はテナント境界、システムコンソールへの越境、Hard Quota、テンプレート上書きの検証、テスト送信の宛先である。

`REQ-TENANCY-014` は通常のテナント管理者がシステムコンソールのテナント一覧へ到達しないことを宣言しており、素通りすれば他テナントの存在そのものが漏れる。

`REQ-TENANCY-018` はテスト送信が操作者本人にしか届かないことを宣言しており、素通りすれば任意の宛先への送信手段になる。

台帳が言えるのは「id を引用したテストが無い」ことだけで、「拒否を確かめるテストが無い」こととは違う。

`mise run report-coverage-debt` はこの差を named / nearby / none に分けるが、named は「同じエラー型名を含む拒否テストが同じパッケージへ到達する」ことしか示さない。

この分類が別の Context で `REQ-APPLICATION-005` に対して指したのは、SAML 署名証明書とは無関係な配信一覧のカーソル検証テストだった。

そして注記だけを足す解消は、この項目では認めない。

拒否の応答だけを確かめるテストに id を書き足せば、検査は通り、台帳は縮み、防護が効いていることは何ひとつ確かめられないまま「検証済み」の見た目だけが残る。

## Scope

- 台帳の Tenancy の 6 件それぞれについて、宣言された拒否に production と同じ入口から到達するテストを用意する。
- 各テストで 2 つを assert する。呼び出し元が観測する応答 (ステータスとエラー種別) と、その拒否が変えなかった状態。
- 各テストのソースに対応する `REQ-TENANCY-NNN` を書く。
- 対応が取れた id を `tools/check/example-coverage-debt.json` から削除する。
- 拒否を実装が持っていないと判明した場合、その防護をこの項目で実装する。
- 防護の追加に新しい設計判断が要る場合は、その id だけを台帳に残し、引き取る work item を切って `depends_on` でつなぐ。
- 完了時点で、この項目が持つ id が台帳から 1 件残らず消えていることを確認する。

## Out of Scope

- 他 Context の拒否。
  Context ごとに別の work item が持つ。
- 102 件を 1 つの単位として消化すること。
  [[wi-399-burn-down-untested-refusal-debt]] がその形で立てられているが、進める単位を未決のまま残し、注記だけの解消も認めている。本項目はその置き換えである。
- 同じ台帳に載る、この項目が持たない id。
  規則は同じだが、引き取る範囲はここに挙げた id に限る。
- R1、R2、R4 の検査そのものの変更。
  [[wi-390-security-control-test-standard-and-gate]] と [[wi-391-refusal-declaration-floor-and-reinventory]] が持つ。
- 既存テストへ `REQ` id を注記するだけで台帳から外すこと。
  この項目ではこれを解消として扱わない。
- システムコンソールの認証強度と再認証時刻。
  [[wi-468-system-console-privileged-session-assurance]] が扱う。
- システムコンソールのテナント横断読出しの有界化。
  [[wi-469-bound-system-console-cross-tenant-reads]] が扱う。
- Quota の値と算入対象そのものの変更。

## Design

### 台帳の単位が Rule から具体例へ移ったこと

この項目は `REQ-TENANCY-NNN` 6 件を持つものとして起票された。

その後 [[wi-491-track-tests-per-example]] が台帳を Rule 単位から具体例単位へ移し、`scenarios.feature.md` は Markdown with Gherkin になった。

台帳の現在の単位は `EX-TENANCY-NNN-MM` であり、Rule の id はもう台帳に載らない。

Rule の採番は移行を越えて保たれているため、起票時に挙げた 6 件は今も同じ拒否を指す。

そこで持ち分を、その 6 Rule が宣言する拒否の具体例へ読み替える。

Rule 005、013、014、017 は Rule 自体が拒否の宣言であり、その `通常経路` が拒否そのものなので、それらの `-01` も持ち分に入る。

### この項目が持つ id

11 件。

| Rule | 具体例 | 宣言している拒否 | 読み直して確かめる対象 |
|---|---|---|---|
| REQ-TENANCY-001 | EX-TENANCY-001-02 | 別テナントの realm を指定した参照を認めず、秘密を返さない | 応答が解決済みテナントの情報だけであり、クライアントシークレット、API トークン、秘密鍵を含まない |
| REQ-TENANCY-005 | EX-TENANCY-005-01 | `javascript:` スキームの footer リンクと SVG ロゴを `InvalidRequestError` で拒否 | branding が保存されておらず、以後の画面がシステムデフォルトを使う |
| REQ-TENANCY-005 | EX-TENANCY-005-02 | label だけの footer リンクを `InvalidRequestError` で拒否 | 片方だけの footer リンクが保存されていない |
| REQ-TENANCY-013 | EX-TENANCY-013-01 | Hard Quota 超過のリソース作成を `QuotaExceededError` で拒否 | リソースが作成されておらず、使用量が増えていない |
| REQ-TENANCY-014 | EX-TENANCY-014-01 | 通常のテナント管理者のシステムコンソール一覧参照を拒否 | 応答に他テナントが 1 件も含まれない |
| REQ-TENANCY-017 | EX-TENANCY-017-01 | 許可集合外の差し込み変数を拒否 | 上書きが保存されておらず、以後も組込みデフォルトが届く |
| REQ-TENANCY-017 | EX-TENANCY-017-02 | 片方だけの本文を拒否 | 片方だけの上書きが作られていない |
| REQ-TENANCY-017 | EX-TENANCY-017-03 | カタログに無い locale を拒否 | その locale の上書きが作られていない |
| REQ-TENANCY-017 | EX-TENANCY-017-04 | 差出人メールアドレスの上書きを拒否 | 送信されるメールの差出人アドレスが変わらず、表示名だけが変わる |
| REQ-TENANCY-018 | EX-TENANCY-018-03 | テスト送信の宛先指定を受け付けない | 指定した宛先へは 1 通も届かず、操作者本人にだけ届く |
| REQ-TENANCY-018 | EX-TENANCY-018-04 | 検証済みメールアドレスを持たない操作者のテスト送信を拒否 | メールが送信されていない |

起票時の表は `REQ-TENANCY-018` を「検証済みメールアドレスを持たない操作者」の 1 件として書いていたが、Motivation は「テスト送信が操作者本人にしか届かない」ことを危険の中心に置いている。

宛先を指定できてしまう形が任意の宛先への送信手段になるので、`EX-TENANCY-018-03` も持ち分に入れる。

`EX-TENANCY-017-04` は「アドレスは上書きできず、表示名だけ上書きできる」という形の拒否なので、効果は「アドレスが変わらないこと」と「表示名は変わること」の両方で読む。

### 解消の条件

1 件の id を台帳から外してよいのは、次の 3 つがすべて成り立つときだけとする。

第 1 に、テストが production の使う入口から入り、宣言された拒否の判断を実際に通ること。

制御面の拒否は、テナント管理者のセッションを付けたシステムコンソールの経路から入る。

第 2 に、拒否が変えなかったものをテスト自身が読み直して確かめること。

第 3 に、具体例の id をテストのソースが引用していること。

### 対照を必ず 1 つ置く

どの拒否のテストにも、同じ入口で同じ操作が通る対照を 1 つ置く。

「拒否されたので何も起きていない」と「そもそも何も起こせない構成だった」は、効果の側だけを読むと同じに見える。

対照が無ければ、配線を落としただけのテストがそのまま緑になる。

### Quota の拒否をどの入口から入るか

`REQ-TENANCY-013` の拒否は Tenancy の Quota が Group の作成経路へ課すものである。

Tenancy の HTTP ハンドラーは Group を作らないので、入口は管理 API の `POST /api/admin/v1/groups` になる。

テストは Tenancy のテストパッケージに置き、`Tenancy.QuotaRepo` を張った合成サーバーから Group の作成経路へ入る。

id は `REQ-TENANCY-013` のものなので、置き場所は Context ではなく「その拒否がどこで効くか」で決める。

`REQ-TENANCY-013` は、拒否が「作成しない」だけでなく「使用量を増やさない」ことまで含む。

使用量だけ増える実装は、以後の正当な作成まで拒否する形で被害が持続するため、両方を読む。

`REQ-TENANCY-018` の「送信されていない」は、送信の抽象の呼び出し回数ではなく、テスト用のメール送信先に何も届いていないことで確かめる。

`REQ-TENANCY-001` は否定形の宣言であり、応答に秘密が含まれないことをフィールド単位で確かめる。

### 却下した進め方

**named の 4 件を注記だけで閉じる案。**

分類の根拠は同じパッケージへ到達する拒否テストにエラー型名が現れることだけである。

**`REQ-TENANCY-014` を認可関数の単体テストで確かめる案。**

システムコンソールの経路は制御面の別ルート上にあり、拒否はその配線に宿る。

関数が正しくても呼ばれていなければ素通りする。

**Quota の拒否を使用量カウンタの単体テストで確かめる案。**

拒否の効果は作成の不在であり、カウンタの値はその一部にすぎない。

## Plan

1. 11 件の具体例を読み、拒否ごとに入口と「変わっていないこと」を確定する。 (完了、上表)
2. 制御面とテナント設定の既存テストを調べ、拡張で足りるか新規が要るかを決める。
3. 越境の 2 件 (001-02、014-01) から着手する。
4. 入力検証の 6 件 (005-01、005-02、017-01 から 017-04) を進める。
5. Quota とテスト送信の 3 件 (013-01、018-03、018-04) を進める。
6. 各テストについて防護を外すと落ちることを確かめる。落ちないものは入口が誤っているので入口からやり直す。
7. 防護そのものが無い id が見つかった場合は実装するか、引き取る work item を切る。
8. 対応の取れた id を台帳から削除し、この項目が持つ id が 1 件も残っていないことを確認する。

## Tasks

- [x] T001 [Inventory] 11 件の入口と「変わっていないこと」を確定し、既存テストの流用可否を判断する。
- [x] T002 [Acceptance] 越境の拒否 (001-02、014-01) を効果まで確かめるテストを書く。
- [x] T003 [Acceptance] 入力検証の拒否 (005-01、005-02、017-01 から 017-04) を効果まで確かめるテストを書く。
- [x] T004 [Acceptance] Quota とテスト送信の拒否 (013-01、018-03、018-04) を効果まで確かめるテストを書く。
- [x] T005 [App] 拒否の実装が欠けていた経路を修正する、または引き取る work item を切って `depends_on` でつなぐ。
      欠けていた経路は 1 つも見つからなかった。実装の変更は無い。
- [x] T006 [Ledger] 対応の取れた id を `example-coverage-debt.json` から削除する。
- [x] T007 [Verify] 防護を意図的に外すと各テストが落ちることを確かめ、検査を通す。

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

制御面の拒否は [[wi-468-system-console-privileged-session-assurance]] と [[wi-469-bound-system-console-cross-tenant-reads]] が触れる範囲と経路を共有する。

同時に進める場合はテストの置き場所が重なるため、先に入った側へ合わせる。

台帳は id 順に 1 エントリー 1 id なので、Context ごとの work item が並行しても衝突はエントリー単位に収まる。

各項目は自分が持つ id のエントリーだけを削除し、コメント配列と他の id には触れない。

## Completion

- **Completed At**: 2026-09-06
- **Summary**:
  `mise run spec-diff` は `no normative specification change against main` を返す。
  規範の宣言は 1 件も変わっていない。変わったのは、宣言済みの拒否 11 件が
  「応答だけ確かめられている」状態から「副作用を止めたことまで確かめられている」状態へ
  移ったことと、その 11 件が網羅台帳から消えたことである。
  Tenancy 分の台帳は 51 件から 39 件へ減った。製品コードの変更は無い。
  防護が欠けている経路は 1 つも見つからなかった。

- **Acceptance RED Evidence**:
  - **Test**: `TestListTenantsRefusesTenantAdminAndReturnsNoOtherTenant`
    (`backend/tenancy/handlers_http/refusal_effects_test.go`)。
    ほかの 10 例も同じ形で観測した。対象の要求は `affected_spec` の 6 件すべてに及ぶ。
  - **Requirement**: REQ-TENANCY-014
  - **Observed Failure**: `Authenticator.RequireControlPlaneUser` から
    `IsControlPlaneActor(actor, RequestTenantID(c))` の判定を外すと、`admin` ロールだけを持つ
    テナント管理者に 200 と全テナントの一覧が返った。本文には `default`、`acme`、`beta` の
    realm と表示名、使用量がそのまま並んだ。テナント名は多くの場合顧客名なので、
    この一覧が読めることは顧客名簿が読めることに等しい。
  - **Detection Reason**: テストは 403 を読むだけでなく、応答本文に `acme` と `beta` の
    文字列が現れないことと、`tenants` 配列が空であることを条件にしている。
    拒否の応答を返しつつ一覧の断片を載せる実装も捕まえる。

- **Unit RED Evidence**:
  - **Test**: `N/A: 防護が欠けている経路が 1 つも見つからなかったため、単体の RED は発生しない。`
    代わりに実施したのは、実装済みの防護を 1 つずつ外して対応するテストが落ちることの確認である
    (下記 Change-Resistance Results)。
  - **Requirement**: N/A: この項目は宣言済みの拒否へ検証を足すもので、新しい製品要求を作らない。
  - **Observed Failure**: `N/A`
  - **Detection Reason**: `N/A`

- **Change-Resistance Results**:
  防護を 1 つずつ外し、対応するテストだけが落ちることを確かめた。

  - `RequireControlPlaneUser` の `IsControlPlaneActor` を外すと `014-01` が落ちた
    (上記 Acceptance RED Evidence)。
  - `TenantBranding.Validate` の `validTenantFooterLink` を外すと `005-01` と `005-02` が
    落ちた。`javascript:alert(1)` の footer リンクも、label だけのリンクも 200 で保存された。
  - `DetectBrandingAssetContentType` の `ErrImageFormat` の写像を素通しに替えると `005-01` が
    落ち、`<svg onload=alert(1)>` がロゴとして保存され、`logo_url` から配信可能になった。
  - `UpdateNotificationTemplate` の `template.ValidateDefinition` を外すと `017-01` と
    `017-02` が落ちた。`{{password}}` を含む本文も、HTML 側が空の上書きも保存された。
  - `resolveTemplate` の `ErrUnknownNotificationTemplate` を既定値へのフォールバックに
    替えると `017-03` が落ち、カタログに無い `fr` の上書きが作られた。
  - `support.DecodeJSON` の `DisallowUnknownFields()` を外すと `017-04` が落ち、
    `from_email` を含む要求が 200 で受理された。
  - `handleSendTestNotification` に本文から宛先を読む分岐を足すと `018-03` が落ち、
    応答の `to` が `victim@example.test` になった。
  - `handleSendTestNotification` の `actor.EmailVerified` を外すと `018-04` が落ち、
    未検証のアドレスへテスト送信が届いた。

  Quota の 1 件は 2 通りの injection で確かめた。
  `CreateGroup` の `CheckQuotaAndAudit` を素通しにすると上限を超えて Group が作られ、
  記憶実装の `CheckAndIncrement` を「上限超過でも使用量だけ足す」形に替えると、
  応答は 422 のまま `usage.groups` が 2 へ動いた。
  後者は応答の側だけを読むテストには見えない。

- **持ち越した観測**:
  `001-02` の越境は 2 層で守られている。
  `Authenticator.ResolveAuthentication` の `user.TenantID != RequestTenantID(c)` だけを外すと
  応答は 401 から 403 へ変わり、拒否そのものは `requireTenantAdmin` の
  `actor.TenantID != support.RequestTenantID(c)` が引き継いだ。
  2 つとも外して初めて 200 と beta の連携エンドポイント一式が返った。
  テストは 401 を固定している。acme のセッションが beta の realm ではセッションとして
  成立しないので、資源サーバとしては 401 が自然だからである。
  宣言は `AccessDeniedError` だが、この不一致は
  [[wi-454-declared-403-body-vs-guard-error-codes]] が持つのと同じ形であり、
  この項目では実装が返す 401 を固定した。

- **やり直した観測**:
  `013-01` の「使用量が増えていない」は、最初に書いた形では成立しないテストだった。
  記憶実装の `GetUsage` は保存中の構造体をそのまま返すので、拒否の前後で返り値を持ち越して
  比べると同じポインターを見ることになり、使用量が動いても必ず一致した。
  障害注入で初めて分かったので、`groups` の値だけを取り出して比べる形へ書き直し、
  同じ注入で落ちることを確かめ直した。

- **Verification Results**:
  - `mise run test-go-race` - passed
  - `mise run check-spec` - passed (711 例中 113 id をテストが名指し。3 項目の実施前は 92)
  - `mise run check-security-controls` - passed
  - `mise run report-coverage-debt` - Tenancy は 51 件から 39 件へ
  - `mise run check-work-items` - passed
  - `mise run check-ids` - passed
  - `mise run verify` - passed
