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
  specification:
    - docs/contexts/provisioning/scenarios.md
    - docs/contexts/provisioning/standards.md
  typespec: [IdMagic.Contract.ProvisioningFeatureFlags]
  source:
    - backend/provisioning/usecases/capture.go
    - backend/provisioning/usecases/deliver.go
    - backend/provisioning/usecases/notify_adapters.go
    - backend/provisioning/source_idmanagement
    - backend/provisioning/ports/capture.go
  tests:
    - backend/provisioning
    - backend/provisioning/usecases
  stop_before_reading: [frontend, backend/oauth2, backend/authentication]
affected_spec:
  - { path: docs/contexts/provisioning/scenarios.md, requirement: REQ-PROVISIONING-013 }
  - { path: spec/contexts/provisioning/models.tsp, symbol: IdMagic.Contract.ProvisioningFeatureFlags }
---

# `push_groups` が配信を生まない状態を解消する

## Motivation

Provisioning の正本文書は Group の送出を能力として宣言している。`glossary.md` は Push Groups を「`ProvisioningFeatureFlags.push_groups` が有効なとき、Group とメンバーシップを下流へ配信する機能」と定義し、`states.md` の配信ライフサイクルは `GroupPushed` と `GroupMembershipPushed` を終端への遷移として持ち、TypeSpec は `push_groups` と `GroupPushConfig` を接続の設定として公開している。

**実装には、Group の配信を生む経路が無い。**

- 配信を捕捉するのは User の変更と割り当ての変更の 2 つだけで、Group の変更を捕捉する通知先はどの Context にも配線されていない。
- Full Resync が配信を作るのは適用範囲の User だけで、割り当ての一覧からも `user` 種別しか拾わない。
- 配信実行時に属性を解決する経路は User の Aggregate しか扱わず、Group を渡すと「対象が無い」として何も送らずに成功で終わる。
- 送出クライアントには Group の作成・更新とメンバーシップの PATCH が実装されているが、呼び出す側がいない。

つまり `push_groups` を有効にしても、下流へは何も起きない。**設定は保存され、画面は有効と表示し、配信は 1 件も生まれない。** 失敗として現れないぶん、動いていないことに気付く手掛かりが無い。

[[wi-403-provisioning-declares-no-scim-conformance]] は外向き SCIM の準拠範囲を宣言したが、Group リソースの行を置けなかった。`excluded` と書けば上記の正本文書と矛盾し、`partial` と書けば動かないものを動くと言うことになるからである。**この非一貫が解けるまで、外向きの Group 送出は規範として宣言できない。**

## Scope

- Group の変更とメンバーシップの変更を配信として捕捉する経路を作る。捕捉は既存の同一トランザクション捕捉のポートに揃える。
- Group の属性を解決する経路を作り、`GroupPushConfig` の表示名の取得元を反映する。
- Full Resync と On-Demand Provision が Group を対象に含む条件を決める。
- `push_groups` が無効なときに Group の配信が生まれないことを、否定テストとして固定する。
- 解決したうえで `docs/contexts/provisioning/standards.md` に Group リソースの行を足す。

## Out of Scope

- Group の入れ子構造の送出。直接所属だけを扱う内向きの方針に揃える。
- 下流の Group を IdMagic へ取り込む向き。`Sourcing` の関心である。
- 能力そのものを取り下げる判断。取り下げるなら `glossary.md`、`states.md`、TypeSpec、画面の 4 つを同時に外すことになり、この work item より大きい。着手時に費用を比較して選ぶ。

## Design

**実装する側を取る** (利用者の判断)。取り下げは `glossary.md`・`states.md`・TypeSpec・画面の 4 つを同時に外すことになり、この work item より大きい。

### 誰が通知するか

`IdManagement` の Group 側である。捕捉すべきは Group そのものの変更とメンバーシップの変更で、どちらも Group 集約の出来事だからである。`Application` の割り当ては「どの主体がそのアプリを使えるか」であって、Group の内容とは別の軸にある。

User 側の先例に揃えて、`groupports.ProvisioningNotifier` を IdManagement 側に置く。Context Map の `depends_on` は Provisioning → IdManagement の向きなので、IdManagement は `backend/provisioning` を import できない。引き金の語彙は IdManagement が持ち、`provisioning/usecases` の `GroupMutationNotifier` が自分の語彙へ翻訳する。

### 2 つの番人

配信が生まれるかどうかは 2 つの独立した条件で決まる。

1. **`push_groups` の機能フラグ** —— `translateTrigger` が落とす。能力そのものが無効なら何も生まれない。
2. **`GroupPushConfig` による対象の選択** —— `groupInScope` が落とす。`explicit` は列挙した id だけ、`assigned_groups` は接続のすべての Group を対象にする (割り当ての主体種別が今は User だけなので、Group の割り当てが入るまではこの読みになる)。**設定そのものが無い (`nil`) 場合は対象なしとする** —— 「まだ設定していない」を「すべて」と読むと、能力を有効にしただけの接続が全 Group を下流へ書き始める。

この 2 つは別々に止めるので、証拠も別々に置く。片方だけを外した誤実装がもう片方に隠れる (実際に最初の版で起きた。Change-Resistance を参照)。

### 属性解決とメンバーシップ

`AttributeSource` は User しか扱えなかったので、`GroupAttributeSource` を足し、両者を `CombinedAttributeSource` で束ねて配送エンジンへ 1 つの値として渡す。`display_name` の取得元は `GroupPushConfig.display_name_source` が選び、既定は Group の名前とする。表示名は fail-closed の判断ではないので、未設定や未知の値は失敗にせず既定へ落とす。

メンバーシップは対応付けの対象にしない。リソース本文は対応付けが作るが、メンバーシップは集合演算であり、`members` に対する増分 PATCH として別に送る。配送エンジンは `GroupMemberSource` という別のポートで読む。

**送るのは既に下流へ provision 済みのメンバーだけである。** 対応関係を持たないメンバーの識別子をこちらで作って送ると、下流はこの接続が持たないリソースを作りかねない。既にいるメンバーへの `add` は下流で no-op なので、IdMagic が下流のメンバー集合を持たなくても収束する。

検討した代替案:

- **`Application` の割り当て側から通知する**: 割り当てが動いたときだけ Group が同期されることになり、Group の名前の変更が下流へ届かない。採用しない。
- **メンバーシップを属性対応付けに載せる**: 対応付けは 1 つのリソース本文を作る仕組みで、全置換になる。下流のメンバーのうち IdMagic が知らない相手を消してしまう。採用しない。

## Plan

1. 実装と取り下げのどちらを取るかを決める。
2. 実装を選ぶなら、捕捉、属性解決、配信の順に 1 挙動ずつ広げる。
3. `standards.md` に Group の行を足し、証拠テストを付ける。

## Tasks

- [x] T001 [Design] 実装する側を確定した (利用者の判断)。
- [x] T002 [Acceptance] Group の変更が配信を生むことの受け入れ検査を RED で置いた。
  `TestE2E_GroupChange_ReachesRealDownstream`。標準行: `RFC7643-OUT-GROUP-RESOURCES`。
- [x] T003 [Adapters] `groupports.ProvisioningNotifier` と `GroupMutationNotifier` を足し、
  IdManagement の Group 変更 5 経路 (作成・更新・削除・メンバー追加・メンバー除去) から呼ぶようにした。
- [x] T004 [App] `translateTrigger` に Group の引き金を足し、`groupInScope` で対象を選ぶようにした。
- [x] T005 [Infrastructure] `GroupAttributeSource` / `CombinedAttributeSource` / `GroupMemberSource` を
  足し、worker の合成根で配線した。`PatchGroupMembers` を `ProvisioningTargetClient` に載せ、
  `deliverGroup` から呼ぶようにした。
- [x] T006 [Acceptance] 2 つの番人を分けた否定検査と、Group 削除の検査を足した
  (変異試験が見逃しを示したため。Change-Resistance を参照)。
- [x] T007 [App] 全体再同期が Group を対象に含むようにした (`resyncGroupIDs`)。捕捉と同じ 2 つの
  番人を同じ順序で適用する。`StartFullResync` には検査が 1 件も無かったので併せて置いた。
- [x] T008 [Spec] `standards.md` に `RFC7643-OUT-GROUP-RESOURCES` を足した。
- [x] T009 [Verify] `mise run verify`。

## Verification

- `mise run check-spec`
- `mise run test-go`
- `mise run verify`

## Risk Notes

リスクは medium。動いていなかった経路が動き出すため、`push_groups` を有効にしたまま放置されている接続があれば、この変更の後に初めて下流へ書き込みが始まる。誤削除ガードと適用範囲の判定が Group にも効くことを、実装より先に確かめる。

## Completion

- **Completed At**: 2026-08-30
- **Summary**:
  `mise run spec-diff` は `no normative specification change against main` を返す。`REQ-` シナリオは動いておらず、変わったのは `docs/contexts/provisioning/standards.md` に新設した `RFC7643-OUT-GROUP-RESOURCES` の行と、実装である。`push_groups` を有効にした接続で、Group の作成・更新・削除とメンバーシップの変更が下流への書き込みまで届くようになった。それまでは 4 つの経路がすべて欠けていた —— Group の変更を捕捉する通知先がどの Context にも配線されておらず、`translateTrigger` に Group の引き金が無く、属性解決が User の集約しか扱わず、`PatchGroupMembers` はクライアントにあってポートにすら載っていなかった。設定は保存され画面は有効と表示しながら、配信は 1 件も生まれていなかった。
  [[wi-403-provisioning-declares-no-scim-conformance]] が置けなかった Group リソースの行を、`partial` として置けるようになった。`partial` に留めたのは、メンバーの除去を送っていないためである (下流の現在のメンバー集合を読み戻していないので、除くべき相手を知る手段が無い)。
- **Acceptance RED Evidence**:
  - **Test**: `TestE2E_GroupChange_ReachesRealDownstream` (`backend/provisioning/e2e_capture_delivery_test.go`)
  - **Requirement**: REQ-PROVISIONING-013
  - **Observed Failure**: `undefined: usecases.GroupMutationNotifier` (build failed)。捕捉の入口そのものが存在しないことがコンパイル段階で現れた。入口を足したあとは `ExecuteDelivery() error = provisioning: resource not found` へ進み、下流が `POST /Groups` を知らないことが次に現れた。
  - **Detection Reason**: 実際の HTTP を話す下流に対して、IdManagement 側の通知から配信の実行までを通す。下流が受け取った要求そのものを見るので、「配信は生まれたが何も送っていない」という元の状態 (属性解決が「対象なし」を返して成功で終わる) と区別できる。`RemoteResourceLink` が残ることも確かめるので、次の変更が作成ではなく更新になる根拠まで固定している。
- **Unit RED Evidence**:
  - **Test**: `TestE2E_GroupChange_EachGuardStopsDeliveryOnItsOwn` と `TestE2E_GroupMembership_PatchesOnlyProvisionedMembers` (同ファイル)
  - **Requirement**: REQ-PROVISIONING-013
  - **Observed Failure**: 番人を片方ずつ外した変異 (M1 / M2 / M3) と、メンバーの相関を無視する変異 (M4) に対して観測した。
  - **Detection Reason**: 配信を止める番人は 2 つあり (機能フラグと対象の選択)、条件を 1 つずつだけ変えた 3 件と、対で「配信される」1 件を置く。こうしないと片方を外した誤実装がもう片方に隠れる —— 実際に隠れた (下記)。メンバーシップの側は、相関を持つメンバーと持たないメンバーを同じ Group に入れ、送られたのが前者だけであることと、操作が `add` であることを見る。`add` を主張しないと、`remove` を送って下流のメンバーを消す実装が通る。
- **Change-Resistance Results**:
  変更した判定を系統的に変異させ、10 件すべてが検出されることを最終的に確認した。
  M1 作成の `push_groups` ゲートを外す → 検出。
  M2 `GroupPushConfig` が無いときを対象ありとする (fail-open) → 検出。
  M3 `explicit` の選択が Group の一覧を無視する → 検出。
  M4 相関を持たないメンバーへ IdMagic の識別子を送る → 検出。
  M5 メンバーシップの変更が配信を生まない → 検出。
  M6 Group の削除を更新として送る → 検出。
  M7 メンバーシップの PATCH を一切送らない → 検出。
  M8 メンバーシップを `add` ではなく `remove` として送る → 検出。
  M9 全体再同期が 2 つの番人を無視する → 検出。
  M10 全体再同期が Group の配信を 1 件も作らない → 検出。
  **方法が見つけたもの**: 最初の版では M1・M2・M6・M8 が生存した。M1 と M2 が生存したのは、否定検査が `push_groups` を無効にすると同時に `GroupPushConfig` も未設定のままにしており、**2 つの番人が互いを隠していた**ためである。片方を外しても、もう片方が同じ配信を止めるので何も落ちない。方法論が「別々に立つ番人は証拠でも別々に」と言う通りの形で、条件を 1 つずつだけ変えた 3 件に分けて閉じた。M6 は Group 削除の検査が 1 件も無かったため、M8 は PATCH の操作名を主張していなかったためで、それぞれ検査を足した。
- **Verification Results**:
  - `mise run verify` - passed (exit 0)
  - `mise run check-spec` - ok
  - `mise run lint-go` - 0 issues
  - `go test ./backend/... -count=1` - 全パッケージ ok
  - `mise run spec-diff` - `no normative specification change against main`

## Follow-up

Risk Notes が求めた「`push_groups` を有効にしたまま放置されている接続があれば、この変更の後に初めて下流へ書き込みが始まる」への対応として、対象の選択を fail-closed にした。`GroupPushConfig` を設定していない接続は、`push_groups` が有効でも配信を生まない。したがって「有効にしただけ」の接続が突然書き始めることはなく、管理者が対象を決めたときにだけ動き出す。この判断は証拠検査 (`push_groups は有効だが対象の設定が無いなら配信しない`) で固定してある。

メンバーの除去を送っていないことは `RFC7643-OUT-GROUP-RESOURCES` を `partial` に留めている理由である。下流の現在のメンバー集合を読み戻す仕組みが要るので、別 work item として扱う。

Full Resync は Group を対象に含めた。捕捉と同じ 2 つの番人を同じ順序で適用するので、`push_groups` が無効な接続や対象を決めていない接続では Group を再同期しない。`explicit` の指定は一覧の引き当てを要さないため `GroupRepo` が未配線でも再同期でき、`assigned_groups` は `GroupRepo` を必要とする (合成根では配線済み)。

**On-Demand Provision は Group を対象に含めていない。** 単一の主体を名指しで送り直す接点で、いまの引数は User の識別子を取る。Group を取れるようにするには接点の形を変えることになり、公開 API の変更を伴う。別 work item として切り出す。
