---
status: pending
authors: [tn]
risk: low
created_at: 2026-11-28
priority: p2
group: development-process
depends_on: [wi-82146-require-spec-record-verification-when-implementing-external-protocols]
change_kind: refactor
spec_impact: { kind: none, reason: "記録の再生と模倣のテストのうち、既存の単体テストと同じ分岐を検査するものを削るか既存のテストへ寄せる。本番コード、規範シナリオ、標準の宣言、TypeSpec シンボルは変えない。" }
---

# 記録の再生と模倣のテストを、既存の単体テストと重ならないように整理する

## 動機

記録の再生と模倣は、四つの項目で実装の後から足した。
SCIM（[wi-422](../done/wi-422-external-contract-and-failure-verification.md)）、上流のトークンエンドポイント（[wi-27706](../done/wi-27706-verify-upstream-oauth-token-endpoints-with-recordings.md)）、上流の OIDC IdP（[wi-65625](../done/wi-65625-verify-upstream-oidc-identity-providers-with-recordings.md)）、上流の MCP サーバー（[wi-49024](../done/wi-49024-verify-upstream-mcp-servers-with-recordings.md)）である。
どの項目も、既存の単体テストを振る舞いごとに並べずに検査を足した。

MCP Gateway の `backend/mcpgateway/internal/catalog_http` では、2026-11-28 に次の重なりを確かめた。

| 足したテスト | 同じ分岐を検査する既存のテスト |
| --- | --- |
| `TestPublisher_HandlesTwinFaults` | `TestSendCatalogMarksPartialListsIncomplete`、`TestSendCatalogSendsNothingWhenTheFirstPageFails` |
| `TestPublisher_SendsNoCatalogWhenTheTwinRefuses` | `TestSendCatalogSendsNothingWhenTheFirstPageFails`（401） |
| `TestPublisher_FollowsTwinPagination` | `TestSendCatalogFollowsCursorsAndSendsOnlyTheDeclaredFields`（空文字列のカーソルを除く） |
| `TestListener_ResendsWhenTheTwinChangesTools` | `TestListenCatalogsAgainWhenToolsChange` |
| `TestListener_StopsOnRecordedAcknowledgmentWithoutTools` | `TestListenCatalogsOnceAndStopsWhenTheUpstreamCannotNotify`（ツールの能力のない確認応答） |
| `TestPublisher_SendsNoCatalogOnRecordedRefusals` の 401 | `TestSendCatalogSendsNothingWhenTheFirstPageFails` の 401 |

同じ分岐を二か所で検査すると、振る舞いを変えたときに直すテストが二倍になる。
また、どちらが期待値の根拠なのかが読み取れなくなる。

ほかの三つの相手は、まだ照らしていない。
対象は `backend/provisioning/internal/client_scim`、`backend/tokenvault/internal/upstream_http`、`backend/authentication/internal/federation/protocol_oidc` の `*replay_test.go` と `*twin_test.go` である。

## 対象範囲

- 四つの相手について、記録の再生と模倣のテストを、同じパッケージの既存の単体テストと振る舞いごとに並べた表を作る。
- [wi-82146](wi-82146-require-spec-record-verification-when-implementing-external-protocols.md) で書く分担に従い、テストを次の三つに分ける。
  - 再生のうち、仕様の形が意味を持つ往復（成功の本文、カーソル、ストリームのメッセージ、本文のあるエラー）は契約の層として残す。
  - 模倣のうち、既存のテストが扱えない状態をまたぐ検査は残す。
  - 既存の単体テストと同じ分岐を検査するものは、どちらか一方に寄せる。
    寄せる先の選び方は設計に書く。
- 残す再生のテストが見つけた分岐のうち、既存の単体テストにないものを既存のテストに一ケースとして足す。
  MCP の空文字列のカーソルを `TestSendCatalogFollowsCursorsAndSendsOnlyTheDeclaredFields` に足す、などである。
  再生のテストは記録の形を固定する役として残す。
- 削ったテストが固定していた故障を、残ったテストが検出することを確かめる。

## 対象外

- 模倣の部品（`testing_httprecord`、`testing_*twin`）の API と、模倣の記録との照合（fidelity と drift のテスト）。後続の項目が使うので変えない。
- 本番コードの変更。整理の途中で規範違反を見つけたら、別の項目として起票する。
- 残りの外部の相手への適用。[wi-94480](wi-94480-decide-recordings-for-the-remaining-external-counterparts.md) と [wi-21837](wi-21837-verify-vault-and-openbao-transit-with-recordings.md) が扱う。

## 設計

重なりを解くときは、一つの応答で足りる分岐を、手書きの関数で応答を返す既存の単体テストへ寄せる。
既存のテストのほうが短く、何を検査しているかを読みやすい。
[テスト方針](../../docs/development/testing.md#外部契約と設定の利用)の「一つの応答だけが要るテストは模倣へ置き換えない」とも合う。
模倣のテストは、ページの列、変更の通知、鍵の回転、組み合わせた障害のように、手書きの関数で作ると状態の管理が膨らむ検査だけに残す。

既存のテストの手書きのスタブを模倣に置き換えて一本化する案は採らない。
スタブが期待値を実装と共有する問題は、再生のテストを契約の層として残すことで扱える。
模倣に寄せると、単純な分岐のテストまで模倣の状態を読む必要が出る。

OAuth2 のツール取得（`backend/oauth2/internal/token/usecases/mcp_discovery_replay_test.go`）は、wi-49024 の前は e2e のテストしかなかった。
ここでの模倣のテストは重なりではなく欠けていた単体テストを埋めているので、重なりの表で確かめてから残すかを決める。

## 計画

1. 相手ごとに重なりの表を作る。
2. 寄せるテストを決め、空文字列のカーソルのように既存のテストにない分岐は既存のテストへ足す。
3. 重なったテストを削る。
4. 削ったテストが固定していた故障を一つずつ手で注入し、残ったテストが落ちることを確かめる。

## タスク

- [ ] T001 [Research] 四つの相手の重なりの表を作る。
- [ ] T002 [App] MCP Gateway と OAuth2 のテストを整理する。
- [ ] T003 [App] SCIM、トークンエンドポイント、OIDC のテストを整理する。
- [ ] T004 [Verify] 削ったテストの故障を残ったテストが検出することを、手で注入して確かめる。
- [ ] T005 [Verify] 実際の差分から選んだ検査を実行する。

## 検証

- 触れたパッケージごとに `mise run test-go-package -- <package>`。
- 削ったテストが検出していた故障を注入し、残ったテストが落ちることを確かめる。
  空文字列のカーソルを終わりと読む故障、二ページ目の 5xx を無視する故障、変更の通知で再送しない故障などである。
- `mise run verify-changed -- main`。

## リスク

重なって見えるテストが、実は別の故障を検出していることがある。
削る前に故障を注入して確かめ、残ったテストが検出しない故障があれば、そのテストは削らずに残す。
