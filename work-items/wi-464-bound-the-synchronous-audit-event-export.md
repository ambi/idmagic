---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-03
priority: p1
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/contexts/audit/scenarios.md, requirement: REQ-AUDIT-001 }
  - { path: spec/contexts/audit/main.tsp, symbol: IdMagic.Audit.Operations.ExportAdminAuditEvents }
---

# 監査イベントのエクスポートを、同時実行と応答量に上限のある処理にする

## Motivation

`GET /api/admin/v1/audit_events/export` は、API プロセスの中で最大 10,000 件の監査イベントを読み、`[]*auditports.AuditEventRecord` と `[]AdminAuditEventResponse` という 2 つの中間表現をメモリに作ってから JSON へ直列化する。ストリーミングしない。

[[wi-459-api-process-plane-separation-decision]] の T007 で API プロセスに残る同期処理を棚卸ししたところ、**上限のない高コスト処理はこの 1 経路だけだった。** ほかの管理系一覧はすべて 200 件、監査の一覧も 999 件で頭打ちになる。利用者とグループのエクスポートは `data_export` として `bulk` レーンのワーカーへ出ており、ダウンロードは `c.Stream` で流す。

**問題は件数ではなく、上限がないことである。**

- `backend/shared/ratelimit` のポリシーは `token`、`authorize`、`par`、`device_authorization`、`backchannel_authentication`、`password_reset` の 6 つで、`/api/admin/v1` を 1 つも保護しない。
- 同時実行数を制限する仕組みは backend 全体に存在しない。`Semaphore` に相当するものが 1 つもない。
- 1 レコード 1 KiB という `docs/capacity.md` の Planning assumption では、1 リクエストにつき約 10 MiB の中間表現が 2 つ載る。本番の Pod は `limits` がメモリ 2 GiB である。

したがって、管理者が数十本のエクスポートを並行に叩けるなら、それを止めるものは何もない。**これは飽和ではなくメモリ枯渇として現れる。** 到達率で見れば管理系は認証系の数%にすぎないので、[[wi-396-prioritize-login-under-saturation]] が入れる入場制御はこの状況を「飽和」と判定しない。入場制御は総容量が足りないときに何を捨てるかを決める機構であって、1 リクエストが際限なく資源を取ることそのものは扱わない。

[[wi-459-api-process-plane-separation-decision]] は API の Deployment を種別ごとに分けない判断を記録した。その再検討条件 C1 は、**分割を検討する前に上限またはワーカー化を評価することを明示的に求めている。** 分割はこの欠陥を認証系から遠ざけるだけで、除去しない。本 work item は原因の側を扱う。

## Scope

- `ExportAdminAuditEvents` の応答が、同時実行数と 1 応答当たりのメモリの両方で上限を持つようにする。
- 上限を超えた要求の扱いを決める。拒否するのか、ページ分割された結果を返すのか、ジョブへ回すのかを Design で選ぶ。
- 上限を運用者が起動時設定で調整できるようにする（`REQ-SYSTEM-016` に従って検証する）。
- `REQ-AUDIT-001` の「絞り込み結果がエクスポートデータとして返る」が、上限の導入後も成立する形を保つ。境界が変わる場合はシナリオを先に更新する。
- 上限に触れたことを観測できるメトリクスを出す。
- TypeSpec の `ExportAdminAuditEvents` に、上限超過時の応答と新しいクエリパラメーターを反映する。

## Out of Scope

- ほかの管理系エンドポイントへの上限追加。棚卸しの結果、上限のない同期処理はこの 1 経路だけだった。SCIM Bulk（[[wi-249-scim-bulk-operations]]）が入れば同じ性質の経路が増えるが、それはその work item が持つ。
- 入場制御、優先度クラス、優先度別の接続予算、自動スケール。[[wi-396-prioritize-login-under-saturation]] が持つ。本 work item は負荷に連動しない固定の上限だけを扱う。
- API プロセスを種別ごとに分けること。[[wi-459-api-process-plane-separation-decision]] が判断を持つ。
- 監査イベントの保持方針、検索属性の副表、`audit_events` のインデックス設計。
- エクスポート形式の追加（CSV など）。現在の JSON のまま扱う。

## Design

### 上限が要るのは 2 つの軸である

同時実行数と 1 応答当たりの量は別の軸で、片方だけでは足りない。1 応答を 1,000 件に絞っても、100 本並行に走れば同じだけメモリを取る。同時実行を 2 本に絞っても、1 応答が無制限ならピークは下がらない。両方に上限を置く。

### 選択肢

| 案 | 内容 | 利点 | 欠点 |
| --- | --- | --- | --- |
| A | 応答を JSON 配列としてストリーミングし、中間表現を持たない | メモリが件数に比例しなくなる。エンドポイントの意味を変えない | データベースのカーソルを応答の間ずっと開くので、接続の占有時間が延びる。エラーを応答の途中で返せない |
| B | 上限を下げ、超過した要求は 400 で拒否してページ分割された一覧へ誘導する | 最も単純。追加の機構が要らない | 「絞り込み結果をエクスポートする」という `REQ-AUDIT-001` の利用者価値が落ちる |
| C | ユーザーとグループのエクスポートと同じく `JobKind` としてワーカーへ出し、成果物を `c.Stream` で配る | API プロセスから費用が完全に消える。既存の `data_export` と同じ形で、実装の前例がある | 同期の応答から非同期の 3 手順（作成・照会・取得）へ変わる。TypeSpec と UI の変更を伴う |
| D | 同時実行数だけをセマフォで絞り、超過は 429 で拒否する | 小さい。ピークメモリの上界が決まる | 1 応答の量は無制限のまま。上界が「同時実行数 × 10 MiB」でしか決まらない |

**C を軸に、D を併用する案を推す。** ユーザーとグループのエクスポートは既に C の形になっており、監査だけが同期で残っているのは経緯であって設計ではない。`data_export` の `JobKind`、`bulk` レーン、`CSVArtifacts` に相当する成果物ストア、`DownloadDataExport` の `c.Stream` という部品はすべて既にある。D を併用するのは、移行期間に同期経路を残す場合と、ジョブ作成そのものが殺到した場合の両方に上界が要るためである。

A は接続の占有時間をレプリカ当たり 16 接続という予算の中で延ばすので、`docs/capacity.md` の接続予算と衝突しうる。B は要件を落とす。D 単独では上界が「同時実行数 × 無制限」になる。

**ただしこの選択は着手時に確定する。** C は `REQ-AUDIT-001` の「エクスポートデータとして返る」を非同期の取得へ変えるため、シナリオの更新を伴う。UI 側の変更量と、既存の API トークン利用者への影響を着手時に測ってから決める。

### 決めておくこと

- 上限を超えたときの状態コード。同期のまま拒否するなら 429 か 413 か。`docs/api-rules.md` と既存の `RateLimitedError` の扱いに合わせる。
- 既定値。`docs/capacity.md` の Peak request profile に管理系の行が入ったので、そこから逆算する。
- 同期経路を残すかどうか。残すなら非推奨として `Deprecation` / `Sunset` ヘッダー（`REQ-SYSTEM-014`）を付ける。

## Plan

1. 現在のピークメモリと同時実行数を測る。`http_requests_in_flight` を `route` で絞り、10,000 件を返す要求 1 本の常駐量を実測する。仮定ではなく数字から上限を決める。
2. C と D の実装量を測り、同期経路を残すかどうかを決める。`REQ-AUDIT-001` の更新が要るかはここで確定する。
3. 規範シナリオと TypeSpec を先に更新する。
4. 上限を実装し、超過時の応答を確かめる。
5. 上限に触れたことを示すメトリクスと、`docs/capacity.md` の管理系の行との対応を確認する。

## Tasks

- [ ] T001 [Research] 10,000 件のエクスポート 1 本の常駐メモリと所要時間を実測し、上限の既定値を決める。
- [ ] T002 [Design] C と D の実装量を測り、同期経路を残すかどうかを確定する。
- [ ] T003 [Spec] `REQ-AUDIT-001` と `ExportAdminAuditEvents` を更新する。
- [ ] T004 [Acceptance] 上限を超えた要求が拒否されるか非同期へ回ることを、観測可能な境界で RED として確かめる。
- [ ] T005 [App] 同時実行の上限を実装する。
- [ ] T006 [App] 1 応答当たりの量の上限、またはジョブ化を実装する。
- [ ] T007 [Config] 上限の起動時設定を追加し、`REQ-SYSTEM-016` の検証に載せる。
- [ ] T008 [UI] 管理コンソールのエクスポート操作を新しい形に合わせる。
- [ ] T009 [Verify] 検査を通す。

## Verification

- `mise run check-spec`
- `mise run check-config-reference`
- `mise run verify`

## Risk Notes

リスクは medium。監査は製品の重要な性質であり、エクスポートは規制対応で使われうるので、上限が低すぎると利用者の要件を満たせなくなる。上限は測った数から決め、運用者が調整できるようにする。

**非同期化を選んだ場合、`REQ-AUDIT-001` の意味が変わる。** 「エクスポートデータとして返る」が同期の応答から取得経路へ移る。シナリオを先に更新し、実装を通すためにシナリオを弱めない。既存の API トークン利用者は同期の応答を前提にしているので、非推奨期間を置くかどうかを T002 で決める。

**上限を入れても、それが正しいことは負荷試験でしか分からない。** 実測は [[wi-282-staging-load-testing-and-capacity-validation]] の基盤を使う。この work item は上限の存在を保証するもので、値の正しさは測定に依存する。

`reversibility` は reversible。上限は設定で戻せる。非同期化を選んだ場合、公開された API の形が変わるので、その部分だけは利用者側の変更を伴う。
