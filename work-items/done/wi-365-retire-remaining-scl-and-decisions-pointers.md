---
status: completed
authors: [tn]
risk: medium
created_at: 2026-08-13
depends_on: []
change_kind: maintenance
initial_context:
  typespec:
    - IdMagic.OAuth2.Operations.ListAdminRolePolicies
  source:
    - backend/shared/spec/enums.go
    - backend/shared/kernel/tenancy.go
    - infra/k8s/base/configmap.yaml
  stop_before_reading:
    - frontend
    - spec/contexts
affected_spec:
  - { path: spec/contexts/oauth2/main.tsp, symbol: ListAdminRolePolicies }
---

# 削除済みの spec/scl.yaml と decisions/ を指したままの記述と設定を実態へ揃える

## Motivation

SCL と `decisions/` は撤去済みだが、そこを指すポインタが残っている。今回の方式整備で「読み手を存在
しない場所へ送る導線」を潰してきたので、同じ基準で残りも片付ける。

- `docs/SPECIFICATION.md` の Structure ツリーが、削除済みの `decisions/` を現在のディレクトリとして
  掲載している。現在設計を記す文書が実態と食い違っている。
- `infra/k8s/base/configmap.yaml`・`infra/docker/docker-compose.dev.yaml`・
  `infra/deploy/gcp/cloudrun-idmagic.yaml` が `SCL_PATH=/app/spec/scl.yaml` を設定している。
  backend に `SCL_PATH` を読むコードは無く、指すファイルも存在しない。実害こそ出ていないが、
  コンテナ設定を読む人には有効な設定に見える。
- Go コメント 5 箇所が `spec/scl.yaml` を規範として参照し、うち 2 箇所は既に存在しない
  coherence test での突き合わせを前提にしている。
- `spec/contexts/oauth2/main.tsp` の `ListAdminRolePolicies` の `@doc` が「SCL authorization から
  導出された」と説明している。生成 OpenAPI 経由で API 利用者にも見える。
- wi-359 で追加した cross-context 配置規約が「root `SPECIFICATION.md` に `REQ-SYSTEM-NNN` を置く」と
  書いているが、実際には root は Overview と Design だけで Scenarios を持たず、context 横断シナリオは
  `docs/contexts/system/SPECIFICATION.md` が所有している（同文書の Overview がそう宣言している）。
  規約の側が実態と食い違っている。

## Scope

- `docs/SPECIFICATION.md` の Structure から `decisions/` を除去する。併せて traceability に関する
  Design 記述に、導出ビューが存在する現状を反映する。
- 3 つの infra 設定から `SCL_PATH` を除去する。
- Go コメントの `spec/scl.yaml` 参照を、現在の所有文書（root / context の `SPECIFICATION.md`、
  TypeSpec）への参照に置き換える。存在しない coherence test への言及は削除する。
- `ListAdminRolePolicies` の `@doc` を現在の説明に直す。
- `tools/README.md` に、今回追加した導出ビューと検査を反映する。
- cross-context 配置規約を実態へ揃える。`SPECIFICATION_FORMAT.md` は「cross-context view の所有を
  宣言している文書に置く」と製品非依存に書き、`spec-change` skill 側でこのリポジトリでの所有者を示す。

## Out of Scope

- `spec/idmagic.openapi.baseline.json` に残る `x-scl-*` 拡張。凍結済みのリリース基準線であり、
  リリース手順の外で書き換えない。
- Go コメントの言語・文体の統一。参照先の修正だけを行う。
- `SCL_PATH` を読む機能の再実装。読み手は存在しない。

## Design

- 参照先の置き換えは「削除」ではなく「現在の所有者への付け替え」を基本にする。コメントが指していた
  情報（context_map の依存方向、パスワードポリシーの既定値）は今も所有者が存在するため、そこへ送る。
- `SCL_PATH` は削除する。読み手が無い設定は、残すと将来の運用者に意味があるものと誤読される。

## Plan

1. `docs/SPECIFICATION.md` と TypeSpec の `@doc` を直す。
2. infra 設定から `SCL_PATH` を除去する。
3. Go コメントの参照を付け替える。
4. `tools/README.md` を更新する。
5. `just check-api-compat` を含む `just verify` を通す。

## Tasks

- [x] T001 [Spec] `docs/SPECIFICATION.md` の Structure と traceability 記述を実態へ揃える。
- [x] T002 [Spec] `ListAdminRolePolicies` の `@doc` を直す。
- [x] T003 [Ops] 3 つの infra 設定から `SCL_PATH` を除去する。
- [x] T004 [App] Go コメント 5 箇所の参照を現在の所有文書へ付け替える。
- [x] T005 [Docs] `tools/README.md` を更新する。
- [x] T006 [Docs] cross-context 配置規約を `SPECIFICATION_FORMAT.md` と `spec-change` skill で
      実態へ揃える。
- [x] T007 [Verify] `just check` と `just verify` を通す。
- [x] T008 [Completion] 完了記録を追加して `work-items/done/` へ移動する。

## Verification

- `just check-api-compat`
- `just check-compose`
- `just check-k8s`
- `just verify`

## Risk Notes

`@doc` の変更は生成 OpenAPI の description を変えるため、`just check-api-compat` で破壊的変更と
判定されないことを確認する。`SCL_PATH` の除去は、リポジトリ外の deployment がこの環境変数を前提に
していた場合に影響し得るが、backend に読み手が無いため挙動は変わらない。

## Completion

- **Completed At**: 2026-08-13
- **Summary**:
  削除済みの `spec/scl.yaml` と `decisions/` を指したままの箇所を実態へ揃えた。root
  `SPECIFICATION.md` の Structure から `decisions/` を除き、work-items が判断履歴も持つことを明記。
  traceability の Design 記述には、導出ビューが citation から生成される現状を追記した。
  `SCL_PATH`（読み手のいない環境変数、削除済みファイルを指す）を compose / k8s / Cloud Run の 3 設定
  から除去した。Go コメント 5 箇所の `spec/scl.yaml` 参照を現在の所有者（root Design の context 関係、
  tenancy の PasswordPolicyDefaults、`just check-boundaries`）へ付け替え、既に存在しない coherence
  test への言及を削除した。`ListAdminRolePolicies` の `@doc` から SCL の語を外した。
  加えて wi-359 で入れた cross-context 配置規約の誤りを修正した。root `SPECIFICATION.md` は Overview
  と Design だけで Scenarios を持たず、context 横断シナリオは `docs/contexts/system/SPECIFICATION.md`
  が所有している。`SPECIFICATION_FORMAT.md` は製品非依存に「cross-context view の所有を宣言している
  文書に置く」とし、`spec-change` skill でこのリポジトリの所有者を示した。この誤りは、wi-360 で入れた
  `initial_context` の参照解決検査が本 work item 自身の `REQ-SYSTEM-001` 参照を落として発覚した。
- **Verification Results**:
  - `just check-api-compat` - passed（baseline に対する破壊的変更なし）
  - `just check-compose` - passed
  - `just check-k8s` - passed（20 resources valid）
  - `just verify` - passed
