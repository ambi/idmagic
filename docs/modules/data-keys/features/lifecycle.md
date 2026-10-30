# DEK のライフサイクル

## 概要

この文書は、テナントの `DataEncryptionKey` の生成、ローテーション、無効化、破棄の仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | 初回の暗号化での DEK の生成、ローテーション、危殆化に備えた即時の無効化、すべての参照の再暗号化の後の破棄 |
| 行為者 | System（ライフサイクルのユースケース、定期実行のバッチ、再暗号化のジョブ）、System 管理者（制御面主体） |
| 扱わないもの | 再暗号化するレコードの読み書きは、項目を持つ各モジュールの `FieldMigrator` が扱う。DEK の状態の一覧は[DEK の健全性の一覧](health.md)が扱う |

## モデル

最初のバージョンは、テナントの項目を初めて暗号化するときに生成し、`MasterKey` のプロバイダーでラップする。
ローテーションは、新しいバージョンを新規の暗号化に使う唯一の 1 本として有効化し、直前のバージョンを復号できるまま残す。
復号できるバージョンが複数あり、暗号化に使えるバージョンが 1 本しかないという非対称が、再暗号化を待つ間も読み取りを止めない根拠である。

破棄はレコードを削除せず、`wrapped_dek` だけを消す。
鍵素材を失った後も、そのバージョンがいつ有効化され、いつ退役し、いつ破棄されたかを参照できるようにするためである。

- **判断**：テナント管理者へ公開しない理由は、[DEK のライフサイクルの操作をテナント管理者に公開しない](../decisions.md#dek-のライフサイクルの操作をテナント管理者に公開しない)。

## 状態遷移

### DataEncryptionKeyLifecycle

`bootstrap` で最初の鍵を `active` として生成する。ローテーションで新しいバージョンが `active` になると、旧バージョンは `retiring` へ遷移する。`retiring` の鍵は `disable` で即時に無効化できる。`retiring` または `disabled` の鍵を `destroyed` へ遷移できるのは、すべての参照を再暗号化したことを `Jobs` 経由で確認した後だけである。`active` の鍵は直接 `disable` または `destroy` できず、先にローテーションする必要がある。

状態遷移表（マトリクス形式）の行は鍵のバージョンの状態、列はそのバージョンへの操作である。
ローテーションはテナントの `active` のバージョンだけを退役させ、ほかのバージョンを変えない。

<!-- spec:state-machine backend/datakeys/testdata/DataEncryptionKeyLifecycle.statemachine.json -->

| State | Kind | Meaning |
| --- | --- | --- |
| active | initial | 新規の暗号化操作に使う。テナントごとに高々 1 本だけ存在する |
| retiring | — | 新規の暗号化には使わないが、既存の暗号文の復号には引き続き使う |
| disabled | — | 鍵素材の危殆化などにより手動で即時に無効化した。以後この鍵による復号はフェイルクローズで拒否する |
| destroyed | terminal | `wrapped_dek` を破棄し、暗号学的消去が成立した。元に戻せない |

| From | Event | Guard | To | Effects |
| --- | --- | --- | --- | --- |
| active | DataEncryptionKeyRotated | — | retiring |  |
| retiring | DataEncryptionKeyDisabled | — | disabled |  |
| retiring | DataEncryptionKeyDestroyed | — | destroyed |  |
| disabled | DataEncryptionKeyDestroyed | — | destroyed |  |

| State | ローテーション | 無効化 | 破棄 | 暗号文の復号 |
| --- | --- | --- | --- | --- |
| active | → retiring | 拒否：InvalidRequestError | 拒否：InvalidRequestError | 何もしない |
| retiring | 何もしない | → disabled | → destroyed（参照の移行が完了）<br>拒否：DataKeyStillReferencedError（未移行の参照が残る） | 何もしない |
| disabled | 何もしない | 拒否：InvalidRequestError | → destroyed（参照の移行が完了）<br>拒否：DataKeyStillReferencedError（未移行の参照が残る） | 拒否：DataKeyUnavailableError |
| destroyed | 何もしない | 拒否：InvalidRequestError | 拒否：InvalidRequestError | 拒否：DataKeyUnavailableError |

<!-- /spec:state-machine -->

## 操作

### テナントの初回の暗号化による DEK の生成

#### REQ-DATAKEYS-001 テナントの初回利用時に DEK を生成する

- テナントに DEK がない間、そのテナントの項目を初めて暗号化するとき、DataKeys は、新しい DEK を `MasterKey` のプロバイダーでラップし、バージョン 1 の `active` の DEK として `wrapped_dek` だけを保存し、`DataEncryptionKeyBootstrapped` を発行する。
- DEK を生成するとき、DataKeys は、平文の DEK を保存せず、イベントにも含めない。
- `MasterKey` のプロバイダーに到達できないか、ラップに失敗した場合、DataKeys は、DataKeyUnavailableError で失敗し、DEK を作らず、イベントを発行しない。
- **例**：EX-DATAKEYS-001-01、EX-DATAKEYS-001-02

### DEK のローテーション

#### REQ-DATAKEYS-002 DEK をローテーションしても既存の暗号文を復号できる

- テナントの DEK をローテーションしたとき、DataKeys は、次のバージョンの DEK を `active` にし、それまでの `active` のバージョンを `retiring` にし、`DataEncryptionKeyRotated` を発行する。
- テナントの DEK をローテーションしたとき、DataKeys は、登録したすべての `FieldMigrator` について、そのテナントの再暗号化のジョブを `Jobs` に予約する。
- 再暗号化のジョブの予約に失敗した場合、DataKeys は、ローテーションを取り消さず、REQ-DATAKEYS-007 の定期実行で後から回収する。
- 再暗号化のために読んだ値が書き戻す前に変わった場合、DataKeys は、新しい値を残し、その行を移行済みに数えない。
- バージョンが `retiring` の間、そのバージョンで暗号化した値を復号するとき、DataKeys は、そのバージョンで復号する。
- DataKeys は、`active` のバージョンだけを新しい暗号化に使う。
- ローテーションの確定の後に始まる暗号化を要求されたとき、DataKeys は、どのプロセスでも新しいバージョンで暗号化する。
- **例**：EX-DATAKEYS-002-01、EX-DATAKEYS-002-02

#### REQ-DATAKEYS-007 周期を過ぎた DEK を定期的にローテーションする

- `idmagic-batch data-key-lifecycle` の実行で、`active` のバージョンの有効化からの経過が周期（デフォルトは 90 日）以上のテナントを処理するとき、DataKeys は、DEK をローテーションする。
- 経過が周期より短いテナントを処理するとき、DataKeys は、ローテーションせず、未移行の行を報告した `FieldMigrator` の再暗号化のジョブだけを `Jobs` に予約する。
- DEK を持たないテナントを処理するとき、DataKeys は、DEK を作らず、何もしない。
- 一つのテナントの処理に失敗した場合、DataKeys は、残りのテナントの処理を続け、バッチを失敗で終了する。
- 周期に正の整数の日数以外を指定された場合、DataKeys は、依存を組み立てる前に設定のエラーでバッチを終了する。
- **判断**：一つのテナントの失敗で止めると、先頭の失敗が解消するまで後続のテナントの DEK が永久に入れ替わらない。

#### REQ-DATAKEYS-008 制御面主体はテナントの DEK をローテーションできる

- 制御面主体が制御面テナントの経路で対象のテナントの DEK のローテーションを要求したとき、DataKeys は、REQ-DATAKEYS-002 のとおりローテーションし、200 で `tenant_id`、`active_version`、`previous_version` を返し、`DataEncryptionKeyRotated` に操作者を `actorUserId` として残す。
- 応答を返すとき、DataKeys は、`wrapped_dek`、平文の DEK、`MasterKey` の識別子を応答に含めない。
- 対象のテナントが存在しない場合、DataKeys は、404 と TenantNotFoundError で拒否する。
- 対象のテナントが DEK を持たない場合、DataKeys は、400 と InvalidRequestError で拒否し、DEK を作らない。
- `system_admin` を持たないか、制御面テナントに所属しないか、要求先が制御面テナントでない利用者が要求した場合、DataKeys は、403 と `access_denied` で拒否し、DEK を変えない。
- API アクセストークンで要求された場合、DataKeys は、トークンのスコープによらず 403 と `insufficient_scope` で拒否する。
- ブラウザーからの要求であることを証明できない場合、DataKeys は、403 で拒否し、DEK を変えない。

### DEK の無効化

#### REQ-DATAKEYS-003 retiring の DEK を即時にロックアウトできる

- バージョンが `retiring` の間、そのバージョンを無効化したとき、DataKeys は、バージョンを `disabled` にし、キャッシュした DEK を捨て、`DataEncryptionKeyDisabled` を発行する。
- バージョンが `disabled` または `destroyed` の間、そのバージョンで暗号化した値の復号を要求されたとき、DataKeys は、どのプロセスでも DataKeyUnavailableError で拒否する。
- 復号の開始が無効化または破棄の確定より前の場合、DataKeys は、その復号を中断しない。
- バージョンの状態を読めない場合、DataKeys は、保持した DEK を使わず、暗号化と復号に失敗する。
- **例**：EX-DATAKEYS-003-01、EX-DATAKEYS-003-02

#### REQ-DATAKEYS-004 active の DEK は直接 disable できない

- `active` のバージョンの無効化または破棄を要求された場合、DataKeys は、InvalidRequestError で拒否し、バージョンを `active` のまま残し、イベントを発行しない。
- `retiring` でないバージョンの無効化を要求された場合、DataKeys は、InvalidRequestError で拒否し、バージョンを変えない。
- **例**：EX-DATAKEYS-004-01

#### REQ-DATAKEYS-009 制御面主体は retiring の DEK を無効化できる

- 制御面主体が制御面テナントの経路で対象のテナントのバージョンの無効化を要求したとき、DataKeys は、REQ-DATAKEYS-003 と REQ-DATAKEYS-004 のとおり処理する。
- 無効化に成功したとき、DataKeys は、204 を返し、`DataEncryptionKeyDisabled` に操作者を `actorUserId` として残す。
- REQ-DATAKEYS-004 で拒否する場合、DataKeys は、400 と InvalidRequestError で応答する。
- 対象のテナントが存在しない場合、DataKeys は、404 と TenantNotFoundError で拒否する。
- 対象のバージョンが存在しない場合、DataKeys は、404 と DataKeyVersionNotFoundError で拒否する。
- 制御面主体でない利用者、API アクセストークン、ブラウザーからの要求であることを証明できない要求の場合、DataKeys は、REQ-DATAKEYS-008 と同じ応答で拒否し、バージョンを変えない。

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

## 実現方式

### アーキテクチャ

破棄してよいかの確認は、このモジュール自身では行わず、項目を持つ各モジュールが登録した `FieldMigrator` の未移行の件数の報告に委ねる。
破棄は、登録したすべての `FieldMigrator` が、アクティブなバージョン以外で暗号化した行を 0 件と報告したときだけ進む。
いずれかが 1 件以上を報告している間は、`DataKeyStillReferencedError` で拒否する。

ローテーションは、確定した後に、登録した `FieldMigrator` ごとに `data_key_reencryption` のジョブを投入する。
ジョブは、テナントと `FieldMigrator` の組を重複排除の鍵にする。

| 段 | 内容 |
| --- | --- |
| 1 回の実行 | 200 行ずつ、最大 25 回の一括の再暗号化を進める |
| 残りがある場合 | 同じ重複排除の鍵で続きのジョブを投入して終わる |
| 別のテナントを指すパラメーター | ジョブのテナントと異なるテナントの再暗号化を拒否する |

### 信頼性

ローテーションの後のジョブの投入に失敗しても、ローテーション自体は確定したままにし、警告のログだけを残す。
取りこぼしは、毎日の `idmagic-batch data-key-lifecycle` が、未移行の行を報告する組にジョブを投入して回収する。
`idmagic-batch data-key-reencryption-sweep` は、運用者が手で起動し、すべてのテナントとすべての `FieldMigrator` の組にジョブを投入する。
どちらも重複排除により、何度実行しても安全である。

DEK を持たないテナントのジョブは、`ErrNoActiveDataKey` で失敗する。
そのテナントで次に項目を暗号化したときに DEK が生成されるので、その後に掃引を実行し直す。

アンラップした DEK は、各プロセスの中に `(テナント, バージョン)` ごとに保持する。
バージョンの状態は保持せず、暗号化と復号のたびに保存先から読む。
ほかのプロセスのライフサイクルの操作は、これで確定の直後から効く。
方式の比較は[判断](../decisions.md#バージョンの状態を暗号化と復号のたびに保存先から読む)、メモリに残る鍵素材は[DataKeys のリスク](../risks.md)が扱う。

状態を読めない間は、保持した DEK があっても暗号化と復号に失敗する。
データベースの停止中は暗号文も読めないので、この失敗で新たに止まる読み取りはない。
