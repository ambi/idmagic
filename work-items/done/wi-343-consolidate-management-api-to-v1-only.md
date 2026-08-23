---
status: completed
authors: [tn]
risk: medium
created_at: 2026-08-08
depends_on: [wi-297-management-api-versioning-and-deprecation-policy]
---

# 管理 API・自己管理 API のパスを `/v1/` のみに一本化する (未リリース時の一括移行)

## Motivation

[[wi-297-management-api-versioning-and-deprecation-policy]] / [[ADR-156-api-versioning-and-deprecation-policy]]
は「既存パスを壊さない」ことを絶対条件に、現行の無版パス (`/api/admin/...`,
`/api/account/...`) を「v1 のエイリアス」として維持し、`/v1/` を追加のエイリアスとして
生やす設計にした。これは**既に外部クライアントが依存している**ことを前提にした安全策である。

しかし IdMagic はまだリリースされておらず、無版パスに依存する外部クライアントは
実在しない。この前提のもとでは、無版パスを残すこと自体が将来のコストになる:

- 無版パスと `/v1/` パスの二重登録・二重の存在は、実際にリリースした後に無版パスを
  消そうとした瞬間、初めて破壊的変更になる。今のうちに `/v1/` へ一本化すれば、
  「無版パスをいつか消さなければならない」という負債を残さずに済む。
- パスが二重に存在すること自体が API 利用者から見て紛らわしい。

未リリースであるという前提のもと、無版パスを直ちに削除し `/v1/` へ一本化する。
`deprecated_since` / `sunset_at` による移行期間は不要 (依存者がいないため)。

## Scope

- `docs/contexts/*.yaml`: `/api/admin/...` を持つ interface の `bindings[].path` を
  `/api/admin/v1/...` へ、`/api/account/...` を `/api/account/v1/...` へ書き換える。
  対象は wi-297 で `admin-api` / `self-api` に分類した interface (`/api/admin` 201 件、
  `/api/account` 31 件)。`/api/branding`、`/api/auth/*`、protocol endpoint、ops endpoint は
  対象外 (ADR-156 の版管理スコープ外のまま)。

## Out of Scope

- `/api/branding`、`/api/auth/*` (browser session flow) のパス変更。ADR-156 の版管理対象外のまま。
- OAuth2/OIDC/SAML/WS-Federation/SCIM/SharedSignals のプロトコル endpoint。
- `/v2/` の導入 (将来、実際に破壊的変更が必要になった時点で別 WI として着手する)。

## Design

- Go 側は `backend/shared/http/support_http.RegisterVersionAliases`
  (`Echo.OnAddRoute` で無版パス登録時に `/v1/` エイリアスを追加で生やす仕組み) を**削除**する。
  一本化後は各 `handlers_http.RegisterRoutes` が最初から `/api/admin/v1/...` /
  `/api/account/v1/...` を直接登録するため、エイリアス機構自体が不要になる。
  `versioning.go` / `versioning_test.go` を削除する。
- `tools/scl-to-openapi`: `versionAliasPath` によるパス複製ロジックを削除する。SCL の
  `bindings[].path` が最初から `/v1/` を含むため、追加の複製生成は不要になる。
- `backend/shared/http/support_http/deprecation.go` の `canonicalizeRuntimePath` にある
  「`/v1/` を剥がして無版パスへ正規化する」分岐は、無版パスが存在しなくなるため到達しなくなる。
  死んだ分岐として残さず削除する (tenant routing 2 形式の正規化は維持する)。
- `spec/idmagic.openapi.baseline.json` は、未リリース状態でのパス形状変更なので
  「破壊的変更として検知させる」のではなく、変更後の形状で**再生成して上書き**する
  (実際のリリース後にこの手法を使わないことを README に明記する ——
  一度リリースしたら同じ手は使えない)。
- [[ADR-156-api-versioning-and-deprecation-policy]] の「決定」を「現行パスは v1 のエイリアス」から
  「パスは最初から `/api/admin/v1/...` / `/api/account/v1/...` を正規形とし、無版パスは
  提供しない」へ更新する (同一 ADR 番号のまま本文を修正。まだリリースされていない決定の
  即時修正であり、外部が依存した決定を覆すものではないため supersede ではなく本文更新とする)。

## Plan

- **SCL が正本**。まず `docs/contexts/*.yaml` の対象 binding path を書き換え、`just check-scl` を通す。
- **Go は機械的に追随**。各 `handlers_http/routes.go` 等の route 登録文字列リテラルを
  同じ規則で書き換える。安全網は既存の `TestAssembledRoutesMatchGeneratedOpenAPI`
  (実際に組み立てた router と生成 OpenAPI の path 集合を突合する) —— Go 側の書き換え漏れは
  ここで機械的に検出される。
- **Go テストも追随**。40 ファイルが対象パスを参照する。`just verify-go` を実行し、
  落ちたテストを一つずつ新パスへ更新する (テストの assertion 漏れは red で検出できる)。
- **frontend も追随**。`frontend/src/api/admin.ts` / `api/account.ts` 等 21 ファイルの
  呼び出しパスと、それを参照するテスト/MSW mock 30 ファイルを書き換える。
  `just verify-ui` で漏れを検出する。
- **最後に baseline を再生成**し、`just check-api-compat` を新しい形状に対して
  ゼロ差分で通す。

## Tasks

- [x] T001 [SCL] `/api/admin/*` interface の binding path を `/api/admin/v1/*` へ、
      `/api/account/*` を `/api/account/v1/*` へ書き換える。`just check-scl` を通す。
      → 232 binding (block-style・flow-style 両方) を書き換え。`just check-scl` 緑。
- [x] T002 [Go] 各 `handlers_http` の route 登録リテラルを新パスへ書き換える。
      `backend/shared/http/support_http/versioning.go` / `versioning_test.go` を削除し、
      `Register()` から `RegisterVersionAliases` 呼び出しを外す。
      `deprecation.go` の `/v1/` 剥がし分岐を削除する。
      → 16 route ファイルの route 登録リテラルを書き換え。`versioning.go`/`versioning_test.go`
      削除。`support_http/auth.go` の scope 判定 (`requiredAccountScope` 等、サブパス基準の
      `strings.Contains`/`HasSuffix`) と `integration_endpoints_handler.go` の
      `ManagementAPIBaseURL`/`AccountAPIBaseURL` も新パスへ追随 (これらは route 登録ではなく
      文字列一致・discovery 応答なので個別に確認・修正が必要だった)。`go build ./...` 緑。
- [x] T003 [Go] 対象パスを参照する既存テスト (約40ファイル) を新パスへ更新する。
      `just verify-go` を通す (`TestAssembledRoutesMatchGeneratedOpenAPI` を含む)。
      → 実際は Go 側 44 ファイル。`/realms/acme/api/admin/...` のように prefix が付く
      literal は最初の機械的書き換えで見落とし、2 パス目で修正 (安全網としての既存テストの
      価値を実証)。`just test-go` は最終的に `TestAssembledRoutesMatchGeneratedOpenAPI` 1 件のみ
      fail (OpenAPI 未再生成が原因、T004 で解消予定) で、他は無傷だった。
- [x] T004 [tooling] `tools/scl-to-openapi` のエイリアス複製ロジックを削除する。
      `just scl-render` で `spec/idmagic.openapi.json` を再生成する。
      → `versionAliasPath` とその呼び出しを削除、対応テスト差し替え。再生成後
      `just test-go` 全緑 (`TestAssembledRoutesMatchGeneratedOpenAPI` 含む)。
- [x] T005 [UI] `frontend/src/api/admin.ts` / `api/account.ts` 等の呼び出しパスと
      関連テスト/mock (約30ファイル) を新パスへ更新する。`just verify-ui` を通す。
      → 実際は 83 ファイル、332 箇所。機械的書き換えが `from '../../api/admin'` のような
      **相対 import 文**まで誤って書き換え (`'../../api/admin/v1'` という存在しないモジュール
      パスを生成) し、`typecheck-ui` の型エラーとして即座に検出・修正した
      (HTTP パス文字列と import 文字列はどちらも `/api/admin` を含むが別物、という
      区別が必要だった)。`just verify-ui` 緑。
- [x] T006 [Docs] [[ADR-156-api-versioning-and-deprecation-policy]] の決定文と README の
      "API Stability, Versioning & Deprecation" 節を「無版パスは存在しない」形へ更新する。
      → ADR-156 の「版の付け方」「却下した代替案」「影響」を更新 (同一 ADR 番号のまま
      本文修正、supersede はしない)。README の Versioning 段落を更新。
- [x] T007 [Baseline] `spec/idmagic.openapi.baseline.json` を新しいパス形状で再生成する
      (リリース後はこの手法を使わない旨を README に明記済みであることを確認する)。
      → 再生成して上書き。`just check-api-compat` がゼロ差分で通ることを確認。
- [x] T008 [Verify] 下記 Verification を緑にする。

## Verification

- `just check` / `just check-scl` / `just check-work-items` / `just check-ids`
- `just check-api-compat` — 再生成後の baseline に対して破壊的差分ゼロ
- `just verify-go` / `just verify-ui`
- 手動: `PERSISTENCE=memory` で実サーバを起動し、無版パス (`/api/admin/settings` 等) が
  404 になり、`/api/admin/v1/settings` が従来通り応答することを確認する。

## Risk Notes

対象ファイル数が多い (SCL 20 ファイル、Go 約60ファイル、frontend 約50ファイル) ため、
書き換え漏れのリスクがある。`TestAssembledRoutesMatchGeneratedOpenAPI` と各層の
既存テストスイートを安全網として使い、機械的に漏れを検出しながら進める。
未リリース前提の一括変更であることを Motivation に明記し、リリース後に同じ手法
(baseline の無条件上書き) を使わないことを README で明示する。

## Completion

- **Completed At**: 2026-08-08
- **Summary**:
  [[wi-297-management-api-versioning-and-deprecation-policy]] が導入した「無版パス = v1 の
  エイリアス」設計を、未リリースであることを理由に「`/v1/` のみ・無版パスは存在しない」へ
  一本化した。対象は `/api/admin/*` (201 interface) と `/api/account/*` (31 interface) の
  bindings path のみで、`/api/branding`・`/api/auth/*`・protocol endpoint・ops endpoint は
  ADR-156 の版管理スコープ外のまま変更していない。
  SCL (232 binding)・Go (route 登録 16 ファイル + 参照テスト 44 ファイル)・frontend (呼び出し
  83 ファイル 332 箇所) を機械的に書き換え、`backend/shared/http/support_http/versioning.go`
  (`RegisterVersionAliases` エイリアス機構) と `tools/scl-to-openapi` のエイリアス複製ロジックを
  削除した (現行版が最初から `/v1/` を含むため、実行時・生成時どちらのエイリアス機構も不要になった)。
  機械的書き換えは 2 種類の見落としを生んだが、どちらも既存の安全網 (Go: 型検査でなく
  test suite の red、UI: `typecheck-ui` の型エラー) で即座に検出できた:
  1. `/realms/xxx/api/admin/...` のように prefix が付く Go の文字列 literal
     (最初の正規表現が文字列の先頭一致だけを見ていたため見逃した)。
  2. frontend の相対 import 文 `from '../../api/admin'` (HTTP パス文字列と同じ
     `/api/admin` という部分文字列を含むが別物)。
  `spec/idmagic.openapi.baseline.json` はこの WI 完了時点のスナップショットで無条件に
  上書きした (これは未リリース前提の一度限りの操作であり、リリース後は行わない)。
  ADR-156 は同一番号のまま本文を更新した (supersede はしない。まだリリースされていない
  決定の即時修正のため)。
- **Verification Results**:
  - `just check` / `just check-scl` / `just check-work-items` / `just check-ids` - passed
  - `just check-api-compat` - 再生成後の baseline に対して破壊的差分ゼロ
  - `just verify-go`(golangci-lint + race) - passed
  - `just verify-ui`(format/lint/typecheck/unit 523件/build) - passed
  - `just verify`(全体) - passed
  - 手動: `PERSISTENCE=memory go run ./backend/cmd/idmagic` で実サーバを起動し、
    `/realms/default/api/admin/settings` と `/realms/default/api/account/profile`
    (無版パス) が 404、`/realms/default/api/admin/v1/settings` と
    `/realms/default/api/account/v1/profile` (`/v1/` パス) が従来通り 401 (未認証) を
    返すことを curl で確認
