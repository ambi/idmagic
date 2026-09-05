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
  - { path: docs/contexts/oauth2/scenarios.md, requirement: REQ-OAUTH2-001 }
  - { path: docs/contexts/oauth2/scenarios.md, requirement: REQ-OAUTH2-002 }
  - { path: docs/contexts/oauth2/scenarios.md, requirement: REQ-OAUTH2-004 }
  - { path: docs/contexts/oauth2/scenarios.md, requirement: REQ-OAUTH2-007 }
  - { path: docs/contexts/oauth2/scenarios.md, requirement: REQ-OAUTH2-009 }
  - { path: docs/contexts/oauth2/scenarios.md, requirement: REQ-OAUTH2-010 }
  - { path: docs/contexts/oauth2/scenarios.md, requirement: REQ-OAUTH2-013 }
  - { path: docs/contexts/oauth2/scenarios.md, requirement: REQ-OAUTH2-015 }
  - { path: docs/contexts/oauth2/scenarios.md, requirement: REQ-OAUTH2-016 }
  - { path: docs/contexts/oauth2/scenarios.md, requirement: REQ-OAUTH2-017 }
  - { path: docs/contexts/oauth2/scenarios.md, requirement: REQ-OAUTH2-018 }
  - { path: docs/contexts/oauth2/scenarios.md, requirement: REQ-OAUTH2-020 }
  - { path: docs/contexts/oauth2/scenarios.md, requirement: REQ-OAUTH2-021 }
  - { path: docs/contexts/oauth2/scenarios.md, requirement: REQ-OAUTH2-023 }
  - { path: docs/contexts/oauth2/scenarios.md, requirement: REQ-OAUTH2-024 }
  - { path: docs/contexts/oauth2/scenarios.md, requirement: REQ-OAUTH2-028 }
  - { path: docs/contexts/oauth2/scenarios.md, requirement: REQ-OAUTH2-029 }
  - { path: docs/contexts/oauth2/scenarios.md, requirement: REQ-OAUTH2-034 }
  - { path: docs/contexts/oauth2/scenarios.md, requirement: REQ-OAUTH2-036 }
  - { path: docs/contexts/oauth2/scenarios.md, requirement: REQ-OAUTH2-037 }
  - { path: docs/contexts/oauth2/scenarios.md, requirement: REQ-OAUTH2-038 }
  - { path: docs/contexts/oauth2/scenarios.md, requirement: REQ-OAUTH2-039 }
  - { path: docs/contexts/oauth2/scenarios.md, requirement: REQ-OAUTH2-040 }
---

# OAuth2 が宣言する未検証の拒否 23 件に効果まで確かめるテストを与え、台帳から外す

## Motivation

`tools/check/scenario-coverage-debt.json` は、どのテストからも id を引用されていない規範 id を保持する、縮むだけの台帳である。

このうち 102 件は [[wi-390-security-control-test-standard-and-gate]] が拒否専用の台帳へ据え置いた分で、[[wi-490-fold-refusal-coverage-into-one-normative-coverage-rule]] が網羅台帳へ統合した。

その 102 件のうち 23 件が OAuth2 の拒否である。

これは全 Context で最大であり、対象はクライアント認証、スコープ、テナント境界、センダー制約、フェイルクローズという、この製品がプロトコルとして外部に約束している防護そのものである。

台帳が言えるのは「id を引用したテストが無い」ことだけで、「拒否を確かめるテストが無い」こととは違う。

`mise run report-coverage-debt` はこの差を named / nearby / none に分けるが、named は「同じエラー型名を含む拒否テストが同じパッケージへ到達する」ことしか示さない。

実際、この分類で `REQ-APPLICATION-005` が指すのは `backend/provisioning/handlers_http/admin_delivery_list_pagination_test.go` の `TestAdminDeliveryListRejectsInvalidCursor` であり、SAML 署名証明書の検証とは何の関係もない。

したがって named の 18 件を「注記が抜けているだけ」と扱うことはできず、1 件ずつ読んで確かめるほかない。

そして注記だけを足す解消は、この項目では認めない。

wi-390 が見つけた欠陥は、テストがあり、カバレッジもあり、それでも防護が素通りしていた形だった。

拒否の応答だけを確かめるテストに id を書き足せば、検査は通り、台帳は縮み、防護が効いていることは何ひとつ確かめられないまま「検証済み」の見た目だけが残る。

台帳が縮むこと自体には価値がなく、価値があるのは拒否が実際に副作用を止めていると分かることである。

## Scope

- 台帳の OAuth2 の 23 件それぞれについて、宣言された拒否に production と同じ入口から到達するテストを用意する。
- 各テストで 2 つを assert する。呼び出し元が観測する応答 (ステータスとエラー種別) と、その拒否が変えなかった状態。
- 各テストのソースに対応する `REQ-OAUTH2-NNN` を書き、シナリオとテストを突き合わせて読めるようにする。
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
- OAuth2 の認可モデル、トークン形式、エンドポイント構成の設計変更。

## Design

### 解消の条件

1 件の id を台帳から外してよいのは、次の 3 つがすべて成り立つときだけとする。

第 1 に、テストが production の使う入口から入り、宣言された拒否の判断を実際に通ること。

トークンエンドポイントの拒否であれば HTTP の境界から入り、use case を直接呼んで拒否の分岐だけを踏むテストは、この条件を満たさない。

配線が外れた実装を、そのテストは検出できないからである。

第 2 に、拒否が変えなかったものをテスト自身が読み直して確かめること。

第 3 に、シナリオの id をテストのソースが引用していること。

### 拒否ごとに「変わっていないこと」として何を読むか

| REQ | 宣言している拒否 | 読み直して確かめる対象 |
|---|---|---|
| REQ-OAUTH2-001 | account スコープを持てない交換を `InvalidScopeError` で拒否 | account スコープを含むアクセストークンが発行されていない |
| REQ-OAUTH2-002 | `account:read` だけの同意 revoke を `AccessDeniedError` で拒否 | 対象の同意が引き続き有効である |
| REQ-OAUTH2-004 | `admin` でも `system_admin` でもない主体のロールポリシー一覧を拒否 | 応答本文にロールポリシーが 1 件も含まれない |
| REQ-OAUTH2-007 | 未知および不正なクライアント認証を `InvalidClientError` で拒否 | トークンが発行されず、認可コードも消費されていない (正しい資格情報での再交換が成功する) |
| REQ-OAUTH2-009 | PAR 必須クライアントの直接送信を `InvalidRequestError` で拒否 | 認可コードも LoginSession も作られていない |
| REQ-OAUTH2-010 | 古い `iat` と `jti` 再使用の DPoP 証明を拒否 | 2 回目でトークンが発行されず、1 回目のトークンの状態も変わっていない |
| REQ-OAUTH2-013 | openid スコープを持たないトークンの UserInfo を拒否 | 応答に `sub` とクレームが含まれない |
| REQ-OAUTH2-015 | 認可コードの並行交換で一方を `InvalidGrantError` で拒否 | 発行されたトークンがちょうど 1 本であり、2 本目が保存されていない |
| REQ-OAUTH2-016 | redirect_uri を持たない動的登録を拒否 | クライアントが作成されていない |
| REQ-OAUTH2-017 | 私有・ループバック・リンクローカル・CGNAT へ解決されるメタデータ取得をフェイルクローズで拒否 | 当該 IP への接続が試みられておらず、メタデータも取り込まれていない |
| REQ-OAUTH2-018 | 絶対有効期限を過ぎたリフレッシュトークンのローテーションを拒否 | 新しいトークンが発行されていない |
| REQ-OAUTH2-020 | 失効したアクセストークンの UserInfo を `InvalidTokenError` で拒否 | 応答にクレームが含まれない |
| REQ-OAUTH2-021 | `offline_access` なしのリフレッシュトークン発行を行わない | 応答に `refresh_token` が無く、保存もされていない |
| REQ-OAUTH2-023 | 未登録の post_logout_redirect_uri を拒否 | 当該 URI へのリダイレクトが発生せず、セッションの終了も起きていない |
| REQ-OAUTH2-024 | `aud` 不一致および署名検証できない `id_token_hint` を拒否 | 対象のセッションが終了していない |
| REQ-OAUTH2-028 | 改ざんされた client_assertion を `InvalidClientError` で拒否 | トークンが発行されていない |
| REQ-OAUTH2-029 | 別証明書での mTLS バインドトークン利用を拒否 | 保護リソースの取得が成功していない |
| REQ-OAUTH2-034 | 他テナントの client_id、refresh、device_code を拒否 | トークンが発行されず、他テナントのレコードが読まれても変更されてもいない |
| REQ-OAUTH2-036 | 範囲外の `expires_in_days`、上限超過、非対応クライアントの追加発行を拒否 | 資格情報が増えておらず、既存の資格情報が変更されていない |
| REQ-OAUTH2-037 | 範囲外の `grace_days` と非対応クライアントのローテーションを拒否 | シークレットがローテーションされておらず、既存のシークレットで認証が引き続き成功する |
| REQ-OAUTH2-038 | 別テナントの同意の参照を拒否 | 応答に別テナントの同意が含まれない |
| REQ-OAUTH2-039 | KeyProvider 障害時の新規発行を `ServerError` で拒否 | 新しい署名が行われておらず、トークンが発行されていない |
| REQ-OAUTH2-040 | 閾値超過と共有カウンタ到達不能をフェイルクローズで拒否 | 認可コード、トークン、device_code のいずれも作られていない |

「変わっていないこと」を保存層まで読むか、後続のプロトコル操作で確かめるかは、拒否ごとに決める。

`REQ-OAUTH2-007` のように「認可コードが消費されていない」を後続の正常交換で示せる場合は、そちらを採る。

保存層を直接覗くテストは、経路が変わると壊れるうえ、素通りした実装を捕まえる力も強くならないためである。

### 却下した進め方

**named の 18 件を注記だけで閉じる案。**

分類の根拠は同じパッケージへ到達する拒否テストにエラー型名が現れることだけであり、上に挙げた誤検出の実例がその弱さを示している。

**23 件を 1 つのテーブル駆動テストにまとめる案。**

入口が `/token`、`/authorize`、`/par`、`/userinfo`、`/end_session`、動的登録、管理 API にまたがるため、共通のテーブルに載せるには入口を抽象化するほかなく、抽象化した時点で「production と同じ入口から入る」条件を失う。

**拒否の実装が無かった id を台帳に残したまま完了する案。**

拒否が宣言だけの状態は、テストが無い状態より悪い。

規模が理由で分ける場合に限り、引き取る work item を先に切り、`depends_on` でつないでから残す。

## Plan

1. 23 件のシナリオを読み、拒否ごとに入口と「変わっていないこと」を確定する。
2. 既存の拒否テストのうち、同じ入口へ到達しているものを特定し、拡張で足りるか新規が要るかを決める。
3. スコープと同意の 4 件 (001、002、013、021) から着手し、テストが RED になることを先に観測する。
4. クライアント認証と資格情報の 5 件 (007、016、028、036、037) を進める。
5. テナント境界の 2 件 (034、038) を進める。
6. トークンの一意性と失効の 3 件 (015、018、020) を進める。
7. センダー制約の 2 件 (010、029) を進める。
8. ログアウトの 2 件 (023、024) を進める。
9. フェイルクローズの 5 件 (004、009、017、039、040) を進める。
10. 防護そのものが無い id が見つかった場合は実装するか、引き取る work item を切る。
11. 対応の取れた id を台帳から削除し、この項目が持つ id が 1 件も残っていないことを確認する。

## Tasks

- [ ] T001 [Inventory] 23 件の入口と「変わっていないこと」を確定し、既存テストの流用可否を判断する。
- [ ] T002 [Acceptance] スコープと同意の拒否 (001、002、013、021) を効果まで確かめるテストを書く。
- [ ] T003 [Acceptance] クライアント認証と資格情報の拒否 (007、016、028、036、037) を効果まで確かめるテストを書く。
- [ ] T004 [Acceptance] テナント境界の拒否 (034、038) を効果まで確かめるテストを書く。
- [ ] T005 [Acceptance] トークンの一意性と失効の拒否 (015、018、020) を効果まで確かめるテストを書く。
- [ ] T006 [Acceptance] センダー制約の拒否 (010、029) を効果まで確かめるテストを書く。
- [ ] T007 [Acceptance] ログアウトの拒否 (023、024) を効果まで確かめるテストを書く。
- [ ] T008 [Acceptance] フェイルクローズの拒否 (004、009、017、039、040) を効果まで確かめるテストを書く。
- [ ] T009 [App] 拒否の実装が欠けていた経路を修正する、または引き取る work item を切って `depends_on` でつなぐ。
- [ ] T010 [Ledger] 対応の取れた id を `scenario-coverage-debt.json` から削除する。
- [ ] T011 [Verify] 防護を意図的に外すと各テストが落ちることを確かめ、検査を通す。

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

台帳は id 順に 1 エントリー 1 id なので、Context ごとの work item が並行しても衝突はエントリー単位に収まる。

各項目は自分が持つ id のエントリーだけを削除し、コメント配列と他の id には触れない。

拒否の実装が無い id が見つかった場合、それは本項目の想定より重い変更になりうる。

その場合に台帳へ残す判断は、引き取る work item を切ってからに限り、「テストが書けなかった」という理由での据え置きは認めない。
