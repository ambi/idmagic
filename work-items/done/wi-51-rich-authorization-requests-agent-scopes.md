---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# Rich Authorization Requests (RFC 9396) でエージェント権限を細粒度に束縛する

ADR-050 に基づき RFC 9396 Rich Authorization Requests を spec-first で実装した。
テナントが事前登録した AuthorizationDetailType の検証スキーマ (フィールドごとに
set / at_most / enum / exact の半順序) に対し、各 authorization_details を
構文・必須・許可値・未登録フィールド拒否で fail-closed に検証する (ValidateAgainstType)。
/authorize・/par・/token で受理し、検証済み詳細を AuthorizationRequest → 同意 Consent →
AuthorizationCodeRecord → access token の authorization_details claim まで貫通させる。
構造化詳細があるときは粗い scope の過去同意で自動スキップせず明示同意を必須とする。
token-exchange (RFC 8693) では要求詳細を subject_token の詳細の部分集合に限定し
(DetailsSubsetOf)、詳細なしの交換は元の詳細を保持のみ・決して拡張しない。introspection は
検証済み JWT から詳細を抽出して返し (RS の検証点)、discovery は Enabled な type を
authorization_details_types_supported として広告する。type 定義は新 permission
AdminAuthorizationDetailTypesManage 配下の admin API / 管理 UI で CRUD でき、同意画面は
登録 type の表示テンプレートで詳細を人間可読に提示する。

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。
