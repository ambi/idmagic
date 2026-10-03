# wi-96676-transcribe-implicit-specifications-of-identity-management

IdManagement がこれまで実装だけで守っていた挙動を、規則として文書で約束するようになった。
振る舞いは変わらない。
利用者が依存してよい境界値、デフォルト値、応答の形が、次の規則で読めるようになった。

- 複数の機能に共通する規則：ロールの正規化、CSV の見出しの照合、列の数が合わない行の扱い、先頭のアポストロフィーの復号、転送ポリシーのデフォルト値と境界、属性値のセルの字句形、エクスポートの開始の検証、一覧、取り消し（[`REQ-IDMANAGEMENT-033`](../../domain/identity-management/README.md)〜`REQ-IDMANAGEMENT-041`）
- ユーザー：作成のユーザー名とパスワード、JIT の項目、期限を過ぎた削除予約の完全削除、更新の `changed_fields`、無効化と必須操作の再実行、削除の予約、復元の猶予期間の境界、完全削除（[`REQ-IDMANAGEMENT-042`](../../domain/identity-management/user/README.md)〜`REQ-IDMANAGEMENT-050`）
- アカウントのセルフサービス：プロフィールの更新の併合、本人へ開示する属性、メールアドレスの変更の起票と確定（[`REQ-IDMANAGEMENT-051`](../../domain/identity-management/account/README.md)〜`REQ-IDMANAGEMENT-054`）
- ユーザー CSV：属性の列の接頭辞、行の対象と重複、組み込み列のセル、CSV で作成する User、所有を判定できないときの拒否（[`REQ-IDMANAGEMENT-055`](../../domain/identity-management/user-csv/README.md)〜`REQ-IDMANAGEMENT-059`）
- グループ：名前と連絡先の正規化、属性スキーマのないテナント、更新の記録、メンバーの追加の対象、下流への通知の失敗（[`REQ-IDMANAGEMENT-060`](../../domain/identity-management/group/README.md)〜`REQ-IDMANAGEMENT-064`）
- 動的グループ：式の制約、規則の版と有効化、一致する User、無効化による所属の解除、古い再評価のジョブ、所属の変化のイベント、プレビューの上限（[`REQ-IDMANAGEMENT-065`](../../domain/identity-management/dynamic-group/README.md)〜`REQ-IDMANAGEMENT-071`）
- グループ CSV：名前の照合、連絡先と動的規則のセル（[`REQ-IDMANAGEMENT-072`](../../domain/identity-management/group-csv/README.md)）
- エージェント：登録の名前と所有者、資格情報の束縛、更新の記録、無効化と再有効化、停止した Agent の操作と削除（[`REQ-IDMANAGEMENT-073`](../../domain/identity-management/agent/README.md)〜`REQ-IDMANAGEMENT-078`）

いくつかの規則には、維持するか是正するかが決まっていない点を要判断として残した。
是正する場合は、別の変更で知らせる。
