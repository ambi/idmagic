---
status: completed
archived_from: 610341bde014725da511dbb909e29fdcfbeaa085
---

# `WriteBrowserError` を削除し Problem Details 移行完了を docs/SPECIFICATION.md に記録する

レガシー `{error, message}` envelope を返す `WriteBrowserError` を
`backend/shared/http/support_http/response.go` から削除した。
`wi-327`〜`wi-339` で 14 コンテキスト・約 400 箇所の呼び出しがすべて
`WriteProblem` へ移行済みで、削除時点の残存参照は定義本体とコメントだけ
だった。`api_error_language_test.go` の tracking map からもエントリを外し、
英語リテラル検査の対象は `WriteProblem` 側で引き継がれる。
`docs/SPECIFICATION.md` の HTTP error responses 節は、起票時に想定していた
「未実装」段落が既に存在しなかった (`ARCHITECTURE.md` から
`SPECIFICATION.md` へ移る際に現在形へ書き直されていた) ため、代わりに
実装完了で確定した内容を反映した: Problem Details を適用しない例外の一覧に
SharedSignals の受信エンドポイント (RFC 8935、`/ssf/streams/{stream_id}/events`)
を加え、境界がパッケージ単位ではなく接点単位で引かれることを明記した。
これは `wi-335` で receiver 用のエラー経路を分離した結果、コード上も
はっきりした事実である。
これで汎用 API のエラーレスポンスは全コンテキストで
`application/problem+json` (`type`/`title`/`status`/`detail`/`instance`) に
統一され、標準がエラー形式を定める接点 (OAuth2 / SCIM / DCR /
SharedSignals 受信エンドポイント) だけが例外として残る。
**対応していないこと** (いずれも `wi-382` に引き継ぎ済み):
- `WriteRateLimited` (429) は `{error, retry_after_seconds, message}` の
まま。`retry_after_seconds` を運ぶ独自形状で、Problem Details 化するなら
拡張メンバーの扱いを決める必要があるため、この work item の
「レガシーヘルパーの削除」とは別種の変更として残す。契約側が 7 operation で
`application/problem+json` を宣言している食い違いは `wi-382` T002。
- SharedSignals 受信エンドポイントの body が RFC 8935 §2.3 の
`{err, description}` と一致していない既知の乖離 (`wi-335` Risk Notes) は
未対応のまま。実装の変更なので `wi-382` の Out of Scope に置き、そこから
別 work item を起票する。
- 移行中に見つかった契約の穴 (`UpdateUserProfile` の error union に
`InvalidUserAttributeError` と 401/404 が無い、
`StartBrowserMfaEnrollment`/`ConfirmBrowserMfaEnrollment` に
`MfaEnrollmentNotAllowedError` が無い、export 系 operation に 429 の宣言が
無く `quota_exceeded` が別概念と code を共有している、
`ReceiveSecurityEvent` に 413/404 の宣言が無い) は `wi-382` の
Motivation 6 と T009 に一覧として移した。

全文は `git show 610341bde014725da511dbb909e29fdcfbeaa085:<このファイルのパス>` で読む。
