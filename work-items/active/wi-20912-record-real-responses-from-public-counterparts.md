---
status: pending
authors: [tn]
risk: low
created_at: 2026-11-28
priority: p3
group: development-process
depends_on: [wi-94480-decide-recordings-for-the-remaining-external-counterparts]
change_kind: tooling
spec_impact: { kind: none, reason: "公開の相手から取った実物の応答を記録にして再生の検査を足す。規範シナリオ、標準の宣言、TypeSpec シンボルを追加も変更もしない。検査が規範に反する実装を見つけたら、その時点で affected_spec を宣言し直す。" }
---

# 公開の相手から実物の応答を取り、記録の再生で検査する

## 動機

[wi-94480](../done/wi-94480-decide-recordings-for-the-remaining-external-counterparts.md) は、HIBP の Range API、Workload の issuer の Discovery、外部の JWKS、OAuth Client ID Metadata Document（CIMD）に記録の再生を置かなかった。
その判断は、一次資料に例があるかと、既存のテストが同じ形を検査しているかだけで比べていた。
既存のテストの応答は、どれも実装を書いた人が手で書いたものである。

2026-11-28 に [外部契約と設定の利用](../../docs/development/testing.md#外部契約と設定の利用) を改め、記録は製品が実際に受け取る応答から選び、公開のサービスに接続できる相手は実物から記録を取ることにした。
この規則に照らすと、上の四つは実物の応答を取れる、または取れるかを調べる相手である。

| 相手 | 送る側 | 実物の取得先の候補 | 手書きの応答では確かめられないこと |
| --- | --- | --- | --- |
| HIBP の Range API | `shared/policy/breaches_hibp` | `https://api.pwnedpasswords.com/range/{接頭辞}` に `Add-Padding: true` を付けた応答 | 改行（文書の範囲の節は定めていない）、詰め物の行の実際の形、Content-Type |
| Workload の issuer の Discovery | `workloadidentity/internal/issuer_metadata` | GitHub Actions、GitLab などの公開の workload の issuer の `/.well-known/openid-configuration` | 実在の issuer の末尾のスラッシュなどの表記と、`issuer` の完全一致の判定が成り立つか |
| 外部の JWKS | `shared/security/tokens_jose` の `JWKResolver` | 上の issuer の `jwks_uri` | `x5c`、`alg`、`use` を持ち、RSA と EC が混ざる実物の鍵を、解決から鍵の解釈まで受けられるか |
| CIMD | `oauth2/internal/client/cimd_http` | 公開されている Client ID Metadata Document（MCP のクライアントなど）があれば | 草案にない欄を持つ実物の文書を、解析器が拒否せず受けるか |

## 対象範囲

- 上の相手ごとに、実物の応答を取れるか、利用規約が記録を許すか、記録に公開の値だけが残るかを確かめる。
- 取れた相手に、実物の往復の記録を `testing_httprecord` の形式で置き、再生の検査を足す。期待値は一次資料の文面から書く。
- 実物が返さないが結果を左右する形があれば、一次資料の例から記録に足す。
- 既存の手書きの応答のテストのうち、再生と同じ分岐を検査するものは、再生へ置き換えるか、記録にない分岐だけを残す。
- CIMD は実物の文書が見つからなければ、その事実と見直す契機を要約に残して置かない。

## 対象外

- Microsoft のレルム探索。既存のテストが観測した実際の応答を転記している。
- 記録を定期的に取り直す仕組み。取り直す契機は記録の取得方法と要約に書く。
- 実在の相手への常時の接続。CI は記録の再生だけを実行する。

## 設計

記録を取る手順は、一度だけの取得なら作業の中で直接実行し、取得方法に送った要求を書く。
繰り返し取り直すと分かった相手だけ、`record-transit` に倣って `mise` タスクにする。
Workload の Discovery と外部の JWKS は同じ issuer から一続きに取れるので、一つの記録にまとめるかを、置き場所のモジュール境界と合わせて決める。
`issuer_metadata` は Workload Identity の内部、`JWKResolver` は `backend/shared` にあるので、両方が使う記録は `backend/shared` に置くことになる。

## 計画

1. 相手ごとに取得先と利用規約を確かめ、記録を取る。
2. 記録の再生で、既存のテストが扱えない分岐を検査する。
3. 見つかった食い違いを分類し、規範に反するものは affected_spec を宣言し直して直す。

## タスク

- [ ] T001 [Research] 相手ごとに取得先、利用規約、記録に残る値を確かめる。
- [ ] T002 [Adapter] 取れた相手に記録と再生を置き、重複する手書きのテストを整理する。
- [ ] T003 [Verify] 実際の差分から選んだ検査を実行する。

## 検証

- 記録の応答を仕様に反する形へ変えると、再生の検査が落ちる。
- 同じ分岐を、再生と既存の単体テストの両方で検査していない。
- `mise run verify-changed -- main`。

## リスク

実物の記録は取得した時点の一つの相手の応答であり、同じ種類のほかの相手や、その後の変化を保証しない。
取得方法に取得先と日付を書き、相手の障害や仕様の改訂を見直す契機にする。
