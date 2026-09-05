---
status: cancelled
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-05
change_kind: maintenance
priority: p1
depends_on: []
affected_spec:
  - { path: docs/contexts/workloadidentity/scenarios.feature.md, requirement: REQ-WORKLOADIDENTITY-002 }
  - { path: docs/contexts/workloadidentity/scenarios.feature.md, requirement: REQ-WORKLOADIDENTITY-003 }
  - { path: docs/contexts/workloadidentity/scenarios.feature.md, requirement: REQ-WORKLOADIDENTITY-004 }
  - { path: docs/contexts/workloadidentity/scenarios.feature.md, requirement: REQ-WORKLOADIDENTITY-005 }
  - { path: docs/contexts/workloadidentity/scenarios.feature.md, requirement: REQ-WORKLOADIDENTITY-006 }
  - { path: docs/contexts/workloadidentity/scenarios.feature.md, requirement: REQ-WORKLOADIDENTITY-007 }
  - { path: docs/contexts/workloadidentity/scenarios.feature.md, requirement: REQ-WORKLOADIDENTITY-009 }
---

# WorkloadIdentity が宣言する未検証の拒否 7 件に効果まで確かめるテストを与え、台帳から外す

## Motivation

`tools/check/example-coverage-debt.json` は、どのテストからも id を引用されていない規範 id を保持する、縮むだけの台帳である。

このうち 102 件は [[wi-390-security-control-test-standard-and-gate]] が拒否専用の台帳へ据え置いた分で、[[wi-490-fold-refusal-coverage-into-one-normative-coverage-rule]] が網羅台帳へ統合した。

その 102 件のうち 7 件が WorkloadIdentity の拒否である。

この Context は台帳に載っている拒否のすべてが未検証であり、しかもそのほとんどが `VerifyWorkloadAttestation` という 1 つの検証点に集まっている。

未登録の発行者、不正な署名、期限切れ、一意に決まらない一致、Killed になった Agent、他テナントの信頼設定は、いずれも「その主体は誰でもない」と判断して交換を止める拒否である。

素通りすれば、アテステーションを提示しただけの主体が Agent の資格情報を受け取る。

この 7 件はいずれも nearby に分類され、同じエラー型名を引用しているテストは 1 件も無い。

これは「注記が抜けている」よりも「拒否の理由ごとの検証が無い」ことを示唆する。

シナリオは `reason=unregistered_issuer` から `reason=agent_not_active` まで理由を区別して宣言しており、理由の取り違えは、どの防護が効いたのかを運用側から見えなくする。

そして注記だけを足す解消は、この項目では認めない。

拒否の応答だけを確かめるテストに id を書き足せば、検査は通り、台帳は縮み、防護が効いていることは何ひとつ確かめられないまま「検証済み」の見た目だけが残る。

## Scope

- 台帳の WorkloadIdentity の 7 件それぞれについて、宣言された拒否に production と同じ入口から到達するテストを用意する。
- 各テストで 2 つを assert する。呼び出し元が観測する応答 (エラー種別と `reason`) と、その拒否が変えなかった状態。
- 拒否に伴って発行されると宣言されているイベントについては、発行されたことと理由が一致することも確かめる。
- 各テストのソースに対応する `REQ-WORKLOADIDENTITY-NNN` を書く。
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
- 対応する発行者の種類の追加と、関連付けの一致規則そのものの変更。

## Design

### 解消の条件

1 件の id を台帳から外してよいのは、次の 3 つがすべて成り立つときだけとする。

第 1 に、テストが production の使う入口から入り、宣言された拒否の判断を実際に通ること。

この Context の拒否はアテステーションの提示から資格情報の交換までの経路にあるため、テストは交換のエンドポイントから入る。

第 2 に、拒否が変えなかったものをテスト自身が読み直して確かめること。

第 3 に、シナリオの id をテストのソースが引用していること。

### 拒否ごとに「変わっていないこと」として何を読むか

| REQ | 宣言している拒否 | 読み直して確かめる対象 |
|---|---|---|
| REQ-WORKLOADIDENTITY-002 | 未登録の発行者を `reason=unregistered_issuer` で拒否 | トークンが発行されておらず、`WorkloadAttestationRejected` が同じ理由で発行されている |
| REQ-WORKLOADIDENTITY-003 | 不正な署名を `reason=invalid_signature` で拒否 | トークンが発行されていない |
| REQ-WORKLOADIDENTITY-004 | 期限切れのアテステーションを `reason=expired` で拒否 | トークンが発行されていない |
| REQ-WORKLOADIDENTITY-005 | 一意に決まらない一致を `reason=ambiguous_match` で拒否 | どの Agent の資格情報も発行されていない |
| REQ-WORKLOADIDENTITY-006 | Killed になった Agent への交換を `reason=agent_not_active` で拒否 | トークンが発行されておらず、Agent が `Killed` のままである |
| REQ-WORKLOADIDENTITY-007 | 他テナントの信頼設定の利用を `reason=unregistered_issuer` で拒否 | トークンが発行されず、他テナントの登録内容が参照されていない |
| REQ-WORKLOADIDENTITY-009 | 他テナントの Agent への関連付け作成を `InvalidRequestError` で拒否 | 関連付けが作成されていない |

理由を区別して assert するのは、この Context に固有の要求である。

`reason` を取り違える実装は、拒否そのものは行うため応答の見た目では区別できないが、運用側は「登録漏れ」と「署名の不一致」を取り違え、誤った是正を行う。

`REQ-WORKLOADIDENTITY-002` はイベントの発行まで宣言しているので、拒否の効果としてイベントの有無と理由を読む。

`REQ-WORKLOADIDENTITY-007` は「他テナントの登録内容を参照しない」という否定の宣言であり、テナント境界の効果として、参照が起きていないことまで確かめる。

### 却下した進め方

**7 件を `VerifyWorkloadAttestation` の単体テストで一括して確かめる案。**

7 件が同じ関数に集まっていることは、単体で済ませてよい理由にはならない。

検証が正しくても交換の経路から呼ばれていなければ素通りするため、テストは交換の入口から入る。

**理由を区別せず「拒否されること」だけを確かめる案。**

シナリオが理由を宣言している以上、理由まで含めて 1 つの規範である。

**別テナントの登録内容を参照していないことを、参照処理のモックで確かめる案。**

モックの呼び出し回数は実装の形に依存し、テナント境界の効果そのものではない。

読める内容が返らないことで確かめる。

## Plan

1. 7 件のシナリオを読み、拒否ごとに入口と「変わっていないこと」を確定する。
2. 交換のエンドポイントから入る既存テストを調べ、拡張で足りるか新規が要るかを決める。
3. アテステーションの妥当性の 3 件 (003、004、002) から着手し、テストが RED になることを先に観測する。
4. 主体の解決の 2 件 (005、006) を進める。
5. テナント境界の 2 件 (007、009) を進める。
6. 拒否時のイベント発行が欠けていた場合は実装する。
7. 対応の取れた id を台帳から削除し、この項目が持つ id が 1 件も残っていないことを確認する。

## Tasks

- [ ] T001 [Inventory] 7 件の入口と「変わっていないこと」を確定し、既存テストの流用可否を判断する。
- [ ] T002 [Acceptance] アテステーションの妥当性の拒否 (002、003、004) を理由と効果まで確かめるテストを書く。
- [ ] T003 [Acceptance] 主体の解決の拒否 (005、006) を理由と効果まで確かめるテストを書く。
- [ ] T004 [Acceptance] テナント境界の拒否 (007、009) を効果まで確かめるテストを書く。
- [ ] T005 [App] 拒否と拒否イベントの実装が欠けていた経路を修正する。
- [ ] T006 [Ledger] 対応の取れた id を `example-coverage-debt.json` から削除する。
- [ ] T007 [Verify] 防護を意図的に外すと各テストが落ちることを確かめ、検査を通す。

## Verification

- `mise run test-go-race`
- `mise run check-spec`
- `mise run check-security-controls`
- `mise run check-event-contract`
- `mise run report-coverage-debt`
- `mise run check-work-items`
- `mise run check-ids`
- `mise run verify`

## Risk Notes

拒否のテストは、防護に到達する前の入力検証で落ちていても同じ応答を返すため、書いた本人にも成立して見える。

これを避けるため、各テストについて防護を外した状態でテストが落ちることを確かめ、落ちなかったものは入口が誤っていると判断してやり直す。

拒否の理由は 6 件が同じエラー型を共有しているため、テストが理由を assert していないと、別の理由で落ちても成立してしまう。

各テストは `reason` の値まで固定する。

台帳は id 順に 1 エントリー 1 id なので、Context ごとの work item が並行しても衝突はエントリー単位に収まる。

各項目は自分が持つ id のエントリーだけを削除し、コメント配列と他の id には触れない。

## Completion

- **Completed At**: 2026-09-06
- **Summary**:
  取り消す。[[wi-491-adopt-markdown-with-gherkin-scenarios]] により被覆の管理単位が規則の `REQ-*` から具体例の `EX-*` へ変わり、本項目の規則単位の計画は古くなった。対象の棚卸し、拒否テスト、無作用の確認、台帳更新は [[wi-392-refusal-tests-assert-the-absent-effect]] に統合し、製品と仕様には変更を加えずに終了する。
