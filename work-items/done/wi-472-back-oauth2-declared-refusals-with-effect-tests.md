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
    - docs/contexts/oauth2/scenarios.feature.md#REQ-OAUTH2-001
    - docs/contexts/oauth2/scenarios.feature.md#REQ-OAUTH2-002
    - docs/contexts/oauth2/scenarios.feature.md#REQ-OAUTH2-004
    - docs/contexts/oauth2/scenarios.feature.md#REQ-OAUTH2-007
    - docs/contexts/oauth2/scenarios.feature.md#REQ-OAUTH2-009
    - docs/contexts/oauth2/scenarios.feature.md#REQ-OAUTH2-010
    - docs/contexts/oauth2/scenarios.feature.md#REQ-OAUTH2-013
    - docs/contexts/oauth2/scenarios.feature.md#REQ-OAUTH2-015
    - docs/contexts/oauth2/scenarios.feature.md#REQ-OAUTH2-016
    - docs/contexts/oauth2/scenarios.feature.md#REQ-OAUTH2-017
    - docs/contexts/oauth2/scenarios.feature.md#REQ-OAUTH2-018
    - docs/contexts/oauth2/scenarios.feature.md#REQ-OAUTH2-020
    - docs/contexts/oauth2/scenarios.feature.md#REQ-OAUTH2-021
    - docs/contexts/oauth2/scenarios.feature.md#REQ-OAUTH2-023
    - docs/contexts/oauth2/scenarios.feature.md#REQ-OAUTH2-024
    - docs/contexts/oauth2/scenarios.feature.md#REQ-OAUTH2-028
    - docs/contexts/oauth2/scenarios.feature.md#REQ-OAUTH2-029
    - docs/contexts/oauth2/scenarios.feature.md#REQ-OAUTH2-034
    - docs/contexts/oauth2/scenarios.feature.md#REQ-OAUTH2-036
    - docs/contexts/oauth2/scenarios.feature.md#REQ-OAUTH2-037
    - docs/contexts/oauth2/scenarios.feature.md#REQ-OAUTH2-038
    - docs/contexts/oauth2/scenarios.feature.md#REQ-OAUTH2-039
    - docs/contexts/oauth2/scenarios.feature.md#REQ-OAUTH2-040
  typespec: []
  source:
    - backend/oauth2/handlers_http
    - backend/oauth2/client/cimd_http
    - backend/oauth2/client/usecases
    - backend/application/handlers_http
    - backend/authentication/handlers_http
    - backend/shared/http/support_http
  tests:
    - backend/oauth2/handlers_http
    - backend/oauth2/client/cimd_http
    - backend/application/handlers_http
    - backend/authentication/handlers_http
    - backend/shared/http/support_http
  stop_before_reading:
    - frontend
    - backend/saml
    - backend/wsfederation
affected_spec:
  - { path: docs/contexts/oauth2/scenarios.feature.md, requirement: REQ-OAUTH2-001 }
  - { path: docs/contexts/oauth2/scenarios.feature.md, requirement: REQ-OAUTH2-002 }
  - { path: docs/contexts/oauth2/scenarios.feature.md, requirement: REQ-OAUTH2-004 }
  - { path: docs/contexts/oauth2/scenarios.feature.md, requirement: REQ-OAUTH2-007 }
  - { path: docs/contexts/oauth2/scenarios.feature.md, requirement: REQ-OAUTH2-009 }
  - { path: docs/contexts/oauth2/scenarios.feature.md, requirement: REQ-OAUTH2-010 }
  - { path: docs/contexts/oauth2/scenarios.feature.md, requirement: REQ-OAUTH2-013 }
  - { path: docs/contexts/oauth2/scenarios.feature.md, requirement: REQ-OAUTH2-015 }
  - { path: docs/contexts/oauth2/scenarios.feature.md, requirement: REQ-OAUTH2-016 }
  - { path: docs/contexts/oauth2/scenarios.feature.md, requirement: REQ-OAUTH2-017 }
  - { path: docs/contexts/oauth2/scenarios.feature.md, requirement: REQ-OAUTH2-018 }
  - { path: docs/contexts/oauth2/scenarios.feature.md, requirement: REQ-OAUTH2-020 }
  - { path: docs/contexts/oauth2/scenarios.feature.md, requirement: REQ-OAUTH2-021 }
  - { path: docs/contexts/oauth2/scenarios.feature.md, requirement: REQ-OAUTH2-023 }
  - { path: docs/contexts/oauth2/scenarios.feature.md, requirement: REQ-OAUTH2-024 }
  - { path: docs/contexts/oauth2/scenarios.feature.md, requirement: REQ-OAUTH2-028 }
  - { path: docs/contexts/oauth2/scenarios.feature.md, requirement: REQ-OAUTH2-029 }
  - { path: docs/contexts/oauth2/scenarios.feature.md, requirement: REQ-OAUTH2-034 }
  - { path: docs/contexts/oauth2/scenarios.feature.md, requirement: REQ-OAUTH2-036 }
  - { path: docs/contexts/oauth2/scenarios.feature.md, requirement: REQ-OAUTH2-037 }
  - { path: docs/contexts/oauth2/scenarios.feature.md, requirement: REQ-OAUTH2-038 }
  - { path: docs/contexts/oauth2/scenarios.feature.md, requirement: REQ-OAUTH2-039 }
  - { path: docs/contexts/oauth2/scenarios.feature.md, requirement: REQ-OAUTH2-040 }
---

# OAuth2 が宣言する未検証の拒否に効果まで確かめるテストを与え、台帳から外す

## Motivation

`tools/check/example-coverage-debt.json` は、どのテストからも id を引用されていない規範 id を保持する、縮むだけの台帳である。

対象は、この製品がプロトコルとして外部に約束している防護そのもの — クライアント認証、スコープ、テナント境界、センダー制約、フェイルクローズ — であり、OAuth2 は全 Context で最大の未検証量を持つ。

台帳が言えるのは「id を引用したテストが無い」ことだけで、「拒否を確かめるテストが無い」こととは違う。

`mise run report-coverage-debt` はこの差を named / nearby / none に分けるが、named は「同じエラー型名を含む拒否テストが同じパッケージへ到達する」ことしか示さない。

実際、この分類で `REQ-APPLICATION-005` が指すのは `backend/provisioning/handlers_http/admin_delivery_list_pagination_test.go` の `TestAdminDeliveryListRejectsInvalidCursor` であり、SAML 署名証明書の検証とは何の関係もない。

したがって named を「注記が抜けているだけ」と扱うことはできず、1 件ずつ読んで確かめるほかない。

そして注記だけを足す解消は、この項目では認めない。

[[wi-390-security-control-test-standard-and-gate]] が見つけた欠陥は、テストがあり、カバレッジもあり、それでも防護が素通りしていた形だった。

拒否の応答だけを確かめるテストに id を書き足せば、検査は通り、台帳は縮み、防護が効いていることは何ひとつ確かめられないまま「検証済み」の見た目だけが残る。

台帳が縮むこと自体には価値がなく、価値があるのは拒否が実際に副作用を止めていると分かることである。

## Scope

- 下記「この項目が持つ id」の 40 件それぞれについて、宣言された拒否に production と同じ入口から到達するテストを用意する。
- 各テストで 2 つを assert する。呼び出し元が観測する応答 (ステータスとエラー種別) と、その拒否が変えなかった状態。
- 各テストのソースに対応する `EX-OAUTH2-NNN-MM` を書き、具体例とテストを突き合わせて読めるようにする。
- 対応が取れた id を `tools/check/example-coverage-debt.json` から削除する。
- 拒否を実装が持っていないと判明した場合、その防護をこの項目で実装する。
- 防護の追加に新しい設計判断が要る場合は、その id だけを台帳に残し、引き取る work item を切って `depends_on` でつなぐ。
- 完了時点で、この項目が持つ id が台帳から 1 件残らず消えていることを確認する。

## Out of Scope

- 他 Context の拒否。
  Context ごとに別の work item が持つ。
- 23 の Rule が持つ正常系の具体例。
  この項目は拒否だけを引き取る。同じ Rule の `通常経路` が台帳に残っていても、それは正常系の網羅を持つ別の項目の対象である。
- 同じ台帳に載る、この項目が持たない id。
  規則は同じだが、引き取る範囲は下記の 40 件に限る。
- R1、R2、R4 の検査そのものの変更。
  [[wi-390-security-control-test-standard-and-gate]] と [[wi-391-refusal-declaration-floor-and-reinventory]] が持つ。
- 既存テストへ id を注記するだけで台帳から外すこと。
  この項目ではこれを解消として扱わない。
- 具体例の Then ステップへ「変わらなかったもの」を規範として書き足すこと。
  下記「拒否ごとに何を読み直すか」を参照。
- OAuth2 の認可モデル、トークン形式、エンドポイント構成の設計変更。

## Design

### 台帳の単位が Rule から具体例へ移ったこと

この項目は `REQ-OAUTH2-NNN` 23 件を持つものとして起票された。

その後 [[wi-491-track-tests-per-example]] が台帳を Rule 単位から具体例単位へ移し、`scenarios.feature.md` は Markdown with Gherkin になった。

台帳の現在の単位は `EX-OAUTH2-NNN-MM` であり、Rule の id はもう台帳に載らない。

Rule の採番は移行を越えて保たれているため、起票時に挙げた 23 件は今も同じ拒否を指す。

そこで持ち分を、その 23 Rule が宣言する拒否の具体例へ読み替える。

正常系の具体例は持たない。

Rule 007、015、018、020、028、038、039 は Rule 自体が拒否の宣言であり、その `通常経路` が拒否そのものなので、これらの `-01` は持ち分に入る。

### この項目が持つ id

40 件。

| Rule | 具体例 | 宣言している拒否 | 読み直して確かめる対象 |
|---|---|---|---|
| REQ-OAUTH2-001 | EX-OAUTH2-001-02 | subject を持たない交換の account スコープ要求を `InvalidScopeError` で拒否 | 応答にアクセストークンが無い |
| REQ-OAUTH2-001 | EX-OAUTH2-001-03 | 許可スコープまたは同意に account が無い | 発行されたトークンの `scope` に account スコープが含まれない |
| REQ-OAUTH2-002 | EX-OAUTH2-002-02 | `account:read` だけの同意 revoke を拒否 | 対象の同意が引き続き `Granted` である |
| REQ-OAUTH2-002 | EX-OAUTH2-002-03 | トークンの user_id が操作対象と一致しない | 他人の同意が `Granted` のまま変わらない |
| REQ-OAUTH2-004 | EX-OAUTH2-004-02 | `admin` でも `system_admin` でもない主体のロールポリシー一覧を拒否 | レスポンスボディにロールポリシーが 1 件も含まれない |
| REQ-OAUTH2-007 | EX-OAUTH2-007-01 | 誤った `client_secret` を `InvalidClientError` で拒否 | 認可コードが消費されておらず、正しい資格情報での再交換が成功する |
| REQ-OAUTH2-007 | EX-OAUTH2-007-02 | 未知の `client_id` を `InvalidClientError` で拒否 | 認可コードが消費されておらず、正しいクライアントでの交換が成功する |
| REQ-OAUTH2-009 | EX-OAUTH2-009-02 | PAR 必須クライアントの直接送信を `InvalidRequestError` で拒否 | 認可コードが発行されておらず、リダイレクトも起きない |
| REQ-OAUTH2-010 | EX-OAUTH2-010-02 | 古い `iat` の DPoP 証明を拒否 | 応答にアクセストークンが無く、認可コードが消費されていない |
| REQ-OAUTH2-010 | EX-OAUTH2-010-03 | `jti` 再使用の DPoP 証明を拒否 | 2 回目でトークンが発行されず、1 回目のトークンは有効なまま |
| REQ-OAUTH2-013 | EX-OAUTH2-013-02 | openid スコープを持たないトークンの UserInfo を拒否 | 応答に `sub` とクレームが含まれない |
| REQ-OAUTH2-015 | EX-OAUTH2-015-01 | 認可コードの並行交換で一方を `InvalidGrantError` で拒否 | 成功はちょうど 1 回で、発行されたアクセストークンが 1 本だけである |
| REQ-OAUTH2-016 | EX-OAUTH2-016-02 | `redirect_uri` を持たない動的登録を拒否 | クライアントが作成されていない |
| REQ-OAUTH2-017 | EX-OAUTH2-017-02 | 私有・ループバック・リンクローカル・CGNAT へ解決される取得をフェイルクローズで拒否 | 当該 IP への接続が試みられておらず、メタデータも取り込まれていない |
| REQ-OAUTH2-018 | EX-OAUTH2-018-01 | 絶対有効期限を過ぎたリフレッシュトークンのローテーションを拒否 | 新しいトークンが発行されず、保存されたトークンも増えていない |
| REQ-OAUTH2-020 | EX-OAUTH2-020-01 | 失効したアクセストークンの UserInfo を `InvalidTokenError` で拒否 | 応答に `sub` とクレームが含まれない |
| REQ-OAUTH2-021 | EX-OAUTH2-021-02 | `offline_access` なしのリフレッシュトークン発行を行わない | 応答に `refresh_token` が無く、保存もされていない |
| REQ-OAUTH2-023 | EX-OAUTH2-023-02 | 未登録の `post_logout_redirect_uri` を拒否 | 当該 URI へのリダイレクトが発生しない |
| REQ-OAUTH2-024 | EX-OAUTH2-024-02 | `aud` 不一致の `id_token_hint` を拒否 | 対象のセッションが終了しておらず、そのリフレッシュトークンも `Active` のまま |
| REQ-OAUTH2-024 | EX-OAUTH2-024-03 | 署名検証できない `id_token_hint` を拒否 | 対象のセッションが終了しておらず、そのリフレッシュトークンも `Active` のまま |
| REQ-OAUTH2-028 | EX-OAUTH2-028-01 | 改ざんされた `client_assertion` を `InvalidClientError` で拒否 | 応答にトークンが無く、認可コードが消費されていない |
| REQ-OAUTH2-029 | EX-OAUTH2-029-02 | 別証明書での mTLS バインドトークン利用を拒否 | 保護リソースの本文が返らない |
| REQ-OAUTH2-034 | EX-OAUTH2-034-01 | 他テナントの認可コード交換を `InvalidGrantError` で拒否 | 応答にトークンが無く、コードは元のテナントで有効なまま |
| REQ-OAUTH2-034 | EX-OAUTH2-034-02 | 他テナントの `client_id` を `InvalidClientError` で拒否 | 応答にトークンが無く、当該クライアントは元のテナントで認証できる |
| REQ-OAUTH2-034 | EX-OAUTH2-034-03 | 他テナントのリフレッシュトークンを `InvalidGrantError` で拒否 | 応答にトークンが無く、当該リフレッシュトークンは元のテナントで使える |
| REQ-OAUTH2-034 | EX-OAUTH2-034-04 | 他テナントの `device_code` を `InvalidGrantError` で拒否 | 応答にトークンが無く、当該 `device_code` は元のテナントで交換できる |
| REQ-OAUTH2-036 | EX-OAUTH2-036-02 | 範囲外の `expires_in_days` を拒否 | 資格情報が増えていない |
| REQ-OAUTH2-036 | EX-OAUTH2-036-03 | `Active` の上限超過を拒否 | 資格情報が増えておらず、既存 2 件の `expires_at` と状態が変わっていない |
| REQ-OAUTH2-036 | EX-OAUTH2-036-04 | 非対応クライアントの追加発行を拒否 | 資格情報が増えていない |
| REQ-OAUTH2-036 | EX-OAUTH2-036-05 | 別クライアントまたは存在しない `credential_id` の失効を拒否 | 別クライアントの資格情報が `Active` のまま |
| REQ-OAUTH2-037 | EX-OAUTH2-037-02 | 範囲外の `grace_days` を拒否 | シークレットがローテーションされておらず、既存のシークレットで認証が引き続き成功する |
| REQ-OAUTH2-037 | EX-OAUTH2-037-03 | 非対応クライアントのローテーションを拒否 | シークレットがローテーションされていない |
| REQ-OAUTH2-038 | EX-OAUTH2-038-01 | 別テナントの同意の参照を拒否 | 応答に別テナントの同意が含まれない |
| REQ-OAUTH2-039 | EX-OAUTH2-039-01 | KeyProvider 障害時の新規発行を `ServerError` で拒否 | 新しい署名が行われておらず、応答にトークンが無い |
| REQ-OAUTH2-040 | EX-OAUTH2-040-01 | 閾値超過を `RateLimitedError` で拒否 | `Retry-After` を伴い、後続の副作用が起きていない |
| REQ-OAUTH2-040 | EX-OAUTH2-040-02 | `/token` の閾値超過を拒否 | トークンが発行されていない |
| REQ-OAUTH2-040 | EX-OAUTH2-040-03 | `/authorize` と `/par` の閾値超過を拒否 | 認可コードも PAR レコードも作られていない |
| REQ-OAUTH2-040 | EX-OAUTH2-040-04 | `/device_authorization` の閾値超過を拒否 | `device_code` が作られていない |
| REQ-OAUTH2-040 | EX-OAUTH2-040-05 | `/bc-authorize` の閾値超過を拒否 | 承認要求が作られていない |
| REQ-OAUTH2-040 | EX-OAUTH2-040-06 | 共有カウンタ到達不能をフェイルクローズで拒否 | 上と同じ副作用がいずれも起きていない |

### 解消の条件

1 件の id を台帳から外してよいのは、次の 3 つがすべて成り立つときだけとする。

第 1 に、テストが production の使う入口から入り、宣言された拒否の判断を実際に通ること。

トークンエンドポイントの拒否であれば HTTP の境界から入り、use case を直接呼んで拒否の分岐だけを踏むテストは、この条件を満たさない。

配線が外れた実装を、そのテストは検出できないからである。

第 2 に、拒否が変えなかったものをテスト自身が読み直して確かめること。

第 3 に、具体例の id をテストのソースが引用していること。

「変わっていないこと」を保存層まで読むか、後続のプロトコル操作で確かめるかは、拒否ごとに決める。

`EX-OAUTH2-007-01` のように「認可コードが消費されていない」を後続の正常交換で示せる場合は、そちらを採る。

保存層を直接覗くテストは、経路が変わると壊れるうえ、素通りした実装を捕まえる力も強くならないためである。

### 拒否ごとに何を読み直すかを仕様へ書き足さない理由

上の表は各拒否の 2 つ目の assert を決めるが、これを具体例の Then ステップへ書き足すことはしない。

「拒否は副作用を止めたことまで assert する」は
[docs/development/specification-first-workflow.md](../../docs/development/specification-first-workflow.md) の検証ラダーが全 Context に課している検証の規範であって、OAuth2 が外部に約束する振る舞いの追加ではない。

同じ内容を 40 の具体例へ書き写せば、規範の記述が 1 か所から 41 か所へ増え、検証方法の変更が仕様の変更に見えるようになる。

拒否が止めるべき副作用そのものを具体例が言っていない箇所があれば、それは仕様の欠落なので、この項目ではなく仕様変更として直す。

### 却下した進め方

**named を注記だけで閉じる案。**

分類の根拠は同じパッケージへ到達する拒否テストにエラー型名が現れることだけであり、上に挙げた誤検出の実例がその弱さを示している。

**40 件を 1 つのテーブル駆動テストにまとめる案。**

入口が `/token`、`/authorize`、`/par`、`/userinfo`、`/end_session`、`/device_authorization`、`/bc-authorize`、動的登録、管理 API、Application API にまたがるため、共通のテーブルに載せるには入口を抽象化するほかなく、抽象化した時点で「production と同じ入口から入る」条件を失う。

**拒否の実装が無かった id を台帳に残したまま完了する案。**

拒否が宣言だけの状態は、テストが無い状態より悪い。

規模が理由で分ける場合に限り、引き取る work item を先に切り、`depends_on` でつないでから残す。

## Plan

1. 40 件の入口と「変わっていないこと」を確定し、既存の拒否テストのうち同じ入口へ到達しているものを特定する。 (完了、上表)
2. スコープと同意の 6 件 (001-02、001-03、002-02、002-03、013-02、021-02) から着手する。
3. クライアント認証と資格情報の 9 件 (007-01、007-02、016-02、028-01、036-02..05、037-02、037-03) を進める。
4. テナント境界の 5 件 (034-01..04、038-01) を進める。
5. トークンの一意性と失効の 3 件 (015-01、018-01、020-01) を進める。
6. センダー制約の 3 件 (010-02、010-03、029-02) を進める。
7. ログアウトの 3 件 (023-02、024-02、024-03) を進める。
8. フェイルクローズの 11 件 (004-02、009-02、017-02、039-01、040-01..06) を進める。
9. 各テストについて防護を外すと落ちることを確かめる。落ちないものは入口が誤っているので入口からやり直す。
10. 対応の取れた id を台帳から削除し、この項目が持つ id が 1 件も残っていないことを確認する。

## Tasks

- [x] T001 [Inventory] 40 件の入口と「変わっていないこと」を確定し、既存テストの流用可否を判断する。
- [x] T002 [Acceptance] スコープと同意の拒否 6 件を効果まで確かめる。
- [x] T003 [Acceptance] クライアント認証と資格情報の拒否 9 件を効果まで確かめる。
- [x] T004 [Acceptance] テナント境界の拒否 5 件を効果まで確かめる。
- [x] T005 [Acceptance] トークンの一意性と失効の拒否 3 件を効果まで確かめる。
- [x] T006 [Acceptance] センダー制約の拒否 3 件を効果まで確かめる。
- [x] T007 [Acceptance] ログアウトの拒否 3 件を効果まで確かめる。
- [x] T008 [Acceptance] フェイルクローズの拒否 11 件を効果まで確かめる。
- [x] T009 [App] 拒否の実装が欠けていた経路を修正する、または引き取る work item を切って `depends_on` でつなぐ。
- [x] T010 [Ledger] 対応の取れた id を `example-coverage-debt.json` から削除する。
- [x] T011 [Verify] 防護を意図的に外すと各テストが落ちることを確かめ、検査を通す。

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
見つからなければ `N/A` とし、代替として上記の障害注入を記録する。

## Risk Notes

拒否のテストは、防護に到達する前の入力検証で落ちていても同じ応答を返すため、書いた本人にも成立して見える。

これを避けるため、各テストについて防護を外した状態でテストが落ちることを確かめ、落ちなかったものは入口が誤っていると判断してやり直す。

台帳は id 順に 1 エントリー 1 id なので、Context ごとの work item が並行しても衝突はエントリー単位に収まる。

各項目は自分が持つ id のエントリーだけを削除し、コメント配列と他の id には触れない。

拒否の実装が無い id が見つかった場合、それは本項目の想定より重い変更になりうる。

その場合に台帳へ残す判断は、引き取る work item を切ってからに限り、「テストが書けなかった」という理由での据え置きは認めない。

## Completion

- **Completed At**: 2026-09-05
- **Summary**:
  `mise run spec-diff` は `no normative specification change against main` を返す。
  規範の宣言は 1 件も変わっていない。変わったのは、宣言済みの拒否 40 件が
  「応答だけ確かめられている」状態から「副作用を止めたことまで確かめられている」状態へ
  移ったことと、その 40 件が網羅台帳から消えたことである。
  `check-spec` の網羅は「テストが名指しする id」19 件から 59 件へ増え、
  `example-coverage-debt.json` は 711 件から 671 件へ、OAuth2 分は 145 件から 105 件へ減った。
  実装の変更は 1 か所だけで、`support_http.CheckRateLimit` が共有カウンターへ到達できないときに
  エラーを伝播して 500 を返していたのを、宣言どおり `RateLimitedError` の 429 で閉じるようにした。

- **Acceptance RED Evidence**:
  - **Test**: `TestTokenRateLimitRefusalIssuesNothingAndLeavesCodeUnredeemed`
    (`backend/oauth2/handlers_http/refusal_effects_test.go`)。
    ほかの 39 例も同じ形で観測した。対象の要求は `affected_spec` の 23 件すべてに及ぶ。
  - **Requirement**: REQ-OAUTH2-040
  - **Observed Failure**: レート制限が「429 を書いてから処理を続ける」形では、
    ステータスと `Retry-After` の検査は通ったまま、認可コードが消費されていないことの検査だけが落ちた。
    障害注入のたびに、対応するテストだけが落ちた。落ちなかったものは無い。
  - **Detection Reason**: どのテストも応答と「拒否が変えなかった状態」の 2 つを assert する。
    応答だけを読む実装との差は障害注入で現れた。とくに UserInfo と account 同意の 3 例では、
    拒否を書いたあとに処理を続ける形 (wi-390 の欠陥そのもの) を注入したところ、
    ステータスとエラー種別の検査は通ったまま、効果側の検査だけが落ちた。

- **Unit RED Evidence**:
  - **Test**: `TestCheckRateLimitStoreErrorFailsClosedAsRateLimited`
    (`backend/shared/http/support_http/rate_limit_test.go`)
  - **Requirement**: REQ-OAUTH2-040
  - **Observed Failure**: `a successful 429 write should not itself error: store unreachable`。
    実装は共有カウンターの障害をそのまま返し、HTTP 層で 500 になっていた。
    `/token` は 400、401、422、429 しか宣言しておらず、500 は契約に無い。
  - **Detection Reason**: 429、`Retry-After`、`blocked=true` の 3 つを見る。
    エラーを伝播する実装はどれも通らず、`RateLimitedError` を書いて停止する実装だけが通る。

- **Change-Resistance Results**:
  防護を 1 つずつ外し、対応するテストだけが落ちることを確かめた。
  結果は 3 つに分かれ、そのうち 2 つはこの項目の想定より重要だった。

  1 つ目は、外した防護に対して素直に落ちたもの。
  PAR 必須 (`009-02`)、未宣言スコープ (`001-03`)、account スコープ (`001-02`)、
  DPoP の `iat` と `jti` (`010-02`、`010-03`)、認可コードの比較交換 (`015-01`)、
  device_code のテナント (`034-04`)、`redirect_uris` 必須 (`016-02`)、
  非公開 IP の判定 (`017-02`)、`expires_in_days` と `grace_days` の範囲 (`036-02`、`037-02`)、
  対応クライアントの判定 (`036-04`、`037-03`)、`Active` の上限 (`036-03`)、
  `credential_id` の所有者判定 (`036-05`)、`post_logout_redirect_uri` の登録判定 (`023-02`)、
  `id_token_hint` の `aud` と署名 (`024-02`、`024-03`)、
  ロールポリシーの `admin` 判定 (`004-02`)、同意管理のテナント (`038-01`)、
  account スコープの検査 (`002-02`)。

  2 つ目は、レート制限の 6 件 (`040-01`..`040-06`)。
  「429 を書いてから処理を続ける」形を注入すると、`/token`、`/authorize`、`/par`、
  `/device_authorization`、`/bc-authorize` の 5 テストが同時に落ちた。
  ステータスの検査は通ったまま、レコードが作られていないことの検査だけが落ちる。
  これは wi-390 の欠陥と同じ形である。

  3 つ目は、防護が二重になっていて 1 本外しただけでは落ちなかったもの。
  これは想定していなかったので、外す範囲を広げて確かめ直した。

  - テナント境界の認可コードとリフレッシュトークン (`034-01`、`034-03`) は、
    レコードのテナント判定と主体のテナント判定の 2 本で守られていた。両方を外すと落ちた。
  - テナント境界の `client_id` (`034-02`) と未知の `client_id` (`007-02`) は、
    `authenticateTokenClient` の照会と `handleToken` の照会の 2 本で守られていた。
    両方を外すと `acme` にしか存在しないクライアントへ `default` の issuer でトークンが発行され、
    テストはその応答で落ちた。
  - リフレッシュトークンの絶対有効期限 (`018-01`) は、use case の判定と
    認可ポリシーの `token_within_absolute_ttl` の 2 本で守られていた。両方を外すと落ちた。
  - `002-03` のテナント境界は、テナントごとの署名鍵、`iss` の照合、主体のテナントの 3 本で
    守られており、1 本ずつでは落ちない。代わりに「拒否を書いてから撤回も続ける」形を注入し、
    `002-02` と `002-03` の両方が落ちることを確かめた。

  `039-01` は、KeyProvider が到達不能なら署名そのものが成立しないため、防護を外す形が取れない。
  代わりに `AccessTokenIssued` の発行を署名の成否判定より前へ動かしたところ、テストが落ちた。

  二重の防護は弱点ではないが、「1 本外して落ちること」を変更耐性の基準にすると、
  この 4 例はいずれも「テストが効いていない」と誤読されうる。
  実際には 4 例とも、防護一式を外せば落ちる。

- **Verification Results**:
  - `mise run test-go-race` - passed
  - `mise run check-spec` - passed (711 例中 59 id をテストが名指し。実施前は 19)
  - `mise run check-security-controls` - passed
  - `mise run report-coverage-debt` - OAuth2 は 145 件から 105 件へ
  - `mise run check-work-items` - passed
  - `mise run check-ids` - passed
  - `mise run verify` - passed
