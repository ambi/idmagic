---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-11-28
priority: p2
group: design-conformance
depends_on: []
change_kind: maintenance
spec_impact: { kind: none, reason: "期限、再送、失効の判定の規則と、記録する時刻の意味は変えない。判定と記録に使う時刻を、アダプターが実時計から読まずに呼び出し側から受け取るだけで、HTTP 契約、永続状態の形、イベントは変えない。" }
---

# ストレージのアダプターが実時計を読まず、時刻を呼び出し側から受け取る

## 動機

呼び出し側が判定の時刻を持っているのに、ストレージのアダプターが期限切れを実時計で判定する誤りが繰り返している。
テストは呼び出し側に記録した時刻や固定の時刻を渡すので、書いた直後は通り、日付が進むと失敗する。

- c1ceb0613：信頼済み端末の一覧で、ユースケースは呼び出し側の時刻で判定するのに、リポジトリが絶対期限を実時計で判定していた。
- 4cb1d8378：HTTP ハンドラーが `time.Now()` で刻んだ実行時刻を、テストが固定の日付で処理し、書いた翌日から失敗した。
- [wi-91138](../done/wi-91138-compare-federation-replay-expiry-with-the-validation-clock.md)：連携の再送の予約が実時計で期限を判定し、記録から 10 分後に SAML の再送テストが失敗した。

規則は、ドメインが時刻を読まないこと（`.golangci.yml` の forbidigo で `domain/` だけに適用）と、時刻の分解能をそろえること（[テストダブル](../../docs/development/testing.md#テストダブル)）までしかない。
[作用の境界](../../docs/design/application/design-guidelines.md#作用の境界)もドメインの計算だけを扱い、アダプターが実時計を読むことを止める規則も検査もない。

2026-11-28 に `rg -c 'time\.Now\(' backend --glob '**/db_memory/**/*.go' --glob '**/db_postgres/**/*.go' --glob '!*_test.go'` で数えると、本番コードに 31 か所ある。
期限を実時計と比べるものとして、oauth2 の PAR、アクセストークンの拒否リスト、同意、承認要求、WebAuthn のセッション、ログインのセッション、tokenvault の試行の削除、連携の再送の予約がある。
SQL でも `federation.sql` の `expires_at<=now()` が同じ判定をする。
`now = time.Now()` を時刻のゼロ値のときだけ補うもの（SAML と oauth2 の再送、ジョブ、署名鍵）と、`updated_at = now()` のような記録用の時刻もある。

## 対象範囲

- [作用の境界](../../docs/design/application/design-guidelines.md#作用の境界)の判定基準に、ストレージのアダプターが判定にも記録にも実時計を読まず、時刻をポートの引数で受け取る行を足す。
- `.golangci.yml` の `time.Now` の禁止を、`db_memory/` と `db_postgres/` の本番コードへ広げる。
- SQL の問い合わせで、期限や順序の判定に `now()` を使うことを拒否する検査を足す。
- 既存の箇所を、ポートに時刻を足して呼び出し側から渡す形へ直す。ゼロ値のときに実時計で補う分岐も取り除く。

## 対象外

- [wi-91138](../done/wi-91138-compare-federation-replay-expiry-with-the-validation-clock.md) が直す連携の再送の予約。main の `test-go` を直すために先に扱い、この項目はその形に合わせる。
- `handlers_http` などストレージ以外のアダプターの `time.Now`。HTTP の入口がユースケースへ渡す時刻を作るのは作用を編成する側の役割である。
- 時刻の分解能の規則。既存の規則を変えない。

## 設計

### 記録用の時刻の扱い

| 案 | 判断 |
| --- | --- |
| A. 判定にも記録にも、アダプターは実時計を読まない。`updated_at` も引数で受け取る | 採る。「アダプターは時計を読まない」の一つの規則で検査でき、判定と記録の区別をレビューに任せずに済む |
| B. 判定だけを禁止し、記録用の `time.Now()` と `now()` は許す | 採らない。判定か記録かは式の形から機械では決められず、例外の注記が増える |

SQL の `updated_at = now()` を引数に替えるかは、案 A を SQL へ当てはめるときに、検査が判定の `now()` と記録の `now()` を区別できるかで決める。
区別できなければ、SQL も記録を含めて引数へ替える。

### 検査の置き場所

Go は既存の forbidigo の設定へ path を足す。
SQL は `.sql` を読む検査が既にあるかを `tools/check` で確かめ、あればそこへ、なければ `check-repository` の検査として足す。

## 計画

1. 規則を作用の境界へ足す。
2. forbidigo の範囲を広げ、検出される箇所の一覧を得る。
3. 一覧をモジュールごとに直し、ポートの変更ごとに局所検査を通す。
4. SQL の検査を足し、同じく直す。

## タスク

- [ ] T001 [Docs] 作用の境界の判定基準に、ストレージのアダプターの時刻の行を足す。
- [ ] T002 [Tooling] forbidigo の `time.Now` の禁止を `db_memory/` と `db_postgres/` へ広げる。
- [ ] T003 [Adapters] 検出された箇所を、ポートの引数で時刻を受け取る形へ直す。
- [ ] T004 [Tooling] SQL の判定の `now()` を拒否する検査を足し、該当箇所を直す。

## 検証

- `mise run lint-go`：広げた禁止が既存の箇所をすべて検出し、修正後に通ること。
- `mise run test-go`：ポートの変更が既存のテストを壊さないこと。DB を使うテストは AGENTS.md の実行方法に従う。
- 検出の RED は、修正前に `lint-go` が箇所を挙げて失敗することで観測する。

## リスク

ポートに時刻を足すと、呼び出し側が渡し忘れたゼロ値で判定が常に期限内になり得る。
ゼロ値を拒否するか、時刻を必須の位置に置いて呼び出し側のテストで確かめる。
