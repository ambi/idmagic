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
    - docs/contexts/seeding/scenarios.feature.md#REQ-SEEDING-003
    - docs/contexts/seeding/scenarios.feature.md#REQ-SEEDING-004
    - docs/contexts/seeding/scenarios.feature.md#REQ-SEEDING-005
    - docs/contexts/seeding/scenarios.feature.md#REQ-SEEDING-007
    - docs/contexts/seeding/scenarios.feature.md#REQ-SEEDING-008
    - docs/contexts/seeding/scenarios.feature.md#REQ-SEEDING-009
  typespec:
    - IdMagic.Contract.SeedRequest
    - IdMagic.Contract.SeedManifest
    - IdMagic.Contract.SeedRejectedError
    - IdMagic.Contract.SeedConflictError
  source:
    - backend/cmd/internal/bootstrap/seeding.go
    - backend/seeding/domain
    - backend/seeding/usecases
    - backend/seeding/manifests_yaml
    - seed/manifests
    - tools/check/example-coverage-debt.json
  tests:
    - backend/cmd/internal/bootstrap
affected_spec:
  - { path: docs/contexts/seeding/scenarios.feature.md, requirement: REQ-SEEDING-003 }
  - { path: docs/contexts/seeding/scenarios.feature.md, requirement: REQ-SEEDING-004 }
  - { path: docs/contexts/seeding/scenarios.feature.md, requirement: REQ-SEEDING-005 }
  - { path: docs/contexts/seeding/scenarios.feature.md, requirement: REQ-SEEDING-007 }
  - { path: docs/contexts/seeding/scenarios.feature.md, requirement: REQ-SEEDING-008 }
  - { path: docs/contexts/seeding/scenarios.feature.md, requirement: REQ-SEEDING-009 }
---

# Seeding が宣言する未検証の拒否 6 件に効果まで確かめるテストを与え、台帳から外す

## Motivation

`tools/check/example-coverage-debt.json` は、どのテストからも id を引用されていない規範 id を保持する、縮むだけの台帳である。

このうち 102 件は [[wi-390-security-control-test-standard-and-gate]] が拒否専用の台帳へ据え置いた分で、[[wi-490-fold-refusal-coverage-into-one-normative-coverage-rule]] が網羅台帳へ統合した。

その 102 件のうち 6 件が Seeding の拒否である。

この Context の拒否は他と性質が違う。

守っているのは実行中のリクエストではなく、**本番環境の初期状態**である。

env シークレットプロバイダーの拒否、development プロファイルの拒否、localhost リダイレクト URI の拒否は、いずれも「本番に開発用の資格情報と設定を書き込ませない」ための防護であり、素通りすれば既知の秘密を持つ管理者が本番に出来上がる。

しかもシナリオは 4 件で「シークレットの解決と書き込みの前に拒否する」と宣言している。

拒否の位置まで規範に書かれているのは、拒否が遅れれば秘密が解決され、部分的に書き込まれるからである。

したがってこの Context のテストは「拒否されたこと」だけでは足りず、「何も書き込まれていないこと」と「秘密の解決が起きていないこと」まで確かめる必要がある。

この 6 件はいずれも nearby に分類され、同じエラー型名を引用しているテストは 1 件も無い。

そして注記だけを足す解消は、この項目では認めない。

## Scope

- 台帳の Seeding の 6 件それぞれについて、宣言された拒否に production と同じ入口 (`SeedData` とマニフェストのローダー) から到達するテストを用意する。
- 各テストで 2 つを assert する。呼び出し元が観測するエラー種別と、その拒否が変えなかった状態。
- 「シークレットの解決と書き込みの前に拒否する」と宣言されている 4 件は、拒否の位置まで確かめる。
- 各テストのソースに対応する `REQ-SEEDING-NNN` を書く。
- 対応が取れた id を `tools/check/example-coverage-debt.json` から削除する。
- 拒否を実装が持っていないと判明した場合、その防護をこの項目で実装する。
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
- マニフェストの書式変更、プロファイルの追加、シークレットプロバイダーの追加。

## Design

### 台帳の単位が Rule から具体例へ移ったこと

この項目は `REQ-SEEDING-NNN` 6 件を持つものとして起票された。

その後 [[wi-491-track-tests-per-example]] が台帳を Rule 単位から具体例単位へ移し、`scenarios.feature.md` は Markdown with Gherkin になった。

台帳の現在の単位は `EX-SEEDING-NNN-MM` であり、Rule の id はもう台帳に載らない。

Rule の採番は移行を越えて保たれているため、起票時に挙げた 6 件は今も同じ拒否を指す。

そこで持ち分を、その 6 Rule が宣言する拒否の具体例へ読み替える。

Rule 003、004、005、007、009 は Rule 自体が拒否の宣言であり、その `通常経路` が拒否そのものなので、それらの `-01` が持ち分に入る。

### この項目が持つ id

7 件。

| Rule | 具体例 | 宣言している拒否 | 読み直して確かめる対象 |
|---|---|---|---|
| REQ-SEEDING-003 | EX-SEEDING-003-01 | マニフェストと指定プロファイルの不一致を書き込み前に拒否 | 対象ストアに 1 件も書き込まれておらず、シークレットの解決へ進んでいない |
| REQ-SEEDING-004 | EX-SEEDING-004-01 | 不正なマニフェストを書き込み前に拒否 | 対象ストアに 1 件も書き込まれておらず、シークレットの解決へ進んでいない |
| REQ-SEEDING-005 | EX-SEEDING-005-01 | 本番での env シークレットプロバイダーを拒否 | 資格情報が作成されておらず、env の解決へ進んでいない |
| REQ-SEEDING-007 | EX-SEEDING-007-01 | 本番での development および performance プロファイルを拒否 | 開発用の利用者、クライアント、グループが 1 件も作成されていない |
| REQ-SEEDING-008 | EX-SEEDING-008-02 | 未指定、localhost、HTTP のリダイレクト URI を書き込み前に拒否 | bootstrap のクライアントが作成されていない |
| REQ-SEEDING-008 | EX-SEEDING-008-01 | (拒否ではなく対照) 明示した https の URI だけが設定される | クライアントのリダイレクト URI が指定したものと完全に一致する |
| REQ-SEEDING-009 | EX-SEEDING-009-01 | 手動変更によるドリフトを上書きしない | 手動で変更された値がそのまま残っている |

`EX-SEEDING-008-01` は起票時の持ち分に入っていなかった。

`EX-SEEDING-008-02` の対照として同じテストが production と同じ入口からこの具体例をそのまま実行し、
「指定した URI だけを持つ」ことまで読むので、注記だけの解消にはならない。

そこで持ち分に加えて台帳から外す。

### 「シークレットの解決が呼ばれていない」をどう観測したか

起票時の案は「解決を担う抽象へ観測点を置き、呼び出しが 0 であることを読む」だった。

`bootstrap.Seed` は `manifestadapter.SecretResolver{Getenv: os.Getenv}` を内側で組み立てるので、
呼び出し回数を外から数える継ぎ目は無い。

継ぎ目を作れば、確かめたい防護そのものの形を変えることになる。

そこで観測の向きを変えた。

確かめたい拒否のマニフェストには必ず解決できない env シークレット参照を置き、その環境変数はどのテストでも設定しない。

解決へ進んだ実装は `seed env secret is unavailable` を返すので、宣言された拒否のエラーが返ったこと自体が「解決の手前で止まった」ことの観測になる。

各テストはこの 2 つを同時に assert する。宣言された拒否の文言を含むことと、シークレット解決の文言を含まないことである。

呼び出し回数を数える案より弱いのは、解決が呼ばれて成功した場合を区別できない点だが、
解決できない参照を置いてあるので成功する経路は存在しない。

### 解消の条件

1 件の id を台帳から外してよいのは、次の 3 つがすべて成り立つときだけとする。

第 1 に、テストが production の使う入口から入り、宣言された拒否の判断を実際に通ること。

この Context は HTTP の境界を持たないため、入口は `SeedData` の呼び出しとマニフェストのローダーである。

検証関数を直接呼ぶテストは、その関数が投入経路から呼ばれていることを示さないため、この条件を満たさない。

第 2 に、拒否が変えなかったものをテスト自身が読み直して確かめること。

第 3 に、シナリオの id をテストのソースが引用していること。

### 「書き込まれていない」として何を読むか

投入対象のストアを、呼び出し元から見える形で 3 つ数える。ファーストパーティークライアント、
利用者、グループである。

投入の前後で両方読む。投入前から空であることを確かめないと、件数の一致は何も示さない。

`EX-SEEDING-009-01` は上書きしないことが拒否の効果なので、手動変更した値が保たれていることを読む。

投入が失敗したかどうかではなく、値が守られたかどうかが規範である。

### ドリフトの判定は 2 か所にある

マニフェスト由来のファーストパーティークライアントは `ensureClient` の `sameClient` が、
デモクライアントは `SeedDemoData` の `sameDemoClient` がドリフトを見ている。

片方だけを確かめると、もう片方が上書きに退化しても気づけない。

`EX-SEEDING-009-01` のテストは両方の論理キーについて同じ手順を通す。

### 却下した進め方

**マニフェストの検証関数の単体テストで 6 件を閉じる案。**

検証が正しくても投入経路から呼ばれていなければ素通りする。

wi-390 の欠陥はまさにその形である。

**本番判定をテスト用のフラグで置き換えて確かめる案。**

本番判定そのものが防護の一部であり、置き換えると何を確かめたのか分からなくなる。

環境の判定に使う設定値を production 相当にして入る。

**書き込みの不在を、投入後のストアの件数だけで確かめる案。**

投入前から空であることを確かめないと、件数の一致は何も示さない。

投入前後の両方を読む。

## Plan

1. 7 件の具体例を読み、拒否ごとに入口と「変わっていないこと」を確定する。 (完了、上表)
2. シークレットの解決に観測点を置ける形かを調べる。 (完了、解決できない参照でエラーを弁別する形にした)
3. 書き込み前の拒否の 4 件 (003-01、004-01、005-01、008-02) を進める。
4. 本番プロファイルの 1 件 (007-01) を進める。
5. ドリフトの 1 件 (009-01) を進める。
6. 各テストについて防護を外すと落ちることを確かめる。落ちないものは入口が誤っているので入口からやり直す。
7. 拒否の位置が宣言より後ろだった場合は、拒否を前へ移す。
8. 対応の取れた id を台帳から削除し、この項目が持つ id が 1 件も残っていないことを確認する。

## Tasks

- [x] T001 [Inventory] 7 件の入口と「変わっていないこと」を確定し、シークレット解決の観測点を決める。
- [x] T002 [Acceptance] 書き込み前の拒否 (003-01、004-01、005-01、008-02) を位置と効果まで確かめるテストを書く。
- [x] T003 [Acceptance] 本番プロファイルの拒否 (007-01) を効果まで確かめるテストを書く。
- [x] T004 [Acceptance] ドリフトを上書きしないこと (009-01) を効果まで確かめるテストを書く。
- [x] T005 [App] 拒否の位置または実装が欠けていた経路を修正する。
      位置が宣言より後ろだった経路も、防護が欠けていた経路も見つからなかった。実装の変更は無い。
- [x] T006 [Ledger] 対応の取れた id を `example-coverage-debt.json` から削除する。
- [x] T007 [Verify] 防護を意図的に外すと各テストが落ちることを確かめ、検査を通す。

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

シークレットの解決が呼ばれていないことの確認は、実装の分解の仕方に依存する。

観測点を置くために境界を作り替えると、確かめたい防護そのものを動かすことになるため、既存の抽象で観測できる範囲に留め、できない場合は「書き込みが無いこと」までで妥協した理由を記録する。

本番相当の設定でテストを走らせるため、テストが実際の外部プロバイダーへ到達しないことを、投入前に確認する。

台帳は id 順に 1 エントリー 1 id なので、Context ごとの work item が並行しても衝突はエントリー単位に収まる。

各項目は自分が持つ id のエントリーだけを削除し、コメント配列と他の id には触れない。

## Completion

- **Completed At**: 2026-09-06
- **Summary**:
  `mise run spec-diff` は `no normative specification change against main` を返す。
  規範の宣言は 1 件も変わっていない。変わったのは、宣言済みの拒否 6 件が
  「エラーだけ確かめられている」状態から「書き込みの不在と拒否の位置まで確かめられている」
  状態へ移ったことと、その 6 件に対照の `EX-SEEDING-008-01` を加えた 7 件が
  網羅台帳から消えたことである。
  Seeding 分の台帳は 12 件から 5 件へ減った。製品コードの変更は無い。
  拒否の位置が宣言より後ろだった経路も、防護が欠けていた経路も見つからなかった。

- **Acceptance RED Evidence**:
  - **Test**: `TestSeedRefusesDevelopmentAndPerformanceProfilesInProduction`
    (`backend/cmd/internal/bootstrap/seeding_refusal_effects_test.go`)。
    ほかの 6 例も同じ形で観測した。
  - **Requirement**: REQ-SEEDING-007
  - **Observed Failure**: `Request.Validate` から
    `Environment == EnvironmentProduction && Profile != ProfileBootstrap` を外すと、
    `performance` プロファイルの本番投入が dry_run と apply の両方で成功した
    (`拒否されるべき投入が成功した`)。apply は合成利用者を実際に作成した。
  - **Detection Reason**: テストはエラーの文言だけでなく、投入前後のクライアント数、
    利用者数、グループ数がすべて 0 のままであることと、デモ利用者 `alice` が
    `FindBySub` で nil のままであることを条件にしている。件数だけでは
    「別の理由で 0 件だった」と区別できないので、名指しの読み直しを重ねている。

- **Unit RED Evidence**:
  - **Test**: `N/A: 防護が欠けている経路が 1 つも見つからなかったため、単体の RED は発生しない。`
    代わりに実施したのは、実装済みの防護を 1 つずつ外して対応するテストが落ちることの確認である
    (下記 Change-Resistance Results)。
  - **Requirement**: N/A: この項目は宣言済みの拒否へ検証を足すもので、新しい製品要求を作らない。
  - **Observed Failure**: `N/A`
  - **Detection Reason**: `N/A`

- **Change-Resistance Results**:
  防護を 1 つずつ外し、対応するテストだけが落ちることを確かめた。

  - `Manifest.ValidateForRequest` の `m.Profile != request.Profile` を外すと `003-01` が落ちた。
  - マニフェストの 5 つの検査をそれぞれ外すと `004-01` の対応する部分検査だけが落ちた。
    `yaml.Strict()` を外すと未知のキー、`validateLogicalKey` の重複判定を外すと重複キー、
    `Manifest.Validate` のスキーマ版判定を外すと未対応版、`loadState.load` の `s.active`
    判定を外すと `include` の循環、`contained` の判定を外すとルート外のパスが、
    それぞれ受理された。
  - `Manifest.ValidateForRequest` の
    `reference.Provider != SecretProviderFile` を外すと `005-01` が dry_run と apply の
    両方で落ちた。
  - `Request.Validate` の本番プロファイル判定を外すと `007-01` が落ちた
    (上記 Acceptance RED Evidence)。
  - `Request.Validate` の `validateProductionRedirectURIs` を外すと `008-02` が
    未指定・localhost・HTTP の 3 つの部分検査すべてで落ちた。
  - ドリフトの判定は 2 か所にあり、両方を別々に確かめた。
    `ensureClient` の `OperationConflict` を上書きの `Save` に替えると `009-01` の
    ファーストパーティークライアント側が落ち、`SeedDemoData` の
    `!sameDemoClient(...)` の分岐を無条件の `Save` に替えるとデモクライアント側が落ちた。
    片方の injection ではもう片方が緑のままだったので、2 つは独立した防護である。

- **持ち越した観測**:
  `007-01` の development プロファイルは、本番プロファイル判定を外しても
  `005-01` の env シークレットプロバイダー判定が拒否を引き継いだ。
  2 つとも外して初めて dry_run が本番で計画に成功し、apply は
  `seed user password violates password policy` まで到達した。
  「本番に開発用の資格情報を入れない」は 3 重に守られている。
  テストは最も外側の判定を固定しており、内側の 2 つはそれぞれ `005-01` と
  パスワードポリシーのテストが持つ。

- **Verification Results**:
  - `mise run test-go-race` - passed
  - `mise run check-spec` - passed (711 例中 113 id をテストが名指し。3 項目の実施前は 92)
  - `mise run check-security-controls` - passed
  - `mise run report-coverage-debt` - Seeding は 12 件から 5 件へ
  - `mise run check-work-items` - passed
  - `mise run check-ids` - passed
  - `mise run verify` - passed
