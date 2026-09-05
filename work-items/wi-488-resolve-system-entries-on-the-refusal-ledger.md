---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-05
change_kind: maintenance
priority: p3
depends_on: []
affected_spec:
  - { path: docs/contexts/system/scenarios.md, requirement: REQ-SYSTEM-001 }
  - { path: docs/contexts/system/scenarios.md, requirement: REQ-SYSTEM-015 }
---

# System の台帳 2 件を、フェイルクローズの検証と誤検出の是正に分けて解消する

## Motivation

`tools/check/scenario-coverage-debt.json` は、どのテストからも id を引用されていない規範 id を保持する、縮むだけの台帳である。

このうち 102 件は [[wi-390-security-control-test-standard-and-gate]] が拒否専用の台帳へ据え置いた分で、[[wi-490-fold-refusal-coverage-into-one-normative-coverage-rule]] が網羅台帳へ統合した。

その 102 件のうち 2 件が System の id である。

この 2 件は、台帳の他の 100 件と扱いが違う。

`mise run report-coverage-debt` はこの 2 件だけを `none` に分類し、所属パッケージも解決できていない。

`backend/system` というディレクトリが存在しないためで、報告タスクは prefix と一致するディレクトリが無い場合に推測しない設計になっている。

読んで確かめたところ、2 件は別々の理由でここに載っている。

**`REQ-SYSTEM-001` は本物のフェイルクローズである。**

「PostgreSQL へ到達できない → `ReadinessProbe` は `unavailable` を返し、API は新規トラフィックを受けない → `LivenessProbe` は `healthy` を維持し、依存障害だけでは再起動しない」と宣言している。

依存先が落ちているときに受付を止めることと、それだけで再起動しないことは、どちらも運用上の防護である。

前者が効かなければ、データベースへ到達できない API が正常なものとしてトラフィックを受け続ける。

後者が効かなければ、依存障害のたびに全プロセスが再起動して障害を長引かせる。

**`REQ-SYSTEM-015` は検出側の誤りである。**

該当行は「ALT 再認可から復旧できない → 再ログイン導線を提示する」であり、拒否として検出されたのは条件節の「できない」である。

結果は再ログイン導線の提示であって、何も拒否していない。

`refusalSteps` が段の行全体に対して語彙を照合しているため、条件節に否定語があるだけで拒否として数えられる。

したがってこの id は、テストを書いて解消するものではなく、**分類そのものを直して台帳から外す**ものである。

## Scope

- `REQ-SYSTEM-001` のフェイルクローズを、プローブの経路から効果まで確かめるテストで裏づける。
- `REQ-SYSTEM-015` が拒否として検出される原因を取り除く。
- 対応が取れた id を `tools/check/scenario-coverage-debt.json` から削除する。
- 検出側を直す場合は、変更の前後で台帳に出入りする id を全 Context で測り、記録する。
- 完了時点で、この項目が持つ id が台帳から 1 件残らず消えていることを確認する。

## Out of Scope

- 他 Context の拒否。
  Context ごとに別の work item が持つ。
- 102 件を 1 つの単位として消化すること。
  [[wi-399-burn-down-untested-refusal-debt]] がその形で立てられているが、進める単位を未決のまま残し、注記だけの解消も認めている。本項目はその置き換えである。
- R4 の判定単位を操作ごとへ細かくすること。
  [[wi-391-refusal-declaration-floor-and-reinventory]] が持つ。
- 「副作用の不在」を機械検査の門にすること。
  [[wi-390-security-control-test-standard-and-gate]] が意図的に見送った判断であり、ここで覆さない。
- 既存テストへ `REQ` id を注記するだけで台帳から外すこと。
  この項目ではこれを解消として扱わない。
- プローブの構成そのものと停止時の排出手順の変更。
  [[wi-98-kubernetes-health-probes-and-graceful-drain]] が扱う。

## Design

### REQ-SYSTEM-001 — プローブの経路から確かめる

テストは、PostgreSQL へ到達できない状態を作ったうえで、`ReadinessProbe` と `LivenessProbe` の両方を実際のエンドポイントから呼ぶ。

読み直して確かめるのは 3 つである。

第 1 に `ReadinessProbe` が `unavailable` を返すこと。

第 2 に、その状態で API が新規のリクエストを受け付けていないこと。

第 3 に `LivenessProbe` が `healthy` を維持すること。

第 2 と第 3 が、この宣言における「拒否が変えなかったもの」に当たる。

第 3 を欠くと、依存障害で両方を落とす実装がテストを通ってしまい、宣言の後半が守られない。

到達不能の状態は、接続先を無効な宛先へ差し替えて作る。

実際のデータベースを停止する形は、他のテストと同居できない。

### REQ-SYSTEM-015 — 検出を結果節に限る、あるいは段を書き換える

案 A は `refusalSteps` の照合を結果節に限ることである。

段は `条件 → 結果 → 結果` の形で書かれているので、最後の `→` の右側だけを語彙の照合対象にする。

エラー型名の照合は現状のまま段全体に残す。

型名は結果側にしか現れないため、条件節で誤って拾う余地が無いからである。

案 B はこの段を書き換え、条件節から否定語を外すことである。

案 A を採る。

案 B は同じ形の段が他にもあるかぎり再発し、しかも「検査を通すために日本語を選ぶ」ことになる。

規範の文章が検査の都合で歪むのは、この repo が仕様を先に置いている理由と正面から反する。

ただし案 A は全 Context の分類を変えうるので、変更の前後で台帳に出入りする id を測ってから決める。

出ていく id が `REQ-SYSTEM-015` だけであれば、そのまま採用する。

本物の拒否が出ていく場合は、案 A を破棄して条件節と結果節の両方を見る形へ改め、それでも解決しなければこの id だけを残して別項目へ送る。

### 却下した進め方

**`REQ-SYSTEM-015` に適当なテストを結び付けて閉じる案。**

拒否でないものに拒否のテストを書くことになり、台帳の意味を壊す。

**`REQ-SYSTEM-001` を単体テストで閉じる案。**

プローブの値が正しくても、受付の停止と結び付いていなければ防護になっていない。

**2 件をまとめて「System には拒否が無い」として台帳から外す案。**

`REQ-SYSTEM-001` は本物のフェイルクローズであり、外す理由が無い。

## Plan

1. `REQ-SYSTEM-001` のプローブ経路を確認し、到達不能状態の作り方を決める。
2. プローブと受付停止を結ぶテストを書き、RED を先に観測する。
3. `refusalSteps` を結果節に限る変更を試作し、全 Context で台帳の出入りを測る。
4. 測定結果に応じて案 A を採用するか、別項目へ送る。
5. 対応の取れた id を台帳から削除し、この項目が持つ id が 1 件も残っていないことを確認する。

## Tasks

- [ ] T001 [Acceptance] PostgreSQL 到達不能時に `ReadinessProbe` が `unavailable` を返し、API が新規リクエストを受けないことを確かめるテストを書く。
- [ ] T002 [Acceptance] 同じ状態で `LivenessProbe` が `healthy` を維持することを確かめるテストを書く。
- [ ] T003 [Tooling] `refusalSteps` の照合を結果節に限る変更を試作し、全 Context の分類差分を測る。
- [ ] T004 [Tooling] 測定結果に基づいて採否を決め、採る場合は検査とその場のテストを更新する。
- [ ] T005 [Ledger] 対応の取れた id を `scenario-coverage-debt.json` から削除する。
- [ ] T006 [Verify] 防護を意図的に外すとプローブのテストが落ちることを確かめ、検査を通す。

## Verification

- `mise run test-go-race`
- `mise run check-spec`
- `mise run check-security-controls`
- `mise run report-coverage-debt`
- `mise run check-work-items`
- `mise run check-ids`
- `mise run verify`

## Risk Notes

`refusalSteps` の照合範囲を狭めると、本物の拒否が台帳と検査の対象から静かに外れうる。

これは検査を緩める方向の変更なので、採否は測定結果だけで決め、差分は work item に記録する。

出入りする id を数えずに採用してはならない。

プローブのテストは、到達不能状態を他のテストへ漏らすと無関係なテストを不安定にする。

状態の差し替えはテストの内側に閉じ、終了時に必ず戻す。

台帳は id 順に 1 エントリー 1 id なので、Context ごとの work item が並行しても衝突はエントリー単位に収まる。

ただしこの項目は検査側にも触れるため、`refusalSteps` を変える前に他の Context の項目の進行状況を確認する。
