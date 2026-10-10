# WI-83020: 主体の名前と User の作成経路を統一する

作業項目は `wi-83020-unify-principal-names-and-user-creation-paths` である。

WI-83020 は、User のユーザー名、Group と Agent の名前を一つの規則で比べ、User を作るすべての経路で同じ検証と動的グループの評価を行う。

ユーザー名は、Group と Agent の名前と同じく、大文字と小文字を区別せずに比べる。
`alice` がいるテナントでの `Alice` の作成は、`POST /api/admin/v1/users` が 409 と `username_conflict` で拒否し、User CSV では対象の照合と重複の判定で同じ User として扱う。
入力された表記は保存し、そのまま表示する。
比較は Unicode の case folding で行うので、`straße` と `STRASSE` のような名前も同じ名前になる。
規則は[IdManagement の値オブジェクト](../../modules/identity-management/README.md#値オブジェクト)が定め、[ユーザー](../../modules/identity-management/user/README.md)の REQ-IDMANAGEMENT-042 と REQ-IDMANAGEMENT-090、[グループ](../../modules/identity-management/group/README.md)の REQ-IDMANAGEMENT-060、[エージェント](../../modules/identity-management/agent/README.md)の REQ-IDMANAGEMENT-073 と REQ-IDMANAGEMENT-075、[グループ CSV](../../modules/identity-management/group-csv/README.md)の REQ-IDMANAGEMENT-072 が参照する。

メールアドレスは、テナントの中で削除されていない User の間で一意になる。
管理者による作成と更新は、ほかの User と大文字と小文字を区別せずに同じメールアドレスを 409 と `email_taken`（`EmailTakenError`）で拒否する。
User CSV は、ほかの User のメールアドレスの行を `email_taken`、前の行と同じメールアドレスの行を `duplicate_email` で `rejected` にする（[ユーザー CSV](../../modules/identity-management/user-csv/README.md)の REQ-IDMANAGEMENT-056）。

フェデレーションの JIT と User CSV の適用で作った User も、作成の時点で動的グループの規則で評価され、一致した Group に所属する。
CSV の適用は、更新した行と変更なしの行の User も評価するので、評価に失敗した適用は同じ CSV の再適用で回収できる。
規範上の条件は[ユーザー](../../modules/identity-management/user/README.md)の REQ-IDMANAGEMENT-089 と REQ-IDMANAGEMENT-043、[ユーザー CSV](../../modules/identity-management/user-csv/README.md)の REQ-IDMANAGEMENT-004 が定める。

PostgreSQL の `users`、`groups`、`agents` の表は、名前とメールアドレスの比較キーの列を持ち、名前の一意性をその列の一意索引で守る。
製品は未リリースなので、データの移行は行わない。
