# 接続

## 概要

この文書は、本人が外部サービスを連携、解除し、IdMagic がトークンを保管する仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | 連携の開始、コールバック、一覧、解除、保管の暗号化 |
| 行為者 | ブラウザーセッションの本人、上流の認可サーバー |
| 扱わないもの | 更新は[代理実行](execution.md)が扱う |

## 状態遷移

### ConnectionLifecycle

`Connection.status` の状態遷移を表す。

<!-- spec:state-machine backend/tokenvault/testdata/ConnectionLifecycle.statemachine.json -->

| State | Kind | Meaning |
| --- | --- | --- |
| active | initial | トークンを保管し、代理実行に使える |
| reauthorization_required | — | 上流がトークンを受け付けなくなり、トークンを消した。本人の再連携を待つ |
| removed | terminal | 本人が解除したか、コネクタを削除した。行は残らない |

| From | Event | Guard | To | Effects |
| --- | --- | --- | --- | --- |
| active | TokenVaultConnectionReauthorizationRequired | `invalid_grant`、または更新トークンのない期限切れ | reauthorization_required | 保管したトークンを消す |
| reauthorization_required | TokenVaultConnectionLinked | — | active |  |
| active | TokenVaultConnectionLinked | 再連携 | active | トークンと付与スコープを置き換える |
| active | TokenVaultConnectionUnlinked | — | removed | 行を消す |
| reauthorization_required | TokenVaultConnectionUnlinked | — | removed | 行を消す |

| State | 再連携 | 更新の成功 | 再認可が必要な更新の失敗 | 解除 | 代理実行 |
| --- | --- | --- | --- | --- | --- |
| active | → active | 何もしない | → reauthorization_required | → removed | 何もしない |
| reauthorization_required | → active | 何もしない | 何もしない | → removed | 拒否：409 connection_reauthorization_required |
| removed | 何もしない（新しい行を作る） | 何もしない | 何もしない | 拒否：404 connection_not_found | 拒否：404 connection_not_found |

<!-- /spec:state-machine -->

## 操作

### 保管

#### REQ-TOKENVAULT-003 保管するシークレットはレコードに束縛して暗号化する

- アクセストークン、更新トークン、クライアントシークレット、PKCE の検証子を保存するとき、TokenVault は、テナント、テーブル、レコード、項目に束縛したテナントの DEK で暗号化する。
- 別のレコードへ複製した暗号文を読むとき、TokenVault は、復号の失敗として扱い、平文へ退避しない。
- DEK を再暗号化するとき、TokenVault は、接続とコネクタの項目を有効なバージョンへ移す。
- 再暗号化の読み取りの後に行が更新されていたとき、TokenVault は、新しい値を上書きしない。

### 本人による連携

#### REQ-TOKENVAULT-005 連携はブラウザーセッションの本人だけが開始できる

- 本人が `active` のコネクタの連携を開始したとき、TokenVault は、一度だけ使える `state`、`S256` の PKCE、コネクタごとのリダイレクト先を持つ連携を 10 分の期限で作り、200 と `authorization_url` を返す。
- 同じ本人とコネクタの連携が進行中のとき、TokenVault は、それを新しい連携で置き換える。
- 要求するスコープが省略されたとき、TokenVault は、コネクタの `allowed_scopes` を要求する。
- `allowed_scopes` にないスコープを要求された場合、TokenVault は、400 と `invalid_scope` で拒否する。
- ブラウザーセッションでない要求を受けた場合、TokenVault は、403 と `access_denied` で拒否する。
- コネクタが存在しないか `disabled` の場合、TokenVault は、404 と `connector_not_found` で拒否する。
- **判断**：トークンで開始させない。Agent が自分で接続を作れてしまうためである。

#### REQ-TOKENVAULT-004 コールバックは開始した本人とコネクタに束縛し、一度だけ受ける

- 開始した本人のセッションで、有効な `state` と `code` を受けたとき、TokenVault は、`state` を消費し、PKCE の検証子でコードを交換し、接続を `active` で保存し、`TokenVaultConnectionLinked` を発行し、アカウントポータルへ `result=linked` で戻す。
- トークンの応答が `scope` を持たないとき、TokenVault は、要求したスコープを付与スコープとして保存する。
- すでに接続があるとき、TokenVault は、同じ接続のトークンと付与スコープを置き換える。
- `state` が未知、消費済み、期限切れ、別のコネクタのものである場合、TokenVault は、接続を変えず、`result=invalid_state` で戻す。
- セッションの本人が開始した本人と異なるかセッションがない場合、TokenVault は、`state` を消費せず、接続を変えず、`result=invalid_state` で戻す。
- 上流が `error` を返した場合、TokenVault は、`state` を消費し、接続を変えず、`result=denied` で戻す。
- コードの交換に失敗するか、応答が `access_token` を持たないか、`token_type` が `Bearer` でない場合、TokenVault は、接続を変えず、`result=exchange_failed` で戻し、`TokenVaultConnectionLinkFailed` を発行する。
- 戻り先へ戻すとき、TokenVault は、上流の値を戻り先に含めない。

### 本人による一覧と解除

#### REQ-TOKENVAULT-006 本人は自分の接続だけを一覧でき、トークンは現れない

- 本人が一覧したとき、TokenVault は、200 と `active` のコネクタごとの名前、接続の状態、付与スコープ、`last_used_at` を返す。
- `active` でないコネクタの接続があるとき、TokenVault は、解除できるようにその接続も一覧に含める。
- 接続を返すとき、TokenVault は、トークンと有効期限を含めない。

#### REQ-TOKENVAULT-007 解除は上流の結果を待たずに次の実行から止める

- 本人が接続を解除したとき、TokenVault は、接続を消し、204 を返し、`TokenVaultConnectionUnlinked` を発行する。
- コネクタが `revocation_endpoint` を持つとき、TokenVault は、解除の確定の後に更新トークンの失効を上流へ一度だけ試し、結果をイベントの `upstreamRevocation` に記す。
- 接続が更新トークンを持たないとき、TokenVault は、代わりにアクセストークンの失効を試す。
- 上流の失効に失敗した場合、TokenVault は、解除を成功のままにする。
- 本人の接続がない場合、TokenVault は、404 と `connection_not_found` で拒否する。
- ブラウザーセッションでない要求を受けた場合、TokenVault は、403 と `access_denied` で拒否する。

## 実現方式

連携の `state` はハッシュだけを保存する。消費は行の削除で原子的に行う。
