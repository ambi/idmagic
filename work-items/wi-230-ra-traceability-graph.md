---
status: pending
authors: ["tn"]
risk: high
created_at: 2026-07-16
depends_on: [wi-229-scl-stable-element-references]
---

# SCL 規範要素・Architecture・実装・検証証跡を直接結ぶ追跡グラフを構築する

## Motivation
SCL、Architecture、Go/TypeScript 実装、テスト、WI completion は個別には存在するが、相互対応を検証する仕組みがない。実装済みだが仕様がない endpoint、仕様はあるが実現・検証がない要素、対象 revision が古い検証結果を現在も有効として扱う問題を自動検出する必要がある。

保証文を別の obligation 台帳へ複製せず、wi-229 が定義する安定した SCL element reference と実現・検証・実行結果を直接結ぶことで、SCL の規範要素を単一の正に保つ。

## Scope
- `verification/manifest.yaml` と schema を追加し、SCL element reference、Architecture module、実行可能 check、`just` recipe、evidence kind の対応を宣言する。
- manifest に対象 selector ごとの coverage policy を持たせ、どの SCL 要素に realization・検証・許容 evidence kind が必要かを明示する。
- CI evidence に対象 source revision、artifact、実行時刻、結果を記録し、宣言された検証と実行結果を分離する。
- `ra` CLI に workspace traceability graph の構築、参照解決、coverage/staleness 検査、機械可読 report を追加する。
- `backend/shared/spec/assurance_manifest.go` の手書き台帳を manifest へ移行し、Go 側レジストリを廃止する。
- Work Item schema に `change_kind`、`initial_context`、直接の SCL element reference を持つ `affected_spec`、`spec_impact` の条件付き必須規則を追加する。
- 既存 pending WI と既存 verification binding を report-only 期間中に移行し、strict gate 導入前の debt を期限付き baseline として分類する。

## Out of Scope
- SCL への `assurance` section、obligation、テストパス、CI 情報の追加。
- 個々の不足テストや不足実装の解消。
- Architecture と実 import の詳細検証。これは wi-232 が所有する。
- 外部 SaaS への evidence upload。
- 自然言語のテスト名や description から保証内容を推測する意味解析。

## Plan
- グラフは `SCL normative element -> Architecture realization -> declared verification -> execution evidence` の四層とし、obligation 中間ノードを設けない。一つの check は複数の SCL 要素を、一つの SCL 要素は複数の check を参照できる。
- coverage policy は section/element 属性による selector と、必要な realization、evidence kind、最小 check 数を宣言する。単に manifest に一度現れたことを coverage とせず、policy を満たすことを合格条件にする。
- report は `realized_without_spec`、`specified_without_realization`、`specified_without_verification`、`verification_without_target`、`missing_evidence`、`stale_evidence` を区別する。
- feature/bugfix/operations WI は `affected_spec` を必須とし、仕様非影響の変更だけ `spec_impact: none` と具体的理由を許可する。旧 `affected_guarantees` を新規入力には使用せず、完了済み WI の履歴は書き換えない。
- 導入は report-only、既存 debt の owner・理由・期限付き baseline 化、新規 drift の strict error 化の順に行う。空 selector、空 check、無期限例外、成功結果のない evidence は coverage と数えない。

## Tasks
- [ ] T001 [Schema] direct SCL target、Architecture realization、check/recipe、coverage policy、execution evidence、期限付き baseline の schema を定義する。
- [ ] T002 [Graph] wi-229 の resolver を使って四層 graph を構築し、不明参照・孤立 node・重複 binding を検査する。
- [ ] T003 [Coverage] selector ごとの realization/evidence 要件と、欠落・stale・target 不在を分類する report を実装する。
- [ ] T004 [WorkItem] `change_kind`、`initial_context`、`affected_spec`、`spec_impact` の条件付き必須規則と direct reference validation を追加する。
- [ ] T005 [Migration] pending WI、恒久 verification binding、既存 debt を manifest/baseline へ移行する。
- [ ] T006 [Go] 手書き `AssuranceManifest` と専用 binding test を廃止し、workspace traceability check へ置換する。
- [ ] T007 [CI] report-only から strict gate へ移行する `just` recipe を追加し、`just verify` に接続する。
- [ ] T008 [Verify] 未知 target、realization/check/evidence 欠落、古い revision、期限切れ baseline の negative test を含め全検証を通す。

## Verification
- `just test-tools`
- `just typecheck-tools`
- `just yaml-check-work-items`
- `just check-ids`
- `just yaml-check`
- `just verify`

## Risk Notes
既存の全 SCL 要素へ一律に同じ evidence を要求すると、意味の薄いテストと大量の false positive を生む。coverage policy は外部 binding を持つ interface、required standard、stateful aggregate、scenario など意味のある selector から段階導入し、例外は owner・理由・期限を必須にする。証跡は check の存在ではなく対象 revision に対する成功結果まで確認する。
