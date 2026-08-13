---
status: pending
authors: [tn]
risk: medium
created_at: 2026-08-13
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: spec/contexts/oauth2/SPECIFICATION.md, requirement: REQ-OAUTH2-010 }
  - { path: spec/contexts/oauth2/models.tsp, symbol: IdMagic.Contract.SenderConstraintCnf }
---

# リソースアクセス時の DPoP proof で ath クレームを検証し、proof を access token に束縛する

## Motivation

DPoP (RFC 9449) の検証は `backend/shared/security/tokens_jose/dpop_verifier.go` に実装済みで、
`typ` / `alg` / jwk header / 署名 / `htm` / `htu` / `iat` スキュー / `jti` リプレイまで見ている。
しかし **`ath` (access token hash) クレームの検証が無い**。リポジトリを検索しても `ath` は 0 件である。

RFC 9449 §4.3 は、protected resource へのアクセスで DPoP proof を用いる場合に
`ath` = base64url(SHA-256(access_token)) の検証を要求する。これが無いと、proof は
「この鍵の持ち主である」ことしか証明せず、**特定の access token に束縛されない**。結果として、
同じ鍵で発行された任意の proof が、同じ `htm` / `htu` の範囲で別の access token に流用できる。
現状 `jti` リプレイストアがあるため同一 proof の再送は防げるが、束縛の欠落そのものは埋まらない。

エージェント文脈ではこの欠落の意味が重い。エージェントは人間のブラウザセッションと違い、
長時間稼働し、多数のリソースサーバーを横断し、委譲チェーンの各段で異なる token を扱う。
proof と token の対応が緩いほど、取り違えと流用の窓が構造的に広がる。idmagic は
`cnf.jkt` による送信者束縛を Token Exchange の結果にも引き継いでいるので、
検証側だけが RFC の要求を満たしていない状態になっている。

この項目は [[wi-369-agent-capability-survey-2026-08]] の棚卸しで P0 (セキュリティ) と判断した。

## Scope

- `spec/contexts/oauth2/SPECIFICATION.md` の DPoP 節に `ath` 検証の normative scenario
  (REQ-OAUTH2-042) と standards 要件行を追加する。
- `tokens_jose.VerifyDPoP` に access token を渡し、`ath` を検証できるようにする。
- リソースアクセス経路の呼び出し側を更新する:
  `backend/shared/http/support_http/auth.go` (`resolveAuthnContext` の DPoP 分岐) と
  `backend/oauth2/handlers_http/userinfo_handler.go`。

## Out of Scope

- **DPoP-Nonce (`DPoP-Nonce` ヘッダ / `use_dpop_nonce` エラー)**。RFC 9449 §8 上は任意であり、
  リプレイ耐性は既存の `DpopReplayStore` (`backend/oauth2/db_memory/replay.go` /
  `db_postgres/replay_store.go`) で既に担保している。nonce はサーバー側の状態と往復を 1 回増やす
  ため、必要性が具体的に出た時点で別 work item にする。
- トークンエンドポイント (`/token` / `/par`) の proof。RFC 9449 は `ath` を protected resource
  アクセス時の要求としており、token 要求時点では対象の access token が存在しない。
- 新しい送信者束縛方式の追加。mTLS 証明書束縛は既に実装済みで本 work item は触らない。

## Design

- **`VerifyDPoP` のシグネチャに access token を足す**。`ath` を検証しない呼び出しを可能にする
  optional な形にすると、リソース経路で検証漏れが起きても型が教えてくれない。
  トークンエンドポイント経路と保護リソース経路を**別の関数として分ける**か、
  access token 引数を必須にして空文字を許さないかを実装時に決める。緩い optional 引数は採らない。
- **既存 DPoP クライアントを壊す変更である**。`ath` を送っていないクライアントはリソース
  アクセスで一斉に 401 になる。移行方針を Plan で明示する。
- 比較は定数時間で行う。既存の PKCE 検証 (`backend/oauth2/authorization/domain/pkce.go` の
  `VerifyPKCES256`) が同じ形の比較を持っているため、そのパターンに揃える。
- `ath` の算出対象は「クライアントが提示した access token 文字列そのもの」であり、
  introspection 後の内部表現ではない。ハンドラで受け取った生の文字列を渡す。

## Plan

- 先に spec (REQ-OAUTH2-042 + standards 行) を確定させ、そのあと実装する。
- **移行**: `ath` 欠落を即時拒否にするか、テナント設定で段階導入するかを T001 で決める。
  既定を「必須」にするのが RFC 準拠だが、稼働中の DPoP クライアントがある環境では
  切り替え時に一斉障害になる。ADR に判断を残し、少なくとも変更内容を README に書く。
- 呼び出し側は 2 箇所だけなので、`VerifyDPoP` 側を変えたときにコンパイルエラーで
  全経路が露出する形を選ぶ (見落としを型で防ぐ)。
- 未決定: `ath` 検証をリソースサーバー側の共通ミドルウェアに寄せるか、現在のまま
  `resolveAuthnContext` に置くか。[[wi-324-sharedsignals-agent-revocation-followups]] が
  `resolveAuthnContext` の統合を別途扱っているため、そちらの結論と衝突しないようにする。

## Tasks

- [ ] T001 [Spec] DPoP 節に `ath` 検証の standards 要件と REQ-OAUTH2-042 を追加し、
      `ath` 欠落時の扱い (即時必須か段階導入か) を決めて記録する。
- [ ] T002 [Domain] `VerifyDPoP` で `ath` を検証する。RED: `ath` 欠落 / 別 token の `ath` /
      正しい `ath` の 3 ケースを先に書く → GREEN。
- [ ] T003 [Adapters] `resolveAuthnContext` と userinfo の呼び出しに access token を渡す。
      RED: 別 access token 用の proof が保護リソースで拒否されるテスト → GREEN。
- [ ] T004 [Docs] README に DPoP クライアント側の `ath` 要求と移行手順を追記する。
- [ ] T005 [Verify] 下記 Verification を緑にする。`just spec-render` を実行する。

## Verification

- `just check` / `just check-spec` / `just check-work-items`
- `just verify-go` / `just test-go-race`
- 手動: `just dev` で (1) 正しい `ath` を持つ proof で保護 API にアクセスできること、
  (2) 別の access token 用の proof が拒否されること、(3) `ath` を欠いた proof が
  決定した方針どおりに扱われること、(4) トークンエンドポイントの proof は従来どおり
  動作すること (回帰) を確認する。

## Risk Notes

リスクは medium。**既存の DPoP クライアントを壊す破壊的変更**であり、`ath` を送らない
実装はリソースアクセスで一斉に失敗する。移行方針を T001 で明示的に決め、README に書く。

検証を追加する側の失敗様式にも注意する。`ath` の算出対象を introspection 後の内部表現に
してしまうと、正しいクライアントを拒否する。ハンドラが受け取った生の token 文字列を
渡すことをテストで固定する。

トークンエンドポイント経路に誤って `ath` 必須を波及させると、DPoP を使う全てのトークン
取得が壊れる。経路の分離をテストで固定する。
