# DEK のライフサイクル

## 概要

この文書は、テナントの `DataEncryptionKey` の生成、ローテーション、無効化、破棄の仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | 初回の暗号化での DEK の生成、ローテーション、危殆化に備えた即時の無効化、すべての参照の再暗号化の後の破棄 |
| 行為者 | System（ライフサイクルのユースケースと再暗号化のジョブ） |
| 扱わないもの | 再暗号化するレコードの読み書きは、項目を持つ各モジュールの `FieldMigrator` が扱う。DEK の状態の一覧は[DEK の健全性の一覧](../health/README.md)が扱う |

## モデル

最初のバージョンは、テナントの項目を初めて暗号化するときに生成し、`MasterKey` のプロバイダーでラップする。
ローテーションは、新しいバージョンを新規の暗号化に使う唯一の 1 本として有効化し、直前のバージョンを復号できるまま残す。
復号できるバージョンが複数あり、暗号化に使えるバージョンが 1 本しかないという非対称が、再暗号化を待つ間も読み取りを止めない根拠である。

破棄はレコードを削除せず、`wrapped_dek` だけを消す。
鍵素材を失った後も、そのバージョンがいつ有効化され、いつ退役し、いつ破棄されたかを参照できるようにするためである。

- **判断**：ライフサイクルの操作を HTTP に公開しない理由は、[DEK のライフサイクルの操作を HTTP に公開しない](../design/decisions.md#dek-のライフサイクルの操作を-http-に公開しない)。

## 状態遷移

### DataEncryptionKeyLifecycle

`bootstrap` で最初の鍵を `active` として生成する。ローテーションで新しいバージョンが `active` になると、旧バージョンは `retiring` へ遷移する。`retiring` の鍵は `disable` で即時に無効化できる。`retiring` または `disabled` の鍵を `destroyed` へ遷移できるのは、すべての参照を再暗号化したことを `Jobs` 経由で確認した後だけである。`active` の鍵は直接 `disable` または `destroy` できず、先にローテーションする必要がある。

| State | Kind | Meaning |
|---|---|---|
| active | initial | 新規の暗号化操作に使う。テナントごとに高々 1 本だけ存在する |
| retiring | — | 新規の暗号化には使わないが、既存の暗号文の復号には引き続き使う |
| disabled | — | 鍵素材の危殆化などにより手動で即時に無効化した。以後この鍵による復号はフェイルクローズで拒否する |
| destroyed | terminal | `wrapped_dek` を破棄し、暗号学的消去が成立した。元に戻せない |

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| active | DataEncryptionKeyRotated | — | retiring |  |
| retiring | DataEncryptionKeyDisabled | — | disabled |  |
| retiring | DataEncryptionKeyDestroyed | — | destroyed |  |
| disabled | DataEncryptionKeyDestroyed | — | destroyed |  |

行は鍵のバージョンの状態、列はそのバージョンへの操作である。
ローテーションはテナントの `active` のバージョンだけを退役させ、ほかのバージョンを変えない。

| State | ローテーション | 無効化 | 破棄 | 暗号文の復号 |
|---|---|---|---|---|
| active | → retiring | 拒否：InvalidRequestError | 拒否：InvalidRequestError | 何もしない |
| retiring | 何もしない | → disabled | → destroyed（参照の移行が完了）<br>拒否：DataKeyStillReferencedError（未移行の参照が残る） | 何もしない |
| disabled | 何もしない | 拒否：InvalidRequestError | → destroyed（参照の移行が完了）<br>拒否：DataKeyStillReferencedError（未移行の参照が残る） | 拒否：DataKeyUnavailableError |
| destroyed | 何もしない | 拒否：InvalidRequestError | 拒否：InvalidRequestError | 拒否：DataKeyUnavailableError |

## 操作

### テナントの初回の暗号化による DEK の生成

#### REQ-DATAKEYS-001 テナントの初回利用時に DEK を生成する

- テナントに DEK がない間、そのテナントの項目を初めて暗号化するとき、DataKeys は、新しい DEK を `MasterKey` のプロバイダーでラップし、バージョン 1 の `active` の DEK として `wrapped_dek` だけを保存し、`DataEncryptionKeyBootstrapped` を発行する。
- DEK を生成するとき、DataKeys は、平文の DEK を保存せず、イベントにも含めない。
- `MasterKey` のプロバイダーに到達できないか、ラップに失敗した場合、DataKeys は、DataKeyUnavailableError で失敗し、DEK を作らず、イベントを発行しない。
- **例**：EX-DATAKEYS-001-01、EX-DATAKEYS-001-02

### System による DEK のローテーション

#### REQ-DATAKEYS-002 DEK をローテーションしても既存の暗号文を復号できる

- System がテナントの DEK をローテーションしたとき、DataKeys は、次のバージョンの DEK を `active` にし、それまでの `active` のバージョンを `retiring` にし、`DataEncryptionKeyRotated` を発行する。
- System がテナントの DEK をローテーションしたとき、DataKeys は、登録したすべての `FieldMigrator` について、そのテナントの再暗号化のジョブを `Jobs` に予約する。
- 再暗号化のジョブの予約に失敗した場合、DataKeys は、ローテーションを取り消さず、定期の再暗号化で後から回収する。
- バージョンが `retiring` の間、そのバージョンで暗号化した値を復号するとき、DataKeys は、そのバージョンで復号する。
- DataKeys は、`active` のバージョンだけを新しい暗号化に使う。
- **例**：EX-DATAKEYS-002-01

### System による DEK の無効化

#### REQ-DATAKEYS-003 retiring の DEK を即時にロックアウトできる

- バージョンが `retiring` の間、System がそのバージョンを無効化したとき、DataKeys は、バージョンを `disabled` にし、キャッシュした DEK を捨て、`DataEncryptionKeyDisabled` を発行する。
- バージョンが `disabled` または `destroyed` の間、そのバージョンで暗号化した値の復号を要求されたとき、DataKeys は、DataKeyUnavailableError で拒否する。
- **例**：EX-DATAKEYS-003-01

#### REQ-DATAKEYS-004 active の DEK は直接 disable できない

- `active` のバージョンの無効化または破棄を要求された場合、DataKeys は、InvalidRequestError で拒否し、バージョンを `active` のまま残し、イベントを発行しない。
- `retiring` でないバージョンの無効化を要求された場合、DataKeys は、InvalidRequestError で拒否し、バージョンを変えない。
- **例**：EX-DATAKEYS-004-01

### System による DEK の破棄

#### REQ-DATAKEYS-005 すべての参照を再暗号化した後に DEK を destroy できる

- 登録したすべての `FieldMigrator` が `active` のバージョンへの未移行の参照を持たない間、System が `retiring` または `disabled` のバージョンを破棄したとき、DataKeys は、バージョンを `destroyed` にし、`wrapped_dek` を消し、記録を残し、キャッシュした DEK を捨て、`DataEncryptionKeyDestroyed` を発行する。
- 未移行の参照が一件でも残る場合、DataKeys は、DataKeyStillReferencedError で拒否し、バージョンと `wrapped_dek` を残し、イベントを発行しない。
- `destroyed` のバージョンの破棄を要求された場合、DataKeys は、InvalidRequestError で拒否する。
- **例**：EX-DATAKEYS-005-01、EX-DATAKEYS-005-02

## セキュリティ上の考慮

危殆化への対応では、再暗号化の完了を待たずに復号を止めたい。
そのための状態が `disabled` であり、破棄と違って再暗号化の完了を求めない代わりに、そのバージョンで暗号化された値は読めなくなる。
無効化と破棄のどちらも `active` のバージョンには適用できず、先にローテーションして退役させる。

DEK を生成するときにプロバイダーへ到達できなければ、テナントは平文の鍵を持たないまま失敗する。
ラップできない鍵を暫定的に平文で保持する経路はない。
