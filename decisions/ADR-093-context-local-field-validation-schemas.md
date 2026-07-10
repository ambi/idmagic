---
status: accepted
authors: [tn]
created_at: 2026-07-11
---

# ADR-093: 業務型の field validation (zog schema) はコンテキスト所有へ、汎用ラッパーのみ共有化する

## コンテキスト

[[ADR-089]] 決定 3 は「SCL ロード基盤は `shared` に残す。`loader.go` / `validation.go` /
`coherence.go` は特定業務に属さない SCL 読込・整合検証の技術基盤である」としている。

しかし [[wi-173]]（oauth2 横展開）の T002 着手時に判明した通り、`backend/shared/spec/validation.go`
（358 行）は SCL YAML の読込・構造検証（`loader.go`/`coherence.go` が担う）とは別物で、
実体は `tenantSchema` / `oauth2ClientSchema` / `userSchema` / `mfaFactorSchema` /
`authorizationDetailTypeSchema` 等、**個々の業務ドメイン型のフィールド検証スキーマ
（`github.com/Oudwins/zog` による）を 1 ファイルに集約したもの**である。呼び出し元は
`oauth2.go` / `tenancy.go` / `users.go` / `groups.go` / `agents.go` / `authentication.go`
の 6 ファイルにまたがり、いずれも [[ADR-089]] 決定 1 により今後 per-context `domain/` へ
移設される予定の型である。

[[ADR-089]] 決定 1（業務型を per-context `domain/` へ移設）と決定 3（`validation.go` を
`shared` に残す）を文字通り両立させると、型が `oauth2/domain` 等の leaf context へ移った後も
`shared/spec` がそれらの型を参照してスキーマ検証する形になり、`shared` が leaf context へ
依存する逆向き依存が発生する。これは [[ADR-089]] 決定 5（コンテキスト間で domain 型を
直接 import しない）にも反する。[[ADR-089]] 決定 3 の「validation.go」という記述は、
本来 SCL 自体の読込・整合検証（`loader.go`/`coherence.go` の技術基盤）を指す意図だったが、
同名の別ファイルが実際には型別スキーマ集約ファイルであったための記述の齟齬と判断する。

## 決定

1. **型別の zog スキーマ変数（`xxxSchema`）は、対応する型と共に所有コンテキストの
   `domain/` へ移設する**。例：`oauth2ClientSchema` / `consentSchema` /
   `authorizationDetailTypeSchema` は `OAuth2Client` / `Consent` /
   `AuthorizationDetailType` と共に `internal/oauth2/domain/`（現 `backend/oauth2/domain/`）へ。
2. **zog の汎用ラッパー（`validate(schema, value) error` / `zogError(issues) error`）は
   `shared` 側に残し、他コンテキストの `domain/` パッケージから再利用できるようエクスポートする**。
   [[ADR-090]] が `shared/adapters/persistence` の `RowScanner`/`TenantKey` をエクスポートして
   per-context adapter に再利用させたのと同じパターン（技術基盤の共有・業務型の非共有）を
   field validation にも適用する。配置は `backend/shared/spec` 直下に `Validate`/`ZogError`
   としてエクスポートする（新規パッケージは起こさない。SCL ロード基盤と同じパッケージに
   同居させて構わない技術的ユーティリティのため）。
3. **`loader.go` / `coherence.go`（真の SCL 読込・整合検証）は [[ADR-089]] 決定 3 の通り
   `shared/spec` に残す**。今回の決定は `validation.go` の解釈のみを是正するものであり、
   [[ADR-089]] の他の決定を変更しない。
4. 移設対象コンテキストの `domain/` パッケージは `z "github.com/Oudwins/zog"` を直接 import し、
   スキーマ定義とその型の `Validate() error` メソッドを保持する。`shared/spec` への依存は
   `Validate`/`ZogError` ラッパー呼び出しのみに限定する。

## 却下した代替案

- 案 A: `validation.go` を [[ADR-089]] 決定 3 の記述通りそのまま `shared` に残し、型移設後も
  `shared/spec` が per-context `domain/` を import してスキーマ検証する。
  → `shared` が leaf context に依存する逆依存が生じ、[[ADR-089]] 決定 5 と
  [[ADR-091]] が目指す「各 context が自己完結する」方向性に反するため却下。
- 案 B: zog スキーマ検証自体を廃止し、Go の素朴な if 文によるバリデーションへ置き換える。
  → 振る舞い不変（[[ADR-089]] 決定 4）に反し、無関係な変更を横展開 WI に混入させるため却下。
  zog 採用の是非は本 ADR の範囲外。

## 影響

- `backend/shared/spec/validation.go` は SCL 読込・整合検証専用ファイルではなくなるため、
  型別スキーマの移設が完了したコンテキストから順にスキーマ変数を削除していく。全コンテキスト
  移設完了時点で `validation.go` は空になるか、削除して `loader.go`/`coherence.go` のみが残る。
- `backend/shared/spec` は `Validate(schema *z.StructSchema, value any) error` と
  `ZogError(issues z.ZogIssueList) error` をエクスポートする。
- [[wi-173]]（oauth2 client/consent/authorization detail type）以降、[[wi-174]]〜[[wi-179]]・
  [[wi-181]]・[[wi-182]] の domain 移設タスクは本 ADR の方針に従う。
- SCL 規範・DB schema・公開 API・HTTP route は変更しない。
