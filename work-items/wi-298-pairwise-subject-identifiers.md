---
status: pending
authors: [tn]
risk: medium
created_at: 2026-07-25
depends_on: []
change_kind: feature
initial_context:
  scl:
    OAuth2:
      - interfaces.UserInfo
      - interfaces.GetOpenidConfiguration
      - interfaces.Token
      - models.OAuth2Client
      - standards.OpenIDConnectCore
    IdManagement:
      - models.User
    Saml:
      - interfaces.SamlSingleSignOn
  decisions:
    - decisions/ADR-039-user-profile-shape.md
    - decisions/ADR-082-user-domain-id-and-tenant-key-policy.md
    - decisions/ADR-059-federation-bounded-context-and-claim-issuance.md
  source:
    - backend/oauth2/token/usecases
    - backend/oauth2/handlers_http
    - backend/claimmapping
    - backend/saml/usecases
  tests:
    - backend/oauth2/token/usecases
    - backend/oauth2/handlers_http
  stop_before_reading:
    - frontend
    - backend/provisioning
affected_spec:
  - { context: OAuth2, kind: interface, element: UserInfo }
  - { context: OAuth2, kind: interface, element: GetOpenidConfiguration }
  - { context: OAuth2, kind: model, element: OAuth2Client }
---

# pairwise subject identifier (相関防止のための per-client sub) に対応する

## Motivation

現在の `sub` は User の canonical な domain 識別子そのものである。SCL の
`IdManagement.models.User.id` の記述は「protocol 境界ではこの id を写像して表現する:
OIDC `sub` claim / SAML NameID / WS-Fed subject / SCIM リソース参照」とし、
「pairwise subject など写像を差し替える場合も id 自体は不変」と将来の余地に言及しているが、
**実装は存在しない**。discovery の `subject_types_supported` は `[public]` 固定である。

これは OpenID Connect Core §8 が定義する 2 つの subject type のうち、
`public` のみを提供している状態である。`pairwise` が無いことの意味:

1. **RP 間でユーザーを名寄せできてしまう**。同じテナントの複数アプリに同一の `sub` が
   出るため、アプリ間で結託すれば「このユーザーは A と B の両方を使っている」と
   突き合わせられる。社内アプリだけなら許容できても、外部 SaaS を多数繋ぐ構成や、
   B2C 的な用途では相関リスクになる。
2. **プライバシー要件を満たせない**。GDPR のデータ最小化の文脈で、
   「RP に渡す識別子は RP ごとに異なるべき」という要求は現実に存在する。
   `System.standards.GDPR` を宣言している以上、選択肢として持つべきである。
3. **識別子の意味が固定化する**。`sub` に内部 ID を直接出していると、
   将来 ID 体系を変えたときに全 RP が壊れる。pairwise の写像層を持つこと自体が、
   内部 ID と外部識別子を分離する構造になる。

競合比較:

- **Keycloak**: client ごとに `pairwise` subject type を選択でき、`sector_identifier_uri` にも対応。
- **Okta**: 既定は public だが、アプリごとの識別子ポリシーを持つ。
- **Entra ID**: アプリ単位の `oid` / `sub` の扱いを分けており、`sub` はアプリ固有である
  (つまり Entra は既定で pairwise 相当)。

つまり **Entra は既定で pairwise 相当、Keycloak は選択可能、IdMagic は不可**という位置にある。

## Scope

- **decision**:
  - 新規 ADR (subject identifier の写像): `public` を既定に維持する理由、`pairwise` の
    算出方式 (OIDC Core §8.1 に従い sector identifier + local account id + salt の
    ハッシュ)、sector identifier の決定規則 (`sector_identifier_uri` を対応するか、
    redirect_uri のホストから導出するか、Application 単位にするか)、salt の保管と
    ローテーション不可性 (salt を変えると全 pairwise sub が変わり RP のユーザー紐付けが
    壊れるため不変とする)、テナント境界との関係 (salt はテナントごと)、
    SAML NameID の `persistent` / `transient` 形式との対応、
    既存クライアントを public のまま維持する後方互換、
    切り替えの不可逆性 (public → pairwise はその RP のユーザー紐付けを壊す) を記録する。
- **scl**:
  - `OAuth2.models.OAuth2Client` に `subject_type` (public / pairwise) と
    `sector_identifier_uri` (対応する場合) を追加する。
  - `subject_types_supported` に `pairwise` を追加し、discovery
    (`GetOpenidConfiguration` / `GetOauthAuthorizationServer`) に反映する。
  - `Token` / `UserInfo` / ID Token の `sub` 算出を「client の subject_type に従う写像」として
    記述する。同一ユーザー・同一 sector で安定、異なる sector で異なることを requires に書く。
  - `ClaimMapping` に subject 写像の位置付けを追記する
    ([[ADR-059-federation-bounded-context-and-claim-issuance]] の claim 発行境界と整合)。
  - SAML の NameID 形式 (`persistent`) と pairwise の対応を `Saml` 側に記述する。
  - `states` に PairwiseSubjectIssued を追加する (初回算出の監査可能性のため)。
  - `scenarios`: pairwise クライアントで同一ユーザーの `sub` が安定する /
    異なる sector のクライアントで `sub` が異なる / 同一 sector の複数クライアントで
    `sub` が一致する / public クライアントの `sub` が従来と変わらない /
    introspection / UserInfo / ID Token で `sub` が一貫する /
    pairwise の `sub` から内部 ID を逆引きできない。
- **go**:
  - subject 写像を `backend/claimmapping` (または OAuth2 の token 発行経路) の
    単一の関数に集約し、ID Token / access token / UserInfo / introspection の
    すべてがそれを通ることを保証する。
  - pairwise の算出は `SHA-256(sector_identifier || local_account_id || tenant_salt)` の
    base64url とし、算出結果を永続化する (毎回算出でも一致するが、監査と逆引き用の
    マッピングを持つ)。
  - **逆引きの必要性**を扱う: 管理者が「この `sub` はどのユーザーか」を調べる必要があるため、
    `(tenant, sector, pairwise_sub) -> user_id` のマッピングを保存し、管理 API から
    解決できるようにする。ただしこのマッピングは管理者しか見られない。
  - SAML の `persistent` NameID にも同じ写像を適用する。
  - トークン検証・セッション紐付け・失効など、内部で `sub` を使っている経路が
    pairwise で壊れないことを確認する (内部処理は内部 ID を使い、`sub` は出力専用に限る)。
- **http**:
  - クライアント登録 (DCR) と管理 API で `subject_type` を受け付ける。
  - 管理 API に pairwise sub からユーザーを解決する参照を追加する。
- **ui**:
  - Application の OIDC 詳細設定に `subject_type` を追加し、切り替えの不可逆性
    (RP 側のユーザー紐付けが壊れる) を警告として表示する。
- **documentation**:
  - README に pairwise の設定、sector identifier の決め方、切り替えの影響を追記する。

## Out of Scope

- 既定を pairwise に変更すること。既定は `public` を維持する (既存 RP を壊さない)。
- `sector_identifier_uri` の外部取得。SSRF 面を作らないため、対応する場合も
  「登録時に管理者が入力した値の検証」に留め、実行時 fetch はしない
  ([[wi-293-request-object-jar-and-jarm-signed-authorization-messages]] と同じ判断)。
- ephemeral / transient な subject (毎回変わる識別子)。
- SCIM リソース参照や Provisioning の外部 ID への pairwise 適用。
  下流 SaaS は内部 ID の mirror であり、相関防止の対象ではない。
- 既存 public クライアントの一括移行ツール。

## Plan

- **写像を 1 箇所に集約するのが設計の核**。`sub` は ID Token / access token / UserInfo /
  introspection の 4 経路で出る。どこか 1 つが内部 ID を素で出すと pairwise の意味が
  消える。単一の写像関数を通すことを構造で強制し、テストで 4 経路の一貫性を固定する。
- **内部処理から `sub` を排除する**。セッション紐付け・失効・監査などの内部処理が
  `sub` 文字列を使っていると、pairwise 導入で静かに壊れる。内部は内部 ID を使い、
  `sub` は境界での出力専用にする。着手時にこの棚卸しを先に行う。
- **salt は不変にする**。salt をローテーションすると全 RP でユーザーが「別人」になり、
  RP 側のローカルアカウント紐付けが全て壊れる。テナント作成時に生成して不変とし、
  ADR に「ローテーション不可」を明記する。鍵ローテーション
  ([[ADR-009-key-rotation-strategy]]) の対象外であることも書く。
- **逆引きマッピングを持つ**。運用上「この sub は誰か」を調べられないと、
  障害調査とサポートが成立しない。マッピングを保存し、管理者のみ参照可能にする。
  これは相関防止 (RP 間) と運用可能性 (IdP 内) の両立である。
- **切り替えの不可逆性を UI で示す**。public → pairwise の変更は RP 側のユーザー紐付けを
  壊す。設定変更時に警告を出し、ADR にも影響として書く。
- **既定を変えないことで回帰面をゼロにする**。既存クライアントは全て public のままなので、
  `sub` の値は変わらない。これを回帰テストで固定する。
- 未決定: sector identifier を `sector_identifier_uri` で持つか、Application 単位にするか。
  この repo は Application を単一の編集面とする方針 ([[ADR-066-application-as-single-editor-surface]])
  なので、**Application を sector の単位とする**のが最も整合する。第一候補とし、
  OIDC 準拠のため `sector_identifier_uri` を将来の拡張余地として ADR に残す。

## Tasks

- [ ] T001 [Survey] `sub` を使っている経路を棚卸しする。出力経路 (ID Token / access token /
      UserInfo / introspection / SAML NameID) と、内部処理で `sub` 文字列を使っている箇所を
      分けて一覧化する。内部利用があれば内部 ID へ置換する対象として記録する。
- [ ] T002 [ADR] subject identifier の写像の ADR を起票する (算出方式・sector の単位・
      salt の不変性・逆引き・SAML NameID との対応・切り替えの不可逆性)。
- [ ] T003 [SCL] `OAuth2Client.subject_type`、`subject_types_supported` への pairwise 追加、
      Token / UserInfo / ID Token の写像記述、ClaimMapping / Saml への追記、event、
      scenario 6 件を追加し `just check-scl` を通す。
- [ ] T004 [Domain] subject 写像関数 (public / pairwise) を実装する。RED: 同一 sector で
      安定、異なる sector で異なる、public は従来と一致するテストを先に書く
      (scenario `OAuth2.pairwise_subject_stable_per_sector`) → GREEN。
- [ ] T005 [Internal ID] T001 で見つかった「内部処理が `sub` を使っている箇所」を内部 ID に
      置換する。RED: 該当経路が pairwise クライアントでも動くテスト → GREEN。
- [ ] T006 [Persistence] tenant salt と `(tenant, sector, pairwise_sub) -> user_id` の
      マッピングを `infra/schema/postgres.sql` に追加し、`just sqlc-generate` を実行する。
      salt はテナント作成時に生成し不変とする。
- [ ] T007 [Output paths] ID Token / access token / UserInfo / introspection の 4 経路が
      同一の写像を通ることを実装で強制する。RED: 4 経路で `sub` が一貫するテスト → GREEN。
- [ ] T008 [SAML] `persistent` NameID に同じ写像を適用する。RED: SAML の NameID が
      pairwise になるテスト → GREEN。
- [ ] T009 [Client] DCR と管理 API で `subject_type` を受け付ける。RED: 不正値の拒否と
      既定 public のテスト → GREEN。
- [ ] T010 [Admin] pairwise sub からユーザーを解決する管理 API を追加する。
      RED: 管理者以外が参照できないテスト → GREEN。
- [ ] T011 [Discovery] `subject_types_supported` に pairwise を追加する。
      RED: discovery の contract テスト → GREEN。
- [ ] T012 [UI] Application の OIDC 詳細設定に `subject_type` と切り替え警告を追加する。
      RED: presentation logic の unit test → GREEN。
- [ ] T013 [Docs] README に pairwise の設定・sector の決め方・切り替えの影響を追記する。
- [ ] T014 [Verify] 下記 Verification を緑にする。`just scl-render` を実行する。

## Verification

- `just check` / `just check-scl` / `just check-work-items` / `just check-ids`
- `just test-go` / `just test-go-race` / `just verify-go`
- `just verify-ui` / `just test-ui-unit`
- `just demo` — 既存の OIDC デモ (public クライアント) の `sub` が変わらないこと
- 手動: `just dev` で (1) pairwise クライアント 2 件 (別 Application) で同一ユーザーの
  `sub` が異なること、(2) 同一 Application の複数 client で一致すること、
  (3) ID Token / UserInfo / introspection の `sub` が一致すること、
  (4) 管理 API で pairwise sub からユーザーを解決できること、
  (5) SAML の persistent NameID が pairwise になること、を確認する。

## Risk Notes

**`sub` は RP 側でユーザーを識別する主キーとして使われる**。写像の実装ミスや salt の
変更は、全 RP でユーザーが「別人」になる不可逆な事故になる。salt を不変とし、
ADR にローテーション不可を明記し、切り替えの不可逆性を UI で警告する。
出力経路が 4 つあるため、どこか 1 つが写像を通らないと相関防止が破れる (かつ気付きにくい)。
4 経路の一貫性をテストで固定するのを完了条件にする。
内部処理が `sub` 文字列に依存している箇所を見落とすと、pairwise クライアントで
セッションや失効が静かに壊れる。T001 の棚卸しを実装前に完了させる。
既定を public に維持することで既存 RP への影響はゼロだが、それを回帰テスト
(`just demo` と contract テスト) で明示的に守る。
