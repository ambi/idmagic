# SSF ストリームの管理

## 概要

この文書は、テナント管理者が送信側と受信側の `SsfStream` を管理する仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | 送信側と受信側のストリームの登録、更新、無効化と再有効化、削除、配送の状況の参照 |
| 行為者 | テナント管理者 |
| 扱わないもの | 受信した SET の検証は[SET の受信](../receiver/README.md)が、配送は[SET の配送](../transmitter/README.md)が扱う |

## モデル

送信側のストリームは、配送先のエンドポイントと、送る CAEP のイベントの種類と、配送の試行の上限を持つ。
受信側のストリームは、信頼する発行者（`trusted_issuer`）とその鍵の取得先を持つ。
送信側と受信側のストリームは、`ssf_streams` の同じ上限を共有する。

## 状態遷移

### SsfStreamLifecycle

登録時に `enabled` として作成する。無効化すると `disabled` に遷移し、それ以降は配送も受信も行わない。再有効化すれば `enabled` に戻せる。削除はレコードと付随する送信側設定と受信側設定を取り除く終端の操作であり、以後の操作は存在しないストリームとして拒否する。

| State | Kind | Meaning |
|---|---|---|
| enabled | initial | 配送と受信を行う |
| disabled | — | 配送も受信も行わない。再有効化できる |
| deleted | terminal | レコードと付随する設定を削除した |

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| enabled | SsfStreamDisabled | — | disabled |  |
| disabled | SsfStreamEnabled | — | enabled |  |
| enabled | SsfStreamDeleted | — | deleted |  |
| disabled | SsfStreamDeleted | — | deleted |  |

| State | 無効化 | 再有効化 | 削除 | SET の受信 |
|---|---|---|---|---|
| enabled | → disabled | 何もしない | → deleted | 何もしない（受信側のストリーム）<br>拒否：400 security_event_rejected（送信側のストリーム） |
| disabled | 何もしない | → enabled | → deleted | 拒否：400 security_event_rejected |
| deleted | 拒否：404 ssf_stream_not_found | 拒否：404 ssf_stream_not_found | 拒否：404 ssf_stream_not_found | 拒否：400 security_event_rejected |

## 操作

### 管理者によるストリームの登録

#### REQ-SHAREDSIGNALS-009 SsfStream の登録は Hard Quota を超えると拒否される

- 管理者が送信側のストリームを登録したとき、SharedSignals は、配送先、audience、購読する CAEP のイベントの種類、配送の試行の上限（デフォルト 8 回）を持つ `enabled` のストリームを作り、テナントの `ssf_streams` の使用量を一つ増やし、201 とストリームを返し、`SsfStreamRegistered` を発行する。
- 管理者が受信側のストリームを登録したとき、SharedSignals は、`trusted_issuer`、鍵の取得先、受理する audience、購読する種類を持つ `enabled` のストリームを作り、テナントの `ssf_streams` の使用量を一つ増やし、201 とストリームを返し、`SsfStreamRegistered` を発行する。
- 管理者がストリームの購読する種類を更新したとき、SharedSignals は、200 とストリームを返し、`SsfStreamUpdated` を発行する。
- 管理者がストリームを削除したとき、SharedSignals は、ストリームと付随する設定を消し、テナントの `ssf_streams` の使用量を一つ減らし、204 を返し、`SsfStreamDeleted` を発行する。
- 管理者がストリームまたは配送の一覧を取得したとき、SharedSignals は、テナントのストリームまたはそのストリームの配送を 200 で返す。
- 送信側と受信側のストリームの合計が `ssf_streams` の上限に達している間、管理者がどちらかのストリームを登録したとき、SharedSignals は、422 と `quota_exceeded` で拒否し、`resource="ssf_streams"` の `QuotaExceeded` を発行し、ストリームも付随する設定も作らない。
- 購読する種類が空か、未知の CAEP のイベントの種類を含む場合、SharedSignals は、422 と `ssf_stream_event_types_required` または `ssf_stream_event_type_invalid` で拒否する。
- 送信側のストリームの配送先が `https://` で始まらないか、audience がない場合、SharedSignals は、422 と `ssf_transmitter_delivery_endpoint_invalid` または `ssf_transmitter_audience_required` で拒否する。
- 受信側のストリームの `trusted_issuer` が `https://` で始まらないか、`jwks_uri` と `jwks` のどちらもないか、受理する audience が空の場合、SharedSignals は、422 と `ssf_receiver_trusted_issuer_invalid`、`ssf_receiver_jwks_required`、`ssf_receiver_accepted_audiences_required` のどれかで拒否する。
- 存在しないか別のテナントのストリームを指定された場合、SharedSignals は、404 と `ssf_stream_not_found` で拒否する。
- 登録を拒否した場合、SharedSignals は、ストリームを作らず、`ssf_streams` の使用量を変えない。
- **例**：EX-SHAREDSIGNALS-009-01、EX-SHAREDSIGNALS-009-02、EX-SHAREDSIGNALS-009-03

#### REQ-SHAREDSIGNALS-011 SsfStream の登録と状態変更は管理者に限られる

- `admin` のロールを持たない利用者がストリームの登録、一覧、取得、更新、無効化、再有効化、削除、配送の一覧を要求した場合、SharedSignals は、403 と `access_denied` で拒否し、ストリームを変えない。
- **例**：EX-SHAREDSIGNALS-011-01、EX-SHAREDSIGNALS-011-02

### 管理者によるストリームの無効化

#### REQ-SHAREDSIGNALS-008 無効化したストリームでは配送も受理も行わない

- 管理者がストリームを無効化または再有効化したとき、SharedSignals は、ストリームを `disabled` または `enabled` にし、204 を返し、`SsfStreamDisabled` または `SsfStreamEnabled` を発行する。
- ストリームがすでにその状態の間、管理者が無効化または再有効化を要求したとき、SharedSignals は、204 を返し、イベントを発行しない。
- 受信側のストリームが `disabled` の間、SET を受けたとき、SharedSignals は、400 と `security_event_rejected` で拒否し、失効を反映しない。
- 送信側のストリームが `disabled` の間、`AgentAccessRevoked` を発行したとき、SharedSignals は、そのストリームの配送を作らない。
- **例**：EX-SHAREDSIGNALS-008-01

## セキュリティ上の考慮

`SsfStream` の登録、更新、有効化、無効化、削除と、配送の状況の参照は、`admin` ロールを持つ、有効かつ認証済みのユーザーだけが、所属テナントに対して行える。
