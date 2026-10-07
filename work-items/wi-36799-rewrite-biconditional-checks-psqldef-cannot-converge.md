---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-10-08
priority: p1
depends_on: []
change_kind: maintenance
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

- [ ] T001 [Acceptance] インデックスを外した状態で収束検査の RED を確かめる。
- [ ] T002 [Schema] 2 つの双条件の `CHECK` を同値な `AND` / `OR` の形へ書き換える。
- [ ] T003 [Docs] スキーマ管理の規則表へ双条件の書き方を加える。
- [ ] T004 [Verify] 同値性を網羅的に検証し、`mise run check-schema` と `mise run verify` を通す。

## 検証

- `mise run check-schema`
- `mise run verify`
- 書き換えた 2 つの式が、関係する列の全組み合わせで元の式と同じ真偽値を返す。

## リスク

リスクは low。
制約を弱める書き換えをすると、これまで拒否していた行が通る。
同値性の網羅検証がこの誤りを直接検出する。
