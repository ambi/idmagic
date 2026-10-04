# 信頼済みデバイスの設計

この文書は、[信頼済みデバイス](README.md)の要件を保証する仕組みを扱う。

## アーキテクチャ

cookie には `selector.verifier` を入れ、サーバーは `selector` と `SHA-256(verifier)` だけを保存する。
`selector` は一意なのでレコードを 1 件だけ引け、`verifier` のハッシュは定数時間で比べる。
全レコードを走査して総当たりの時間差を晒すことも、cookie の平文を保存することもない。
cookie は realm ごとの名前とパスを使う既存の部品で発行するので、あるテナントで記憶した端末が、別のテナントのログインへ持ち込まれることはない。
属性は `HttpOnly`、`SameSite=Lax`、発行者が HTTPS なら `Secure` とする。

評価は、サインインポリシーが MFA を求め、かつセッションがまだ第二要素を持たないときにだけ行う。
照合に成功するたびに verifier を回転させて cookie を再発行するので、盗まれた古い cookie は、正規の利用者が次に同じ端末でログインした時点で無効になる。
回転は正規の利用者の側にも観測できる作用を残すので、盗難が静かに続く状態を作らない。

ステップアップの再認証は `StepUpMethod` に列挙した要素だけで成り立ち、`tdev` はその選択肢に入らない。

## データ

`trusted_devices` も `authentication_sessions` と同じ理由で、通常の[`tenant_id` の保持区分](../../../design/data/database.md#tenant_id-の保持区分)の例外として `tenant_id` を保持する。
このレコードを引く鍵は不透明な cookie の `selector` であり、ログインのたびにテナントの境界を確かめる条件が、`users` への結合ではなくレコードそのものに要るからである。
親が全体で一意なので、外部キーは `users(id)` への単一のカラムとする。
`verifier_hash` は SHA-256 の 16 進である。
索引は `(tenant_id, user_id, last_used_at DESC)` の部分索引 1 本で、本人の一覧と一括の失効の両方を賄う。

## 検証

信頼済みデバイスの cookie の `ParseCookie` は、Go のネイティブのファジングで `FormatCookie` との往復を表明する。
最初ではなく最後の `.` で切るような変更を入れると、selector と verifier の対応がずれて、別の端末の記録に対して verifier を照合することになる。
