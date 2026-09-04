# 入場制御が発動したとき

`ApiAdmissionSheddingInteractiveAuth`、`ApiAdmissionShedding`、`ApiAdmissionUnclassifiedRoute` の対応手順である。機構そのものは [deployment.md](../deployment.md#load-shedding-under-saturation)、判断の理由は [contexts/system/decisions.md](../contexts/system/decisions.md#load-shedding-by-priority-class) が持つ。

## 何が起きているか

API プロセスは、実行中の要求数が優先度クラスごとの上限を超えると、そのクラスの要求を 503 で拒否する。拒否はハンドラーの手前で起きるので、拒否された要求は状態を一切変えていない。

**どの経路がどのクラスに属するかは [ROUTE_PRIORITY.md](../../ROUTE_PRIORITY.md) を引く。** 「この操作はなぜ 503 になったのか」「この操作は次にどのステージで落ちるのか」はそこで直接引ける。分類はコードに 1 箇所あり、この生成物はそこから導かれるので、実装を読んで前方一致の優先順位を解く必要はない。

| アラート | 意味 |
| --- | --- |
| `ApiAdmissionShedding` | `management` または `management_bulk` を捨てている。設計どおりの縮退であり、対話的な認証はまだ守られている |
| `ApiAdmissionSheddingInteractiveAuth` | 対話的な認証まで捨てている。縮退の最終ステージであり、`SLO-PRIMARY-ERRORS` と `SLO-OAUTH2-AVAILABILITY` はこの時点で失われている |
| `ApiAdmissionUnclassifiedRoute` | 優先度クラスの無い経路が配信されている。縮退の順序がその経路に及ばない |

## 最初に見るもの

1. `http_admission_in_flight_requests` と、`ADMISSION_*` の 3 つの上限を並べる。実行中数がどの上限に張り付いているかで、どのステージにいるかが分かる。
2. `rate(http_admission_decisions_total{outcome="shed"}[5m])` をクラス別に見る。`management_bulk` だけなら想定内である。
3. `idmagic:http_request_duration_seconds:p99_5m` を認証系の経路で見る。入場制御が効いていれば、ここは劣化していないはずである。効いていなければ、束縛条件は同時実行数ではなく別の場所にある。
4. API レプリカ数と HorizontalPodAutoscaler の状態を見る。上限 (`maxReplicas`) に張り付いているなら、増やせる余地がもう無い。

## ステージごとの対応

**`management_bulk` だけを捨てている場合。** 一括処理が集中している。誰が何を走らせているかを `http_requests_total` の `route` から特定する。運用上の締切がなければそのまま収束を待つ。急ぐなら、その処理を発行している側の同時実行を落とす。**入場制御の閾値を上げて通すのは、認証を守るための余白を削ることなので、認証系のレイテンシーを確認せずに行わない。**

**`management` まで捨てている場合。** 管理コンソールとポータルが使えなくなっている。障害対応そのものに管理 API が要る場合は、`ADMISSION_MANAGEMENT_MAX_CONCURRENT_REQUESTS` を一時的に引き上げるより、レプリカを増やすほうが安全である。前者は認証の余白を削るが、後者は総容量を増やす。ただしレプリカを増やすと論理接続予算も増えるので、PostgreSQL 側の接続使用率を先に確認する。

**`interactive_auth` まで捨てている場合。** 総容量が足りていない。優先度の付け替えでは解決しない。
- HorizontalPodAutoscaler が上限に達しているなら、上限を上げられるかを [capacity.md](../capacity.md#sizing-rules) の接続予算と 70% 規則で確かめてから上げる。
- PostgreSQL 側が束縛条件なら、レプリカを増やしても悪化する。接続の待ち時間と `DB_MAX_CONNS` を先に見る。
- 収まった後、[capacity.md](../capacity.md#degradation-order) の縮退順序に照らして、この事象が [contexts/system/decisions.md](../contexts/system/decisions.md#no-api-plane-separation) の再検討条件 (a) に当たるかを判断する。当たるなら記録を残す。

**分類の無い経路が現れた場合。** 経路を足したときに分類を足し忘れている。`TestEveryAssembledRouteDeclaresAPriorityClass` が本来これを配備前に落とす。落ちずにここまで来たなら、その検査が回っていないか、経路の登録が検査の見ている router を通っていない。どちらも配備の前に直す問題である。

## やってはいけないこと

- **`ADMISSION_CONTROL_ENABLED=false` で切って様子を見ること。** 切れば拒否は止まるが、飽和は止まらない。先着順に戻るので、次に落ちるのはログインである。切ってよいのは、閾値が明らかに誤っていて平常時に発動していると分かっている場合だけである。
- **閾値の順序を入れ替えること。** `management_bulk` の上限を `interactive_auth` より上にすると縮退の順序が逆になる。起動時設定の検証がこれを拒否するので、プロセスは起動しない。
- **拒否を 4xx に見せかけること。** ステージ 5 の 503 が `SLO-PRIMARY-ERRORS` の失敗として数えられるのは正しい。認証を受け付けられなかったことは失敗である。

## 収束後

平常時に 1 件も拒否が出ない状態へ戻ったことを `rate(http_admission_decisions_total{outcome="shed"}[5m])` で確かめる。閾値を触った場合は、既定値が Planning assumption であることを踏まえ、容量検証で置き直すまでの暫定であることを記録に残す。
