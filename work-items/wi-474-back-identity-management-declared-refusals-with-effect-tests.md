---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-05
change_kind: maintenance
priority: p1
depends_on: []
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

# IdentityManagement が宣言する未検証の拒否 9 件に効果まで確かめるテストを与え、台帳から外す

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

- 台帳の IdentityManagement の 9 件それぞれについて、宣言された拒否に production と同じ入口から到達するテストを用意する。
- 各テストで 2 つを assert する。呼び出し元が観測する応答 (ステータスとエラー種別) と、その拒否が変えなかった状態。
- 各テストのソースに対応する `REQ-IDMANAGEMENT-NNN` を書く。
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
- 管理 API のスコープ体系そのものの見直しと、CSV エクスポートの形式変更。

## Design

### 解消の条件

1 件の id を台帳から外してよいのは、次の 3 つがすべて成り立つときだけとする。

第 1 に、テストが production の使う入口から入り、宣言された拒否の判断を実際に通ること。

管理 API のスコープとロールの拒否は、実際のトークンを付けた HTTP の境界から入る。

第 2 に、拒否が変えなかったものをテスト自身が読み直して確かめること。

第 3 に、シナリオの id をテストのソースが引用していること。

### 拒否ごとに「変わっていないこと」として何を読むか

| REQ | 宣言している拒否 | 読み直して確かめる対象 |
|---|---|---|
| REQ-IDMANAGEMENT-002 | `account:read` だけの変更操作と不一致の `user_id` を拒否 | 対象ユーザーの属性が変更されていない |
| REQ-IDMANAGEMENT-003 | CSRF トークンと Cookie の不一致でメール確認を拒否 | メールアドレスが確認済みになっていない |
| REQ-IDMANAGEMENT-005 | 非管理者の一覧取得と、別テナント・改ざん・条件不一致のカーソルを拒否 | 応答にユーザーが 1 件も含まれず、別テナントのページも返らない |
| REQ-IDMANAGEMENT-006 | 許可一覧にない列、期限切れ、種別とテナントを越えた参照を拒否 | エクスポートが作成されておらず、ファイルもダウンロードできない |
| REQ-IDMANAGEMENT-008 | `group_id` 未指定と別グループのパスでの参照を拒否 | エクスポートが作成されておらず、他グループのメンバーが取得できない |
| REQ-IDMANAGEMENT-013 | 自身が admin である場合の削除の予約、復元、完全削除を拒否 | 対象ユーザーが削除予約状態にならず、在籍したままである |
| REQ-IDMANAGEMENT-014 | ロールを持たないユーザーの管理 API 呼び出しを拒否 | 応答に管理対象の一覧が含まれない |
| REQ-IDMANAGEMENT-022 | 不正な CEL の保存と動的グループの手動メンバー操作を拒否 | 規則が保存されておらず、メンバーシップも変わっていない |
| REQ-IDMANAGEMENT-025 | 読み取りスコープだけの変更、種別を越えた操作を拒否 | User、Group、Agent が 1 件も作成、更新、削除されていない |

`REQ-IDMANAGEMENT-025` は、シナリオ自身が「`Group` は 1 件も作成、更新、削除されない」と副作用の不在まで書いている。

宣言がすでにこの形をしているのに引用するテストが無いことは、規範として書いた側と確かめる側が接続していないことを示している。

`REQ-IDMANAGEMENT-006` の列指定の拒否は、拒否後に生成物が存在しないことまで読む。

エクスポートのジョブが作られてから拒否される実装は、応答だけを見るテストでは成功と区別できない。

### 却下した進め方

**named の 7 件を注記だけで閉じる案。**

分類の根拠は同じパッケージへ到達する拒否テストにエラー型名が現れることだけである。

**CSV エクスポートの拒否をハンドラーの単体テストで確かめる案。**

エクスポートは非同期のジョブを伴うため、拒否が「ジョブを作らない」ことまで含む。

ハンドラー単体では、作られたジョブが後で失敗する実装と区別できない。

**動的グループの手動操作の拒否を、規則の評価器の単体テストで確かめる案。**

拒否はメンバーシップを変える入口に置かれるべきものであり、評価器はその入口ではない。

## Plan

1. 9 件のシナリオを読み、拒否ごとに入口と「変わっていないこと」を確定する。
2. 既存の管理 API テストのうち、同じ入口へ到達しているものを特定し、拡張で足りるか新規が要るかを決める。
3. ロールとスコープの 4 件 (002、014、025、005) から着手し、テストが RED になることを先に観測する。
4. エクスポートの 2 件 (006、008) を進める。
5. CSRF と自己削除の 2 件 (003、013) を進める。
6. 動的グループの 1 件 (022) を進める。
7. 防護そのものが無い id が見つかった場合は実装するか、引き取る work item を切る。
8. 対応の取れた id を台帳から削除し、この項目が持つ id が 1 件も残っていないことを確認する。

## Tasks

- [ ] T001 [Inventory] 9 件の入口と「変わっていないこと」を確定し、既存テストの流用可否を判断する。
- [ ] T002 [Acceptance] ロールとスコープの拒否 (002、005、014、025) を効果まで確かめるテストを書く。
- [ ] T003 [Acceptance] CSV エクスポートの拒否 (006、008) を効果まで確かめるテストを書く。
- [ ] T004 [Acceptance] CSRF と自己削除の拒否 (003、013) を効果まで確かめるテストを書く。
- [ ] T005 [Acceptance] 動的グループの手動操作の拒否 (022) を効果まで確かめるテストを書く。
- [ ] T006 [App] 拒否の実装が欠けていた経路を修正する、または引き取る work item を切って `depends_on` でつなぐ。
- [ ] T007 [Ledger] 対応の取れた id を `example-coverage-debt.json` から削除する。
- [ ] T008 [Verify] 防護を意図的に外すと各テストが落ちることを確かめ、検査を通す。

## Verification

- `mise run test-go-race`
- `mise run check-spec`
- `mise run check-security-controls`
- `mise run report-coverage-debt`
- `mise run check-work-items`
- `mise run check-ids`
- `mise run verify`

## Risk Notes

拒否のテストは、防護に到達する前の入力検証で落ちていても同じ応答を返すため、書いた本人にも成立して見える。

これを避けるため、各テストについて防護を外した状態でテストが落ちることを確かめ、落ちなかったものは入口が誤っていると判断してやり直す。

エクスポートの拒否は非同期の生成と隣り合うため、「ジョブが作られていない」ことと「ジョブが作られたが失敗した」ことを取り違えやすい。

テストは前者を確かめる。

台帳は id 順に 1 エントリー 1 id なので、Context ごとの work item が並行しても衝突はエントリー単位に収まる。

各項目は自分が持つ id のエントリーだけを削除し、コメント配列と他の id には触れない。
