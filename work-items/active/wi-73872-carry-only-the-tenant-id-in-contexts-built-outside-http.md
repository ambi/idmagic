---
status: in_progress
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-09
priority: p3
depends_on: []
change_kind: refactor
spec_impact:
  kind: none
  reason: "HTTP を経ない処理が文脈に入れるテナントの表現を変えるだけで、各処理が使うテナントの ID、パスワードポリシーの解決結果、Cookie の名前とパス、要件、永続状態、発行するイベント、外向きの呼び出しは変えない。"
---

# HTTP を経ない文脈には ID だけのテナントの代わりにテナントの ID を入れる

## 動機

ジョブ、インポート、通知の配信、署名などの HTTP を経ない処理は、`tenancy.WithTenant(ctx, &tenancydomain.Tenant{ID: id}, "", "")` で文脈を作る。
着手時の本番コードに 13 箇所ある。
この `Tenant` は ID 以外のフィールドが空なので、文脈から `tenancy.Tenant(ctx)` でテナント全体を読む処理は、パスワードポリシーの上書きや `endpoint_style` などを「設定なし」として扱う。

設定の読み手は、パスワードポリシーの解決、`account_context_handler.go` と Tenancy が提供する Cookie の属性である。
これらは nil の場合を扱い、ID 以外の設定が空の Tenant と nil で結果は変わらない。
しかし、ジョブの経路でテナントの設定値を文脈から読む処理を足すと、上書きが黙って空として扱われ、型からもコメントからもそれに気づけない。
テナントを解決していない文脈で default テナントへ落ちる処理を調べたときに見つけた。

## 対象範囲

- HTTP を経ない処理がテナントの ID だけを文脈へ入れる手段を設け、13 箇所をそれに置き換える。
- そうした文脈では `tenancy.Tenant(ctx)` が nil を返し、`tenancy.TenantID(ctx)` は ID を返す形にする。

## 対象外

- テナントのパスワードポリシーをジョブの中で適用する変更。
  今はジョブからパスワードを検証する経路がない。
- テナントの解決の規則の変更。

## 設計

### 共有ライブラリの負債との整合

2026-10-18 の棚卸しで、wi-73604 が同じテナント ID の運搬を変更することを再確認した。
両項目で別々のキー、getter、未解決時の拒否を実装しないよう、共有側へ渡す ID と Tenancy が持つ解決済み設定の関係を一緒に決める。
この負債整理では文脈を変更せず、今の読み手が HTTP の設定を必要とする箇所も確認し直してから着手する。

### 採る案

利用者の承認により wi-73604 と同じブランチで実装する。
当初の別キー案は採らず、wi-73604 に記録した一つの格納先を使う。
ID の読取りと未解決時の拒否を共有側の一か所へ集め、ID だけの値に上書きすると古い設定と HTTP の位置情報も消える設計にする。

`backend/tenancy/ports/resolved_tenant.go` に `WithTenantID(ctx context.Context, tenantID string) context.Context` を加え、共有の一つのキーへ ID だけの値を格納する。

- `TenantID(ctx)` は、その格納先の現在の値から ID を読み、未設定または空の ID なら panic する。
- `Tenant(ctx)` は、解決済みの `Tenant` だけを返す。ID だけの文脈では nil を返す。
- テナントの設定値が要る処理は、nil を見て、リポジトリから引くか全体の既定値を使うかを自分で決める。

この形なら、ID だけの文脈を「設定のないテナント」と取り違える余地がなくなる。
現在の読み手はすべて nil の場合を扱っているので、結果は変わらない。

### 採らない案

| 案 | 採らない理由 |
| --- | --- |
| 11 箇所でテナントをリポジトリから引き、全体を文脈へ入れる | 各処理に `TenantRepository` の依存と読み取りが増える。停止中や削除済みのテナントの Job をどう扱うかという、振る舞いの判断も新たに要る |
| 現状のまま、コメントで注意を促す | 型が誤った使い方を許したままで、読み手がコメントを読む保証がない |

### 着手時に決めること

- 停止中のテナントの Job の扱いを変えないことを、採る案で確かめる（採る案は ID しか運ばないので変わらない見込みである）。

## 計画

1. `WithTenantID` と `TenantID`、`Tenant` の振る舞いを `backend/tenancy/ports/resolved_tenant_test.go` で先に固定する。
2. 現在の 13 箇所を置き換える。起票後に加わった所属変更と UserLifecycleCommands の文脈も対象にする。
3. `testing_tenant.Default` は ID だけの文脈にする。HTTP の位置が必要なテストには `ResolveDefaultAt` で明示的に解決済みの値を渡す。

## タスク

- [x] T001 [App] `WithTenantID` を加え、`TenantID` と `Tenant` の振る舞いを `TestTenantIDOnlyContext` と `TestTenantIDOnlyContextReplacesResolvedSettings` で固定する。`REQ-TENANCY-006` の空 ID の拒否も同じテストで検査する。
- [x] T002 [App] HTTP を経ない 13 箇所を `WithTenantID` に置き換える。
- [x] T003 [Verify] `test-go-package`、変更行の `test-go-mutation` と `verify` で文脈と呼び出し元を検査する。

## 検証

- `mise run test-go-package -- ./backend/tenancy/ports`
- `mise run test-go-mutation -- backend/tenancy/ports 2 off /tmp/wi-73604-ports-mutation-final.json 60b18f90d`
- `mise run verify`

## リスク

- ID だけの文脈で `Tenant(ctx)` が nil になるため、nil を想定せずにフィールドを読む処理があれば panic する。
  着手時に `tenancy.Tenant(` の読み手を洗い出し直す。

## 要約

wi-73604 と同じブランチで、`WithTenantID` と共有の一つの格納先を導入した。
HTTP を経ない13箇所と `testing_tenant.Default` は ID だけを運び、`Tenant` は nil を返す。
解決済みの文脈を ID だけの文脈へ上書きすると古い設定と HTTP の位置情報も消える。
`spec-diff` は規範差分なしを示した。対象 ID、パスワードポリシーの結果、Cookie、永続状態とイベントは変えていない。
停止中の Job の扱いと、背景処理のために設定を追加解決する変更は対象外とした。
