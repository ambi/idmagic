---
status: accepted
authors: [tn]
created_at: 2026-07-15
---

# ADR-111: 動的グループ所属ルールに制限付き CEL を採用する

## コンテキスト

部署、雇用区分、勤務地などの User 属性から Group membership を自動管理したい。
独自 DSL は parser、tooling、利用者学習を恒久的に所有することになり、JavaScript や
Lua は membership が認可へ直結する用途に対して実行能力が広すぎる。Okta と Entra ID
が備える比較・論理演算・文字列・配列条件を満たしつつ、安全に組み込める既存言語が必要である。

## 決定

1. `github.com/google/cel-go` の CEL を採用し、保存する rule は Boolean を返す単一式とする。
2. activation は `user` だけを公開する。core profile と Tenancy の実効
   `UserAttributeDef` を flat field とし、roles、membership、credential、他 tenant は公開しない。
3. CEL AST を allowlist 検査し、論理・比較・文字列・RE2 regex・`lowerAscii`・
   `exists` / `all` だけを許可する。式長、AST、regex、list、runtime cost に上限を置く。
4. 文字列は CEL 標準の case-sensitive を既定とし、case-insensitive は
   `lowerAscii()` を明示する。欠損値は null、評価 error は false とする。
5. Group の membership type は `manual` / `dynamic` の排他的かつ不変な値とする。
   dynamic group への手動 include / exclude は許可しない。
6. DynamicGroupRule は version を持ち、dynamic membership は同じ version のときだけ有効。
   rule 更新・disable は旧所属を直ちに無効化する。単一 User 更新は同期評価、rule 単位の
   全件評価は Jobs の `dynamic_group_reconcile` として実行する。
7. CEL source を正として保存し、保存時と worker 実行時に compile する。compile 済み
   program は `(tenant, group, version)` で process-local cache できるが永続 AST は持たない。

## 却下した代替案

- 独自 JSON/構造化 DSL: Builder には適するが、表現力拡張ごとに独自言語を保守するため不採用。
- Rego / Cedar: 汎用 policy / authorization engine であり、User 1件を Boolean 判定する用途には過大。
- JavaScript / Lua: sandbox と資源制限の責任が大きく、非 Turing complete な CEL より攻撃面が広い。
- manual と dynamic の混在: 例外運用は柔軟だが、所属理由、削除権限、監査の説明を複雑にする。

## 影響

- SCL は `models.DynamicGroupRule` / `models.DynamicMembershipEvaluation`、
  `interfaces.UpdateDynamicGroupRule` / `PreviewDynamicGroupRule` /
  `EnableDynamicGroupRule` / `DisableDynamicGroupRule` と関連 events / scenarios を持つ。
- `spec/contexts/jobs.yaml` に `dynamic_group_reconcile` を追加する。
- `group_members` は source と rule version を保持し、認可・Application assignment は有効 version だけを読む。
- 管理 UI は条件 Builder と CEL editor を同じ source expression に対して提供する。
