---
status: in_progress
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-10
priority: p3
depends_on:
  - wi-97546-move-modules-without-private-callers-behind-go-internal
  - wi-33994-reinventory-the-remaining-boundary-debt
  - wi-35767-publish-claim-issuance-as-claimmapping-public-operations
  - wi-60465-publish-job-enqueue-and-handler-registration-as-jobs-ports
  - wi-87746-publish-authentication-operations-other-modules-use
  - wi-93464-move-authorization-login-steps-into-authentication
  - wi-13438-publish-oauth2-consent-revocation-and-client-administration
  - wi-93579-publish-idmanagement-user-operations-and-transactional-writer
  - wi-90942-relocate-saml-assertion-building-out-of-wsfederation
  - wi-28791-publish-application-sign-in-policy-evaluation
  - wi-74257-move-tenancy-claimmapping-and-jobs-behind-go-internal
change_kind: refactor
affected_spec:
  - { path: spec/modules/application/models.tsp, symbol: IdMagic.Contract.FapiProfile, impact: modifies }
---

# 残りのモジュールを Go の internal/ へ移し、legacy の公開方式を撤去する

## 動機

[外から非公開パッケージへの import がないモジュールの移行](../done/wi-97546-move-modules-without-private-callers-behind-go-internal.md)の後には、ほかのモジュールから非公開パッケージへ import される 8 個のモジュール（起票時は Tenancy、Authentication、ClaimMapping、Jobs、OAuth2、IdManagement、WsFederation、Application）が `legacy` のまま残る。
これらの依存は[負債の棚卸し](../done/wi-33994-reinventory-the-remaining-boundary-debt.md)が起票する解消の項目と、[テナントの公開契約](../done/wi-39119-publish-the-resolved-tenant-as-tenancy-public-language.md)で取り除く。
取り除いた後に移す作業と、全モジュールの移行後に `legacy` の命名による判定を消す作業を受け持つ項目がない。
依存が先に解消した Tenancy、ClaimMapping、Jobs は [3 モジュールの移行](../done/wi-74257-move-tenancy-claimmapping-and-jobs-behind-go-internal.md)が移し、この項目は残る 5 モジュールを受け持つ。

## 対象範囲

- 外から非公開パッケージへの import が解消したモジュールを `internal/` へ移し、責務表の公開方式を `internal` にする。
- 全モジュールが `internal` になった後、境界検査、責務表、構造の文書から `legacy` の公開方式と、組み立て地点からの非公開パッケージへの import を移行期間だけ許す規則を外す。
  公開目的と外側に残る実装を責務表と照合する規則は残す。
- 負債の基準 revision の比較に使う旧形式の互換処理のうち、`legacy` を前提にするものを、基準 revision が新形式になった後で外せるか判断する。

## 対象外

- 外から非公開パッケージへの import そのものの解消。負債の棚卸しが起票する項目とテナントの公開契約の項目で行う。
- モジュールの分割と統合。

## 設計

着手の条件は、対象のモジュールへの `private-import` が台帳から消えていることである。
一部のモジュールの依存だけが先に解消した場合は、そのモジュールから先に移してよい。
全モジュールの移行前に `legacy` を撤去しない。

公開パッケージの決め方は、先行する移行の項目と同じく[境界を選ぶ判断手順](../../docs/design/application/design-guidelines.md#境界を選ぶ判断手順)を適用する。

### 着手時の棚卸し

2026-10-17 の台帳で残る `private-import` は `Saml->WsFederation` と `Tenancy->WsFederation` の 2 件で、どちらも [SAML のアサーション生成の移設](../done/wi-90942-relocate-saml-assertion-building-out-of-wsfederation.md)（`pending`）が解消する。
IdManagement、Authentication、OAuth2、Application は今回移せる。
WsFederation の移行と `legacy` の撤去（T003、T004）は wi-90942 の完了を待つ。

### 境界の判断の記録

入力は責務表、本番コードの import の辺、組み立て地点（`backend/cmd`、`server_http`、`testing_stack`、`fixtures_postgres`）が名指す非公開パッケージである。
適用する制約は[公開範囲と `internal/`](../../docs/design/application/backend.md#公開範囲と-internal)の最終形で、組み立て地点はルートパッケージと公開パッケージだけを import できる。

| 判断 | 候補 | 採否と理由 |
| --- | --- | --- |
| 公開パッケージ | `domain` と `ports` の区画のうち、ほかのモジュールの本番コードが import するものだけを列挙する | 採用。`legacy` が暗黙に公開していた区画のうち、利用者のないものは公開しない |
| 組み立て地点が名指す保存先 | ルートパッケージに永続層ごとの `NewMemoryModule` と `NewPostgresModule` を置く | 採用。Tenancy、Jobs と同じ形で、組み立て地点は保存先の型を名指さない |
| 同 | 保存先のアダプターを公開パッケージにする | 不採用。アダプターはほかのモジュールの利用目的を持たない |
| ほかのモジュールのテストが使うメモリの保存先 | ルートの `NewMemoryModule` が返す `Module` のポート型のフィールドを使う | 採用。Tenancy の移行と同じで、公開を増やさない |
| `testing_stack` の具体型のフィールド | ポート型に変える | 採用。組み立て地点が非公開の型を名指さない |

### 実装で決めたこと

組み立て地点に残っていた業務の組み立ては、各モジュールのルートパッケージの操作へ移した。

| モジュール | 公開パッケージ | ルートへ移した組み立て |
| --- | --- | --- |
| Application | `domain`、`ports` | 永続層ごとの `Module` の生成 |
| OAuth2 | `domain`、`ports`、`client/ports`、`consent/ports`、`logout/ports` | 永続層ごとの生成、CIMD の解決とイベントの発行先、経路の登録、サインインの続き、リソースのアクセストークンの検証、back-channel logout のクライアント |
| IdManagement | `domain`、`user`、`group`、`agent` の `domain` と `ports` | 永続層ごとの生成、CSV のジョブの登録、ライフサイクルと所属の操作、CSV の成果物の掃除、経路の登録、外部 IdP の利用者の作成、主体の解決 |
| Authentication | `domain`、`mfa/domain`、`signin/ports`、`password`、`recovery`、`session`、`totp`、`trusteddevice`、`webauthn` の公開ポートと、そのポートが使う `domain` | 永続層ごとの生成、再暗号化の移行器の登録、WebAuthn の RP、セッション管理、セキュリティ通知、保持期間の掃除、経路と外部 IdP のログインの登録 |

着手時の仮定に当たった個所の扱いは次のとおりである。

| 状況 | 扱い | 理由 |
| --- | --- | --- |
| ほかのモジュールのテストがメモリの保存先の `Seed` を使う | ポートの `Save` へ置き換えた | `Seed` は `Save` を呼ぶだけである |
| ほかのモジュールの結合テストが IdManagement の User の作成、変更、削除を呼ぶ | ルートに `AdminUsers` を置いた | 利用者が判断した。ルートは組み立て地点しか import できないので、ほかのモジュールの本番コードへの公開は増えない |
| 組み立て地点の型が非公開の型を名指す（`SessionManager`、`AuthEventBucketStore`、外部 IdP の主張、利用者の作成の入力） | ルートに型の別名を置いた | 組み立て地点だけが名指す |
| テストだけが使う解決器と計算（`DemoHeaderResolver`、`GenerateTOTP`） | ルートに別名と関数を置いた | 同上 |
| `fixtures_postgres` がモジュールのルートを import すると、そのモジュールの内側のテストと循環する | `SeedUser`、`SeedGroup`、`SeedClient` は保存先を引数で受け取る | 所有モジュールの内側のテストからも同じ fixture を使える |
| 具体型の時計や `ChangePassword` など、所有モジュールの内側でしか触れられない振る舞いを外から確かめていたテスト | 所有モジュールの側へ移した | 観測する振る舞いは変えない |
| 公開ポートの型が使うドメインの型（`totp`、`webauthn`、`recovery`、`trusteddevice` の `domain`、OAuth2 の承認要求と logout） | 公開した、または `oauth2/domain` の別名に加えた | 公開ポートの値をほかのモジュールと組み立て地点が組み立てる |

負債の比較検査は `private-import` の利用元だけを読み替えていたため、`table-write` の SQL のパスも同じ規則で読み替えるよう直した。

## タスク

- [x] T001 [Inventory] 対象のモジュールへの `private-import` が解消済みかを確かめ、未解消のものの解消の項目を特定する。
  未解消は WsFederation への 2 件で、wi-90942 が解消する。
- [ ] T002 [App] 解消したモジュールから順に `internal/` へ移し、責務表を `internal` にする。
  Application、OAuth2、IdManagement、Authentication を移した。
  WsFederation は wi-90942 の完了後に移す。
  検証は `mise run lint-go`、`mise run check-boundaries`、`mise run check-boundary-debt-ratchet`、`mise run test-go-changed`。
  Authentication のルートの経路依存から `SessionManager` を外す誤実装を注入し、`server_http` のテスト 87 件が失敗することを確かめた。
  負債の比較検査の読み替えは、`table-write` の移動を扱うテストを先に失敗させてから直した。
- [ ] T003 [Tooling] 全モジュールの移行後に `legacy` の判定と移行期間の規則を境界検査から外し、fixture を更新する。
- [ ] T004 [Docs] 責務表と構造の文書から `legacy` の公開方式を外す。
- [ ] T005 [Verify] 変更を検証する。

## 検証

- `mise run check-boundaries`
- `mise run check-boundary-debt-ratchet`
- `mise run verify`

## リスク

- 依存の解消が遅れると、この項目が長く着手できない。
  解消したモジュールから移せるようにし、`legacy` の撤去だけを最後に回す。
- `legacy` の撤去と同時に、公開目的の照合まで外してしまうおそれがある。
  撤去するのは命名による判定だけとし、`internal` の照合の fixture を残す。
