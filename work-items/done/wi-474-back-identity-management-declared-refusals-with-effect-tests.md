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
    - docs/contexts/identity-management/scenarios.feature.md#REQ-IDMANAGEMENT-002
    - docs/contexts/identity-management/scenarios.feature.md#REQ-IDMANAGEMENT-003
    - docs/contexts/identity-management/scenarios.feature.md#REQ-IDMANAGEMENT-005
    - docs/contexts/identity-management/scenarios.feature.md#REQ-IDMANAGEMENT-006
    - docs/contexts/identity-management/scenarios.feature.md#REQ-IDMANAGEMENT-008
    - docs/contexts/identity-management/scenarios.feature.md#REQ-IDMANAGEMENT-013
    - docs/contexts/identity-management/scenarios.feature.md#REQ-IDMANAGEMENT-014
    - docs/contexts/identity-management/scenarios.feature.md#REQ-IDMANAGEMENT-022
    - docs/contexts/identity-management/scenarios.feature.md#REQ-IDMANAGEMENT-025
  typespec:
    - IdMagic.IdManagement.Operations.ListAdminUsers
    - IdMagic.IdManagement.Operations.StartGroupMemberCsvExport
    - IdMagic.IdManagement.Operations.AddGroupMember
  source:
    - backend/idmanagement/handlers_http
    - backend/idmanagement/usecases
    - backend/idmanagement/user/handlers_http
    - backend/idmanagement/user/usecases
    - backend/idmanagement/group/handlers_http
    - backend/idmanagement/group/usecases
    - backend/idmanagement/agent/handlers_http
    - backend/shared/http/support_http
    - backend/shared/http/server_http
    - tools/check/src
affected_spec:
  - { path: docs/contexts/identity-management/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-002 }
  - { path: docs/contexts/identity-management/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-003 }
  - { path: docs/contexts/identity-management/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-005 }
  - { path: docs/contexts/identity-management/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-006 }
  - { path: docs/contexts/identity-management/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-008 }
  - { path: docs/contexts/identity-management/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-013 }
  - { path: docs/contexts/identity-management/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-014 }
  - { path: docs/contexts/identity-management/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-022 }
  - { path: docs/contexts/identity-management/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-025 }
---

# IdentityManagement が宣言する未検証の拒否に効果まで確かめるテストを与え、台帳から外す

## Motivation

`tools/check/example-coverage-debt.json` は、どのテストからも id を引用されていない規範 id を保持する、縮むだけの台帳である。

このうち 102 件は [[wi-390-security-control-test-standard-and-gate]] が拒否専用の台帳へ据え置いた分で、[[wi-490-fold-refusal-coverage-into-one-normative-coverage-rule]] が網羅台帳へ統合した。

その 102 件のうち 9 件が IdentityManagement の拒否である。

対象は管理 API のロールとスコープ、CSV エクスポートの列とテナントの境界、自己削除の禁止、動的グループの手動操作の禁止である。

いずれも素通りすれば、利用者の属性そのものが読み出されるか、書き換えられる。

`REQ-IDMANAGEMENT-006` の `password_hash` を含む列指定の拒否は、素通りしたときの結果が最も分かりやすい例である。

台帳が言えるのは「id を引用したテストが無い」ことだけで、「拒否を確かめるテストが無い」こととは違う。

`mise run report-coverage-debt` はこの差を named / nearby / none に分けるが、named は「同じエラー型名を含む拒否テストが同じパッケージへ到達する」ことしか示さない。

この分類が別の Context で `REQ-APPLICATION-005` に対して指したのは、SAML 署名証明書とは無関係な配信一覧のカーソル検証テストだった。

そして注記だけを足す解消は、この項目では認めない。

wi-390 が見つけた欠陥は、テストがあり、カバレッジもあり、それでも防護が素通りしていた形だった。

拒否の応答だけを確かめるテストに id を書き足せば、検査は通り、台帳は縮み、防護が効いていることは何ひとつ確かめられないまま「検証済み」の見た目だけが残る。

## Scope

- 下記「この項目が持つ id」の 17 件それぞれについて、宣言された拒否に production と同じ入口から到達するテストを用意する。
- 各テストで 2 つを assert する。呼び出し元が観測する応答 (ステータスとエラー種別) と、その拒否が変えなかった状態。
- 各テストのソースに対応する `EX-IDMANAGEMENT-NNN-MM` を書き、具体例とテストを突き合わせて読めるようにする。
- 対応が取れた id を `tools/check/example-coverage-debt.json` から削除する。
- 拒否を実装が持っていないと判明した場合、その防護をこの項目で実装する。
- 防護の追加に新しい設計判断が要る場合は、その id だけを台帳に残し、引き取る work item を切って `depends_on` でつなぐ。
- 完了時点で、この項目が持つ id が台帳から 1 件残らず消えていることを確認する。

## Out of Scope

- 他 Context の拒否。
  Context ごとに別の work item が持つ。
- 102 件を 1 つの単位として消化すること。
  [[wi-399-burn-down-untested-refusal-debt]] がその形で立てられているが、進める単位を未決のまま残し、注記だけの解消も認めている。本項目はその置き換えである。
- 9 つの Rule が持つ正常系の具体例。
  この項目は拒否だけを引き取る。同じ Rule の `通常経路` が台帳に残っていても、それは正常系の網羅を持つ別の項目の対象である。
- 同じ台帳に載る、この項目が持たない id。
  規則は同じだが、引き取る範囲は下記の 17 件に限る。
- R1、R2、R4 の検査そのものの変更。
  [[wi-390-security-control-test-standard-and-gate]] と [[wi-391-refusal-declaration-floor-and-reinventory]] が持つ。
- 既存テストへ id を注記するだけで台帳から外すこと。
  この項目ではこれを解消として扱わない。
- 具体例の Then ステップへ「変わらなかったもの」を規範として書き足すこと。
  下記「拒否ごとに何を読み直すか」を参照。
- 管理 API のスコープ体系そのものの見直しと、CSV エクスポートの形式変更。
- `EX-IDMANAGEMENT-005-04` の「正確な件数の取得に失敗する」。
  これは拒否ではなくフェイルクローズの縮退であり、起票時の表にも入っていない。正常系の網羅を持つ項目が引き取る。

## Design

### 台帳の単位が Rule から具体例へ移ったこと

この項目は `REQ-IDMANAGEMENT-NNN` 9 件を持つものとして起票された。

その後 [[wi-491-track-tests-per-example]] が台帳を Rule 単位から具体例単位へ移し、`scenarios.feature.md` は Markdown with Gherkin になった。

台帳の現在の単位は `EX-IDMANAGEMENT-NNN-MM` であり、Rule の id はもう台帳に載らない。

Rule の採番は移行を越えて保たれているため、起票時に挙げた 9 件は今も同じ拒否を指す。

そこで持ち分を、その 9 Rule が宣言する拒否の具体例へ読み替える。

正常系の具体例は持たない。

Rule 022 は Rule 自体が拒否の宣言であり、その `通常経路` が拒否そのものなので、その `-01` は持ち分に入る。

### この項目が持つ id

17 件。

| Rule | 具体例 | 宣言している拒否 | 読み直して確かめる対象 |
|---|---|---|---|
| REQ-IDMANAGEMENT-002 | EX-IDMANAGEMENT-002-02 | `account:read` だけの変更操作を拒否 | プロフィールもメールアドレス変更申請も残っていない |
| REQ-IDMANAGEMENT-002 | EX-IDMANAGEMENT-002-03 | トークンのテナントが操作対象と一致しない要求を拒否 | 対象ユーザーの属性が変更されていない |
| REQ-IDMANAGEMENT-003 | EX-IDMANAGEMENT-003-02 | CSRF トークンと Cookie の不一致でメール確認を拒否 | メールアドレスが確認済みにならず、同じトークンは正しい CSRF で今も使える |
| REQ-IDMANAGEMENT-005 | EX-IDMANAGEMENT-005-05 | 非管理者の一覧取得を拒否 | 応答にユーザーが 1 件も含まれない |
| REQ-IDMANAGEMENT-005 | EX-IDMANAGEMENT-005-06 | 別テナント発行・改ざん・条件不一致のカーソルを拒否 | 別テナントのページが返らない |
| REQ-IDMANAGEMENT-006 | EX-IDMANAGEMENT-006-02 | 許可一覧にない列 (`password_hash`) を拒否 | エクスポートのジョブが作られておらず、ファイルも生まれない |
| REQ-IDMANAGEMENT-006 | EX-IDMANAGEMENT-006-05 | 保持期限を過ぎたダウンロードを拒否 | CSV の中身が応答に出ず、`downloadable` も false |
| REQ-IDMANAGEMENT-006 | EX-IDMANAGEMENT-006-06 | 種別とテナントを越えた参照・ダウンロード・取り消しを拒否 | CSV の中身が応答に出ず、取り消しも効かない |
| REQ-IDMANAGEMENT-008 | EX-IDMANAGEMENT-008-02 | `group_id` 未指定のメンバーエクスポート開始を拒否 | エクスポートのジョブが作られていない |
| REQ-IDMANAGEMENT-008 | EX-IDMANAGEMENT-008-03 | 別グループのパスでの参照とダウンロードを拒否 | 他グループのメンバーが応答に出ない |
| REQ-IDMANAGEMENT-013 | EX-IDMANAGEMENT-013-02 | 自身が admin である場合の削除の予約、復元、完全削除を拒否 | 対象ユーザーが `Active` のまま在籍し、削除予約状態にならない |
| REQ-IDMANAGEMENT-014 | EX-IDMANAGEMENT-014-02 | ロールを持たないユーザーの管理 API 呼び出しを拒否 | 応答に管理対象の一覧が含まれず、作成も通らない |
| REQ-IDMANAGEMENT-022 | EX-IDMANAGEMENT-022-01 | 不正な CEL の保存と動的グループの手動メンバー操作を拒否 | 規則が保存されておらず、メンバーシップも変わっていない |
| REQ-IDMANAGEMENT-025 | EX-IDMANAGEMENT-025-02 | `users:read` だけでの User 変更と CSV インポートを拒否 | User が変更されず、インポートのジョブも作られない |
| REQ-IDMANAGEMENT-025 | EX-IDMANAGEMENT-025-03 | `groups:read` だけでの Group CSV インポートと適用を拒否 | Group が 1 件も作成、更新、削除されない |
| REQ-IDMANAGEMENT-025 | EX-IDMANAGEMENT-025-04 | `users:*` だけでの Group と Agent の操作を拒否 | Group も Agent も作成されない |
| REQ-IDMANAGEMENT-025 | EX-IDMANAGEMENT-025-05 | `agents:read` だけでの Agent のキルと削除を拒否 | 対象 Agent が在籍し、失効エポックも進まない |

`EX-IDMANAGEMENT-025-03` は、シナリオ自身が「`Group` は 1 件も作成、更新、削除されない」と副作用の不在まで書いている。

宣言がすでにこの形をしているのに引用するテストが無いことは、規範として書いた側と確かめる側が接続していないことを示している。

`EX-IDMANAGEMENT-006-02` の列指定の拒否は、拒否後に生成物が存在しないことまで読む。

エクスポートのジョブが作られてから拒否される実装は、応答だけを見るテストでは成功と区別できない。

### 解消の条件

1 件の id を台帳から外してよいのは、次の 3 つがすべて成り立つときだけとする。

第 1 に、テストが production の使う入口から入り、宣言された拒否の判断を実際に通ること。

管理 API のスコープとロールの拒否は、実際に発行した API アクセストークンを付けた HTTP の境界から入る。

ブラウザーの経路で宣言された拒否は、セッション Cookie、Origin、CSRF トークンを伴う HTTP の境界から入る。

use case を直接呼ぶテストは、これらの判定を一つも通らないため、この条件を満たさない。

第 2 に、拒否が変えなかったものをテスト自身が読み直して確かめること。

第 3 に、具体例の id をテストのソースが引用していること。

### 対照を必ず 1 つ置く

どの拒否のテストにも、同じ入口で同じ操作が通る対照を 1 つ置く。

「拒否されたので何も起きていない」と「そもそも何も起こせない構成だった」は、効果の側だけを読むと同じに見える。

対照が無ければ、配線を落としただけのテストがそのまま緑になる。

### 拒否ごとに何を読み直すかを仕様へ書き足さない理由

上の表は各拒否の 2 つ目の assert を決めるが、これを具体例の Then ステップへ書き足すことはしない。

「拒否は副作用を止めたことまで assert する」は
[docs/development/specification-first-workflow.md](../../docs/development/specification-first-workflow.md) の検証ラダーが全 Context に課している検証の規範であって、IdentityManagement が外部に約束する振る舞いの追加ではない。

同じ内容を 17 の具体例へ書き写せば、規範の記述が 1 か所から 18 か所へ増え、検証方法の変更が仕様の変更に見えるようになる。

### エクスポートの保持期限をテストから動かす方法

`EX-IDMANAGEMENT-006-05` の期限切れは `job.CreatedAt + DataExportTTL` と現在時刻の比較で決まり、HTTP のハンドラーは時計を注入していない。

そこで `jobs` の Repository を包む装飾をテスト側に置き、`Enqueue` の `Now` だけを過去へずらす。

production の時計にも判定にも触れないまま、期限を過ぎたエクスポートが HTTP の境界から観測できる。

装飾を外した同じ流れがダウンロードできることを対照に置く。

### 却下した進め方

**named の 11 件を注記だけで閉じる案。**

分類の根拠は同じパッケージへ到達する拒否テストにエラー型名が現れることだけである。

**CSV エクスポートの拒否をハンドラーの単体テストで確かめる案。**

エクスポートは非同期のジョブを伴うため、拒否が「ジョブを作らない」ことまで含む。

ハンドラー単体では、作られたジョブが後で失敗する実装と区別できない。

**動的グループの手動操作の拒否を、規則の評価器の単体テストで確かめる案。**

拒否はメンバーシップを変える入口に置かれるべきものであり、評価器はその入口ではない。

**粒度スコープの 4 件を偽の introspector で確かめる案。**

API アクセストークンのテナント束縛は `aud` と `jti` の照合としてイントロスペクションの側にあり、
偽の introspector で置き換えるとその照合ごと消える。本物の署名器で発行したトークンを使う。

## Plan

1. 17 件の入口と「変わっていないこと」を確定する。 (完了、上表)
2. ロールとスコープの 7 件 (005-05、005-06、014-02、025-02、025-03、025-04、025-05) を進める。
3. エクスポートの 5 件 (006-02、006-05、006-06、008-02、008-03) を進める。
4. アカウント API と CSRF の 3 件 (002-02、002-03、003-02) を進める。
5. 自己削除と動的グループの 2 件 (013-02、022-01) を進める。
6. 各テストについて防護を外すと落ちることを確かめる。落ちないものは入口が誤っているので入口からやり直す。
7. 防護そのものが無い id が見つかった場合は実装するか、引き取る work item を切る。
8. 対応の取れた id を台帳から削除し、この項目が持つ id が 1 件も残っていないことを確認する。

## Tasks

- [x] T001 [Inventory] 17 件の入口と「変わっていないこと」を確定し、既存テストの流用可否を判断する。
- [x] T002 [Acceptance] ロールとスコープの拒否 7 件を効果まで確かめる。
- [x] T003 [Acceptance] CSV エクスポートの拒否 5 件を効果まで確かめる。
- [x] T004 [Acceptance] アカウント API と CSRF の拒否 3 件を効果まで確かめる。
- [x] T005 [Acceptance] 自己削除と動的グループの拒否 2 件を効果まで確かめる。
- [x] T006 [App] 拒否の実装が欠けていた経路を修正する、または引き取る work item を切って `depends_on` でつなぐ。
      欠けていた経路は 1 つも見つからなかった。実装の変更は無い。
- [x] T007 [Ledger] 対応の取れた id を `example-coverage-debt.json` から削除する。
- [x] T008 [Verify] 防護を意図的に外すと各テストが落ちることを確かめ、検査を通す。

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

エクスポートの拒否は非同期の生成と隣り合うため、「ジョブが作られていない」ことと「ジョブが作られたが失敗した」ことを取り違えやすい。

テストは前者を確かめる。

台帳は id 順に 1 エントリー 1 id なので、Context ごとの work item が並行しても衝突はエントリー単位に収まる。

各項目は自分が持つ id のエントリーだけを削除し、コメント配列と他の id には触れない。

## Completion

- **Completed At**: 2026-09-05
- **Summary**:
  `mise run spec-diff` は `no normative specification change against main` を返す。
  規範の宣言は 1 件も変わっていない。変わったのは、宣言済みの拒否 17 件が
  「応答だけ確かめられている」状態から「副作用を止めたことまで確かめられている」状態へ
  移ったことと、その 17 件が網羅台帳から消えたことである。
  `check-spec` の網羅は「テストが名指しする id」75 件から 92 件へ増え、
  `example-coverage-debt.json` は 655 件から 638 件へ、IdentityManagement 分は
  84 件から 67 件へ減った。
  製品コードの変更は無い。防護が欠けている経路は 1 つも見つからなかった。

- **Acceptance RED Evidence**:
  - **Test**: `TestUserExportAcrossTypeAndTenantReturnsNoCSVAndCancelsNothing`
    (`backend/idmanagement/handlers_http/export_refusal_effects_test.go`)。
    ほかの 17 例も同じ形で観測した。対象の要求は `affected_spec` の 9 件すべてに及ぶ。
  - **Requirement**: REQ-IDMANAGEMENT-006
  - **Observed Failure**: `loadScopedExport` からテナントと種別の判定を外すと、
    `別テナントでのダウンロード` が 200 を返し、本文に
    `preferred_username,email / admin-default,... / user-alice,... / user-bob,...` の
    CSV がそのまま現れた。acme のレルムから default 全利用者の一覧が読み出せた形である。
    障害注入のたびに、対応するテストだけが落ちた。
  - **Detection Reason**: どのテストも応答と「拒否が変えなかった状態」の 2 つを assert する。
    このテストは 404 を読むだけでなく、レスポンスボディに対象利用者の識別子が現れないことと、
    取り消しのあとに正しい経路から `succeeded` を読み直すことを条件にしている。
    拒否の応答を返しつつ本体を返す実装も、応答を返しつつ取り消しを実行する実装も捕まえる。

- **Unit RED Evidence**:
  - **Test**: `N/A: 防護が欠けている経路が 1 つも見つからなかったため、単体の RED は発生しない。`
    代わりに実施したのは、実装済みの防護を 1 つずつ外して対応するテストが落ちることの確認である
    (下記 Change-Resistance Results)。
  - **Requirement**: N/A: この項目は宣言済みの拒否へ検証を足すもので、新しい製品要求を作らない。
  - **Observed Failure**: `N/A`
  - **Detection Reason**: `N/A`

- **Change-Resistance Results**:
  防護を 1 つずつ外し、対応するテストだけが落ちることを確かめた。
  結果は 2 つに分かれる。

  1 つ目は、外した防護に対して素直に落ちたもの。

  - `Authenticator.RequireAdmin` の `admin` ロール判定を外すと `014-02` と `005-05` が落ちた。
    `014-02` はロールを持たない利用者に利用者一覧全件が返り、`005-05` は総件数まで返った。
  - `ParsePageRequest` のカーソル復号失敗を先頭ページへの縮退に替えると `005-06` が
    3 つの部分検査すべてで落ちた。
  - `UserCSVExporter.ValidateUserCSVColumns` を素通しにすると `006-02` が落ち、
    `password_hash` を含むエクスポートが 202 で受理された。
  - `mapExportStatus` の期限判定を外すと `006-05` が落ち、期限切れが `succeeded` /
    `downloadable=true` になった。
  - `loadScopedExport` のテナントと種別の判定を外すと `006-06` と `008-03` が落ちた。
  - `validateExportFilter` の `group_id` 必須判定を外すと `008-02` が落ちた。
  - `SoftDeleteUser` / `RestoreUser` / `DeleteUser` の `ErrSelfDeleteForbidden` を 3 か所とも
    外すと `013-02` が落ちた。削除の予約が 204 で通り、その後の要求は 401 になった。
    自分を消した管理者が締め出される形がそのまま観測できた。
  - `requireAdminApiTokenScope` を素通しにすると `025-02`、`025-03`、`025-04`、`025-05` が落ちた。
  - `hasRequiredAccountScope` の判定を外すと `002-02` が落ちた。
  - `VerifyBrowserRequest` の二重送信 CSRF の比較を外すと `003-02` が落ちた。
  - `CompileDynamicGroupRule` の属性と関数の検査、および `AddMember` / `RemoveMember` の
    `ErrDynamicMembershipManaged` を外すと `022-01` が 4 つの部分検査すべてで落ちた。
  - API アクセストークンの `user_id` 整合の 2 判定 (`AuthenticateClaims` の
    `token.UserID != claims.Sub` と `resolveAuthnContext` の `principal.UserID != res.Sub`) を
    外すと `002-03` の `user_id` の側が落ち、記録上 bob のトークンで alice が改名された。

  2 つ目は、単一の判定を外しても落ちなかったもの。想定していなかったので、
  外す範囲を広げて確かめ直した。

  `002-03` のテナントの側は次の 6 本で守られていた。

  1. `tokens_jose.JWTSigner.IntrospectAccessToken` の `iss` の照合
  2. `apitoken.AuthenticateClaims` の `aud` の照合
  3. `apitoken` Repository の `FindByJTI` がテナントで絞ること
  4. `Authenticator.ResolveAuthentication` の `user.TenantID != RequestTenantID(c)`
  5. `user/usecases.loadSelf` の `user.TenantID != tenancy.TenantID(ctx)`
  6. 署名鍵ストアがテナントごとに分かれていること

  1 から 5 をすべて外しても越境した要求は拒否されたままだった。6 番目、すなわち
  default の鍵で署名したトークンは acme の鍵では検証できないためである。
  6 は「1 行外すと落ちる」形の防護ではなく、鍵の分離という設計そのものなので、
  障害注入で RED を作ることはできない。
  そこで同じ具体例の `user_id` の側に、取り外せる判定に対する変更耐性を持たせた
  (上記の最後の項目)。テナントの側のテストが固定するのは境界が保たれていることであり、
  特定の 1 行の有無ではない。この読み替えはテストのソースにも書いてある。

- **持ち越した観測**:
  `EX-IDMANAGEMENT-002-03` が宣言する拒否は `AccessDeniedError` だが、越境したトークンに
  対して実装が返すのは 401 `invalid_token` である。RFC 6750 の資源サーバとしては
  「提示されたトークンがこの資源では無効」なので 401 が自然であり、拒否そのものは効いて
  副作用も止まっている。宣言と本体型の不一致は
  [[wi-454-declared-403-body-vs-guard-error-codes]] が 403 について持つのと同じ形であり、
  401 へ広げる判断はその項目に属する。テストは実装が返す 401 を固定した。

- **Verification Results**:
  - `mise run test-go-race` - passed
  - `mise run check-spec` - passed (711 例中 92 id をテストが名指し。実施前は 75)
  - `mise run check-security-controls` - passed
  - `mise run report-coverage-debt` - IdentityManagement は 84 件から 67 件へ
  - `mise run check-work-items` - passed
  - `mise run check-ids` - passed
  - `mise run verify` - passed
