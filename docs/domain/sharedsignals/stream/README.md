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

登録時に `enabled` として作成する。無効化すると `disabled` に遷移し、それ以降は配送も受信も行わない。再有効化すれば `enabled` に戻せる。削除は状態遷移ではなくレコードそのものを取り除く終端操作であり、付随する送信側設定と受信側設定をカスケード削除する。

| State | Kind | Meaning |
|---|---|---|
| enabled | initial | 配送と受信を行う |
| disabled | — | 配送も受信も行わない。再有効化できる |

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| enabled | SsfStreamDisabled | — | disabled |  |
| disabled | SsfStreamEnabled | — | enabled |  |

## 操作

### 管理者によるストリームの登録

#### REQ-SHAREDSIGNALS-009 SsfStream の登録は Hard Quota を超えると拒否される

#### REQ-SHAREDSIGNALS-011 SsfStream の登録と状態変更は管理者に限られる

### 管理者によるストリームの無効化

#### REQ-SHAREDSIGNALS-008 無効化したストリームでは配送も受理も行わない

## セキュリティ上の考慮

`SsfStream` の登録、更新、有効化、無効化、削除と、配送の状況の参照は、`admin` ロールを持つ、有効かつ認証済みのユーザーだけが、所属テナントに対して行える。
