---
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-10
priority: p2
depends_on: []
change_kind: refactor
evidence_policy: risk-based-v4
documentation_impact:
  level: none
  reason: クレームの発行の関数を置くパッケージと、属性スキーマを読む場所だけを変えるので、発行するクレーム、開示の下限、HTTP の応答は変わらず、リリースの読者へ知らせることがない。
  references: []
initial_context:
  specification: []
  typespec: []
  source:
    - backend/claimmapping/domain/policy.go
    - backend/claimmapping/domain/attribute_defs.go
    - backend/claimmapping/domain/floor.go
    - backend/claimmapping/domain/claims.go
    - docs/design/architecture/logical.md
    - docs/modules/claim-mapping/design/architecture.md
    - tools/check/boundary-debt.json
  tests:
    - backend/claimmapping/domain/floor_test.go
  stop_before_reading: [frontend, spec]
spec_impact: { kind: none, reason: "クレームの発行と開示規則の検証を呼ぶ経路を、ClaimMapping の公開パッケージへ変えるだけである。発行するクレーム、開示の下限、HTTP の応答、ドメインイベントは変えない。" }
---

# クレームの発行と開示規則の検証を ClaimMapping の公開操作にする

## 動機

`tools/check/boundary-debt.json` の `private-import` のうち 11 件は、Application、OAuth2、Saml、WsFederation から `backend/claimmapping/usecases` への依存である。
四つのプロトコルは、同じ開示規則でクレームを発行するために `IssueClaimsWithFloor`、`ResolveUserAttributes`、`ResolveTenantAttributeDefs`、`ValidateClaimReleaseRules`、`ClaimIssuanceResult`、`TenantAttributeSchemaRepo` を直接使っている。
ClaimMapping の非公開の実装に依存しているので、ClaimMapping の内部を変えると四つのプロトコルへ変更が波及する。

## 対象範囲

- クレームの発行、利用者の属性の解決、属性の定義の解決、開示規則の検証を、ClaimMapping の公開パッケージの操作として公開する。
- 四つのプロトコルの呼び出しを公開操作の経由へ直す。
- 解消した違反 ID を台帳から消す。

## 対象外

- 開示規則と発行するクレームの変更。
- ClaimMapping の残りのパッケージの `internal/` への移動。[残るモジュールの移行](../active/wi-75383-move-remaining-modules-behind-go-internal-and-retire-legacy.md)で行う。

## 設計

[境界の負債の順位付け](wi-33994-reinventory-the-remaining-boundary-debt.md)の判断を引き継ぐ。

- D1：クレームの開示規則と開示の下限（floor）の所有者は ClaimMapping なので、ClaimMapping を担当とする。
- D3：四つのプロトコルが使う操作は、発行、属性の解決、定義の解決、規則の検証に限られる。この範囲だけを公開し、属性スキーマの保存先は公開しない。

| 案 | 判断 |
| --- | --- |
| 発行と検証を公開操作にする | 採る。開示の下限を通らずにクレームを組み立てる経路をなくせる |
| `TenantAttributeSchemaRepo` などの保存先を公開し、各プロトコルが組み立てる | 採らない。開示の下限を迂回できる公開範囲が残る |
| ClaimMapping を各プロトコルへ分けて複製する | 採らない。同じ規則を四か所で保つことになる |

公開パッケージの名前と、`TenantAttributeSchemaRepo` の型を公開操作の引数から外せるかは、着手時に `ports` と専用の公開パッケージを比べて決める。

### 置き場所の決定

`usecases` の関数は、`ResolveTenantAttributeDefs` が属性スキーマの保存先を読むほかは、入力だけで結果が決まる計算である。
`structure.md` は決定論的な計算を `domain` に置くと定めるので、計算を `domain` へ移す。
ClaimMapping は `legacy` の公開方式なので、`domain` と `ports` が公開パッケージになる。

| 現在の `usecases` | 移した後 | 公開 |
| --- | --- | --- |
| `IssueClaimsWithFloor`、`ValidateClaimReleaseRules`、`ResolveUserAttributes`、`ClaimIssuanceResult`、`ClaimReleaseDeniedError`、属性の鍵の定数 | `domain` | する |
| `IssueClaims`（下限を通らない発行） | `domain` の非公開の関数 `issueClaims` | しない。公開すると、開示の下限を迂回してクレームを組み立てる経路が残る。WsFederation の `domain` に置かれていた射影の規則のテストは、ClaimMapping の `domain` の内部テストへ移した |
| `IsAttributeReleasable`、`IsReservedClaimType` | `domain` | する。判定を返すだけで、下限を迂回してクレームを組み立てる経路にならない。下限のテストが直接使う |
| `TenantAttributeSchemaRepo` | `ports` | する。プロトコルのモジュールの依存の型として使う |
| `ResolveTenantAttributeDefs(ctx, tenantID, repo)` | 保存先を読む部分は `ports.ResolveTenantAttributeDefs`、組み込みの定義との合成は `domain.MergeTenantAttributeDefs(schema)` | する。呼び出し側に「保存先がなければ組み込みの定義だけ」という判断を複製しないため、保存先を読む部分を一か所に残す |

| 案 | 判断 |
| --- | --- |
| 計算を `domain` へ、保存先の型を `ports` へ移す | 採る。`legacy` の公開の規則のままで公開でき、`domain` は作用を持たない |
| ClaimMapping を `internal` の公開方式にし、`usecases` を公開パッケージとして宣言する | 採らない。下限を通らない `IssueClaims` まで公開され、`internal` への移行は[残るモジュールの移行](../active/wi-75383-move-remaining-modules-behind-go-internal-and-retire-legacy.md)が扱う |

`usecases` のパッケージは空になるので消す。
クレームの発行の設計（`docs/modules/claim-mapping/design/architecture.md`）のパッケージの表を、移した後の構成に合わせる。

### 証拠

テストの参照先のパッケージを書き換えるので、振る舞いを保つ変更には当たらない。

| 証拠 | 内容 |
| --- | --- |
| 受け入れ RED | N/A: 製品の振る舞いを変えない配置の変更であり、対応する REQ がない。代替として、台帳から ClaimMapping への `private-import` を先に消し、`mise run check-boundaries` が未記録の違反として失敗することを確かめる |
| 単体 RED | N/A: 振る舞いを変えないので、実装前に失敗する単体テストはない。下限のテスト（`floor_test.go`）は表明を変えずに `domain` へ移す |
| 変更耐性 | `mise run test-go-mutation -- backend/claimmapping/domain` で、移した関数への変異を移したテストが検出することを確かめる |

## タスク

- [x] T001 [Design] 公開する操作の一覧と引数の型を決める。
  - 設計の「置き場所の決定」に記録した
- [x] T002 [App] 公開操作を置き、四つのプロトコルの呼び出しを書き換える。
  - `usecases` の 4 ファイルを `domain` へ移して `usecases` のパッケージを消し、保存先の型と読み取りを `ports` に置いた
  - 呼び出し側 21 ファイルの参照を `claimdomain` と `claimports` へ書き換えた
  - `floor_test.go` は表明を変えずに `domain` の外部テストへ移し、WsFederation の `domain` にあった `IssueClaims` のテストは、ClaimMapping の `domain` の内部テスト `issue_claims_test.go` へ移した
  - `MergeTenantAttributeDefs` のテストを足した
  - 検査: `mise run lint-go`、`mise run test-go-changed`
- [x] T003 [Tooling] 解消した違反 ID を台帳から消す。
  - ClaimMapping への `private-import` の 4 項目（11 件）を消した。台帳は 128 件から 117 件になった
  - `module-cycle` の X->ClaimMapping と、`tokens_jose` からの `shared-dependency` は残る
  - 検査: `mise run check-boundaries`、`mise run check-boundary-debt-ratchet -- main`
- [x] T004 [Verify] 変更を検証する。

## 検証

- `mise run check-boundaries`
- `mise run check-boundary-debt-ratchet`
- `mise run verify`

## リスク

- 発行の経路を書き換えるときに、開示の下限を適用しない経路が生じるおそれがある。
  各プロトコルの発行のテストが、下限を固定しているかを着手時に確かめ、固定していなければ特性化テストを先に置く。

## 完了

- **完了日**: 2026-10-10
- **要約**:
  `mise run spec-diff` は規範の差分なしを報告する。
  ClaimMapping の入力だけで決まる計算（`IssueClaimsWithFloor`、`ValidateClaimReleaseRules`、`ResolveUserAttributes`、属性の定義の合成）を `usecases` から公開パッケージの `domain` へ移し、属性スキーマの保存先の型と読み取りを `ports` に置いた。
  下限を通らない発行は `domain` の非公開の `issueClaims` にし、ほかのモジュールからは `IssueClaimsWithFloor` しか呼べない。
  `usecases` のパッケージは消した。
  OIDC、SAML、WS-Federation、Application の呼び出し側は、関数の中身を変えず修飾子だけを書き換えた。
  境界の負債の台帳から ClaimMapping への `private-import` の 11 件を消し、台帳は 128 件から 117 件になった。
  発行するクレーム、開示の下限、HTTP の応答は変えていない。
- **受け入れ RED の証拠**:
  - **テスト**: `mise run check-boundaries`
  - **要件**: N/A: 規範上の振る舞いを変えない配置の変更であり、対応する REQ がない。
  - **観測した失敗**: 台帳から ClaimMapping への `private-import` の 4 項目を先に消すと、`mise run check-boundaries` が 11 件を `violation(s) are absent from boundary-debt.json` として失敗した。移した後は通った。
  - **検出できる理由**: 配置を変える作業なので、製品の振る舞いの RED は存在しない。代わりに、リファクタリングを必要にした構造ゲートが変更の前後の違いを観測した。
- **単体 RED の証拠**:
  - **テスト**: `floor_test.go` の下限のテスト（`REQ-CLAIMMAPPING-001` を引く）、`issue_claims_test.go`、`attribute_defs_test.go`
  - **要件**: N/A: 規範上の振る舞いを変えない配置の変更であり、対応する REQ がない。
  - **観測した失敗**: 該当なし。振る舞いを変えないので、実装前に失敗する単体テストはない。下限のテストと射影の規則のテストは、表明を変えずに移して通った。
  - **検出できる理由**: 移した関数は中身を変えておらず、移したテストが同じ表明でそれを確かめる。リスクの節が求めた「下限を適用しない経路」については、呼び出し側の変更が修飾子だけであり、下限を通らない `issueClaims` をモジュールの外から呼べなくしたことで、新しい経路は生じない。
- **変更耐性の結果**:
  `mise run test-go-mutation -- backend/claimmapping/domain` は、テストが実行する 14 件をすべて検出した。
  `ResolveUserAttributes` と値の文字列化の 16 件は、ClaimMapping 自身のテストから実行されない（`NOT COVERED`）。移す前の `usecases` でも同じで、WsFederation などほかのモジュールのテストが実行している。今回は名前と形を変えた `MergeTenantAttributeDefs` のテストだけを足した。
- **検証結果**:
  - `mise run check-boundaries` - 成功
  - `mise run check-boundary-debt-ratchet -- main` - 成功（private-import 48 → 37）
  - `mise run lint-go` - 成功
  - `mise run test-go-changed` - 成功
  - `mise run verify` - 成功
  - `mise run test-ui-e2e` - 成功（39 件）
