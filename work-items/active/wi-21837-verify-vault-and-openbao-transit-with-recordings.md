---
status: pending
authors: [tn]
risk: high
created_at: 2026-11-26
priority: p2
group: development-process
depends_on: [wi-27706-verify-upstream-oauth-token-endpoints-with-recordings]
change_kind: tooling
spec_impact: { kind: none, reason: "Vault と OpenBao の Transit に対する前提を、実物から取った記録の再生と状態を持つ模倣で確かめる検査を足す。規範シナリオ、標準の宣言、TypeSpec シンボルを追加も変更もしない。検査が規範に反する実装を見つけたら、その時点で affected_spec を宣言し直す。" }
---

# Vault と OpenBao の Transit への前提を、実物から取った記録の再生と模倣で確かめる

## 動機

署名鍵は Vault の Transit で署名し、エンベロープ暗号のマスター鍵は OpenBao の Transit で包む。
Transit の応答の形が想定と違うと、トークンの署名とデータの復号が止まる。

どちらのアダプターのテストも `httptest` で一つの応答を返す形である（2026-11-26 に確かめた）。
相手の実際の応答の形を固定する検査はない。

ほかの外部の相手と違い、Vault と OpenBao は開発用のサーバーを手元で起動できる。
[wi-422](../done/wi-422-external-contract-and-failure-verification.md) の SCIM は実物に接続できず、一次資料の例を転記して記録にしたので、保証は一次資料の示す応答を解釈できることまでだった。
Transit では実物の往復を記録でき、製品の版ごとの差まで固定できる。

## 対象範囲

- Vault と OpenBao の開発用サーバーから往復を記録する手順。手順は `mise` のタスクにし、記録には取得日、サーバーの版、取得方法を載せる。
- 署名、検証、鍵の作成と回転、暗号化と復号、データ鍵の生成の再生の検査。
- 状態を持つ Transit の模倣。鍵の版の回転、`min_decryption_version`、存在しない鍵、権限のない要求、`5xx`、不正な本文を返せる。記録の再生と同じ応答を返すことを検査する。
- 記録と模倣の部品は [wi-27706](../done/wi-27706-verify-upstream-oauth-token-endpoints-with-recordings.md) が決めた共有の置き場所のものを使う。
- 見つかった誤りの修正。製品の結果を変えるものは、着手後に利用者に確かめてから `affected_spec` を宣言する。

## 対象外

- 鍵管理サービスの遮断器。[wi-615](wi-615-circuit-breaker-for-key-management-service.md) が扱う。模倣の障害の注入はその検査にも使える。
- 本番の Vault と OpenBao への接続。
- CI での実物の起動。記録を取る手順は手元で使い、CI は記録の再生と模倣だけを実行する。

## 設計

記録を取るのは手元の意図的な作業とし、CI のゲートに入れない。
実物の版を上げたら記録を取り直し、差分を読んで、相手が変わったのか記録が古いのかを判断する。
記録に鍵の材料や秘密情報が入らないことを、記録を保存する前に検査する。

## 計画

1. 開発用サーバーから記録を取るタスクを作り、Vault と OpenBao のそれぞれで記録する。
2. 再生の検査を置く。
3. Transit の模倣を置き、記録との照合と、鍵の回転と障害の検査を置く。

## タスク

- [ ] T001 [Tooling] 記録を取るタスクと、記録の秘密情報の検査を作る。
- [ ] T002 [Adapter] Vault と OpenBao の記録と再生の検査を置く。
- [ ] T003 [Adapter] Transit の模倣と照合を置く。
- [ ] T004 [Verify] 実際の差分から選んだ検査を実行する。

## 検証

- 記録の応答を実物の形から外すと、再生の検査が落ちる。
- 模倣の応答を記録から外すと、照合が食い違いを報告する。
- 鍵の材料を含む記録を、保存前の検査が拒否する。
- `mise run verify-changed -- main`。

## リスク

開発用サーバーは本番の構成（監査デバイス、名前空間、ポリシー）と違うので、記録はその差を含まない。
記録の取得方法に開発用サーバーの構成を書き、本番の構成での保証ではないことを明示する。
