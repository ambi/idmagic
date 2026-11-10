---
status: completed
---

# 評価の不許可から承認の開始まで Agent を導く

承認がないか使えないことで不許可にした評価が、Agent が承認を求めるための明細を返し、MCP ゲートウェイがそれを MCP クライアントへ伝えるようになった。

- REQ-OAUTH2-065：理由が `approval_required`、`approval_expired`、`approved_operation_mismatch` の不許可は、要求と登録の写しの明細を `authorization_remediation` に返す。使用済みの承認と、読めない引数には返さない。
- REQ-OAUTH2-062、065：`requires_approved_operation` の規則で明細のないトークンは、`authorization_detail_mismatch` ではなく、関係の判定の後に `approval_required` になる。許可の範囲は変わらないが、理由の文字列で分岐するリソースサーバーは影響を受ける。
- REQ-MCPGATEWAY-003：明細を伴う不許可を、403、`WWW-Authenticate: Bearer error="insufficient_authorization"` と base64url の `authorization_remediation`、JSON-RPC の `-32000` と `error.data.authorization_remediation` で返す。符号化した値が 8 KiB を超えるとヘッダーから除く。
- TypeSpec：`ResourceAccessEvaluationResponse.authorization_remediation` と `AuthorizationRemediation` を加えた。
- 標準：`docs/modules/mcp-gateway/standards.md` を新設し、draft-ietf-oauth-rar-metadata-remediation-00 に固定した。MCP の 401 の意味と衝突するので、草案の 401 ではなく 403 を使い、一回限りの承認なので `authorization_reference` を返さない。
- 主要 IdP（Auth0、Okta）と同じく、Agent のアプリが CIBA と RAR で承認を求める経路である。CIBA を呼ばない汎用の MCP クライアントの経路は wi-62682 が扱う。
- 検査ツール：新しい `standards.md` の表の見出し行 `ID` を、追加された標準要件として読む欠陥を直した。

重要な保証は、受け入れのテストが、ゲートウェイの示した明細をそのまま CIBA で要求して一回だけ実行できることと、実装から独立に書いた明細の形で確かめる。
実際の MCP SDK のクライアントが 403 のヘッダーと `error.data` を呼び出し元へ渡すかは確かめていない。
草案は変わり得るので、版を上げるときは標準の表と応答を見直す。
