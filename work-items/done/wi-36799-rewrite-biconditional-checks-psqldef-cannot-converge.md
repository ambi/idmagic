---
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-10-08
priority: p1
depends_on: []
change_kind: maintenance
evidence_policy: risk-based-v4
documentation_impact:
  level: none
  reason: 制約の論理も外部から観測できる振る舞いも変えないので、リリースの読者へ知らせることがない。規則表の追記は一次情報文書の更新である。
  references: []
initial_context:
  specification: []
  typespec: []
  source: [infra/schema/postgres.sql, docs/design/data/schema-management.md]
  tests: [infra/schema/check-convergence.sh]
  stop_before_reading: [backend, frontend, tools]
spec_impact:
  kind: none
  reason: >-
    制約の論理を変えない。2 つの CHECK は同値な AND / OR の形へ書き換えるだけで、
    provisioning_scheduled_deprovisions と provisioning_full_resyncs が受理し拒否する行は同じままである。
---

# psqldef が収束しない双条件の CHECK 2 つを AND / OR の形へ書き換える

## 動機

`psqldef` を 3.11.23 から 3.11.26 へ上げた後、CI の `Verify psqldef schema convergence` ジョブが 3 回続けて失敗した。
空のデータベースへ適用した直後のプレビューが、次の制約の削除と再追加を出す。

- `provisioning_scheduled_deprovisions_task_consistent`: `CHECK ((status = 'materialized') = (task_id IS NOT NULL))`
- `provisioning_full_resyncs_completed_consistent`: `CHECK ((status = 'completed') = (completed_at IS NOT NULL))`

望ましいスキーマは `CREATE UNLOGGED TABLE` のために汎用パーサーで読めず、pgquery で解析される。
一方、データベースから読み戻した現在のスキーマは汎用パーサーで解析される。
両者で双条件 `(P) = (Q)` の解析結果が一致しないため、差分が収束しない。

最新のコミットでは同じジョブが合格したが、修正されたわけではない。
`users_tenant_purge_candidates_idx` の述語にある `?` 演算子を汎用パーサーが読めず、現在のスキーマの側も pgquery へ落ちるようになった。
その結果、比較が偶然一致しているだけである。
このインデックスを変えれば再び失敗する。

双条件を `AND` / `OR` の形へ書き換える方針は、完了済みの作業項目で一度確立している。
上の 2 つの制約はその後に双条件の形のまま追加された。
`docs/design/data/schema-management.md` の規則表がこの書き方を規則として載せていないことも、再発の一因である。

## 対象範囲

- 2 つの `CHECK` を `(P AND Q) OR (NOT P AND NOT Q)` の形へ書き換える。
- 書き換えた式が元の式と同値であることを PostgreSQL 上で網羅的に確かめる。
- `docs/design/data/schema-management.md` の規則表へ「双条件の `CHECK` は `AND` / `OR` で書く」を加える。

## 対象外

- `CREATE UNLOGGED TABLE` と `?` 演算子による代替パーサーへの切り替え。前者は意図した耐久性の選択であり、後者はクエリの述語とインデックスの述語を一致させる必要がある。
- `psqldef` の版の固定や戻し。
- 双条件を機械的に拒否する検査の追加。規則と収束検査で足りるかを、この作業の後で判断する。

## 設計

関係する列は `status` が `NOT NULL`、もう一方が `IS NOT NULL` を通るので、式が NULL になることはなく、三値論理による差は生じない。
同値性は、`status` の取りうる値と NULL / 非 NULL の全組み合わせを `IS DISTINCT FROM` で突き合わせて確かめる。

収束の確認は、汎用パーサーで読めない `?` があるために偶然合格する状態では意味がない。
書き換えの効果を示すため、`users_tenant_purge_candidates_idx` を一時的に外した状態で `mise run check-schema` が書き換え前に失敗し、書き換え後に合格することを確かめる。

## 計画

1. インデックスを一時的に外した状態で `mise run check-schema` を走らせ、2 つの制約の削除と再追加が出ることを確かめる。
2. 2 つの `CHECK` を書き換え、同じ状態で合格することを確かめる。
3. インデックスを戻して合格することを確かめる。
4. 同値性を網羅的に検証し、規則表を更新する。

## タスク

- [x] T001 [Acceptance] インデックスを外した状態で収束検査の RED を確かめる。
- [x] T002 [Schema] 2 つの双条件の `CHECK` を同値な `AND` / `OR` の形へ書き換える。
- [x] T003 [Docs] スキーマ管理の規則表へ双条件の書き方を加える。
- [x] T004 [Verify] 同値性を網羅的に検証し、`mise run check-schema` と `mise run verify` を通す。

## 検証

- `mise run check-schema`
- `mise run verify`
- 書き換えた 2 つの式が、関係する列の全組み合わせで元の式と同じ真偽値を返す。

## リスク

リスクは low。
制約を弱める書き換えをすると、これまで拒否していた行が通る。
同値性の網羅検証がこの誤りを直接検出する。

## 完了

- **完了日**: 2026-10-08
- **要約**:
  `mise run spec-diff` は規範の差分なしを報告する。
  `provisioning_scheduled_deprovisions_task_consistent` と `provisioning_full_resyncs_completed_consistent` を、同値な `AND` / `OR` の形へ書き換えた。
  `psqldef` 3.11.26 でも、空のデータベースに対する収束検査が、偶然の合格に頼らず通るようになった。
  スキーマ管理の規則表に、双条件の `CHECK` の書き方と、検査だけに頼れない理由を加えた。
- **受け入れ RED の証拠**:
  - **テスト**: `mise run check-schema`（`users_tenant_purge_candidates_idx` を一時的に外した状態）
  - **要件**: N/A: 規範上の振る舞いを変えないスキーマ記法の変更であり、対応する REQ がない。
  - **観測した失敗**: 変更前の `main` では収束検査が合格した。インデックスを外すと、空のデータベースへ適用した直後のプレビューが 2 つの制約の `DROP CONSTRAINT` と `ADD CONSTRAINT` を出して失敗した。CI で 3 回続いた失敗と同じ出力である。
  - **検出できる理由**: インデックスの `?` 演算子が現在のスキーマの側も代替パーサーへ落とし、比較を偶然一致させていた。インデックスを外すと、両側が別のパーサーで解析される CI の失敗時の状態に戻る。書き換え後は、インデックスの有無のどちらでも合格した。
- **単体 RED の証拠**:
  - **テスト**: PostgreSQL 18 上での式の同値検証。
  - **要件**: N/A: 規範上の振る舞いを変えないスキーマ記法の変更であり、対応する REQ がない。
  - **観測した失敗**: 該当なし。書き換えの前に失敗する検査がないので、書き換えの後で同値性を網羅的に確かめた。`status` の取りうる値に NULL を加えた値と、相手の列の NULL / 非 NULL の全組み合わせ（8 通りと 6 通り）で、不一致は 0 件だった。
  - **検出できる理由**: `IS DISTINCT FROM` は両辺が NULL の場合も一致として扱うので、三値論理のずれを見落とさない。定義域が小さいので全数検証になり、標本ではない。
- **変更耐性の結果**:
  書き換えた式を持つ表へ実際に行を入れ、`('completed', NULL)` と `('running', now())` が `violates check constraint` で拒否され、整合した 2 行が受理されることを確かめた。弱い制約に書き換えていたら、拒否されるはずの行が通っていた。
- **検証結果**:
  - `mise run check-schema` - passed
  - `mise run verify` - passed
