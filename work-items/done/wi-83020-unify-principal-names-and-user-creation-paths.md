---
status: completed
archived_from: 8d750a4bcdde9dab091b3d106572abf0c387ca9d
---

# User、Group、Agent の名前を一つの値オブジェクトで比較し、User を作るすべての経路で同じ検証と動的グループの評価を行う

`mise run spec-diff -- main` は、REQ-IDMANAGEMENT-089 と REQ-IDMANAGEMENT-090 の追加、REQ-IDMANAGEMENT-004、042、043、056、060、072、073、075 の変更、`EmailTakenError` の追加、`CreateAdminUser`、`UpdateAdminUser`、`UsernameConflictError`、`GroupNameConflictError`、`AgentNameConflictError` の変更を示す。
User のユーザー名、Group と Agent の名前は、IdManagement の値オブジェクト（名前）として、前後の空白を除き、表記を保ったまま Unicode の case folding で比較する。
メールアドレスは同じテナントの削除されていない User の間で一意になり、管理者による作成と更新は 409 と `email_taken`、User CSV は `email_taken` と `duplicate_email` で重複を拒否する。
管理者による作成とフェデレーションの JIT は一つの作成の関数を通り、JIT と User CSV の適用で作った User も動的グループの規則で評価する。CSV の適用は、更新と変更なしの行も評価し、評価に失敗した適用を再適用で回収できる。
PostgreSQL は `users`、`groups`、`agents` に比較キーの列を持ち、名前の一意性をその列の一意索引で守る。

全文は `git show 8d750a4bcdde9dab091b3d106572abf0c387ca9d:<このファイルのパス>` で読む。
