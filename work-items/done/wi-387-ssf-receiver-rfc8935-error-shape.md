---
status: completed
authors: [tn]
risk: low
created_at: 2026-08-19
change_kind: bugfix
priority: p3
depends_on: []
evidence_policy: risk-based-v2
initial_context:
  specification: [docs/contexts/sharedsignals/standards.md]
  typespec:
    - IdMagic.Contract.SecurityEventRejectedError
    - IdMagic.Contract.SecurityEventTokenTooLargeError
    - IdMagic.Contract.SecurityEventStreamNotFoundError
  source:
    - backend/sharedsignals/handlers_http/routes.go
  tests:
    - backend/sharedsignals/handlers_http/routes_test.go
  stop_before_reading: [frontend, backend/oauth2, backend/idmanagement]
affected_spec:
  - { path: spec/contexts/sharedsignals/models.tsp, symbol: IdMagic.Contract.SecurityEventRejectedError }
  - { path: spec/contexts/sharedsignals/models.tsp, symbol: IdMagic.Contract.SecurityEventTokenTooLargeError }
  - { path: spec/contexts/sharedsignals/models.tsp, symbol: IdMagic.Contract.SecurityEventStreamNotFoundError }
---

# SharedSignals 受信エンドポイントのエラー本体を RFC 8935 §2.3 の形にする

## Motivation

`POST /ssf/streams/{stream_id}/events` の拒否応答は `writeSecurityEventReceiverError` が `{"error": code, "message": description}` を返す。RFC 8935 §2.3 が定めるのは `{"err": ..., "description": ...}` である。この接点は「標準が形を定めるから Problem Details を適用しない」という理由で例外扱いにしているのに、その標準の形になっていない。

`wi-335` の Risk Notes がこれを記録し、`wi-382` は「契約が実装に合わせる」という原則の外にあるとして Out of Scope に置いたうえで、実装がいま返している `{error, message}` のほうを契約に書いた。契約を先に正しい形にすると、どちらが正なのか読めなくなるためである。

## Scope

- `writeSecurityEventReceiverError` の出力を `{"err": ..., "description": ...}` に変える。
- `SecurityEventRejectedError` / `SecurityEventTokenTooLargeError` / `SecurityEventStreamNotFoundError` の 3 モデルを新しい形に合わせ、`@doc` から「RFC に追従できていない」という注記を外す。
- 受信側の適合を確認する試験を足す。

## Out of Scope

- 送信側 (transmitter) の SET 形式。RFC 8417 に従っており変更しない。
- 管理 API 側のエラー本体。Problem Details のままとする。
- `SecurityEventStreamNotFoundError` (404) がこの接点から返らないこと。`ReceiveSecurityEvent` は stream を引けない場合も `ErrSecurityEventRejected` を返すので、`writeReceivedSecurityEventError` の `ErrStreamNotFound` 分岐へは到達しない (`ErrStreamNotFound` を返すのは `admin_streams.go` だけである)。本 work item は 3 モデルの形を揃えるところまでで、到達しない応答の宣言そのものは [[wi-386-declared-status-code-audit]] が持つ。**なお到達しないこと自体は欠陥ではない** —— stream の存在を呼び出し側へ漏らさない fail-closed の設計であり、証拠検査でその性質を固定した。

## Design

`writeSecurityEventReceiverError` の欄名を `error` / `message` から RFC 8935 §2.3 の `err` / `description` へ変え、3 モデルの欄名と `@doc` を揃える。この接点を Problem Details の外へ置いている根拠が「標準が形を定めるから」である以上、標準の形になっていなければ例外扱いの理由が成り立たない。`docs/api-rules.md` は既に「各標準が定めるエラーレスポンスを返す」と書いており、文書は正しく実装だけが取り残されていた。

検討した代替案:

- **両方の欄を並べて出す (`err` と `error` を両方書く)**: 既存の読み手を壊さずに済むが、RFC 8935 §2.3 が定めていない欄を足すことになり、標準への適合を目的とする変更の意味が失われる。移行期間を設けるなら work item を分けて期限を決めるべきで、既定として恒久的に残す形ではない。採用しない。
- **接点を版で分ける**: 受信エンドポイントの URL は transmitter 側に登録済みで、版を切ると相手全員に再登録を求めることになる。欄名の是正より重い。採用しない。

## Verification

- `mise run verify`
- 手動確認: 不正な SET を送ると `{"err": ..., "description": ...}` が返る。
- 手動確認: 生成された OpenAPI の 400 / 404 / 413 が同じ形を記述している。

## Risk Notes

`{error, message}` を読んでいる外部 transmitter があれば壊れる。この接点は外部から呼ばれるため、変更前に配備先で実際に受信している相手を確認する。相手が居ない配備では単純な置換で済む。

## Tasks

- [x] T001 [Test] 受信側が RFC 8935 §2.3 の形を返すことの検査を RED で置いた。
  `TestReceiveSecurityEventUsesRFC8935ErrorShape`。
- [x] T002 [Spec] 3 モデルの欄名を `err` / `description` にし、`@doc` から追従できていない旨の注記を外した。
- [x] T003 [Adapters] `writeSecurityEventReceiverError` の出力を `{"err": ..., "description": ...}` に変えた。
- [x] T004 [Verify] `mise run verify` と `mise run check-api-compat` (ベースライン更新後)。

## Completion

- **Completed At**: 2026-08-30
- **Summary**:
  `mise run spec-diff` は `no normative specification change against main` を返す。規範シナリオも標準行も動いておらず、変わったのは受信エンドポイントが返すエラー本体の欄名と、それを記述する 3 モデルである。`{"error": ..., "message": ...}` から RFC 8935 §2.3 が定める `{"err": ..., "description": ...}` へ変えた。この接点を Problem Details の外に置いている根拠が「標準が形を定めるから」であり、`docs/api-rules.md` も「各標準が定めるエラーレスポンスを返す」と書いていたので、文書が正しく実装だけが取り残されていた状態を解いたことになる。
  **これは実際に線を越える変更である。** wi-397 が直した「返らない応答の宣言」とは違い、サーバーは本当に `{error, message}` を送っていた。`check-api-compat` は 6 件の破壊的変更を報告し (400 / 404 / 413 それぞれの `error` と `message` の削除)、この 6 件だけを凍結し直した。
- **Acceptance RED Evidence**:
  - **Test**: `TestReceiveSecurityEventUsesRFC8935ErrorShape` (`backend/sharedsignals/handlers_http/routes_test.go`)
  - **Requirement**: N/A: 該当する `REQ-` シナリオは無い。規範は RFC 8935 §2.3 と、それを受けた `docs/api-rules.md` の「標準が定めるエラーレスポンスを返す」という宣言である。
  - **Observed Failure**: 2 つの部分試験がいずれも `body={"error":"...","message":"..."} must carry err and description (RFC 8935 §2.3)` で失敗した。
  - **Detection Reason**: 新しい欄 (`err` / `description`) が在ることと、古い欄 (`error` / `message`) が無いことを対で主張する。前者だけでは、両方の欄を並べて出す実装 —— 標準に無い欄を足したままの状態 —— が通ってしまう。実測でもその変異は検出された (下記)。状態コードも併せて固定するので、形だけ直して分類を崩す実装は分かれる。
- **Unit RED Evidence**:
  - **Test**: 同上 (`writeSecurityEventReceiverError` は 1 つの小さな写像で、HTTP の境界より内側に独立した計算を持たない)
  - **Requirement**: N/A: 上と同じ。
  - **Observed Failure**: 同上。単体の境界が別に立たないので、代わりに `mise run check-api-compat` が実装と契約の食い違いを落とした。契約側だけを直して実装を放置すると 6 件の破壊的変更が残り、実装側だけを直して契約を放置しても同じ 6 件が残る。両方を揃えたときにだけ通る。
  - **Detection Reason**: 内側に単体境界が無いことを理由に検査を省くのではなく、契約と実装の一致を落とす別の検査を代替として使った。`check-api-compat` は生成された OpenAPI と凍結ベースラインの欄名を直接突き合わせるので、片側だけの修正では必ず残差が出る。
- **Change-Resistance Results**:
  代表的な誤実装を 2 つ実測した。
  M1 欄名を `error` / `message` へ戻す → 2 つの部分試験が落ちる。
  M2 新旧両方の欄を並べて出す (`err`, `description`, `error`, `message` の 4 つ) → 2 つの部分試験が落ちる。古い欄の不在を主張していなければ生存していた変異で、これが「新しい欄が在ること」だけを見る検査との差を作っている。
- **Verification Results**:
  - `mise run verify` - passed (exit 0)
  - `mise run check-spec` - ok (148 document(s), 333 operation(s), 845 TypeSpec symbol(s))
  - `mise run check-api-compat` - 修正直後は 6 件の破壊的変更 (`POST /ssf/streams/{stream_id}/events` の 400 / 404 / 413 それぞれの `error` と `message` の削除)。この 1 接点と 3 schema だけをベースラインへ反映して `no breaking changes`。差分は 30 行で、他の接点には触れていない。
  - `mise run test-go-package -- ./backend/sharedsignals/...` - 全パッケージ ok
  - `mise run spec-diff` - `no normative specification change against main`

## Follow-up

Risk Notes が求めた「変更前に配備先で実際に受信している相手を確認する」は**実施していない**。この作業からは配備先を観測できないためで、判断は標準への適合を優先した。`{error, message}` を読んでいる外部 transmitter が居る配備では、この変更で相手のエラー処理が壊れる (拒否そのものは status で伝わるので、壊れるのは理由の読み取りである)。配備前に相手を確認すること。
