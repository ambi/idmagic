---
status: pending
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
本番コードに 11 箇所ある。
この `Tenant` は ID 以外のフィールドが空なので、文脈から `tenancy.Tenant(ctx)` でテナント全体を読む処理は、パスワードポリシーの上書きや `endpoint_style` などを「設定なし」として扱う。

現在 `tenancy.Tenant(ctx)` を読むのは、`ResolveTenantPolicy`、`account_context_handler.go`、Cookie の名前と属性を決める関数だけであり、どれも HTTP の経路からだけ呼ばれる。
したがって今は誤った結果は出ない。
しかし、ジョブの経路でテナントの設定値を文脈から読む処理を足すと、上書きが黙って空として扱われ、型からもコメントからもそれに気づけない。
テナントを解決していない文脈で default テナントへ落ちる処理を調べたときに見つけた。

## 対象範囲

- HTTP を経ない処理がテナントの ID だけを文脈へ入れる手段を設け、11 箇所をそれに置き換える。
- そうした文脈では `tenancy.Tenant(ctx)` が nil を返し、`tenancy.TenantID(ctx)` は ID を返す形にする。

## 対象外

- テナントのパスワードポリシーをジョブの中で適用する変更。
  今はジョブからパスワードを検証する経路がない。
- テナントの解決の規則の変更。

## 設計

### 採る案

`backend/tenancy/context.go` に `WithTenantID(ctx context.Context, tenantID string) context.Context` を加え、テナントの ID を `Tenant` とは別のキーで運ぶ。

- `TenantID(ctx)` は、解決済みの `Tenant` の ID、なければ `WithTenantID` の ID を返し、どちらもなければ panic する。
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

1. `WithTenantID` と `TenantID`、`Tenant` の振る舞いを `backend/tenancy/context_test.go` で先に固定する。
2. 11 箇所を置き換える。
3. `testing_tenant.Default` を ID だけの文脈にするかを、テストが `Tenant(ctx)` を読むかで判断する。

## タスク

- [ ] T001 [App] `WithTenantID` を加え、`TenantID` と `Tenant` の振る舞いをテストで固定する。
- [ ] T002 [App] HTTP を経ない 11 箇所を `WithTenantID` に置き換える。
- [ ] T003 [Verify] 変更を検証する。

## 検証

- `mise run test-go-package -- ./backend/tenancy`
- `mise run test-go-mutation -- backend/tenancy`
- `mise run verify`

## リスク

- ID だけの文脈で `Tenant(ctx)` が nil になるため、nil を想定せずにフィールドを読む処理があれば panic する。
  着手時に `tenancy.Tenant(` の読み手を洗い出し直す。
