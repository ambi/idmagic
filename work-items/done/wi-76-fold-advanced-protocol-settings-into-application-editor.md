---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# 高度なプロトコル設定をアプリケーション編集画面へ畳み、低レベル client / RP 画面を撤去する

OIDC / WS-Federation の advanced 設定を Application 編集画面に畳み、低レベルの AdminClients /
AdminWsFedRelyingParties 画面を撤去した。ADR-066 を起票。
- OIDC: ApplicationOidcConfig / *UpdateRequest / AdminApplicationCreateRequest に
grant_types / response_types / PAR / DPoP と作成時の client_type / 認証方式 / jwks_uri /
tls_client_auth_subject_dn を追加。詳細リゾルバと UpdateClient 委譲で編集、作成フローで
認証方式を確定 (以後不変)。token_endpoint_auth_method 等は読み取り専用表示。
- WS-Fed: ApplicationWsFedConfig / *UpdateRequest に audience / token_type / claim 規則を
追加し、Application 編集画面に畳んだ。Entra domain federation はテナント/ドメイン単位の
操作のため、専用テナント設定画面 AdminEntraFederation (/admin/federation/entra) に分離。
- 画面撤去: AdminClients (/admin/clients) と AdminWsFedRelyingParties
(/admin/wsfed/relying-parties) の screen 定義・UI route・page・未使用 API ラッパを撤去。
低レベル HTTP API (/api/admin/clients, /api/admin/wsfed) は内部 API として存置。
- 旧導線 (ダッシュボード等) を /admin/applications へ修正。spec twin と生成 HTML を
lockstep 更新。scl.yaml には wi / ADR 参照を入れず純粋仕様文で記述。

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。
