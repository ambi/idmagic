---
depends_on: []
status: completed
authors: [tn]
risk: low
reversibility: irreversible
created_at: 2026-08-23
priority: p2
change_kind: docs
evidence_policy: risk-based-v2
initial_context:
  specification:
    - docs/contexts/provisioning/README.md
    - docs/contexts/provisioning/scenarios.md#REQ-PROVISIONING-002
    - docs/contexts/provisioning/scenarios.md#REQ-PROVISIONING-007
    - docs/contexts/provisioning/scenarios.md#REQ-PROVISIONING-009
    - docs/contexts/provisioning/internals.md
    - docs/contexts/provisioning/decisions.md
    - docs/contexts/sourcing/standards.md
  typespec:
    - IdMagic.Contract.ProvisioningCapabilities
    - IdMagic.Contract.ProvisioningAuthMethod
    - IdMagic.Contract.AttributeMappingRule
  source:
    - backend/provisioning/client_scim
    - backend/provisioning/usecases
    - backend/provisioning/source_idmanagement
  tests:
    - backend/provisioning/client_scim
    - backend/provisioning/usecases
  stop_before_reading:
    - backend/sourcing
    - frontend
    - backend/provisioning/db_postgres
affected_spec:
  - { path: docs/contexts/provisioning/standards.md, requirement: RFC7643-OUT-CORE-RESOURCES }
  - { path: docs/contexts/provisioning/standards.md, requirement: RFC7643-OUT-EXTERNAL-ID }
  - { path: docs/contexts/provisioning/standards.md, requirement: RFC7643-OUT-SCHEMA-EXTENSIONS }
  - { path: docs/contexts/provisioning/standards.md, requirement: RFC7644-OUT-RESOURCE-OPERATIONS }
  - { path: docs/contexts/provisioning/standards.md, requirement: RFC7644-OUT-PATCH }
  - { path: docs/contexts/provisioning/standards.md, requirement: RFC7644-OUT-DISCOVERY }
  - { path: docs/contexts/provisioning/standards.md, requirement: RFC7644-OUT-FILTERING }
  - { path: docs/contexts/provisioning/standards.md, requirement: RFC7644-OUT-ERROR-RESPONSE }
  - { path: docs/contexts/provisioning/standards.md, requirement: RFC7644-OUT-AUTHENTICATION }
  - { path: docs/contexts/provisioning/standards.md, requirement: RFC7644-OUT-BULK }
  - { path: docs/contexts/provisioning/standards.md, requirement: RFC7644-OUT-SORT }
  - { path: docs/contexts/provisioning/standards.md, requirement: RFC7644-OUT-ETAG }
---

# Provisioning に `standards.md` を置き、外向き SCIM の準拠範囲を宣言する

## Motivation

IdMagic は SCIM 2.0 を両方向で扱う。内向き（`Sourcing`、SCIM サーバー）と外向き（`Provisioning`、SCIM クライアント）である。**準拠範囲を宣言しているのは内向きだけである。**

`docs/contexts/sourcing/standards.md` は RFC 7643 と RFC 7644 に 8 行の `Adoption` / `Strength` / `Statement` を与えている。`docs/contexts/provisioning/` に `standards.md` は無く、`README.md` の索引にも行が無い。

宣言が無いだけで、外向きも同じ RFC が形を定める部分を実装している。`provisioning/internals.md` 自身がそう書いている。

> 外向き側には、フィルター文字列の組み立てと、対応付けに基づく広い属性集合 (`externalId`、Enterprise 拡張) の直列化が必要になる。重なるのは Discovery の構造体と RFC が定めるスキーマ URN 程度

フィルター構文、`externalId`、Enterprise 拡張、Discovery、スキーマ URN——いずれも RFC 7643 / 7644 が定める。**どこまで従い、どこを送らないかは、いまコードにしか無い。**

これが効くのは連携先を増やすときである。下流 SaaS の SCIM 実装には差があり、PATCH を受けない相手、Enterprise 拡張を無視する相手、`externalId` で相関しない相手が実在する。**「IdMagic は PATCH を送るのか PUT を送るのか」「Enterprise 拡張を常に送るのか」は連携の可否を決める問いだが、答えは仕様のどこにも書かれていない。**

[SPECIFICATION_FORMAT.md](../../SPECIFICATION_FORMAT.md) §5 と [DOCUMENTATION_GUIDE.md](../../DOCUMENTATION_GUIDE.md) §5.3 は、規範の各行に証拠となるテストを要求し、`excluded` の行には否定テストを要求する。**宣言が無い規範には、この要求が一切かからない。** 送らないと決めたものが送られるようになっても、それを嘘だと言う記述が存在しない。

## Scope

- `docs/contexts/provisioning/standards.md` を作り、外向き SCIM クライアントとしての RFC 7643 / RFC 7644 の準拠範囲を宣言する。
- 少なくとも次を `Adoption` / `Strength` / `Statement` の行として決める。
  - User / Group リソースの作成・置換・削除。
  - 部分更新に PATCH を使うか PUT を使うか、接続設定で選べるのか。
  - Enterprise 拡張（`employeeNumber`、`department`、`manager`）を送るかどうか。
  - `externalId` による相関。
  - 連携先の `ServiceProviderConfig` / `ResourceTypes` / `Schemas` の探索と、探索結果に従うのか無視するのか。
  - フィルターを使う場面と、組み立てる構文の範囲。
  - 連携先が返す SCIM エラーレスポンスの解釈（どの状態を再試行し、どれを隔離するか）。
- `docs/contexts/provisioning/README.md` の索引に行を足す。
- 各行に対応するテストを確かめ、無い行にはテストを足す。

## Out of Scope

- `Sourcing` の `standards.md` の見直し。内向きの宣言は既にあり、この変更では触れない。
- 新しいプロトコルの連携先（`internals.md` が将来として挙げる `entraid`、`googledir`）。それらの規範は、その機能単位を作る変更が持つ。
- 実装の変更。いま送っているものを宣言することに限る。**宣言しようとして「これは仕様として妥当でない」と分かった場合は、欠陥として個別の work item へ切り出す。**
- 他の `standards.md` を持たない Context（audit、jobs、tenancy など）の点検。それらが外部規範に準拠しているかは別の問いであり、Provisioning のように「同じ RFC を宣言している片割れが隣にいる」という明確な非対称は無い。

## Design

### 1. 送出側での `Adoption` の読みと、行ごとに要る証拠

DOCUMENTATION_GUIDE §3.3 の 4 値は提供者側の語彙なので、送出側では次のように読む。併せて、その読みから決まる証拠の形を置く。証拠の形まで決めないと、`partial` の行が「一部やっている」としか言わない検証不能な行になる。

| 値 | 送出側での読み | 要求する証拠 |
|---|---|---|
| `required` | 条件を問わず常に送る、または常に行う | 送っていることを確かめるテスト |
| `optional` | 条件が満たされたときだけ送る | 条件が真のときに送ること、偽のときに送らないこと |
| `partial` | 定めた範囲の中だけを送る | 範囲の中を送ること、範囲の外を組み立てないこと |
| `excluded` | 送らない | 送っていないことを確かめる否定テスト |

`optional` の行の `Statement` には、**切り替えの条件と、条件が偽のときに代わりに何を送るか**を必ず書く。書かなければ「有効時と無効時」の 2 本を書き分けられない。この読みが効いた行は `RFC7644-OUT-PATCH` 1 件で、条件は下流の `patch.supported`、偽のときの代替は PUT である。

### 2. ID 空間

`RFC<番号>-OUT-<名前>` とする。`Sourcing` の既存 ID は改名しない。

規範 ID をテストから名指しする運用では、ID はリポジトリ全体で 1 つの保証を指していなければ検索の意味が無い。`RFC7643-CORE-RESOURCES` は既に内向きサーバーの保証を指しているので、外向きクライアントの保証には別の文字列が要る。先頭の `RFC7643` / `RFC7644` を残すのは、ID がどの標準の話かを ID 自身に言わせるためであり、`SCIM-CLIENT-*` のような独自接頭辞ではそれが失われる。

非対称——内向きが無印、外向きが `-OUT-`——は意図的に残す。参照済みの規範 ID の改名は、`backend/sourcing` のテストとコメント、および完了済み work item の参照を巻き込むうえ、`Sourcing` の `standards.md` は本 work item の Out of Scope である。無印の側が「IdMagic 自身が提供する面」、印の付いた側が「IdMagic が他所の実装に対して送る面」という読みで区別は成り立つ。SCIM を話す 3 つ目の Context が現れたら、そのときに 3 者の命名をまとめて見直す。

### 3. 証拠テストの過不足

名指しの追加だけで済んだ行と、テストを書いた行は次のとおり。既存テストは `backend/provisioning/client_scim` にあり、規範 ID をどこにも書いていなかった。名指しは `Sourcing` の先例に合わせ、テスト関数の直前のコメントに ID を置く形にする（テスト名は Go の識別子で、日本語の規範文と噛み合わない）。

| 行 | 既存テスト | 追加したもの |
|---|---|---|
| `RFC7643-OUT-CORE-RESOURCES` | `TestBuildResource_*` | 名指しのみ |
| `RFC7643-OUT-EXTERNAL-ID` | `TestBuildResource_CreateOnlySkippedOnUpdate` | 既定の対応付けが `externalId` を作成時だけ送ることのテスト |
| `RFC7643-OUT-SCHEMA-EXTENSIONS` | 無し | 否定テスト（拡張 URN で修飾したパスを与えても本文に拡張が現れない） |
| `RFC7644-OUT-RESOURCE-OPERATIONS` | `TestClient_CreateUser_*`、`TestClient_DeleteUser_*` | メディアタイプの表明 |
| `RFC7644-OUT-PATCH` | `TestClient_UpdateUser_UsesPatchWhenSupported`、同 `_FallsBackToPutWhenPatchUnsupported` | 名指しのみ |
| `RFC7644-OUT-DISCOVERY` | `TestClient_Discover_ParsesCapabilities` | 否定側（`/ResourceTypes` と `/Schemas` を要求しない） |
| `RFC7644-OUT-FILTERING` | `TestClient_SearchUserByAttribute_*` | 範囲外（論理演算子を組み立てない）の表明 |
| `RFC7644-OUT-ERROR-RESPONSE` | `TestClient_CreateUser_ReturnsConflictErrorOn409` ほか | 2xx 以外の未知の状態を再試行しないことのテスト |
| `RFC7644-OUT-AUTHENTICATION` | 無し | 資格情報を得るための追加要求を送らないことのテスト |
| `RFC7644-OUT-BULK` | 無し | 否定テスト |
| `RFC7644-OUT-SORT` | 無し | 否定テスト |
| `RFC7644-OUT-ETAG` | 無し | 否定テスト |

### 4. 宣言しなかったもの

実装を読んで宣言できないと判断した 2 件は、行を作らずに欠陥として切り出す。**`Statement` は製品が何をするかを書く欄であって、直っていない実装を規範に昇格させる欄ではない。**

- **リソース本文の `schemas` 属性**。`BuildResource` は属性対応付けの結果だけで本文を組み立てるため、`POST` / `PUT` の本文にも PATCH の `value` にも `schemas` が現れない。RFC 7643 §3 はリソース表現に `schemas` を要求しており、`excluded` として宣言すれば非準拠を方針として書くことになる。`RFC7643-OUT-CORE-RESOURCES` を `partial` とし、本文に入るものを列挙するに留めた。→ [[wi-439-outbound-scim-resource-body-omits-schemas]]
- **Group リソースの送出**。`glossary.md` の Push Groups、`states.md` の `GroupPushed` / `GroupMembershipPushed`、`feature_flags.push_groups` はいずれも Group の送出を能力として宣言しているが、Group の変更を捕捉する経路が無く、属性解決も User 型しか扱わないため、Group の配信は生まれない。`excluded` と書けば同じ Context の他の正本文書と矛盾し、`partial` と書けば動かないものを動くと言うことになる。行を作らず、`RFC7643-OUT-CORE-RESOURCES` の `Statement` を User リソースに限定して、沈黙が見えるようにした。→ [[wi-441-push-groups-produces-no-delivery]]

`RFC7644-OUT-AUTHENTICATION` を `partial` としたのも同じ判断の系である。接続は `oauth2_client_credentials` を受け付けるのにトークンエンドポイントを呼ばないので、「認証方式は `Authorization: Bearer` の 1 つだけ」という真の宣言を置き、受け付ける値との食い違いは → [[wi-440-outbound-scim-oauth2-client-credentials-unimplemented]] が持つ。

## Plan

- まず外向きの実装が実際に何を送っているかを読み、宣言の草案を作る。**宣言を先に書いて実装を後から確かめる順序にしない。** 願望が仕様になる。
- 草案ができた時点で 1 と 2 を確定する。ID の形が決まらないとテストの名指しが書けない。
- 行を入れるたびに対応するテストを確かめる。証拠の無い行を先にまとめて入れない。
- 「送っているが仕様として妥当でない」が出たら切り出して先へ進む。

## Tasks

- [x] T001 [Spec] 外向き SCIM の実装を読み、いま何を送り何を送っていないかを列挙する。
- [x] T002 [Design] `Adoption` の読み替え、ID 空間の分け方、証拠テストの過不足を確定し `## Design` に記録する。
- [x] T003 [Spec] `docs/contexts/provisioning/standards.md` を作る。
- [x] T004 [Spec] `docs/contexts/provisioning/README.md` の索引に行を足す。
- [x] T005 [Test] 各行の証拠テストを確かめ、名指しを足す。無い行にはテストを書く。`backend/provisioning/client_scim`、`backend/provisioning/usecases`。
- [x] T006 [Test] `excluded` の行に否定テストを置く。`RFC7643-OUT-SCHEMA-EXTENSIONS`、`RFC7644-OUT-BULK`、`RFC7644-OUT-SORT`、`RFC7644-OUT-ETAG`。
- [x] T007 [Triage] 実装が仕様として妥当でない箇所を、欠陥として個別の work item へ切り出す。wi-439、wi-440、wi-441。
- [x] T008 [Verify] `mise run check-spec`、`mise run test-go`、`mise run verify` を通す。

## Verification

- `mise run check-spec`
  - reason: `standards.md` は書式が *(checked)* である。`Adoption` と `Strength` の値、`excluded` に `MUST` を置いていないこと、ID の一意性がここで落ちる。
- `mise run test-go`
- `mise run verify`
- 手動: `excluded` と宣言した行を 1 つ選び、その機能を実際に送るよう実装を一時的に変えて、否定テストが落ちることを確認する。落ちなければ、その行は誰も守っていない。
- 手動: `docs/contexts/sourcing/standards.md` と並べて読み、内向きと外向きで同じ RFC の同じ条項について矛盾した宣言をしていないことを確認する。

## Risk Notes

リスクは low。文書の追加とテストの名指しであり、製品の振る舞いは変えない。

**この作業の価値は、書いた行の数ではなく `excluded` の行の数で決まる。** 「やっていること」を並べるだけなら、コードを読めば分かることの写しになる（SPECIFICATION_FORMAT §3 が禁じる形）。仕様として意味を持つのは「**やらないと決めたこと**」と、その否定テストである。送らないものを 1 つも挙げずに終わったら、この文書はほぼ無価値だと考えてよい。

もう 1 つの失敗は、**実装を確かめずに RFC の目次から行を起こすこと**である。そうすると `Statement` が標準の側から書かれ（§5 が名指しで警告している形）、製品が実際にやっていないことを宣言することになる。T001 を T003 より先に置いているのはそのためである。

## Completion

- **Completed At**: 2026-08-29
- **Summary**:
  `mise run spec-diff` は `no normative specification change against main` を返した。この道具が見ているのは規範シナリオ、状態遷移の行、TypeSpec の宣言の 3 つで、`standards.md` の行は対象外だからである。**本 work item の成果物は全体が規範の差分に映らなかった。** これ自体が発見であり、[[wi-442-spec-diff-does-not-see-standards-rows]] へ切り出した。
  意味上の差分は次のとおり。`docs/contexts/provisioning/standards.md` が新設され、外向き SCIM クライアントとしての準拠範囲が 12 行の規範として存在するようになった。RFC 7643 に 3 行（`RFC7643-OUT-CORE-RESOURCES`、`RFC7643-OUT-EXTERNAL-ID`、`RFC7643-OUT-SCHEMA-EXTENSIONS`）、RFC 7644 に 9 行（`RFC7644-OUT-RESOURCE-OPERATIONS`、`-PATCH`、`-DISCOVERY`、`-FILTERING`、`-ERROR-RESPONSE`、`-AUTHENTICATION`、`-BULK`、`-SORT`、`-ETAG`）。うち 4 行が `excluded` であり、送らないと決めたものが初めて記述された。`SPECIFICATION_FORMAT.md` §5 には、標準を提供する側ではなく消費する側で `Adoption` をどう読むかの段落が加わった。
- **Acceptance RED Evidence**:
  - **Test**: `N/A: 文書の追加であり、利用者が観測できる製品の振る舞いを変えていない。`
  - **Requirement**: N/A: 新しい REQ を起こしていない。宣言したのは既存の実装が既に従っている外部規範である。
  - **Observed Failure**: 代わりに観測した検査は `mise run check-spec`。`RFC7644-OUT-BULK` の `Strength` を `MUST NOT` から `MUST` へ一時的に変えて実行し、`RFC7644-OUT-BULK is excluded, so it cannot carry the obligation "MUST"` で落ちることを確かめた。
  - **Detection Reason**: `excluded` の行に義務を付ける形は、「提供しない能力について守るべき義務がある」という意味を成さない宣言であり、書式の検査が落とせる唯一の種類の誤りである。`Statement` が真かどうかは書式検査では落とせないため、そちらの証拠は下の Unit RED が担う。
- **Unit RED Evidence**:
  - **Test**: `TestClient_SendsNoBulkRequest`、`TestClient_SendsNoSortParameters`、`TestClient_SendsNoConditionalRequestHeaders`、`TestClient_DiscoversOnlyServiceProviderConfig`、`TestClient_SendsOneAuthenticatedRequestPerOperation`、`TestBuildResource_OmitsExtensionSchemaAttributes`、`TestRegisterConnection_DefaultMappingSendsExternalIdOnCreateOnly`
  - **Requirement**: N/A: 規範シナリオではなく規範 ID を証拠の対象とする。各テストは docs/contexts/provisioning/standards.md の 1 行を名指ししている。
  - **Observed Failure**: 宣言した振る舞いは既に成立しているため、テストは書いた時点で緑である。RED は実装を宣言に反する側へ一時的に変えて観測した。下の Change-Resistance Results が観測した失敗そのものである。
  - **Detection Reason**: 否定テストは「送っていないこと」を主張するので、何も送らなくても通ってしまう。`fullLifecycleRequests` が送出経路 6 件を通したうえで要求数を確かめてから各主張を評価するのは、この空振りを塞ぐためである。実際 `RFC7644-OUT-DISCOVERY` の最初の版はこの件数の錘に先に落ち、意図した `/Schemas` の主張が評価されていなかったので、探索だけを通す独立したテストへ書き直した。
- **Change-Resistance Results**:
  規範 1 行につき 1 つ、宣言に反する実装を入れて否定テストが落ちることを確かめた。6 件すべてが検出された。
  - `RFC7644-OUT-ETAG`: 全要求に `If-Match: "v1"` を付ける → `TestClient_SendsNoConditionalRequestHeaders` が 6 件すべてで失敗。
  - `RFC7644-OUT-SORT`: 照会に `sortBy=userName` を付ける → `TestClient_SendsNoSortParameters` が失敗。
  - `RFC7644-OUT-BULK`: User の作成を `/Bulk` へ送る → `TestClient_SendsNoBulkRequest` が失敗。
  - `RFC7643-OUT-SCHEMA-EXTENSIONS`: `urn:` で始まる対象パスを拡張オブジェクトとして組み立てる → `TestBuildResource_OmitsExtensionSchemaAttributes` が失敗。
  - `RFC7644-OUT-AUTHENTICATION`: 要求ごとに `/token` を先に叩く → `TestClient_SendsOneAuthenticatedRequestPerOperation` が要求 12 件を検出して失敗。
  - `RFC7644-OUT-DISCOVERY`: 探索で `/Schemas` も取得する → `TestClient_DiscoversOnlyServiceProviderConfig` が失敗。
  - `RFC7643-OUT-EXTERNAL-ID`: 既定の対応付けの `externalId` を `create_and_update` にする → `TestRegisterConnection_DefaultMappingSendsExternalIdOnCreateOnly` が失敗。
  検出できない範囲も記録する。`RFC7643-OUT-CORE-RESOURCES` と `RFC7644-OUT-RESOURCE-OPERATIONS` と `RFC7644-OUT-ERROR-RESPONSE` は肯定側の宣言であり、既存テストが同じ形の変更で落ちることは確かめていない。また、いずれのテストも実在する下流の SCIM 実装ではなく `httptest` の模擬を相手にしているため、**宣言が RFC の読みとして正しいかまでは確かめていない。** 相手の実装との突き合わせは [[wi-422-external-contract-and-failure-verification]] の関心である。
- **Verification Results**:
  - `mise run check-spec` - passed
  - `mise run check-links` - passed
  - `mise run check-work-items` - passed
  - `mise run check-ids` - passed
  - `mise run lint-go` - passed (0 issues)
  - `mise run verify` - passed
  - 手動: `docs/contexts/sourcing/standards.md` と並べて読み、同じ RFC の同じ条項について内向きと外向きが矛盾していないことを確かめた。重なるのは PATCH と Enterprise 拡張とフィルターで、内向きは提供者として `partial`、外向きは消費者として PATCH を `optional`、拡張を `excluded`、フィルターを `partial` と宣言しており、どちらも相手の宣言を否定していない。
