---
id: idp-wi-105-security-critical-parser-fuzzing
title: "SAML XML・JWT/JWE・redirect_uri・PKCE 等のセキュリティクリティカルなパーサに Go native fuzzing を導入する"
created_at: 2026-07-04
authors: ["tn"]
status: pending
risk: medium
---

# Motivation
idmagic は攻撃者制御の入力を直接パースする箇所を多数持つ。SAML/WS-Fed の
受信 XML（署名検証・XXE・XML canonicalization）、JWT/JWE のデコード
（private_key_jwt / DPoP / client assertion）、`redirect_uri` の完全一致照合、
PKCE code_verifier、PAR/authorization request のパラメータ解析などである。
IdP のこれらは 1 つのパース欠陥が認証バイパス・SSRF・DoS・署名回避に直結する
最重要攻撃面だが、現状の検証はテーブル駆動の単体テストが中心で、
想定外入力の網羅は人手依存になっている。

Go は標準ツールチェインに native fuzzing（`testing.F` / `go test -fuzz`）を持ち、
Go プロジェクト自身が stdlib のパーサをファジングしている。idmagic も
セキュリティクリティカルなパーサに fuzz target を追加し、パニック・
無限ループ・過大メモリ・正規化の不一致（parse→serialize→parse の非等価）を
自動探索し、発見した corpus を回帰として固定すべきである。

# Scope
- **go**:
  - 対象パーサに `Fuzz*` を追加する: SAML/WS-Fed の受信 XML と署名前処理、 JWT/JWE デコード、redirect_uri 照合、PKCE、authorization/PAR パラメータ解析。
  - 不変条件を assertion 化する: パニックしない、境界時間内に返る（DoS 耐性）、 parse の冪等性、redirect_uri 照合が仕様どおり厳密一致であること。
  - 発見した crash/差分を seed corpus として testdata に固定し、回帰テスト化する。
  - XML パースの entity 展開・外部参照禁止（XXE 対策）を fuzz と assertion で担保する。
- **ci**:
  - CI で短時間の fuzz smoke（`-fuzz` を制限時間付き）を回し、蓄積 corpus は 通常テストとして常時実行する。長時間 fuzz は任意ジョブに分ける。
- **documentation**:
  - どのパーサが fuzz 対象で、どの不変条件を守るかを対象 context 近傍に記す。

# Out of Scope
- パーサ実装そのものの全面書き換え。
- OSS-Fuzz 等外部継続ファジング基盤への登録（将来検討）。
- プロトコル準拠テスト（WI-33 が扱う conformance CI）。

# Verification
- `just test-go-race`
- `just test-go-fuzz ./internal/saml/... 30s`
- 手動: 既知の悪性入力（XXE ペイロード、alg=none JWT、部分一致 redirect_uri）で fuzz target が期待どおり拒否/非パニックであることを確認する。

# Risk Notes
fuzzing は本番挙動を変えないが、発見した欠陥の修正がプロトコル互換に影響し得る。
まず不変条件（非パニック・時間境界・厳密一致）に絞って target を作り、
発見バグは個別に評価してから修正する。CI では時間制限付き smoke に留め、
ビルド時間を圧迫しない。
