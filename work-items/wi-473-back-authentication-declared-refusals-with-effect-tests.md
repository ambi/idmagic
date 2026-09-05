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
  - { path: docs/contexts/authentication/scenarios.md, requirement: REQ-AUTHENTICATION-002 }
  - { path: docs/contexts/authentication/scenarios.md, requirement: REQ-AUTHENTICATION-003 }
  - { path: docs/contexts/authentication/scenarios.md, requirement: REQ-AUTHENTICATION-004 }
  - { path: docs/contexts/authentication/scenarios.md, requirement: REQ-AUTHENTICATION-005 }
  - { path: docs/contexts/authentication/scenarios.md, requirement: REQ-AUTHENTICATION-006 }
  - { path: docs/contexts/authentication/scenarios.md, requirement: REQ-AUTHENTICATION-008 }
  - { path: docs/contexts/authentication/scenarios.md, requirement: REQ-AUTHENTICATION-009 }
  - { path: docs/contexts/authentication/scenarios.md, requirement: REQ-AUTHENTICATION-017 }
  - { path: docs/contexts/authentication/scenarios.md, requirement: REQ-AUTHENTICATION-020 }
  - { path: docs/contexts/authentication/scenarios.md, requirement: REQ-AUTHENTICATION-021 }
  - { path: docs/contexts/authentication/scenarios.md, requirement: REQ-AUTHENTICATION-022 }
  - { path: docs/contexts/authentication/scenarios.md, requirement: REQ-AUTHENTICATION-025 }
---

# Authentication が宣言する未検証の拒否 12 件に効果まで確かめるテストを与え、台帳から外す

## Motivation

`tools/check/scenario-coverage-debt.json` は、どのテストからも id を引用されていない規範 id を保持する、縮むだけの台帳である。

このうち 102 件は [[wi-390-security-control-test-standard-and-gate]] が拒否専用の台帳へ据え置いた分で、[[wi-490-fold-refusal-coverage-into-one-normative-coverage-rule]] が網羅台帳へ統合した。

その 102 件のうち 12 件が Authentication の拒否である。

対象は自動リンク、ステップアップ認証、CSRF、流量制限、無効化ユーザーの締め出しという、ログインの入口そのものを守る防護である。

ここが素通りする欠陥は、他の Context がどれだけ正しく認可していても手前で無効化する。

台帳が言えるのは「id を引用したテストが無い」ことだけで、「拒否を確かめるテストが無い」こととは違う。

`mise run report-coverage-debt` はこの差を named / nearby / none に分けるが、named は「同じエラー型名を含む拒否テストが同じパッケージへ到達する」ことしか示さない。

この分類が別の Context で `REQ-APPLICATION-005` に対して指したのは、SAML 署名証明書とは無関係な配信一覧のカーソル検証テストだった。

したがって named の 7 件を「注記が抜けているだけ」と扱うことはできない。

そして注記だけを足す解消は、この項目では認めない。

wi-390 が見つけた欠陥は、テストがあり、カバレッジもあり、それでも防護が素通りしていた形だった。

拒否の応答だけを確かめるテストに id を書き足せば、検査は通り、台帳は縮み、防護が効いていることは何ひとつ確かめられないまま「検証済み」の見た目だけが残る。

## Scope

- 台帳の Authentication の 12 件それぞれについて、宣言された拒否に production と同じ入口から到達するテストを用意する。
- 各テストで 2 つを assert する。呼び出し元が観測する応答 (ステータスとエラー種別) と、その拒否が変えなかった状態。
- 各テストのソースに対応する `REQ-AUTHENTICATION-NNN` を書く。
- 対応が取れた id を `tools/check/scenario-coverage-debt.json` から削除する。
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
- 認証方式そのものの追加、ステップアップ認証の要件変更、流量制限の閾値の見直し。

## Design

### 解消の条件

1 件の id を台帳から外してよいのは、次の 3 つがすべて成り立つときだけとする。

第 1 に、テストが production の使う入口から入り、宣言された拒否の判断を実際に通ること。

ブラウザーの経路で宣言された拒否は、Cookie とヘッダーを伴う HTTP の境界から入る。

use case を直接呼ぶテストは、CSRF や Origin の検証を通らないため、この条件を満たさない。

第 2 に、拒否が変えなかったものをテスト自身が読み直して確かめること。

第 3 に、シナリオの id をテストのソースが引用していること。

### 拒否ごとに「変わっていないこと」として何を読むか

| REQ | 宣言している拒否 | 読み直して確かめる対象 |
|---|---|---|
| REQ-AUTHENTICATION-002 | ポリシー `None`、未検証メール、一意でない一致では自動リンクを拒否 | 外部アイデンティティのリンクが作られておらず、LoginSession も発行されていない |
| REQ-AUTHENTICATION-003 | ステップアップが古い、または締め出しになる解除を拒否 | リンクと解除のいずれも反映されておらず、資格情報の構成が変わっていない |
| REQ-AUTHENTICATION-004 | 非対応スコープと不一致の `user_id`、API トークンでのステップアップ要求を拒否 | 機密操作の対象となる認証情報が変更されていない |
| REQ-AUTHENTICATION-005 | 未認証・認証途中、スコープを持たない Bearer のアカウントコンテキスト取得を拒否 | 応答にアカウント情報と CSRF トークンが含まれない |
| REQ-AUTHENTICATION-006 | CSRF 不一致と WebAuthn 不可時のチャレンジ発行を拒否 | チャレンジが 1 件も保存されていない |
| REQ-AUTHENTICATION-008 | 識別子と IP の組で上限に達した再要求を `RateLimitedError` で拒否 | リセットトークンが追加発行されておらず、メールも送信されていない |
| REQ-AUTHENTICATION-009 | 無効なユーザーの新規ログインと既存セッションを拒否 | セッションが確立されず、既存セッションでの保護リソース取得も成功しない |
| REQ-AUTHENTICATION-017 | 誤った TOTP コードを `InvalidRequestError` で拒否 | LoginSession が `authentication_pending` のままである |
| REQ-AUTHENTICATION-020 | 登録待ちセッションからの通常リソースへのアクセスを未認証として拒否 | 対象リソースの内容が応答に含まれない |
| REQ-AUTHENTICATION-021 | 他テナントの管理者によるセッション操作を拒否 | 対象ユーザーのセッションが失効していない |
| REQ-AUTHENTICATION-022 | 他テナントおよび非 admin の認証器リセットを拒否 | 対象ユーザーの認証器が変更されていない |
| REQ-AUTHENTICATION-025 | API トークンによる外部 IdP 接続の管理を `insufficient_scope` で拒否 | 接続設定が作成も更新もされていない |

`REQ-AUTHENTICATION-009` と `REQ-AUTHENTICATION-021` は、拒否の効果が「セッションが使えないまま」であることなので、拒否の応答だけでなく、その後の保護リソース取得が成功しないことまで確かめる。

`REQ-AUTHENTICATION-008` の流量制限は、拒否のたびに副作用が起きていないこと、すなわちメール送信とトークン発行が増えていないことを数で確かめる。

### 却下した進め方

**named の 7 件を注記だけで閉じる案。**

分類の根拠は同じパッケージへ到達する拒否テストにエラー型名が現れることだけであり、別 Context での誤検出の実例がその弱さを示している。

**拒否の応答だけを確かめて完了とする案。**

wi-390 の欠陥そのものであり、この項目が存在する理由と正面から矛盾する。

**ステップアップと CSRF をモックで置き換えて単体で確かめる案。**

置き換えた時点で、配線が外れた実装を検出できなくなる。

これらの防護はミドルウェアとハンドラーの結線に宿るため、境界を通らないテストでは意味を持たない。

## Plan

1. 12 件のシナリオを読み、拒否ごとに入口と「変わっていないこと」を確定する。
2. 既存の認証テストのうち、同じ入口へ到達しているものを特定し、拡張で足りるか新規が要るかを決める。
3. セッションと締め出しの 3 件 (009、020、017) から着手し、テストが RED になることを先に観測する。
4. スコープと主体の一致の 3 件 (004、005、025) を進める。
5. ステップアップと自動リンクの 2 件 (002、003) を進める。
6. CSRF と流量制限の 2 件 (006、008) を進める。
7. 管理操作のテナント境界の 2 件 (021、022) を進める。
8. 防護そのものが無い id が見つかった場合は実装するか、引き取る work item を切る。
9. 対応の取れた id を台帳から削除し、この項目が持つ id が 1 件も残っていないことを確認する。

## Tasks

- [ ] T001 [Inventory] 12 件の入口と「変わっていないこと」を確定し、既存テストの流用可否を判断する。
- [ ] T002 [Acceptance] セッションと締め出しの拒否 (009、017、020) を効果まで確かめるテストを書く。
- [ ] T003 [Acceptance] スコープと主体の一致の拒否 (004、005、025) を効果まで確かめるテストを書く。
- [ ] T004 [Acceptance] ステップアップと自動リンクの拒否 (002、003) を効果まで確かめるテストを書く。
- [ ] T005 [Acceptance] CSRF と流量制限の拒否 (006、008) を効果まで確かめるテストを書く。
- [ ] T006 [Acceptance] 管理操作のテナント境界の拒否 (021、022) を効果まで確かめるテストを書く。
- [ ] T007 [App] 拒否の実装が欠けていた経路を修正する、または引き取る work item を切って `depends_on` でつなぐ。
- [ ] T008 [Ledger] 対応の取れた id を `scenario-coverage-debt.json` から削除する。
- [ ] T009 [Verify] 防護を意図的に外すと各テストが落ちることを確かめ、検査を通す。

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

流量制限の拒否は共有カウンタの状態に依存するため、テストが互いの計数を汚さないよう、識別子と IP の組をテストごとに分ける。

台帳は id 順に 1 エントリー 1 id なので、Context ごとの work item が並行しても衝突はエントリー単位に収まる。

各項目は自分が持つ id のエントリーだけを削除し、コメント配列と他の id には触れない。
