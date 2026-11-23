---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-11-23
priority: p3
group: development-process
depends_on: []
change_kind: maintenance
spec_impact: { kind: none, reason: "Directory Connector の LDAP の接続、TLS の検証、bind、ページング、差分取得の規範（REQ-DIRECTORYCONNECTOR-002 ほか）を変えず、それを確かめる OpenLDAP 試験のサーバーの入手方法だけを変える。" }
---

# OpenLDAP 試験のサーバーを Debian の slapd から自前で組み立てる

## 動機

OpenLDAP の結合試験と Connector の E2E は、第三者のイメージ `osixia/openldap:1.5.0` を起動している（`backend/directoryconnector/internal/testing_openldap/server.go` の定数 `image`）。
このイメージには次の問題がある。

- 2021-02-19 のリリースが最後の安定版で、同梱の OpenLDAP は 2.4.57 である。現行の安定系列は 2.6 系であり、試験は利用者が使う系列より古いサーバーで振る舞いを確かめている。
- 同梱の証明書は期限が切れている。`Start` は試験ごとに CA とサーバーの証明書を作り、コンテナ内の `/container/service/slapd/assets/certs/` へ差し込んで回避している。
- 2021 年のビルドなので、ベースの OS のセキュリティ更新が入っていない。
- 上流は `osixia/container-openldap` に改名し、v2 を作り直している。2026-11-23 時点で公開されているのは `2.6.15-rc` までの prerelease だけで、README は v2 を破壊的リリースとし、v1 からの移行を手動とする。
- イメージ名が Go の文字列定数にあるため、Renovate も `check-container-images` も追跡しない。

2026-11-23 の依存関係の一括更新で、この項目を最新版の分類から見送った。

## 対象範囲

- `infra/test/openldap/` に、`debian:trixie-slim` の digest 固定と Debian の `slapd` パッケージ（trixie では 2.6.10）から試験用のイメージを作る `Dockerfile` と起動スクリプトを置く。
- `testing_openldap` パッケージの `Start` を、このイメージを組み立てて起動する形に改める。
- 既存の OpenLDAP 試験（`testing_openldap`、`ldap_goldap` の結合試験、`connector_e2e_test.go`、`delegation_e2e_test.go` の 7 件）を、変更なしで新しいサーバーに対して通す。
- OpenLDAP 試験の前提環境を書いた文書（`docs/development/testing.md`、`docs/development/local-development.md`、`docs/development/continuous-integration.md`）を、イメージの組み立てを含む前提に合わせる。

## 対象外

- OpenLDAP 試験の実行区分（`openldap` ビルドタグ、`test-go-openldap*` のタスク、CI の Go 検証での実行）の変更。区分は [[wi-32030-separate-openldap-tests-from-default-go-verification]] が定めたとおりに保つ。
- 試験の追加と表明の変更。サーバーを替えて既存の表明が通らない場合は、製品の欠陥か試験の前提の誤りかを切り分け、製品の欠陥なら別の項目に起票する。
- AD 方言の試験。AD の属性と Kerberos は Samba AD の試験（`test-kerberos-ad`）と [[wi-19734-provision-users-from-idmagic-into-active-directory]]、[[wi-90960-write-password-changes-back-to-active-directory]] が扱う。

## 設計

### 採用する構成

Samba AD の試験用イメージ（`infra/test/kerberos/`）と同じく、Debian のパッケージから試験用のイメージを組み立てる。
OpenLDAP の更新は Debian のセキュリティ更新が担い、ベースイメージの digest は Renovate が追跡する。

- **Dockerfile**：`debian:trixie-slim@sha256:…`（`infra/test/kerberos/Dockerfile` と同じ digest）に `slapd` を入れる。パッケージの初期設定（debconf が作る `/etc/ldap/slapd.d`）は使わず消す。
- **起動スクリプト**：起動時に cn=config の LDIF を `slapadd -n0` で入れてから `slapd -d` を前面で動かす。LDIF は次を持つ。
  - `back_mdb` のモジュールと、`dc=example,dc=test` の mdb データベース。rootdn は `cn=admin,dc=example,dc=test` で、パスワードは `slappasswd` で作ったハッシュにする。
  - `core`、`cosine`、`inetorgperson` のスキーマ。
  - `olcTLSCertificateFile`、`olcTLSCertificateKeyFile`、`olcTLSCACertificateFile` を、試験が差し込む証明書のパスへ向ける。
  - ACL は `userPassword` を本人の書き込みと匿名の認証だけに許し、それ以外を読み取りに許す。委譲の E2E が使う simple bind と、管理者の書き込みが通る最小の構成にする。
  - 待ち受けは `ldap:///`（StartTLS 用）と `ldaps:///` にする。
- **`Start` の流れ**：今の `docker create`、証明書の `docker cp`、`docker start` の順序を保つ。証明書を起動前に差し込めるので、期限切れの同梱証明書を避ける回避策は不要になり、差し込む先を自前のイメージのパスへ変えるだけで済む。osixia 固有の環境変数（`LDAP_ORGANISATION` など）は消す。準備完了の判定は今と同じく、試験用 CA で検証した LDAPS の管理者 bind で行う。
- **イメージの組み立て**：`Start` が `docker build --quiet --tag idmagic-openldap-test <リポジトリの infra/test/openldap>` を実行する。ディレクトリは `runtime.Caller` で得たソースの位置から解決する。2 回目以降は Docker のビルドキャッシュが効く。
- **検査**：起動スクリプトは `lint-repo` の shellcheck（追跡中のすべての `*.sh` が対象）が検査するので、検査のタスクは足さない。

### 採らない案

- **osixia の 1.5.0 のまま**：作業はないが、2.4 系での検証、期限切れ証明書の回避、OS の更新の欠如がすべて残る。
- **osixia の v2 の安定版を待って移行する**：環境変数で設定する今の使い方に近いが、安定版の時期が決まっていない。破壊的変更により `Start` の手順と証明書の置き場所の書き直しは避けられず、作業量は自前のイメージと大差ない。第三者が再び更新を止める可能性も残る。
- **Bitnami の OpenLDAP イメージ**：Bitnami は 2025 年に無償の公開カタログの更新を止めたと理解している。この項目の起票時には確かめておらず、採るなら着手時に一次情報で確認する必要がある。上の理由で自前のイメージを選ぶので、確認はしない。
- **組み立てを mise のタスクに分け、試験はタグだけを参照する**：`go test -tags=openldap` を直接実行したときに、古いイメージを使うか見つからずに失敗する。組み立てを `Start` に置けば、どの入口から実行しても同じイメージになる。

## 計画

1. `infra/test/openldap/` に Dockerfile と起動スクリプトを書き、手元で起動して LDAPS と StartTLS の管理者 bind、`entryUUID` と `modifyTimestamp` の読み取り、simple paged results の応答を確かめる。
2. `Start` を書き換え、`mise run test-go-openldap` を通す。
3. 文書の前提環境を合わせる。

未解決の問いはない。
Debian の `slapd` の postinst が debconf の回答なしに失敗する場合は、`DEBIAN_FRONTEND=noninteractive` と、パッケージの設定を消す手順で吸収する。

## タスク

- [ ] T001 [Infra] `infra/test/openldap/` に Dockerfile と起動スクリプトを置く。
- [ ] T002 [Test] `testing_openldap.Start` を自前のイメージを組み立てて起動する形に改め、osixia 固有の環境変数と証明書の回避策を消す。
- [ ] T003 [Docs] OpenLDAP 試験の前提環境を書いた開発文書を合わせる。
- [ ] T004 [Verify] 変更を検証する。

## 検証

- `mise run test-go-openldap`：変更の中心である。既存の結合試験と E2E の 7 件と、タグ付きコードの lint が、新しいサーバーに対して Skip なしで通ることを確かめる。
- `mise run lint-repo`：起動スクリプトの shellcheck。
- `mise run check-links`：文書を変えるため。
- `mise run verify`：通常の検証が OpenLDAP のコンテナを起動しないまま通ることを確かめる。

## リスク

- Debian の slapd と osixia の構成の違い（既定の ACL、スキーマ、サイズ制限）で、既存の表明が変わる可能性がある。ページングの試験が作るエントリは 5 件で、slapd の既定の `olcSizeLimit`（500）を下回るので、サイズ制限は効かない見込みである。既定の ACL とスキーマの違いは、計画の 1 で手元の起動により確かめる。
- CI のジョブごとにイメージの組み立てが走り、Debian のミラーへの到達に依存する。Samba AD の試験と同じ前提であり、組み立ての失敗は診断付きで試験の失敗になる。
