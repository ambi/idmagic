---
status: pending
authors: [tn]
risk: medium
created_at: 2026-07-18
depends_on: []
change_kind: feature
initial_context:
  scl:
    Scim:
      - standards.RFC7644.RFC7644-ERROR-RESPONSE
      - interfaces.ListScimUsers
      - interfaces.ListScimGroups
  source:
    - backend/scim/domain/filter.go
    - backend/scim/usecases/users.go
    - backend/scim/usecases/groups.go
    - backend/scim/usecases/list.go
  tests:
    - backend/scim/domain/filter_test.go
    - backend/scim/adapters/http/scim_test.go
  stop_before_reading:
    - frontend
affected_spec:
  - { context: Scim, kind: standard_requirement, standard: RFC7644, requirement: RFC7644-ERROR-RESPONSE }
  - { context: Scim, kind: interface, element: ListScimUsers }
  - { context: Scim, kind: interface, element: ListScimGroups }
---

# SCIM inbound filter grammar の対応範囲を広げ、adoption の隙間を縮める

## Motivation

wi-238 は RFC 7644 §3.4.2.2 filter grammar のうち、属性・演算子の allowlist に閉じたサブセット
だけを実装した(`adoption: partial`、[[ADR-121-scope-narrowing-disclosure-obligation]] の開示義務に基づき理由を明記済み)。allowlist 化
自体は injection・過剰計算量対策として正当だが、現状は次の実務上価値の高いユースケースが塞がれている。

- `gt`/`ge`/`lt`/`le` はどの属性にも許可しておらず、構文としては認識するが常に `invalidFilter` に
  なる。SCIM の実運用で最も頻出する gt/lt の用途は `meta.lastModified gt "..."` による増分同期
  (delta sync) であり、これが使えないと外部 IdP は毎回全件を取得するしかない。
- schema URN プレフィックス付き属性名 (`urn:ietf:params:scim:schemas:core:2.0:User:userName eq "x"`)
  は RFC 7644 grammar で許容されるが未対応。一部クライアント(特に enterprise extension を持つ実装)
  はデフォルトで URN 修飾した属性名を送る。

「SCIM 2.0 filter に完全対応する」ために残る作業のうち、この work item は上記2点(dateTime 比較・
URN プレフィックス)に絞る。複数値属性の複合フィルタ(bracket 構文、例 `emails[type eq "work"]`)は
User の email 表現が単一値に簡略化されている現状([[wi-239-scim-inbound-resource-contract-conformance]]
が対応)に依存するため、この work item では扱わない。

## Scope

- `spec/contexts/scim.yaml` の `standards.RFC7644.requirements` に `RFC7644-FILTERING` を新規追加
  する([[ADR-121-scope-narrowing-disclosure-obligation]] の `adoption: partial` 機構を初めて適用する)。`adoption: partial` とし、
  `reason` に「dateTime 属性の gt/ge/lt/le と schema URN プレフィックスまでは対応するが、複数値
  属性の複合フィルタ・custom schema 属性は未対応」と明記する。wi-238 時点では自由記述の interface
  description にしか書かれていなかった対応範囲を、この work item で構造化事実として初めて記録する。
- `backend/scim/domain/filter.go` に `AttrDateTime` 種別を追加し、`meta.created` / `meta.lastModified`
  へ `gt`/`ge`/`lt`/`le`/`eq`/`ne` を許可する。比較は RFC3339 の文字列辞書順ではなく `time.Parse` した
  実時刻同士で行う(タイムゾーン表記差異による誤判定を避ける)。不正な dateTime literal は
  `invalidFilter` にする。
- `attrPath` の schema URN プレフィックス (`urn:ietf:params:scim:schemas:core:2.0:User:` /
  `...:Group:`) をパーサーに追加し、prefix を剥がした後の属性名を既存の allowlist で解決する。
  未知の URN prefix は `invalidFilter` にする。
- `userFilterAttrs` / `groupFilterAttrs` (`backend/scim/usecases/users.go` / `groups.go`) に
  `meta.created` / `meta.lastModified` を追加する。

## Out of Scope

- 複数値属性の複合フィルタ(bracket 構文)。User の `emails` が単一値表現である現状に依存するため、
  [[wi-239-scim-inbound-resource-contract-conformance]] が multi-valued 対応した後に別途扱う。
- custom / extension schema (enterprise extension 等) への属性検索。属性の schema 拡張性自体が
  未実装であり、この work item の範囲を超える。
- `sort`、`attributes`/`excludedAttributes` による projection(wi-238 と同様に対象外のまま)。

## Plan

- dateTime 比較は文字列比較ではなく実時刻でのパースを行う。`filter.go` の `compareExpr` に
  `AttrDateTime` kind を追加し、parse 時に compValue の文字列を `time.RFC3339` で検証・保持し、
  評価時に属性側の値も同様にパースして比較する。
- schema URN プレフィックスは lexer/parser ではなく `parseAttrExpr` の属性解決段階で処理する
  (`urn:...:User:` を認識したら prefix を切り落として既存の allowlist 解決へ渡す)。同一属性が
  prefix あり/なし両方で解決できることをテストで固定する。
- `RFC7644-FILTERING` の `adoption` は `partial` のまま維持する(複数値複合フィルタ・custom schema
  属性が引き続き未対応のため)。`reason` を更新して対応範囲の拡大を反映する([[ADR-121-scope-narrowing-disclosure-obligation]])。

## Tasks

- [ ] T001 [SCL] `RFC7644-FILTERING` requirement (`adoption: partial`) を新規追加し、
      `ListScimUsers`/`ListScimGroups` の description を dateTime 比較・URN プレフィックス対応まで
      更新する。
- [ ] T002 [Domain] RED: `meta.lastModified gt "..."` の dateTime 比較と、無効な dateTime literal が
      `invalidFilter` になる filter_test.go を先に失敗させ、`AttrDateTime` を実装して GREEN にする
      (interfaces.ListScimUsers)。
- [ ] T003 [Domain] RED: schema URN プレフィックス付き属性名の解決テストを先に失敗させ、
      prefix 剥がしを実装して GREEN にする。
- [ ] T004 [Usecase/Adapter] RED: `/Users?filter=meta.lastModified gt "..."` の HTTP contract test
      (境界値・不正 dateTime・複数 timezone 表記の等価性) を先に失敗させ、
      `userFilterAttrs`/`groupFilterAttrs` へ `meta.created`/`meta.lastModified` を追加して GREEN にする。
- [ ] T005 [Verify] `just yaml-check`、`just test-go`、`just verify-go` を実行する。

## Verification

- `just yaml-check`
- `just test-go`
- `just verify-go`
- 手動: `/scim/v2/Users?filter=meta.lastModified%20gt%20%222020-01-01T00%3A00%3A00Z%22` が
  期待どおりの部分集合を返すことを確認する。
- 手動: `/scim/v2/Users?filter=urn%3Aietf%3Aparams%3Ascim%3Aschemas%3Acore%3A2.0%3AUser%3AuserName%20eq%20%22alice%22`
  が prefix なしの `userName eq "alice"` と同じ結果を返すことを確認する。

## Risk Notes

dateTime 比較の実装ミス(文字列辞書順比較のまま放置する等)は、タイムゾーン表記の異なる
レコード間で誤った順序判定を招き、増分同期クライアントに欠落や重複を発生させる。RFC3339 の
`time.Parse` による実時刻比較で固定し、異なる offset 表記の等価性をテストで固定する。
URN プレフィックスの誤った剥がし方は allowlist を迂回する経路になり得るため、prefix 解決後に
必ず同じ allowlist 検証を通す(新しい検証経路を作らない)。
