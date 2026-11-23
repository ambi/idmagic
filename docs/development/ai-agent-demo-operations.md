# Agent の停止、棚卸し、監査

この文書は、Agent の停止と復旧、所有者の変更、資格情報の交換、棚卸し、外部観測の照合、監査と障害時の確認を説明する。
[ガイドの入口](ai-agent-demo.md)から、認証、委任、認可、接続の各手順へ進める。

## 停止と復旧

一時的な停止は「無効化」、恒久的な緊急停止は「緊急停止 (kill)」を使う。
kill は取り消せず、再有効化、編集、資格情報の追加、削除ができないため、一時停止の代わりには使わない。

1. 管理画面の Agent 一覧で対象の所有者と認証用クライアントを確認する。
2. Agent を無効化する。
3. 既発行トークンで Gateway を呼び、上流へ届かないことを確認する。
4. 新しいトークンの発行も拒否され、イントロスペクションが `active=false` になることを確認する。
5. 復旧するときは、所有者が有効であることを確かめて Agent を有効化し、新しいトークンを取得する。

停止前のトークンは、Agent を再有効化しても復活しない。
停止は次のオンライン評価から効き、すでに始まった業務処理を巻き戻さない。
Gateway を使わず JWT の署名と `exp` だけを検査するリソースには、停止の情報が即時には届かない。
即時停止が必要な操作は、IdMagic の評価または同じ失効判定を通るイントロスペクションを業務処理の直前に行う。

## 変更と反映時点

| 操作 | 対象範囲 | 反映時点と復旧 |
| --- | --- | --- |
| MCP 操作や対象の割当を取り消す | その Agent の該当操作と対象 | 既発行トークンでも次の評価から拒否。再割当で次の評価から変更 |
| CEL や実行回数の上限を変える | 規則に対応する操作 | 次の評価から適用。回数の窓と単位の変更は新しい計数になる |
| 発行ガードレールを変える | 関わる Agent の以後の発行 | 次の発行から適用。既発行トークンは取り消さない |
| 本人が Agent の付与を撤回する | 本人とその Agent が関わる代行 | 次の交換とオンライン評価で停止。他の利用者の付与は維持。再付与しても古いトークンは復活しない |
| 本人がアプリへの OAuth 同意を撤回する | 本人と OAuth クライアント | 将来の認可に作用。Agent の付与取消や、既発行トークンの即時失効と同じ操作ではない |
| 外部 OAuth 接続を切断する | 本人とコネクタ | 次の代理実行から停止。ほかの本人の接続は維持 |
| TokenVault コネクタを無効化する | 全利用者のそのコネクタ | 新しい連携と次の代理実行を停止。接続を保持する |
| ワークロードの信頼や関連付けを無効化する | その証明を使う以後の認証と交換 | 発行済みトークンを止める操作は別途必要 |
| Agent の資格情報の紐付けを解除する | その紐付けと、発行時に Agent が関わったトークン | Agent の失効エポックが進み、既発行トークンもオンライン判定で失効。解除した OAuth クライアント自体の管理も別に行う |
| Agent を無効化する | その Agent と、発行に関わる委任の連鎖 | 新しい発行と既発行トークンのオンライン利用を停止。有効化後は新しい発行が必要 |
| Agent を kill する | 同上 | 恒久停止。再有効化不可。記録は残る |
| Agent を削除する | 有効または無効の Agent と紐付け | ID と紐付けを削除し、既発行トークンもオンラインで失効。kill 済みは削除不可 |
| 所有者を無効化または論理削除する | そのテナントで所有する Agent | 所有する有効な Agent を無効化し、失効も反映。所有者を復旧しても Agent は自動復旧しない |
| MCP リソースサーバーを無効化する | その接続先 | 新しい対象トークンの発行と Gateway の次の評価・転送を停止。JWT 単体を一括失効させる操作ではない |

設定が保存されたこと、発行が止まったこと、既存の業務要求が止まったことをそれぞれ確認する。
権限の説明に使う `impact` の変更だけでは許可や拒否は変わらない。

## 所有者と資格情報の管理

### 所有者の変更

Agent の編集で、引き継ぐ有効な利用者を所有者に指定する。
`AgentOwnerChanged` を確認し、新しい所有者でガードレールの承認や棚卸しを行う。
退職者などを停止してから所有 Agent を見落とすことを防ぐため、停止前に棚卸しを行い、必要な Agent の所有者を引き継ぐ。
所有者の変更は、利用者が各自に与えた Agent の付与を自動で作成する操作ではない。

### 公開鍵と認証用クライアントの交換

1. 新しい鍵を Agent 側で作り、新しい公開鍵を登録した認証用クライアントを準備するか、既存クライアントの公開 JWKS を更新する。
2. 新しいクライアントを使う場合は同じ Agent に紐付け、必要なグラントとスコープを設定する。
3. 新しい資格情報でトークンの発行と対象操作を確認する。
4. 古いクライアントの紐付けを解除し、古い OAuth クライアントも運用上不要なら無効化または削除する。
5. 紐付け解除は Agent の既発行トークンを失効させるため、切り替え後に新しいトークンを取り直す。

公開鍵の更新だけで、以前に発行したアクセストークンまで即時に失効するとは判断しない。
漏洩時は Agent の無効化で利用を止め、鍵の交換と確認を終えてから新しい発行で再開する。
JWKS URL の取得とキャッシュ、ワークロードの信頼鍵の更新は別の仕組みのため、各認証方式の更新結果も確認する。

## 棚卸しと権限の資料

管理画面の「エージェント」から「棚卸し」を開く。
API では `GET /api/admin/v1/agent-inventory` を使い、`dormant_days` で休眠の判定日数を指定する。
デフォルトは 90 日、範囲は 1 日から 365 日である。

| 所見 | 意味 | 対応 |
| --- | --- | --- |
| `owner_missing` | 同じテナントの有効な所有者を確認できない | 所有者を引き継ぐか Agent を停止する |
| `dormant` | 有効だが、登録と最終発行の新しい方から指定日数を過ぎている | 用途を確認し、不要なら無効化する |
| `unbound` | 認証用クライアントがない、または紐付け先が存在しない | 意図した未設定か、削除や構成変更の漏れかを確認する |

最終発行は `AccessTokenIssued` と `TokenExchanged` の監査記録から求める。
最後の業務実行時刻とは異なり、休眠の所見だけで外部システムで動いていないとは断定しない。
無効化済みと kill 済みの Agent は、動かなくなったことを休眠として扱わない。

Agent の権限の資料には、紐付くクライアントのスコープ、本人ごとの付与と期限・最終利用、操作と対象の割当、条件、承認、回数制限、ガードレールなどを集める。
API は `GET /api/admin/v1/agents/{agent_id}/access-summary` である。
これは判定の材料であり、「この Agent が今すべて実行できる操作」の保証ではない。
実際の利用者、対象、引数、トークン、時刻を指定した認可評価で許可を確認する。

## 外部で観測した Agent の照合

外部の発見器が報告した主体を取り込み、管理済み Agent と突き合わせる。
IdMagic 自体が外部 SaaS を巡回したり、ネットワークを走査してすべての Agent を発見したりする機能ではない。

1. 発見器用に `agent-observations:import` だけを持つ API アクセストークンを発行する。
2. 発見器が `POST /api/admin/v1/agent-observations/import` に 1 件から 100 件のバッチを送る。
3. 管理者が観測一覧で、issuer、subject、発見元、時刻と照合候補の根拠を確認する。
4. 管理済み Agent と一致するなら確認し、対象外なら除外する。誤った決定を直す場合は再開して未決定に戻す。
5. 確認した観測が棚卸しに添えられ、Agent の所有者や資格情報が変わっていないことを確認する。

要求は次の形で、`observed_at` は実際の観測時刻に置き換える。
観測元の ID は本文で自己申告せず、取り込みに使った API トークンから記録される。

```json
{
  "observations": [
    {
      "issuer": "https://workload.example.com",
      "subject": "repo:example/doc-agent:job:nightly",
      "display_name": "文書集計 Agent",
      "resources": ["documents"],
      "observed_at": "2026-11-23T00:00:00Z"
    }
  ]
}
```

候補の根拠はワークロードの関連付け、または `external_registration_id` と Agent に紐付くクライアント ID の一致であり、表示名の一致は使わない。
候補が一つでも自動確定しない。
管理者は候補外の Agent への確認もできるため、根拠を確認して決定する。
取り込み、確認、除外、再開のどれも、Agent の新規登録、所有者の変更、認証の紐付け、権限の付与を行わない。

取り込みだけのトークンでは一覧や確認はできず、一覧には `agents:read`、決定には `agents:write` が必要である。
30 日を過ぎた観測は古いものとして表示し、180 日を過ぎた未決定と除外は保持期限の削除の対象になる。
確認済みの観測はこの期限では削除しない。

## 監査と相関

管理画面の「監査イベント」で Agent と期間を絞り、発行から操作の判定まで追う。
Gateway のログには評価応答と同じリクエスト ID が残るため、IdMagic のイベントと突き合わせる。

| 確認する段階 | 主なイベントや記録 |
| --- | --- |
| 登録と資格情報 | `AgentRegistered`、`AgentCredentialBound`、`AgentCredentialUnbound`、`AgentOwnerChanged` |
| 発行と委任 | `AccessTokenIssued`、`TokenExchanged`、`TokenExchangeRejected`、`AgentDelegationGranted`、`AgentDelegationRevoked` |
| ワークロード | `WorkloadTokenExchanged` と信頼設定、関連付け、Agent、audience |
| 発行の制限 | `AgentGuardrailEvaluated` の結果、理由、バージョン、リクエスト ID |
| 人の承認 | `BackchannelAuthRequested`、`BackchannelAuthApproved`、評価結果の承認 ID と `operation_digest` |
| 操作の判定 | `ResourceAccessEvaluated` の `requestId`、主体、Agent、行為者の連鎖、条件のバージョンと結果 |
| Gateway | `forwarded`、`denied`、`input_required`、`approval_outcome_unknown` など。Envoy 構成では `permitted` |
| 外部サービス | `TokenVaultExecutionPerformed`、`TokenVaultExecutionDenied`、本人の接続の状態 |
| アプリ間 | 発行元の `IdJagIssued`、受け入れ先の `IdJagRedeemed` または `IdJagRejected` |
| 停止 | `AgentDisabled`、`AgentEnabled`、`AgentKilled`、`RevocationEpochAdvanced`、`AgentAccessRevoked` |
| 影響の説明 | `PermissionImpactChanged` と対象の登録内容 |
| 外部観測 | `AgentObservationsImported`、`AgentObservationConfirmed`、`AgentObservationExcluded`、`AgentObservationReopened` |

`ResourceAccessEvaluated` の許可や Gateway の転送記録は、上流の業務成功とは別に確認する。
承認した操作は承認 ID と引数の digest を業務結果に結び付け、結果不明を成功や安全な未実行と推定しない。
トークン、秘密鍵、外部 OAuth の資格情報をログへ追加して追跡しない。

### 外部システムへの失効通知

Shared Signals の送信ストリームを設定すると、Agent の失効を購読先へ SET として配送できる。
受信ストリームでは、登録した信頼先の署名、宛先、主体などを検証して受理した通知から Agent または利用者の失効を反映する。
送信先への到達失敗は再試行の対象となり、IdMagic 内の停止の反映は配送完了を待たない。

利用する場合はストリームの方向、イベント種別、署名鍵、audience、配信先と主体の解決を設定し、送信失敗時もローカルの `active=false` が維持されることを確認する。
送信は `POST /api/admin/v1/shared-signals/streams/transmitter`、受信は同じ配下の `/receiver` で登録する。
送信では配送先の HTTPS URL と audience、`session-revoked` など購読するイベントを指定し、受信では `trusted_issuer`、JWKS、`accepted_audiences` とイベントを指定する。
作成した `stream_id` の `/deliveries` で配送状況を確認し、必要なら `/deliveries/redeliver` で再配送する。
外部への配送成功だけで、受信先のすべての業務操作が即時停止したとは判断しない。
設定と検証の契約は [Shared Signals](../modules/sharedsignals/README.md)にある。

## 期待する結果にならない場合

| 症状 | 確認する順序 |
| --- | --- |
| `invalid_client` | Agent ID ではなくクライアント ID か、登録した公開鍵と署名鍵が一致するか、アサーションの `aud`、時刻、`jti` が正しいか |
| `invalid_target` | 登録 URI と `resource` が完全一致するか、接続先が有効か、通常の Token Exchange で `audience` を送っていないか |
| `invalid_scope` | クライアント、対象、元トークンの上限。カタログ方式の `mcp` をツール別権限と取り違えていないか |
| `unauthorized_client` | 監督下 Agent の通常発行か、グラント未登録か、ガードレールが承認を要求したか |
| 代理実行が `invalid_grant` | 本人がすべての関与 Agent に付与したか、期限とスコープ、元トークンの失効、`may_act`、委任の深さ、送信者の証明 |
| `decision=false` | `reason` を確認し、操作登録、現在の割当、対象、明細、条件、承認、回数制限を順に照合する |
| `condition_indeterminate` | 引数を属性に渡しているか、属性の名前と型が式に合うか |
| ツールが実行できるはずなのに拒否 | 公開と Agent の割当は両方あるか、対象を許可したか、トークンのスコープの上限に含まれるか |
| 停止したのに動く | Gateway を迂回していないか、判定をキャッシュしていないか、JWT の署名だけで許可していないか、停止前に開始した処理ではないか |
| `approval_already_used` または結果不明 | 自動再送を止め、承認 ID と digest で上流の実行結果を確認する |
| TokenVault の 404 / 409 | 本人の接続とコネクタがあるか、有効か、再認可待ちか |
| マネージド MCP の上流接続に失敗 | HTTPS の公開アドレスか、DNS が非公開アドレスを含まないか、上流認証と専用 JKT、外部接続の規則が合うか |
| 再起動後のデモだけ失敗 | 使い捨てのデータと鍵が作り直されるため、古い ID やトークンを使わず設定からやり直す |

発行ガードレールの詳細な拒否理由は発行応答に出さず、監査で確認する。
エラーを無視して上流へ送る回避策は作らず、判定した接続先と実際の実行先を照合する。

## 検証の組み合わせ

| 環境 | 実行するもの | 確認できる範囲 |
| --- | --- | --- |
| 初回の実演 | `mise run demo-agent` | 管理画面での登録、割当、対象別制限、停止、CIBA 発行 |
| 実在の SDK と Gateway | `mise run test-mcp-client-walkthrough` | トークン取得、許可、拒否、取消、Agent 停止。LLM の判断は含まない |
| 画面の通常の操作 | `mise run test-ui-e2e-file -- tests/e2e/agent-demo-operations.spec.ts` | 閲覧、更新、削除を別々に割り当てる操作 |
| 設定済みの開始点 | `mise run test-ui-e2e-file -- tests/e2e/agent-demo-preconfigured.spec.ts` | 一括準備した値と対象別の拒否 |
| 実在の Envoy | `mise run test-envoy-mcp-gateway` | 配布設定の拒否、ヘッダー、経路、Gateway 障害時の挙動。Docker が必要 |

本人の付与、ID-JAG、操作承認、マネージド MCP、TokenVault、ワークロードの個別検証は各ページのコマンドを使う。
ブラウザーと HTTP 統合テストには localhost の待受けが必要であり、実行環境のサンドボックスが禁止する場合は、その外で実行する。
テスト用 IdP や上流の合格と、実際の外部サービスへ接続した結果は区別して記録する。

契約の詳細は [Agent](../modules/identity-management/features/agent.md)、[外部観測](../modules/identity-management/features/agent-observation.md)、[トークンの失効](../modules/oauth2/features/token.md)、[失効エポック](../modules/sharedsignals/features/revocation.md)を参照する。
