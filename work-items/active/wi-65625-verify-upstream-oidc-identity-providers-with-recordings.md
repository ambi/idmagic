---
status: pending
authors: [tn]
risk: high
created_at: 2026-11-26
priority: p2
group: development-process
depends_on: [wi-27706-verify-upstream-oauth-token-endpoints-with-recordings]
change_kind: tooling
spec_impact: { kind: none, reason: "連携先の OIDC IdP に対する前提を、記録の再生と状態を持つ模倣で確かめる検査を足す。規範シナリオ、標準の宣言、TypeSpec シンボルを追加も変更もしない。検査が規範に反する実装を見つけたら、その時点で affected_spec を宣言し直す。" }
---

# 連携先の OIDC IdP への前提を、記録の再生と模倣で確かめる

## 動機

Authentication は、連携ログインで上流の OIDC IdP の Discovery、認可コードの交換、JWKS を呼ぶ RP である。
上流の応答の形が想定と違うと、その IdP を使う利用者がログインできなくなる。
IdP の実装は製品ごとの差が大きく、上流の変更で前提が崩れやすい。

上流を呼ぶクライアントのテストは `RoundTripper` を差し替えて一つの応答を返す形であり、ID トークンの解析にはファズがある（2026-11-26 に確かめた）。
相手の実際の応答の形を固定する検査はない。
IdMagic を OP とする適合試験は [wi-33](wi-33-protocol-conformance-ci.md) が扱い、RP として呼ぶ側を扱う項目はない。

[wi-422](../done/wi-422-external-contract-and-failure-verification.md) の SCIM では、記録の再生と模倣が二つの誤りを見つけた。
同じ方法を連携ログインへ広げる。

## 対象範囲

- OpenID Connect Core 1.0 と OpenID Connect Discovery 1.0 の要求と応答の例を転記した記録と、Discovery、コードの交換、JWKS の取得の再生の検査。
- 状態を持つ OIDC IdP の模倣。Discovery、認可コードの発行と交換、ID トークンの署名、鍵の回転、`kid` の欠落、`5xx`、不正な本文を返せる。記録の再生と同じ応答を返すことを検査する。
- 記録と模倣の部品は [wi-27706](wi-27706-verify-upstream-oauth-token-endpoints-with-recordings.md) が決めた共有の置き場所のものを使う。
- 見つかった誤りの修正。製品の結果を変えるものは、着手後に利用者に確かめてから `affected_spec` を宣言する。

## 対象外

- 特定の IdP（Entra ID、Google、Okta など）ごとの記録と模倣。汎用の模倣で費用と検出できるものを確かめてから、相手ごとの差が実際に問題になったものだけを起票する。
- IdMagic を OP とする適合試験。[wi-33](wi-33-protocol-conformance-ci.md) が扱う。
- SAML と WS-Federation の連携。

## 設計

記録の出どころは一次資料の例の転記にし、取得方法に節番号を書く（[テスト方針](../../docs/development/testing.md#外部契約と設定の利用)）。
ID トークンの例は署名が検証できないので、模倣との照合では署名の値を揮発する値として扱い、署名の検証は模倣が自分の鍵で署名した値で確かめる。
この扱いが照合を空虚にしないかは、模倣を記録から外す否定の検査で確かめる。

## 計画

1. OIDC Core と Discovery の例を転記し、再生の検査を置く。
2. OIDC IdP の模倣を置き、記録との照合と、鍵の回転と障害の検査を置く。

## タスク

- [ ] T001 [Adapter] 記録と再生の検査を置く。
- [ ] T002 [Adapter] OIDC IdP の模倣と照合を置く。
- [ ] T003 [Verify] 実際の差分から選んだ検査を実行する。

## 検証

- 記録の応答を仕様に反する形へ変えると、再生の検査が落ちる。
- 模倣の応答を記録から外すと、照合が食い違いを報告する。
- `mise run verify-changed -- main`。

## リスク

模倣が自分で署名した ID トークンだけで検証すると、実際の IdP が使う署名方式や鍵の形の差を見落とす。
記録の JWKS の形と模倣の JWKS の形を照合で比べる。
