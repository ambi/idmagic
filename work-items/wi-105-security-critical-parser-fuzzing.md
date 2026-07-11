---
depends_on: []
status: pending
authors: ["tn"]
risk: medium
created_at: 2026-07-04
---

# SAML XML・JWT/JWE・redirect_uri・PKCE 等のセキュリティクリティカルなパーサに Go native fuzzing を導入する

## Motivation
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

## Scope
- **go**:
  - 対象パーサに `Fuzz*` を追加する: SAML/WS-Fed の受信 XML と署名前処理、 JWT/JWE デコード、redirect_uri 照合、PKCE、authorization/PAR パラメータ解析。
  - 不変条件を assertion 化する: パニックしない、境界時間内に返る（DoS 耐性）、 parse の冪等性、redirect_uri 照合が仕様どおり厳密一致であること。
  - 発見した crash/差分を seed corpus として testdata に固定し、回帰テスト化する。
  - XML パースの entity 展開・外部参照禁止（XXE 対策）を fuzz と assertion で担保する。
- **ci**:
  - CI で短時間の fuzz smoke（`-fuzz` を制限時間付き）を回し、蓄積 corpus は 通常テストとして常時実行する。長時間 fuzz は任意ジョブに分ける。
- **documentation**:
  - どのパーサが fuzz 対象で、どの不変条件を守るかを対象 context 近傍に記す。

## Out of Scope
- パーサ実装そのものの全面書き換え。
- OSS-Fuzz 等外部継続ファジング基盤への登録（将来検討）。
- プロトコル準拠テスト（WI-33 が扱う conformance CI）。

## Plan
- targetは現行実装の境界へ置く: SAML AuthnRequest/LogoutRequest/metadata/XML signature、WS-Trust SOAP、JWT/JWE/DPoP/client assertion、OAuth redirect URI/PAR/authorization_details/PKCE。HTTP server全体をfuzzして原因を曖昧にしない。
- 各targetは「panicしない」だけでなく、parse→serialize→parseの保持、既知valid acceptance、署名/issuer/audience/destination/redirect strictness、size/depth/time上限をoracleにする。暗号署名自体をランダムinputごとに生成して速度を浪費しない。
- corpusはSCL scenario/既存unit fixture、最小valid、過去のmalformed regression、protocol official vectorから作り、secret/実tenant dataを含めない。crash inputは最小化して通常regression testへ昇格する。
- PR CIは各targetのseed replayと短時間fuzz、nightlyはtargetごとの時間budget、週次は長時間/raceを回す。Go version/corpus/artifact hashを記録し、failure artifactを信頼できないinputとして扱う。
- XML/entity expansion、JWT segment、JSON nesting、URL length等のpre-parse body limitもtargetに含め、OOM/hangをprocess timeoutだけに頼らずguardで防ぐ。

## Tasks
- [ ] T001 [Matrix] parser function、trust boundary、existing fixture、oracle、max size/depth/timeをtarget matrixにし、重複targetを除く。
- [ ] T002 [XML] SAML request/logout/metadata/signatureとWS-Trust envelopeのfuzz target/corpus/resource guardを追加する。
- [ ] T003 [JWT/Crypto] JWT/JWE、DPoP、private_key_jwt/client assertionのstructure/claim verifier targetとalgorithm-confusion corpusを追加する。
- [ ] T004 [OAuth Input] redirect URI、authorization_details/PAR、PKCE、form/query parserのtargetとroundtrip/strict-match oracleを追加する。
- [ ] T005 [Regression] crash/mismatchをminimizeしてnamed regression testへ保存するhelper/recipeを追加し、corpus provenanceを記録する。
- [ ] T006 [CI] seed replay、PR短時間、nightly budget、weekly race workflowとartifact retention/timeoutを追加する。
- [ ] T007 [Verify/Docs] 全targetを固定seedで再現し、panic/OOM/hang/valid rejectionのtriage・security disclosure手順を記載する。

## Verification
- `just test-go-race`
- `just test-go-fuzz ./internal/saml/... 30s`
- 手動: 既知の悪性入力（XXE ペイロード、alg=none JWT、部分一致 redirect_uri）で fuzz target が期待どおり拒否/非パニックであることを確認する。

## Risk Notes
fuzzing は本番挙動を変えないが、発見した欠陥の修正がプロトコル互換に影響し得る。
まず不変条件（非パニック・時間境界・厳密一致）に絞って target を作り、
発見バグは個別に評価してから修正する。CI では時間制限付き smoke に留め、
ビルド時間を圧迫しない。
