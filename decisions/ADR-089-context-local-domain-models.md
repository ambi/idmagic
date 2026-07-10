---
status: suggested
authors: [tn]
created_at: 2026-07-10
---

# ADR-089: ドメインモデル（SCL 双子定義）をコンテキスト所有へ

## コンテキスト

RA §3.6 は「トップでコンテキスト分割し、各コンテキスト内部で層を繰り返す。境界づけら
れたコンテキストは再生成と AI への文脈投入の自然な単位」と規定する。[[ADR-047]] は
Layer 4 アダプタを、[[ADR-070]] は横断アダプタと SCL Go binding の置き場を整えた。

しかし SCL 側は既に `spec/contexts/<context>.yaml` へコンテキスト分割済みである一方、
その Go 双子定義（entity / value object / enum / state / event の型）は依然として
`internal/shared/spec/`（35 file・約 6,900 行）に集中している。

- `OAuth2Client` は `shared/spec/oauth2.go`、`User` は `shared/spec/users.go` に居座り、
  各コンテキストの `domain/` は抜け殻（oauth2:10, authentication:2, application:1,
  identitymanagement:0 file）。
- `shared/spec` は全 Go ファイルの約 62%（316 file）が import する巨大共有カーネルと化し、
  1 コンテキストの変更が `shared/spec` の横断編集を誘発する。`events.go`(969) /
  `enums.go`(436) / `policy.go`(450) の一点変更が全コンテキストへ波及する。

結果として「1 機能 = 1 コンテキストディレクトリ読解」という RA の狙いが、ドメイン型の
中央集権により崩れている。[[ADR-070]] は `shared` を technical shared context と定義した
が、そこに *業務ドメイン型* が居るのは Shared Kernel の肥大であり本来の意図と逆行する。

## 決定

SCL のコンテキスト分割に Go 双子定義を追従させ、業務ドメイン型を所有コンテキストの
`domain/` へ局所化する。`shared` からは業務型を排除し、真の published language のみを残す。

1. **業務ドメイン型を per-context `domain/` へ移設する**。`shared/spec/oauth2.go` →
   `internal/oauth2/domain/`、`users.go` → `internal/identitymanagement/domain/`、
   `authentication.go` → `internal/authentication/domain/` … と、既存の空 `domain/`
   パッケージへ移す。SCL が既にコンテキスト分割済みのため対応は 1:1 で付く。
2. **`enums.go` / `events.go` / `policy.go` はコンテキスト別に割る**。複数コンテキストで
   *真に共有される* 型のみ `internal/shared/kernel/`（published language）へ残す。何を
   kernel に残すかの判断基準は `spec/context-map.yaml` の各コンテキストの `publishes` と
   `depends_on.uses` とする。ここに載らない型は共有しない。
3. **SCL ロード基盤は `shared` に残す**。`loader.go` / `validation.go` / `coherence.go`
   は特定業務に属さない SCL 読込・整合検証の技術基盤であり `internal/shared/spec`
   （または `internal/shared/scl`）に留める。
4. **移設は import パス付け替え中心とする**。双子定義は手書きであり `ra render` は Go を
   生成しない（HTML / JSON Schema / OpenAPI のみ）。よって本 ADR の範囲では型定義の
   物理移動と参照更新に限り、振る舞い・SCL・wire 契約・DB schema は変更しない。
5. **コンテキスト間で domain 型を直接 import しない**。他コンテキストの型が要る箇所は
   `shared/kernel` の published language 経由、または adapter 境界での変換に限定する。

## SCL→Go 双子定義生成の将来余地

現状は双子定義を手で維持しており移設後も手書き乖離のリスクは残る。将来 SCL→Go の
双子定義生成をコンテキスト別に行い各 `domain/` へ吐けば乖離を根絶できる。本 ADR は
その方向性のみ示し、実装は別 Work Item とする（本 ADR の決定には含めない）。

## 却下した代替案

- **現状維持（`shared/spec` に業務型を集約）**: import path は短いが、[[ADR-047]] /
  [[ADR-070]] が志向した context-first 局所化がドメイン層で達成されず、shotgun surgery が
  残る。
- **`shared/spec` を残しつつ per-context `domain/` に薄いラッパを置く**: 型が二重化し
  真実の所在が二箇所になる。かえって認知負荷が増える。
- **kernel を設けず全型を各 domain に閉じ込める**: 真に共有される published language
  （`TenantRef` 等）まで各所へ複製され、コンテキスト間の共通語彙が失われ整合が崩れる。

## 影響

- Go import path が `idmagic/internal/shared/spec/*`（業務型）から
  `idmagic/internal/<context>/domain/*` および `idmagic/internal/shared/kernel/*` へ変わる。
- `internal/shared/spec` には SCL ロード基盤（`loader` / `validation` / `coherence`）のみ
  残る。業務エンティティ型はゼロになる。
- `spec/context-map.yaml` の `publishes` が kernel 収録判定の規範となる（相互参照）。
- 振る舞い・SCL 規範・HTTP route・DB schema・公開 API は変更しない。
- [[ADR-047]] / [[ADR-070]] を supersede せず extend する（横断アダプタの置き場は不変、
  本 ADR は *ドメイン型* の所有境界のみを前進させる）。
- 移行は Work Item で段階実施（パイロット 1 コンテキスト → 依存の少ない順に横展開）。
