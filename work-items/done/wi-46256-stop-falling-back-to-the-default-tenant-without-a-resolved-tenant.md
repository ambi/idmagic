---
status: completed
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-10-01
priority: p2
depends_on: []
change_kind: bugfix
evidence_policy: risk-based-v4
documentation_impact:
  level: none
  reason: 本番でテナントを解決していない文脈から `tenancy.TenantID` へ届く経路はなく、HTTP の応答、ジョブの結果、永続状態、設定、運用手順のどれも変わらないので、リリースの読者へ知らせる差分がない。
  references: []
initial_context:
  specification: [docs/domain/tenancy/resolution/README.md#REQ-TENANCY-006]
  typespec: []
  source:
    - backend/tenancy/context.go
    - backend/shared/http/support_http/tenant_middleware.go
    - backend/shared/http/server_http/routes.go
    - backend/jobs/usecases/runner.go
    - backend/sharedsignals/usecases/revocation.go
    - backend/idmanagement/deps_http/deps.go
    - backend/cmd/internal/bootstrap
  tests:
    - backend/tenancy/context_test.go
  stop_before_reading: [frontend]
affected_spec:
  - { path: docs/domain/tenancy/resolution/README.md, requirement: REQ-TENANCY-006, impact: conforms }
primary_use_cases:
  - id: refuse-unresolved-tenant-context
    requirement: REQ-TENANCY-006
    observable_result: テナントを解決していない文脈からテナントの ID を読むと、default テナントの ID を返さずに panic し、呼び出し元はどのテナントのデータも読み書きしない。
    boundary: unit
    test: { path: backend/tenancy/context_test.go, name: TestTenantIDPanicsWithoutResolvedTenant, task: test-go-race }
    fault_model: 文脈にテナントがないとき、`tenancy.TenantID` が default テナントの ID を返して処理を続けさせる。
---

# テナントを解決していない文脈で default テナントへ落ちない

## 動機

テナントの解決は、どの経路にも一致しないリクエストを default テナントへフォールバックさせないと定める（REQ-TENANCY-006）。
しかし、リクエストの文脈からテナント ID を読む `tenancy.TenantID` は、文脈にテナントがないとき default テナントの ID を返す。
本番コードには、この関数の呼び出しが 254 箇所ある。

テナントを解決するミドルウェアを通らない経路や、解決の後に文脈を作り直す処理がこの関数を呼ぶと、その処理は default テナントのデータを読み書きする。
Tenancy の既存コードを書き起こしたときに見つけたが、観測できる境界での結果として規則に書くには、呼び出し元の経路を調べる必要がある。

## 対象範囲

- `tenancy.TenantID` の呼び出し元のうち、テナントを解決していない文脈で呼ばれ得るものを洗い出す。
- 文脈にテナントがないときに失敗させる形へ変え、default テナントへ暗黙に落ちる経路をなくす。
- 意図して default テナントを使う経路は、default テナントを明示して呼ぶ。

## 対象外

- テナントの解決の規則そのものの変更。
- `tenancy.TenantID` の戻り値に error を加え、254 箇所の呼び出し元へ伝播させる形。
  到達する経路がないので、どの呼び出し元にも実行されない分岐が増えるだけである。
- ジョブの文脈が ID だけのテナントを運ぶため、`ResolveTenantPolicy` がジョブの中ではテナントのパスワードポリシーの上書きを適用しない疑い。
  ユーザーのインポートが管理者の設定したパスワードを検証する経路に当たるかを含め、別の記録で調べる。
- `mise run test-go-mutation -- backend/authentication/password/usecases` が残した、`request_password_reset.go` の 91 行目の算術の変異。
  今回触れていないコードである。

## 設計

### 洗い出しの結果

呼び出し元を二つの方法で調べた。

| 方法 | 対象 | 結果 |
| --- | --- | --- |
| 実測 | 全 Go テスト（PostgreSQL を含む）。文脈にテナントがないときの呼び出しスタックを一時的に記録した | 1012 回。すべてテストの関数から始まり、本番コードが起こしたゴルーチンから始まるものはない |
| 静的な確認 | HTTP の経路、ジョブ、起動処理、反応器 | 下の表のとおり、どれもテナントを解決するか明示する |

| 経路 | テナントの与え方 |
| --- | --- |
| HTTP | テナントに属する経路は `ResolveHostTenant` か `ResolvePathTenant` の後ろにだけ登録される。`/health` などの運用の経路は `TenantID` を呼ばない |
| ジョブ | `jobs.Runner.execute` と各インポート、エクスポートのハンドラーが Job の `TenantID` で文脈を作る |
| バッチと保持期間の掃除 | `idmagic-batch` と `bootstrap/retention.go` がテナントごとに文脈を作る |
| 起動処理とシード | リポジトリへ `tenancydomain.DefaultTenantID` を引数で渡し、文脈のテナントを読まない |
| 反応器と通知 | `context.Background()` から作り直すが、イベントの `TenantID` を引数で渡すか、`sign_jose.Signer` と `securitynotification.Dispatch` が文脈を作り直す |

既定テナントへ落ちて動いていたのはテストだけであり、約 80 のテストファイルが `context.Background()` のままユースケース、リポジトリ、またはミドルウェアを通さないハンドラーを呼んでいた。

### 失敗の形

`func TenantID(ctx context.Context) string` のシグネチャは変えず、文脈にテナントがないか、テナントの ID が空なら panic する。

- テナントのない文脈がテナントに属する処理へ届くのは配線の誤りであり、入力から生じる失敗ではない。
  設計ガイドラインが値として返すことを求めるのは入力から生じうる失敗なので、この不変条件の違反には panic を使う。
- HTTP では `support_http` の recover が 500 にし、どのテナントのデータも読み書きしない。
- `""` を返す形は採らない。
  メモリ実装のリポジトリは空の ID の下へ保存できてしまい、確実に失敗する保証がない。

テストは `backend/tenancy/testing_tenant` の `Default` で default テナントを明示する。
ミドルウェアを通さずにハンドラーを登録するテストは、同じパッケージの `ResolveDefault` を組み立てに挟む。
このパッケージは本番の入口から import されないので、本番コードに数えない。

### 実装で見つけた分岐の分類

`password.usecases.ResolveTenantPolicy(ctx, repo)` は、文脈にテナントがないとき `tenancy.TenantID` の返す ID でリポジトリを引いていた。
その ID は常に default テナントの ID なので、この分岐は default テナントへの暗黙のフォールバックそのものだった。
本番の 3 つの呼び出し元（パスワードのリセットと変更、ログイン後の有効期限の判定、管理者によるユーザーの作成）は、どれもテナントを持つ文脈から呼ぶ。

(c) 実装を直すと分類し、分岐とリポジトリの引数を削除して `ResolveTenantPolicy(ctx context.Context) PasswordPolicySnapshot` にした。
テナントのない文脈では、テナントのデータを読まずに全体の既定値を返す。
委譲だけになった `resolvePasswordPolicy` も削除し、`tools/check/boundary-debt.json` から解消した依存の記録を除いた。

設計文書では、[テナントの解決の設計](../../docs/modules/tenancy/resolution/design.md)のリクエストコンテキストの責務に、テナントのない文脈から ID を読むと panic することを書いた。

## 計画

1. テストへテナントの文脈を明示する変更を、本番コードを変えない先のコミットにする。
2. `TestTenantIDPanicsWithoutResolvedTenant` で RED を確かめ、`tenancy.TenantID` を panic させる。

## タスク

- [x] T001 [Plan] 呼び出し元を洗い出す。
- [x] T002 [Test] テストが default テナントを明示する。
  N/A: 本番コードを変えないテストの変更である。
  代わりに、T003 の変更を入れた状態の `mise run test-go` で 30 パッケージの失敗を観測し、この変更だけを持つ状態では変更前の本番コードのまま `mise run test-go` が通ることを確かめた。
- [x] T003 [App] 暗黙の default テナントをなくす。
  REQ-TENANCY-006、`TestTenantIDPanicsWithoutResolvedTenant`（`mise run test-go-test -- ./backend/tenancy TestTenantIDPanicsWithoutResolvedTenant`）。
- [x] T004 [Verify] 変更を検証する。

## 検証

- `mise run test-go-mutation -- backend/tenancy`
- `mise run verify`

## リスク

- 現在 default テナントへ落ちることで動いている処理が、失敗に変わる。
  洗い出しで本番の経路がないことを確かめた。
  テストが通らない経路が残っていれば、HTTP では 500、ジョブでは worker の停止として現れる。

## 完了

- **完了日**: 2026-10-09
- **要約**:
  `mise run spec-diff` は main に対する規範仕様の差分を報告しない。
  REQ-TENANCY-006 は変えず、実装をそれに合わせた。
  `tenancy.TenantID` は、文脈にテナントがないか ID が空のとき default テナントの ID を返さずに panic する。
  `password.usecases.ResolveTenantPolicy` から default テナントをリポジトリで引く分岐を除いた。
  テストは `testing_tenant` で default テナントを明示する。
  洗い出しの結果、本番の経路にテナントのない文脈から `TenantID` を呼ぶものはなく、HTTP の応答、ジョブの結果、永続状態は変わらない。
- **主要ユースケースの証拠**:
  - id: refuse-unresolved-tenant-context
    red: 実装前に `TestTenantIDPanicsWithoutResolvedTenant` が、テナントがない場合と ID が空の場合の両方で `TenantID() = "00000000-0000-4000-8000-000000000000", want panic` と失敗した。
    fault_injection: 判定から `tenant.ID != ""` を除くと、`TestTenantIDPanicsWithoutResolvedTenant/ID_が空` が `TenantID() = "", want panic` で失敗した。
- **変更耐性の結果**:
  `mise run test-go-mutation -- backend/tenancy` は 3 件の変異をすべて検出した。
  `mise run test-go-mutation -- backend/authentication/password/usecases` は 51 件中 48 件を検出し、生き残った 1 件は今回触れていない `request_password_reset.go` の 91 行目の除算を乗算へ変えるもので、対象外へ記録した。
  変異器が表せない故障として、ID が空のテナントを受け入れる判定を手で入れ、上のとおり検出した。
  T002 のテストの変更は、本番コードを変更前に戻した worktree で `mise run test-go` が通ることを確かめ、本番の振る舞いを変えないことを確認した。
- **検証結果**:
  - `mise run verify` - 成功
  - `mise run test-ui-e2e` - 成功（39 件、テナントのない文脈による panic のログなし）
