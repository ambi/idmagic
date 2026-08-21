---
status: completed
authors: [tn]
risk: medium
created_at: 2026-08-13
depends_on: []
change_kind: bugfix
initial_context:
  specification:
    - spec/contexts/oauth2/SPECIFICATION.md#REQ-OAUTH2-010
    - spec/contexts/oauth2/SPECIFICATION.md#REQ-OAUTH2-045
  typespec: [IdMagic.Contract.SenderConstraintCnf]
  source:
    - backend/shared/security/tokens_jose/dpop_verifier.go
    - backend/shared/http/support_http/auth.go
    - backend/oauth2/handlers_http/userinfo_handler.go
    - backend/oauth2/handlers_http/token_handler.go
    - backend/oauth2/authorization/domain/pkce.go
  tests:
    - backend/shared/security/tokens_jose/dpop_verifier_test.go
    - backend/oauth2/handlers_http/userinfo_handler_test.go
  stop_before_reading: [frontend, infra]
affected_spec:
  - { path: spec/contexts/oauth2/scenarios.md, requirement: REQ-OAUTH2-045 }
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
  (REQ-OAUTH2-045) と standards 要件行 (RFC9449-ATH) を追加する。
  起票時に想定した REQ-OAUTH2-042 は CIBA の承認要求に既に使われていたため 045 を採番した。
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

- [x] T001 [Spec] DPoP 節に `ath` 検証の standards 要件 (RFC9449-ATH) と REQ-OAUTH2-045 を追加し、
      TypeSpec に `DpopProofClaims` を起こした。`ath` 欠落は**即時拒否**とする (段階導入を採らない)。
      根拠: 猶予フラグは既定値を「束縛の無い proof を受理する」側に置くことになり、本項目が
      閉じようとしている失敗そのものを設定可能な状態として残す。IdMagic は未リリースであり、
      リポジトリ内に DPoP proof を生成するクライアント実装も存在しない (frontend は
      `dpop_bound_access_tokens` の設定 UI のみ) ため、移行対象となる稼働クライアントはいない。
- [x] T002 [Domain] `VerifyDPoP` を経路別の `VerifyDPoPForToken` / `VerifyDPoPForResource` に
      分け、後者で `ath` を必須検証する (`AccessTokenHash` + `subtle.ConstantTimeCompare`)。
      RED → GREEN: `TestVerifyDPoPForResourceBindsProofToAccessToken` (REQ-OAUTH2-045、
      正しい `ath` / `ath` 欠落 / 別 token の `ath` / proof 不在 / access token 未指定) と
      `TestVerifyDPoPForTokenAcceptsProofWithoutATH` (REQ-OAUTH2-045 の ALT、token 経路の回帰)。
      `backend/shared/security/tokens_jose/dpop_verifier_test.go`。
- [x] T003 [Adapters] `resolveAuthnContext` / userinfo に生の access token を渡し、token
      エンドポイントは `VerifyDPoPForToken` に振り分けた。RED → GREEN:
      `TestResourceDPoPProofBindsToPresentedAccessToken`
      (`backend/shared/http/support_http/auth_test.go`、REQ-OAUTH2-045) と
      `TestUserInfoDPoPBoundRequiresMatchingProof` に追加した missing ath /
      ath of another access token の 2 ケース
      (`backend/oauth2/handlers_http/userinfo_handler_test.go`、REQ-OAUTH2-045)。
- [x] T004 [Docs] 取りやめ。IdMagic は未リリースで移行対象の稼働 DPoP クライアントが存在しない
      ため、README に移行手順は書かない。`ath` の要求はクライアント向けの規範として
      REQ-OAUTH2-045 と RFC9449-ATH に置き、README の機能一覧は変更しない。
- [x] T005 [Verify] 下記 Verification を緑にした。`just spec-render` を実行済み
      (生成物は untracked)。手動 `just dev` 確認は未実施 — Completion 参照。

## Verification

- `just check` / `just check-spec` / `just check-work-items`
- `just verify-go` / `just test-go-race`
- 手動: `just dev` で (1) 正しい `ath` を持つ proof で保護 API にアクセスできること、
  (2) 別の access token 用の proof が拒否されること、(3) `ath` を欠いた proof が
  決定した方針どおりに扱われること、(4) トークンエンドポイントの proof は従来どおり
  動作すること (回帰) を確認する。

## Risk Notes

リスクは medium。**既存の DPoP クライアントを壊す破壊的変更**であり、`ath` を送らない
実装はリソースアクセスで一斉に失敗する。IdMagic は未リリースで移行対象の稼働クライアントが
いないため、T001 で即時必須と決め、README への移行手順は書かない。

検証を追加する側の失敗様式にも注意する。`ath` の算出対象を introspection 後の内部表現に
してしまうと、正しいクライアントを拒否する。ハンドラが受け取った生の token 文字列を
渡すことをテストで固定する。

## Completion

- **Completed At**: 2026-08-14
- **Summary**:
  保護リソースへの DPoP proof が access token に束縛されるようになった。`just spec-diff` の
  差分は normative scenario **REQ-OAUTH2-045** の追加と TypeSpec 宣言 **DpopProofClaims** の追加
  (加えて standards 行 RFC9449-ATH と Design 節の DPoP 段落)。仕様上の意味の差は、
  「DPoP proof は鍵の所有を示す」から「保護リソースに提示する proof は
  `ath` = base64url(SHA-256(提示 access token)) を持ち、持たない proof は拒否される」への強化で、
  トークンエンドポイントは対象 token が存在しないため従来どおり `ath` を要求しない。
  実装では `VerifyDPoP` を経路別の `VerifyDPoPForToken` / `VerifyDPoPForResource` に分割した。
  optional 引数ではなく関数分割にしたのは、access token を渡し忘れた呼び出しが
  「ath 検証が黙って無効化された保護リソース」にならないようにするため。3 つの呼び出し側は
  改名によるコンパイルエラーで全て露出させて振り分けた。比較は `subtle.ConstantTimeCompare`。
- **Verification Results**:
  - `just check` - passed (`just verify` に含めて実行)
  - `just check-spec` - passed
  - `just check-work-items` - passed
  - `just check-api-compat` - passed (breaking change 無し)
  - `just verify` - passed
  - `just verify-go` (lint-go + test-go-race) - passed
  - `just spec-render` - 実行済み
  - 手動 `just dev` 確認 - **未実施**。(1)〜(4) は httptest でハンドラ経路を通す自動テストで
    等価に押さえた: `TestUserInfoDPoPBoundRequiresMatchingProof` と
    `TestResourceDPoPProofBindsToPresentedAccessToken` が (1)(2)(3)、
    `TestVerifyDPoPForTokenAcceptsProofWithoutATH` と既存 token 経路テストが (4)。
    リポジトリに DPoP proof を生成するクライアントが無く、手動確認には PS256 proof を
    その場で作る使い捨てスクリプトが必要になるため見送った。

トークンエンドポイント経路に誤って `ath` 必須を波及させると、DPoP を使う全てのトークン
取得が壊れる。経路の分離をテストで固定する。
