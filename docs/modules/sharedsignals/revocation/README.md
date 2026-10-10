# 失効エポック

## 概要

この文書は、Agent ごとの失効エポックを進め、発行済みのトークンをイントロスペクションで即時に無効にする仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | Agent のイベントと受信した SET を契機とする失効エポックの前進、イントロスペクションへの反映、外部への伝播との切り離し |
| 行為者 | テナント管理者（Agent の強制終了）、System（イベントへの反応） |
| 扱わないもの | Agent の状態の変更は `IdManagement` が、外部への配送は[SET の配送](../transmitter/README.md)が扱う |

## モデル

失効エポックは単調にしか進まない。
既存の値と同じ時刻への前進も進めず、`RevocationEpochAdvanced` と `AgentAccessRevoked` を発行しない。
既存の値より前の時刻へ戻す経路はなく、遅れて到着した SET や再送によって、一度失効させたトークンが再び有効になることはない。

| 契機 | 進める対象 |
| --- | --- |
| `KillAgent`、`DisableAgent`、`UnbindAgentCredential` | その Agent |
| 所有者（`owner_user_id`）の無効化と削除 | 対象テナントの中で `owner_user_id` が一致するすべての Agent |
| 受理した SET | SET が指す Agent |

所有者の停止と、それに伴う Agent の無効化（`IdManagement` の `AgentDisabled`）は同じ時刻を持つ。
そのため、Agent ごとの失効は一度だけ記録され、理由は先に届いた所有者の停止になる。

- **判断**：外部への伝播をローカルの失効の後に行う理由は、[ローカルの失効を外部への伝播より先に確定させる](../design/decisions.md#ローカルの失効を外部への伝播より先に確定させる)。

## 操作

### 管理者による Agent の強制終了

#### REQ-SHAREDSIGNALS-001 キルスイッチは発行済みトークンをイントロスペクションで即時無効化する

- 管理者が Agent を強制終了、無効化するか、資格情報の束縛を解除したとき、SharedSignals は、その Agent の失効エポックを現在時刻へ進め、`RevocationEpochAdvanced` と `AgentAccessRevoked` を発行する。
- 所有者の User を無効化または削除したとき、SharedSignals は、テナントの中で所有者が一致するすべての Agent の失効エポックを進める。
- 発行時刻が Agent の失効エポックより前のアクセストークンのイントロスペクションを受けたとき、SharedSignals は、トークンを `active=false` として判定させる。
- 発行時刻が失効エポック以後のトークンのイントロスペクションを受けたとき、SharedSignals は、そのトークンを失効の対象にしない。
- 失効エポックを既存の値と同じかそれより前の時刻へ進める要求を受けた場合、SharedSignals は、エポックを変えず、`RevocationEpochAdvanced` と `AgentAccessRevoked` を発行しない。
- 失効エポックを判定できない場合、SharedSignals は、トークンを無効とみなさせる。
- **例**：EX-SHAREDSIGNALS-001-01、EX-SHAREDSIGNALS-001-02

#### REQ-SHAREDSIGNALS-007 受信側の障害はローカル失効を遅らせない

- 送信側のストリームの受信側へ到達できない間、Agent を失効させたとき、SharedSignals は、配送の成否を待たずに失効エポックを進めてイントロスペクションへ反映し、そのストリームの配送を `pending` のまま再試行の対象にする。
- **例**：EX-SHAREDSIGNALS-007-01

### 所有者のオフボーディング

#### REQ-SHAREDSIGNALS-002 所有者のオフボードは配下エージェント群を一括失効する (superseded by REQ-PLATFORM-001)

引き金の `DisableAdminUser` は IdManagement の操作である。所有者の無効化がログイン、既存セッション、配下エージェントのトークンを同時に閉じることを、REQ-PLATFORM-001 が 1 つの保証として述べる。

## セキュリティ上の考慮

失効エポックの前進（`AdvanceRevocationEpoch`）と参照（`CheckRevocationEpoch`）は HTTP に公開せず、ドメインイベントと `OAuth2` の `Introspect` からの内部の呼び出しに限る。
エポックを巻き戻す操作は、どの権限にも存在しない。
失効エポックを判定できない場合は、トークンを無効とみなす。
