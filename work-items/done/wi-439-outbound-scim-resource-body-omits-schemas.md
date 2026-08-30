---
depends_on: []
status: completed
authors: [tn]
risk: medium
created_at: 2026-08-29
priority: p2
change_kind: bugfix
evidence_policy: risk-based-v2
initial_context:
  specification: [docs/contexts/provisioning/standards.md]
  typespec: [IdMagic.Contract.AttributeMappingRule]
  source:
    - backend/provisioning/client_scim/client.go
    - backend/provisioning/client_scim/mapping.go
  tests:
    - backend/provisioning/client_scim
  stop_before_reading: [frontend, backend/oauth2, backend/authentication]
affected_spec:
  - { path: docs/contexts/provisioning/standards.md, requirement: RFC7643-OUT-CORE-RESOURCES }
  - { path: spec/contexts/provisioning/models.tsp, symbol: IdMagic.Contract.AttributeMappingRule }
---

# 外向き SCIM が送るリソース本文に `schemas` を入れる

## Motivation

下流へ送る User リソースの本文は、接続の属性対応付けが解決した属性だけで組み立てている。対応付けに `schemas` は無く、書こうとしても対象パスの文法が配列リテラルを取らない。結果として `POST /Users` の本文にも `PUT /Users/{id}` の本文にも `schemas` が現れない。

RFC 7643 §3 は、リソース表現が `schemas` を持つことを要求している。値は、そのリソースが従うスキーマの URN の集合であり、User なら `urn:ietf:params:scim:schemas:core:2.0:User` である。§8 の例はすべてこれを含む。

PATCH の本文だけは `urn:ietf:params:scim:api:messages:2.0:PatchOp` を持つので、欠けているのはリソース表現の側だけである。この非対称は、`schemas` を「メッセージの種別を言う欄」としてだけ扱い、リソース表現の必須属性としては扱っていないことを示している。

**これは連携先を選ぶ欠陥である。** 受け取り側が `schemas` の有無を検証する実装なら、作成も置換も 400 で拒否される。拒否は `RFC7644-OUT-ERROR-RESPONSE` が「再試行しない失敗」として扱う経路に落ちるため、配信は試行上限を待たずに死に、原因は下流のエラー本文にしか残らない。IdMagic 自身の内向きサーバーは `schemas` を読み取り専用属性として無視するので、IdMagic どうしを繋いだ試験では再現しない。

[docs/contexts/provisioning/standards.md](../../docs/contexts/provisioning/standards.md) の `RFC7643-OUT-CORE-RESOURCES` を `partial` に留めているのはこの欠落のためである。直った時点で、この行の `Statement` は本文が `schemas` を持つことを言えるようになる。

## Scope

- 送出するリソース表現に `schemas` を入れる。User は core スキーマの URN を持つ。
- 属性対応付けが解決する属性とは別の層で入れるか、対応付けの既定に加えるかを決める。対応付けは管理者が編集できるので、必須属性を編集可能な表に置いてよいかがこの判断の要点になる。
- `RFC7643-OUT-CORE-RESOURCES` の `Statement` を更新し、証拠テストを足す。

## Out of Scope

- 拡張スキーマの URN を `schemas` に加えること。拡張属性は [[wi-403-provisioning-declares-no-scim-conformance]] が `RFC7643-OUT-SCHEMA-EXTENSIONS` として `excluded` を宣言しており、送らないものの URN を広告する理由が無い。
- Group **リソースの送出経路**。捕捉・属性解決・配信を作るのは [[wi-441-push-groups-produces-no-delivery]] が持つ。**ただし送出クライアントの `CreateGroup` / `UpdateGroup` は既に存在し、同じ欠陥を持つ**ので、本文の組み立てという 1 つの欠陥として両方直す。Out of Scope が指しているのは配信経路であって、既にあるクライアントのメソッドではない。半分だけ直すと、wi-441 が配線した瞬間に同じ欠陥が現れる。
- 下流が返す `schemas` の検証。取り込みは行っていない。

## Design

**属性対応付けの外側に置く。** `schemas` はリソース表現の必須属性であって、対応付けの対象ではない。管理者が編集できる表に置けば消せてしまい、消えた時点でこの work item が直した欠陥がそのまま戻る。そもそも対応付けの対象パスの文法は配列リテラルを取れないので、表に書くこと自体ができない。

送出クライアント (`client_scim.Client`) の `withSchemas(doc, urn)` で載せる。リソースの種別は呼び出し側 (`CreateUser` / `CreateGroup` / `UpdateUser` / `UpdateGroup`) が知っているので、URN はそこから渡す。

**載せるのはリソース表現だけである。** 作成 (POST) と置換 (PUT) の本文は完全なリソース表現なので core スキーマの URN を持つ。PATCH の本文は `PatchOp` というメッセージで、既に自分の URN (`urn:ietf:params:scim:api:messages:2.0:PatchOp`) を持っている。その `Operations[].value` は部分断片であってリソース表現ではないので、`schemas` を持たない。この非対称は RFC 7644 §3.5.2 のとおりで、Motivation が言う「`schemas` をメッセージの種別を言う欄としてだけ扱っていた」という誤りの裏返しにならないよう、証拠検査で両方向を固定する。

検討した代替案:

- **既定の属性対応付けに加える**: 管理者が消せる場所に必須属性を置くことになる。対応付けの文法が配列リテラルを取れないという実務的な障害もある。採用しない。
- **`BuildResource` の中で載せる**: `BuildResource` は PATCH の `value` を組み立てる経路でも使われるので、ここで載せると部分断片にまで `schemas` が付く。採用しない。

## Plan

1. 送出本文の組み立てのどこに置くかを決める。
2. 作成と置換の本文が `schemas` を持つことのテストを RED で置く。
3. 実装し、`RFC7643-OUT-CORE-RESOURCES` の `Statement` を更新する。

## Tasks

- [x] T001 [Design] 必須属性の置き場所を確定した。属性対応付けの外側、送出クライアントの `withSchemas`。
- [x] T002 [Test] 本文が `schemas` を持つことのテストを RED で置いた。
  `TestClient_SendsSchemasOnResourceRepresentations`。標準行: `RFC7643-OUT-CORE-RESOURCES`。
- [x] T003 [App] `withSchemas` を実装し、作成と置換の 4 メソッドに適用した。
- [x] T004 [Spec] `RFC7643-OUT-CORE-RESOURCES` の `Statement` を更新した。
- [x] T005 [Verify] `mise run check-spec`、`mise run verify`。

## Verification

- `mise run check-spec`
- `mise run test-go`
- `mise run verify`

## Risk Notes

リスクは medium。送出する本文の形が変わるため、既に連携している下流の受け取り方が変わる。`schemas` を無視する実装では無害だが、未知の属性を拒否する実装では逆に壊れうる。既存の接続がある状態で送出形を変えることになるので、変更前後の本文を並べて確認する。

## Completion

- **Completed At**: 2026-08-30
- **Summary**:
  `mise run spec-diff` は `no normative specification change against main` を返す。`REQ-` シナリオは動いておらず、変わったのは `RFC7643-OUT-CORE-RESOURCES` の `Statement` と、下流へ送る本文である。作成 (POST) と置換 (PUT) のリソース表現が `schemas` を持つようになった。User は `urn:ietf:params:scim:schemas:core:2.0:User`、Group は `...:Group` の 1 要素で、属性対応付けの外側 (`withSchemas`) で載せている。PATCH の本文は従来どおり `PatchOp` の URN を持ち、その `Operations[].value` は部分断片なので `schemas` を持たない。行の `Adoption` は `partial` のままとした。`schemas` の欠落は解けたが、対応付けで書ける範囲 (拡張スキーマを送らないこと) の制約は変わっていないためである。
- **Acceptance RED Evidence**:
  - **Test**: `TestClient_SendsSchemasOnResourceRepresentations` (`backend/provisioning/client_scim/conformance_test.go`)
  - **Requirement**: N/A: 該当する `REQ-` シナリオは無い。規範は `docs/contexts/provisioning/standards.md` の標準行 `RFC7643-OUT-CORE-RESOURCES` (MUST) と RFC 7643 §3 である。
  - **Observed Failure**: `POST /Users の schemas = [], want [urn:ietf:params:scim:schemas:core:2.0:User] (body={"userName":"alice"})`
  - **Detection Reason**: 下流が実際に受け取った要求の本文を、既存の `fullLifecycleRequests` が通す 6 経路すべてについて見る。POST と PUT には core スキーマの URN が 1 要素だけ在ること、PATCH には `PatchOp` の URN が在ることを、方法ごとに別々に主張する。URN を照合するので種別を取り違えた実装は落ち、要素数を見るので余計な URN を足した実装も落ちる。さらに `Operations[].value` に `schemas` が**無い**ことを主張するので、「リソース表現に載せる」を「本文ならどこでも載せる」と取り違えた実装が分かれる。この最後の主張が無ければ、RFC 7644 §3.5.2 に反する PATCH を送る実装が通ってしまう。
- **Unit RED Evidence**:
  - **Test**: 同上 (`withSchemas` は 3 行の写像で、HTTP の境界より内側に独立した判断を持たない)
  - **Requirement**: N/A: 上と同じ理由。
  - **Observed Failure**: 同上。
  - **Detection Reason**: 判断はどの経路にどの URN を載せるかだけで、それは送出された本文にそのまま現れる。`withSchemas` を単体で呼び直す検査を足しても、実装をもう一度書き写すだけで識別力が増えない。代わりに、経路ごとの差 (POST/PUT と PATCH と `value`) を受け入れ側の 1 つの検査で分けている。
- **Change-Resistance Results**:
  代表的な誤実装を 3 つ入れ、いずれも検出されることを実測した。
  M1 作成から `withSchemas` を外す (元の欠陥) → 落ちる。
  M2 `Operations[].value` にも `withSchemas` を適用する → 落ちる。`value` の否定主張だけが捕まえる変異で、「新しい欄が在ること」しか見ない検査なら生存していた。
  M3 User の送出に Group の URN を使う → 落ちる。URN の値まで照合しているため。
- **Verification Results**:
  - `mise run verify` - passed (exit 0)
  - `mise run check-spec` - ok
  - `mise run lint-go` - 0 issues
  - `mise run test-go-package -- ./backend/provisioning/client_scim/...` - ok
  - `mise run spec-diff` - `no normative specification change against main`

## Follow-up

Risk Notes が求めた「既存の接続がある状態で、変更前後の本文を並べて確認する」のうち、**変更前後の本文の比較は証拠検査で行ったが、実在する下流に対する確認は行っていない** (この作業からは配備先を観測できない)。`schemas` を無視する下流には無害で、RFC 7643 に従って検証する下流ではむしろこの変更で初めて通るようになる。未知の属性を拒否する実装が相手の場合にだけ影響が出る。
