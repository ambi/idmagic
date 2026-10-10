# アカウントポータルの設計

この文書は、[アカウントポータル](README.md)の要件を保証する仕組みを扱う。

## アーキテクチャ

ステップアップ認証が必要な操作は TypeSpec の注釈で宣言し、テスト（`TestStepUpAnnotatedInterfacesMatchGatedHandlers`）が、注釈と実際に制御されるハンドラーの不一致を防ぐ。
ログインの直後のセッションもステップアップ認証済みであり、Google や Okta で利用者が慣れている再認証の形と一致する。
`POST /api/account/step_up/complete` が成功すると、UI は元の要求を送り直す。

認証の新しさは `LoginSession` のレコード自体に `step_up_at` として保存するので、Cookie とともに別の端末へ引き継がれることはない。
