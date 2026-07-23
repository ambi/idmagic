---
status: completed
authors: [tn]
risk: low
created_at: 2026-07-24
depends_on: []
---

# APIエラーメッセージを汎用英語から具体的な英語に置き換え、EnglishErrorTextを削除する

## Motivation
現在、`backend/shared/kernel/error_text.go` に定義された `EnglishErrorText` 関数が API 境界で用いられており、呼び出し元が渡した日本語のエラーメッセージを検知すると、一律で汎用的な英語メッセージ (`The request could not be completed.`) に置換しています。
この仕様により、呼び出し元が詳細な日本語エラーメッセージを渡しても、結局すべて同じ汎用的な英語メッセージに潰されてしまい、API クライアントにとって具体的な原因が全くわからない状態になっています。
これを改善するため、呼び出し側から具体的な英語のエラーメッセージを直接渡すように修正し、無意味な動的置換を行う `EnglishErrorText` を削除します。

## Scope
- `spec/contexts/system.yaml` の `BackendErrorText`、`BackendErrorResponse`、シナリオ「バックエンドAPIエラーは英語で返る」
- `backend/shared/kernel/error_text.go` の `EnglishErrorText` および関連テストの削除。
- `EnglishErrorText` を使用しているエラー構築関数（`NewOAuthError`, `OAuthErrorBody` など）からの同関数呼び出しの削除。
- 上記のエラー構築関数を呼び出している各ハンドラーやユースケースにおいて、渡されている日本語のメッセージを具体的な英語メッセージへと書き換える作業。

## Out of Scope
- UI が表示する画面内のバリデーション文言やエラー文言の変更。
- API が返す機械可読なエラーコード（`error` や HTTP ステータスコード）自体の変更。

## Plan
- バックエンドのコードベース全体を対象に、APIレスポンスの生成箇所（HTTP、OAuth/OIDC、SAML、SCIM など）でエラーメッセージとして渡されている日本語文字列を抽出する。
- それらの日本語メッセージを、意図を正確に反映した具体的な英語メッセージに書き換える。
- API 境界での安全網として導入されていた `EnglishErrorText` を取り除き、実装側で直接英語メッセージを指定する形に純化する。

## Tasks
- [x] T000 [SCL] `BackendErrorResponse` とシナリオ「バックエンドAPIエラーは英語で返る」が要求をすでに表現していることを確認する。
- [x] T001 [Backend] 各種エラー生成関数（`NewOAuthError`, `OAuthErrorBody`、`WriteBrowserError` など）の呼び出し元を特定し、日本語で渡されているエラーメッセージを具体的な英語メッセージに置き換える。
- [x] T002 [Backend] `NewOAuthError` や `OAuthErrorBody` などに残っている `EnglishErrorText` の適用を削除する。
- [x] T003 [Backend] `backend/shared/kernel/error_text.go` とそのテストを削除する。
- [x] T004 [Adapter] RED: `TestAPIErrorLiteralTextIsEnglish` が日本語 API エラーリテラルを検出して失敗することを `just test-go-package ./backend/shared/http/support_http` で先に確認（SCL `BackendErrorResponse` / シナリオ「バックエンドAPIエラーは英語で返る」）→ 全対象の英語化後に GREEN。
- [x] T005 [Verify] エラーメッセージ変更に伴い影響を受けるテストの期待値を更新し、`just verify` を通す。

## Verification
- `just yaml-check-work-items`
- `just check-ids`
- `just verify`

## Risk Notes
既存の API クライアントが `error_description` に設定された汎用英語メッセージ (`The request could not be completed.`) の完全一致に依存している場合は影響を受けます。しかし、クライアントは本来 `error` のコード値に依存するべきであり、記述内容の具体化によるリスクは仕様上許容されます。

## Completion
- **Completed At**: 2026-07-24
- **Summary**:
  `WriteBrowserError`、`NewOAuthError`、OAuth 認可 redirect、SCIM error、SAML / WS-Federation の拒否本文へ渡す日本語リテラルを具体的な英語へ置換した。
  `err.Error()` 経由で外部へ露出していた監査検索、認可トランザクション、動的な認可拒否理由の日本語発生源も英語化した。
  `TestAPIErrorLiteralTextIsEnglish` を追加し、既知の API / protocol error sink に日本語リテラルが再混入した場合に検出する。

### Affected Guarantees State
- `BackendErrorText`: API 境界から返す人間可読メッセージを具体的な英語へ統一した。
- `BackendErrorResponse`: HTTP、OAuth/OIDC、SAML、SCIM、WS-Federation のエラーレスポンスに日本語リテラルが混入しないことを回帰テストで保証した。

### Evidence
- API エラー生成箇所を検査する `TestAPIErrorLiteralTextIsEnglish` を追加し、実装前の RED と英語化後の GREEN を確認した。
- wi-216 の completion schema と work item 依存参照を修正し、リポジトリ全体の work item 検証を通過させた。
- `ARCHITECTURE.md` の依存関係を現行実装へ同期し、管理ユーザー属性エディターを分割して architecture complexity gate を通過させた。

- **Verification Results**:
  - `just yaml-check-scl` — passed
  - `just test-go-package ./backend/shared/http/support_http` — RED を確認後、GREEN
  - `just test-go` — passed
  - `just verify-go` — passed
  - `just verify-ui` — passed
  - `just test-tools` — passed
  - `just typecheck-tools` — passed
  - `just traceability-strict` — passed
  - `just yaml-check-architecture` — passed
  - `just verify` — passed
