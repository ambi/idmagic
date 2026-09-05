---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-05
change_kind: maintenance
priority: p2
depends_on: []
affected_spec:
  - { path: docs/contexts/application/scenarios.feature.md, requirement: REQ-APPLICATION-003 }
  - { path: docs/contexts/application/scenarios.feature.md, requirement: REQ-APPLICATION-004 }
  - { path: docs/contexts/application/scenarios.feature.md, requirement: REQ-APPLICATION-005 }
  - { path: docs/contexts/application/scenarios.feature.md, requirement: REQ-APPLICATION-006 }
  - { path: docs/contexts/application/scenarios.feature.md, requirement: REQ-APPLICATION-008 }
  - { path: docs/contexts/application/scenarios.feature.md, requirement: REQ-APPLICATION-009 }
  - { path: docs/contexts/application/scenarios.feature.md, requirement: REQ-APPLICATION-011 }
  - { path: docs/contexts/application/scenarios.feature.md, requirement: REQ-APPLICATION-013 }
---

# Application が宣言する未検証の拒否 8 件に効果まで確かめるテストを与え、台帳から外す

## Motivation

`tools/check/example-coverage-debt.json` は、どのテストからも id を引用されていない規範 id を保持する、縮むだけの台帳である。

このうち 102 件は [[wi-390-security-control-test-standard-and-gate]] が拒否専用の台帳へ据え置いた分で、[[wi-490-fold-refusal-coverage-into-one-normative-coverage-rule]] が網羅台帳へ統合した。

その 102 件のうち 8 件が Application の拒否である。

対象は管理 API のスコープとロール、割り当てのない主体のフェデレーション拒否、クレーム公開の下限、アイコンのアップロード検証である。

`REQ-APPLICATION-006` のクレーム公開の下限は、素通りすればパスワード関連の内部属性がトークンへ載る。

`REQ-APPLICATION-011` の割り当て検査は、素通りすれば割り当てを設定していない利用者がアプリケーションへログインできる。

この Context は、報告タスクの named 分類が当てにならないことを最も明確に示した場所でもある。

`mise run report-coverage-debt` は `REQ-APPLICATION-005` に対して `backend/provisioning/handlers_http/admin_delivery_list_pagination_test.go` の `TestAdminDeliveryListRejectsInvalidCursor` を挙げるが、これは配信一覧のカーソル検証であり、AuthnRequest 署名証明書の検証とは何の関係もない。

named が示しているのは、同じエラー型名を含む拒否テストが import 経由で同じパッケージに到達することだけである。

そして注記だけを足す解消は、この項目では認めない。

wi-390 が見つけた欠陥は、テストがあり、カバレッジもあり、それでも防護が素通りしていた形だった。

拒否の応答だけを確かめるテストに id を書き足せば、検査は通り、台帳は縮み、防護が効いていることは何ひとつ確かめられないまま「検証済み」の見た目だけが残る。

## Scope

- 台帳の Application の 8 件それぞれについて、宣言された拒否に production と同じ入口から到達するテストを用意する。
- 各テストで 2 つを assert する。呼び出し元が観測する応答 (ステータスとエラー種別) と、その拒否が変えなかった状態。
- 各テストのソースに対応する `REQ-APPLICATION-NNN` を書く。
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
- クレーム公開の下限そのものの定義変更と、サインインポリシーの条件追加。

## Design

### 解消の条件

1 件の id を台帳から外してよいのは、次の 3 つがすべて成り立つときだけとする。

第 1 に、テストが production の使う入口から入り、宣言された拒否の判断を実際に通ること。

フェデレーションの拒否は、管理 API ではなくプロトコルの入口から入る。

`REQ-APPLICATION-011` は割り当てのない主体が OIDC、SAML、WS-Federation のいずれかでアプリケーションへ到達しようとする経路であり、そこを通らないテストは配線の欠落を検出できない。

第 2 に、拒否が変えなかったものをテスト自身が読み直して確かめること。

第 3 に、シナリオの id をテストのソースが引用していること。

### 拒否ごとに「変わっていないこと」として何を読むか

| REQ | 宣言している拒否 | 読み直して確かめる対象 |
|---|---|---|
| REQ-APPLICATION-003 | 不一致のテナントまたは `user_id`、`account:read` だけの操作を拒否 | 対象のポータルアプリケーションが変更されていない |
| REQ-APPLICATION-004 | `applications:read` だけの変更と、テナント不一致を拒否 | Application が作成も更新も削除もされていない |
| REQ-APPLICATION-005 | 検証可能な証明書のない AuthnRequest 署名必須設定を拒否 | SAML プロトコル設定が保存前の値のままである |
| REQ-APPLICATION-006 | `visibility=Private` の属性と予約済みクレーム型の指定を拒否 | クレーム公開規則が保存されておらず、以後発行されるトークンにも当該クレームが載らない |
| REQ-APPLICATION-008 | 非画像・上限超過のアップロードと別テナントのアイコン取得を拒否 | 既存のアイコンが置き換わっておらず、別テナントのアセットも取得できない |
| REQ-APPLICATION-009 | 管理者以外のポリシー更新と、許可 CIDR 外のフェデレーションを拒否 | ポリシーが更新されておらず、フェデレーションではトークンが発行されていない |
| REQ-APPLICATION-011 | 割り当てのない主体のフェデレーションを拒否 | 認可コードもアサーションも発行されておらず、アプリケーションへのセッションが成立していない |
| REQ-APPLICATION-013 | admin ロールを持たない利用者の Application 操作を拒否 | Application の一覧が応答に含まれず、変更も反映されていない |

`REQ-APPLICATION-006` は、保存の拒否だけでなく、その後に発行されるトークンへ当該クレームが現れないことまで確かめる。

拒否が保存の手前で効いていることの証明は、最終的にトークンの中身で読むのが最も直接的である。

`REQ-APPLICATION-009` は 2 つの異なる拒否を宣言しており、片方だけを確かめて閉じない。

管理経路の拒否とフェデレーション経路の拒否は、入口も守っている資産も別である。

### 却下した進め方

**named の 7 件を注記だけで閉じる案。**

この Context の誤検出は実測で確認されており、注記だけで閉じれば誤った検証済み表示がそのまま残る。

**フェデレーションの拒否を割り当て判定関数の単体テストで確かめる案。**

判定が正しくても呼ばれていなければ素通りする。

wi-390 の欠陥はまさにその形であり、同じ形のテストで閉じることはできない。

**アイコンのアップロード検証をファイル種別の判定関数だけで確かめる案。**

拒否の効果は「既存のアイコンが置き換わらない」ことであり、判定関数はその効果を持たない。

## Plan

1. 8 件のシナリオを読み、拒否ごとに入口と「変わっていないこと」を確定する。
2. 既存の Application テストと、プロトコル側の既存 E2E を調べ、拡張で足りるか新規が要るかを決める。
3. 管理 API のスコープとロールの 3 件 (003、004、013) から着手し、テストが RED になることを先に観測する。
4. フェデレーションの 2 件 (009、011) を進める。
5. 設定の妥当性の 3 件 (005、006、008) を進める。
6. 防護そのものが無い id が見つかった場合は実装するか、引き取る work item を切る。
7. 対応の取れた id を台帳から削除し、この項目が持つ id が 1 件も残っていないことを確認する。

## Tasks

- [ ] T001 [Inventory] 8 件の入口と「変わっていないこと」を確定し、既存テストの流用可否を判断する。
- [ ] T002 [Acceptance] 管理 API のスコープとロールの拒否 (003、004、013) を効果まで確かめるテストを書く。
- [ ] T003 [Acceptance] フェデレーションの拒否 (009、011) をプロトコルの入口から効果まで確かめるテストを書く。
- [ ] T004 [Acceptance] 設定の妥当性の拒否 (005、006、008) を効果まで確かめるテストを書く。
- [ ] T005 [App] 拒否の実装が欠けていた経路を修正する、または引き取る work item を切って `depends_on` でつなぐ。
- [ ] T006 [Ledger] 対応の取れた id を `example-coverage-debt.json` から削除する。
- [ ] T007 [Verify] 防護を意図的に外すと各テストが落ちることを確かめ、検査を通す。

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

`REQ-APPLICATION-011` はプロトコルごとに別の入口を持つため、1 つのプロトコルだけを確かめて完了とすると、残りのプロトコルで割り当て検査が欠けていても見えない。

対応しているプロトコルすべてで確かめる。

台帳は id 順に 1 エントリー 1 id なので、Context ごとの work item が並行しても衝突はエントリー単位に収まる。

各項目は自分が持つ id のエントリーだけを削除し、コメント配列と他の id には触れない。
