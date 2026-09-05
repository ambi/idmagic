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
  - { path: docs/contexts/provisioning/scenarios.feature.md, requirement: REQ-PROVISIONING-001 }
  - { path: docs/contexts/provisioning/scenarios.feature.md, requirement: REQ-PROVISIONING-010 }
  - { path: docs/contexts/provisioning/scenarios.feature.md, requirement: REQ-PROVISIONING-011 }
  - { path: docs/contexts/provisioning/scenarios.feature.md, requirement: REQ-PROVISIONING-012 }
  - { path: docs/contexts/provisioning/scenarios.feature.md, requirement: REQ-PROVISIONING-015 }
---

# Provisioning が宣言する未検証の拒否 5 件に効果まで確かめるテストを与え、台帳から外す

## Motivation

`tools/check/example-coverage-debt.json` は、どのテストからも id を引用されていない規範 id を保持する、縮むだけの台帳である。

このうち 102 件は [[wi-390-security-control-test-standard-and-gate]] が拒否専用の台帳へ据え置いた分で、[[wi-490-fold-refusal-coverage-into-one-normative-coverage-rule]] が網羅台帳へ統合した。

その 102 件のうち 5 件が Provisioning の拒否である。

対象はスコープとテナント境界、再試行できない配信の拒否、隔離されていない接続の再開の拒否、scope 外の subject の試験配信の拒否である。

この Context の拒否が素通りしたときの結果は、外部システムへの書き込みである。

拒否されたはずの配信が実行されれば、取り消しは相手側の運用に依存し、こちらから戻せない。

`REQ-PROVISIONING-011` は、シナリオ自身が「`InvalidRequestError` 相当の拒否として動作せず対象が無いため何も変化しない」と、拒否の実装がどう成り立っているかを注記付きで宣言している。

宣言がこの形をしていることは、拒否が明示的な検査ではなく条件付きの更新に依存していることを意味する。

その依存が壊れたとき、応答だけを見るテストは何も気づかない。

`mise run report-coverage-debt` の分類では 4 件が nearby、1 件が named だが、named は「同じエラー型名を含む拒否テストが同じパッケージへ到達する」ことしか示さない。

そして注記だけを足す解消は、この項目では認めない。

## Scope

- 台帳の Provisioning の 5 件それぞれについて、宣言された拒否に production と同じ入口から到達するテストを用意する。
- 各テストで 2 つを assert する。呼び出し元が観測する応答 (ステータスとエラー種別) と、その拒否が変えなかった状態。
- 外部システムへの書き込みを伴う拒否は、配信が 1 件も送出されていないことまで確かめる。
- 各テストのソースに対応する `REQ-PROVISIONING-NNN` を書く。
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
- 誤削除ガードの閾値、再試行の上限、scope の判定規則そのものの変更。

## Design

### 解消の条件

1 件の id を台帳から外してよいのは、次の 3 つがすべて成り立つときだけとする。

第 1 に、テストが production の使う入口から入り、宣言された拒否の判断を実際に通ること。

管理 API の拒否はトークンを付けた HTTP の境界から入り、配信の拒否は配信を起動する経路から入る。

第 2 に、拒否が変えなかったものをテスト自身が読み直して確かめること。

第 3 に、シナリオの id をテストのソースが引用していること。

### 拒否ごとに「変わっていないこと」として何を読むか

| REQ | 宣言している拒否 | 読み直して確かめる対象 |
|---|---|---|
| REQ-PROVISIONING-001 | `provisioning:read` だけの変更と、テナント不一致を `AccessDeniedError` で拒否 | 接続が変更されておらず、配信も起動していない |
| REQ-PROVISIONING-010 | dead_letter でない配信の手動再試行を拒否 | 配信の状態と試行回数が変わっておらず、外部への送出も起きていない |
| REQ-PROVISIONING-011 | quarantined でない接続の再開を拒否 | 接続の health が変わっておらず、停止していた配信が再開していない |
| REQ-PROVISIONING-012 | scope 外の subject の試験配信を拒否 | 外部システムへ 1 件も送出されていない |
| REQ-PROVISIONING-015 | 他テナントの接続と配信の参照を NotFound で拒否 | 他テナントの接続と配信が応答に現れず、変更もされていない |

`REQ-PROVISIONING-010` と `REQ-PROVISIONING-012` は、拒否の効果が外部への送出の不在である。

送出の抽象に観測点を置き、拒否のときに 1 件も呼ばれていないことを読む。

配信キューへ積まれていないことも併せて確かめる。

積んだうえで後段で捨てる実装は、応答だけを見ると拒否と区別できない。

`REQ-PROVISIONING-011` は、条件付きの更新に依存する拒否であることが宣言に書かれているため、更新条件が外れた場合に何が起きるかを確かめる。

具体的には、隔離されていない接続に対する再開が health を変えないことと、隔離中の接続に対する再開が正しく成功することの両方を置く。

後者を欠くと、常に失敗する実装でもテストが成立してしまう。

### 却下した進め方

**5 件を use case の単体テストで確かめる案。**

スコープとテナント境界の拒否はハンドラー層の配線に宿るため、use case からは検出できない。

**外部送出の不在を、配信レコードの状態だけで確かめる案。**

状態が `pending` のままでも、送出そのものは起きているかもしれない。

送出の抽象を観測する。

**`REQ-PROVISIONING-011` を拒否の応答だけで確かめる案。**

条件付き更新に依存する拒否は、条件が常に外れる実装でも同じ応答を返す。

成功経路と対で置くことでのみ区別できる。

## Plan

1. 5 件のシナリオを読み、拒否ごとに入口と「変わっていないこと」を確定する。
2. 外部送出の観測点を確認し、無ければ既存の抽象で観測できる形を決める。
3. スコープとテナント境界の 2 件 (001、015) から着手し、テストが RED になることを先に観測する。
4. 配信操作の 2 件 (010、012) を進める。
5. 隔離解除の 1 件 (011) を、成功経路と対で進める。
6. 拒否の実装が欠けていた場合は修正する。
7. 対応の取れた id を台帳から削除し、この項目が持つ id が 1 件も残っていないことを確認する。

## Tasks

- [ ] T001 [Inventory] 5 件の入口と「変わっていないこと」を確定し、外部送出の観測点を決める。
- [ ] T002 [Acceptance] スコープとテナント境界の拒否 (001、015) を効果まで確かめるテストを書く。
- [ ] T003 [Acceptance] 配信操作の拒否 (010、012) を送出の不在まで確かめるテストを書く。
- [ ] T004 [Acceptance] 隔離解除の拒否 (011) を成功経路と対で確かめるテストを書く。
- [ ] T005 [App] 拒否の実装が欠けていた経路を修正する。
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

条件付き更新に依存する拒否は、常に失敗する実装でもテストが通る。

対になる成功経路を必ず置く。

台帳は id 順に 1 エントリー 1 id なので、Context ごとの work item が並行しても衝突はエントリー単位に収まる。

各項目は自分が持つ id のエントリーだけを削除し、コメント配列と他の id には触れない。
