---
depends_on: []
status: completed
authors: [tn]
risk: medium
reversibility: irreversible
created_at: 2026-08-27
priority: p2
change_kind: docs
evidence_policy: risk-based-v2
initial_context:
  specification:
    - docs/contexts/signing-keys/scenarios.md#REQ-SIGNINGKEYS-012
    - docs/contexts/signing-keys/scenarios.md#REQ-SIGNINGKEYS-007
    - docs/contexts/seeding/scenarios.md#REQ-SEEDING-005
  typespec: [IdMagic.Contract.KeyProvider]
  source: [backend/cmd/internal/bootstrap, backend/signingkeys/db_postgres, infra/deploy, infra/k8s, infra/docker, infra/backup]
  tests: [backend/cmd/internal/bootstrap]
  stop_before_reading: [frontend, backend/datakeys]
affected_spec:
  - { path: docs/contexts/signing-keys/scenarios.md, requirement: REQ-SIGNINGKEYS-012 }
  - { path: spec/contexts/signing-keys/models.tsp, symbol: IdMagic.Contract.KeyProvider }
---

# バックアップに含まれる鍵素材の保護を、手順ではなく規範として持つ

## Motivation

`docs/runbooks/backup-restore-dr.md` は、`Local` と `Postgres` の鍵提供元では「秘密鍵は `signing_keys.private_jwk` に平文で入り、PostgreSQL のバックアップにそのまま含まれる」と述べ、保存先の暗号化を指示している。指示の内容は正しい。

問題は、それが **runbook にしかない**ことである。`docs/README.md` は「手順はこの平面には置かない。障害時の手順は runbooks にある」と述べており、runbook は事象の最中に読む手順であって、製品が従う規範ではない。したがってこの保護には、対応する規範 ID が無く、テストが無く、`mise run check-security-controls` の対象にもならない。手順が読まれなければ保護は存在しない。

対照的に、可逆なシークレットについては `docs/database.md` がエンベロープ暗号化を規範として定め、`REQ-DATAKEYS-*` が対応するシナリオを持つ。署名鍵だけが、保護の根拠を手順に置いたまま残っている。

これは [docs/threat-model.md](../../docs/threat-model.md) の THREAT-062 が指す欠落である。

着手時の調査で、規範が無いことの帰結が実物として 2 つ見つかった。`KEY_PROVIDER` は未指定を許す任意の設定であり、`PERSISTENCE=postgres` でも未指定のまま起動できる。つまり秘密鍵がバックアップに入る構成は、誰も選ばないまま既定として成立する。そして `infra/deploy/gcp/cloudrun-idmagic.yaml` は `KEY_PROVIDER: "db"` を設定しており、これは許可された値（`local` / `vault`）ではないので、この本番向けサンプルは起動時設定の検証で落ちる。

## Scope

- 平文の鍵素材が永続化される条件と、そのときに要求する保護を規範として書く。owner は `docs/contexts/signing-keys/decisions.md` とする。
- 鍵提供元ごとに、鍵素材がデータベースに平文で置かれるかどうかを明示する。TypeSpec の `KeyProvider` の doc も実態に合わせる。
- 平文で置かれる提供元を選んだ場合の扱いを決め、規範シナリオ REQ-SIGNINGKEYS-012 として書く。
- 規範を起動時設定で強制する。`PERSISTENCE=postgres` では `KEY_PROVIDER` の明示を要求する。
- 明示を要求する側に回る配備アセット（Compose、Kubernetes、Cloud Run、`dev.sh`、復元訓練）へ `KEY_PROVIDER` を書き、`cloudrun-idmagic.yaml` の不正な値を正す。
- runbook からは規範を参照する形にし、同じ内容を二重に持たない。
- THREAT-062 の `Controls` を、現に存在する制御へ更新する。

## Out of Scope

- バックアップの取得と復旧の手順そのもの。runbook が持ち続ける。
- 保存先（ボリューム、バケット）の暗号化の実装。配備側の責務である。
- `Vault` / `OpenBao` 提供元への移行の強制。
- `signing_keys.private_jwk` のエンベロープ暗号化。可逆なシークレットの規範を署名鍵へ広げるかどうかは、`EnvelopeCrypto` と `KeyProvider` の役割分担を決め直す変更であり、保護の所在を書き留める本件とは別の判断である。
- 起動時設定への配備環境（本番かどうか）の宣言の導入と、それに基づく提供元の拒否。判定の材料が今は存在せず、導入は seeding に閉じた `SEED_ENVIRONMENT` の意味を変える別の変更になる。決定と再検討の条件は `decisions.md` に残した。
- 配備アセットの設定値を起動時設定の語彙と突き合わせる検査。`KEY_PROVIDER: "db"` を通した穴はこれだが、`check-config-reference` は `CONFIGURATION.md` と loader の一致しか見ておらず、マニフェスト側を読む仕組みは無い。別の work item が扱う。

## Design

規範の owner は `docs/contexts/signing-keys/decisions.md` とする。平文で置かれるかどうかは提供元の性質であり、`SigningKeys` が持つ判断だからである。`docs/database.md` は列型と保持区分とエンベロープ暗号化という横断的な方針を持つ場所であり、提供元ごとの差はそこには収まらない。ただしエンベロープ暗号化の記述と近接するため、`database.md` からは「署名鍵の秘密鍵はこの規範の対象ではない」という 1 文で相互に参照する。

本番で平文の提供元を選んだ場合の扱いは、拒否と警告の両案を検討して**どちらも採らない**。前提としていた本番判定が存在しないからである。`REQ-SEEDING-005` が拒否できるのは `SEED_ENVIRONMENT` が seed 要求の引数として与えられるからであって、API と worker のプロセスには配備環境を表す起動時設定が無い。無いものを新設して拒否を組み立てれば、判定の正しさを誰も検証できないまま拒否が働くことになる。

代わりに採るのは、**平文で永続化する構成を既定にしない**という一段弱く、しかし判定を必要としない規範である。鍵素材が永続化されるのは `PERSISTENCE=postgres` のときだけなので、その配備では `KEY_PROVIDER` の明示を要求し、未指定なら起動を拒否する。運用者は `local` と `vault` のどちらかを書く必要が生じ、書いた時点で鍵素材の所在を選んだことになる。`PERSISTENCE=memory` では鍵素材がプロセスの外へ出ないため、何も要求しない。これは `DATABASE_URL` が `PERSISTENCE=postgres` でだけ必須になるのと同じ形であり、既存の `ConfigLoader.RequiredWhen` と `Require` でそのまま表現できる。

効果は起動時設定という入力の境界にとどまり、`SigningKeys` の domain と usecases には及ばない。変更する計算は `LoadSharedConfig` の検証だけであり、時刻、乱数、識別子生成には触れない。`selectKeyStore(cfg SharedConfig, fallback signingports.KeyStore) signingports.KeyStore` は変更しないが、この規範が守るのはまさにその関数の出力であり、`cfg.KeyProvider == ""` で平文の KeyStore が返る経路へ到達しないことが、拒否が防いだ効果になる。

Acceptance と Unit の境界は次のように分ける。Unit は変わった計算そのもの、すなわち `LoadSharedConfig` が `KEY_PROVIDER` を名指しする設定エラーを記録することを見る。Acceptance は `Run()` と同じ順序（読み込み → `Err()` → アダプター選択）を辿り、拒否が何を残さなかったかを見る。すなわち `KEY_PROVIDER` を書かない限り、平文で永続化する KeyStore はどの環境変数の組み合わせからも組み立てられないことを見る。後者が [Testing a refusal](../../docs/development/specification-first-workflow.md#testing-a-refusal) の言う「拒否が触れずに残したもの」にあたる。

## Plan

1. 鍵提供元ごとに、鍵素材がどこにどの形で置かれるかを確認する。→ 完了。`Local`（`PERSISTENCE=memory`）はプロセス内のみ、`Database`（`PERSISTENCE=postgres` かつ `KEY_PROVIDER` が `vault` 以外）は `signing_keys.private_jwk` に平文、`VaultTransit` は Vault 内に閉じる。
2. 本番の判定が起動時設定から一意に決まるかを確認する。→ 完了。決まらない。配備環境を表す起動時設定は存在しない。
3. 規範を `decisions.md` へ書き、REQ-SIGNINGKEYS-012 を足し、TypeSpec の `KeyProvider` の doc を実態に合わせる。
4. Acceptance と Unit の失敗を観測してから、`LoadSharedConfig` に要求を入れる。
5. 配備アセットへ `KEY_PROVIDER` を明示し、`cloudrun-idmagic.yaml` の `db` を正す。設定リファレンスを再生成する。
6. runbook を、規範を参照する形へ書き換える。THREAT-062 の `Controls` を更新する。

## Tasks

- [x] T001 [Baseline] 提供元ごとの鍵素材の置かれ方と、本番判定の可否を確認する。
- [x] T002 [Spec] 規範の owner を決め、判断と REQ-SIGNINGKEYS-012 を書き、TypeSpec の `KeyProvider` の doc を実態に合わせる。
- [x] T003 [Acceptance] `KEY_PROVIDER` を書かない `PERSISTENCE=postgres` の環境から、平文で永続化する KeyStore が現に組み立てられることを観測する（`TestPlaintextKeyCustodyIsNeverImplicit`、REQ-SIGNINGKEYS-012）。
- [x] T004 [App] `LoadSharedConfig` が `KEY_PROVIDER` を要求しないことを観測してから要求を入れる（`TestLoadSharedConfigPostgresRequiresExplicitKeyProvider`、REQ-SIGNINGKEYS-012）。
- [x] T005 [Ops] 配備アセットへ `KEY_PROVIDER` を明示し、`cloudrun-idmagic.yaml` の不正な値を正し、設定リファレンスを再生成する。
- [x] T006 [Spec] runbook を規範の参照へ書き換え、内容の重複を除く。THREAT-062 の `Controls` を更新する。
- [x] T007 [Verify] 変更耐性を確認し、`mise run verify` を通す。

## Verification

- 鍵素材が平文で永続化される条件が、正本文書に書かれている。
- `PERSISTENCE=postgres` で `KEY_PROVIDER` を書かないと起動時設定が拒否し、平文の KeyStore は組み立てられない。
- runbook が同じ内容を二重に持たない。
- `mise run check-spec`、`mise run check-security-controls`、`mise run verify`

## Risk Notes

本番の判定を厳しくしすぎると、開発と評価の構成を壊す。今回は判定そのものを持たず、`PERSISTENCE=postgres` という既に存在する条件だけを使うため、`PERSISTENCE=memory` の経路は何も変わらない。壊れうるのは PostgreSQL で永続化する既存の配備であり、`KEY_PROVIDER` を 1 行足すまで起動しなくなる。これは意図した結果だが、後方互換のない起動時設定の変更なので、リポジトリ内の配備アセットをすべて追随させる。

規範を書いても、保存先の暗号化そのものは配備側の責務であり、製品が強制できない。規範が保証するのは「どの構成で鍵素材が平文になるかが分かること」までであり、それを超える保証を主張しない。THREAT-062 が `planned` のままなのはこのためである。

`REQ-SIGNINGKEYS-012` は永続的な ID であり、`KEY_PROVIDER` を必須にする条件は運用者の既存環境に対する要求でもあるため、この判断は取り消せない前提で書く。

## Completion

- **Completed At**: 2026-08-29
- **Summary**:
  `mise run spec-diff` が示す規範上の差分は `REQ-SIGNINGKEYS-012` の追加 1 件である。秘密鍵素材の保管先が提供元で決まること、平文で永続化する構成を既定にしないこと、保存先の暗号化は配備側の責務であること、本番判定に基づく提供元の拒否は判定材料が無いため採らないことを `docs/contexts/signing-keys/decisions.md` に規範として置いた。runbook は保護を要求する側から手段を示す側へ退き、同じ内容を二重に持たなくなった。起動時設定は `PERSISTENCE=postgres` で `KEY_PROVIDER` の明示を要求するようになり、未指定のまま平文の KeyStore が組み立てられる経路が閉じた。TypeSpec の `KeyProvider` は `Database` が鍵素材をプロセス内に持つと述べていたが、実際にはデータベースへ平文で置くので doc を実態に合わせた。副産物として `infra/deploy/gcp/cloudrun-idmagic.yaml` の `KEY_PROVIDER: "db"` が許可されない値であること、すなわちこの本番向けサンプルがそもそも起動しないことが分かり、`local` に正した。
- **Acceptance RED Evidence**:
  - **Test**: `TestPlaintextKeyCustodyIsNeverImplicit`（`backend/cmd/internal/bootstrap/keystore_test.go`）
  - **Requirement**: REQ-SIGNINGKEYS-012
  - **Observed Failure**: `keystore_test.go:49: PERSISTENCE=postgres without KEY_PROVIDER must be refused before any key store is assembled`。実装前は `PERSISTENCE=postgres` と `DATABASE_URL` だけで起動時設定が通り、`selectKeyStore` が永続層の KeyStore をそのまま返していた。
  - **Detection Reason**: 起動時設定の拒否だけでなく、`Run()` と同じ順序でアダプターの選択まで辿り、返った KeyStore が永続層のもの（`PERSISTENCE=postgres` では秘密 JWK を平文で持つ）と同一かどうかを見る。エラーだけを見るテストは、エラーを記録しながら KeyStore を組み立てる実装を通してしまう。`KEY_PROVIDER=local` と `KEY_PROVIDER=vault` を並べているので、拒否を一律に強めた実装（明示しても通らない、あるいは vault でも平文へ落ちる）も落ちる。
- **Unit RED Evidence**:
  - **Test**: `TestLoadSharedConfigPostgresRequiresExplicitKeyProvider`（`backend/cmd/internal/bootstrap/sharedconfig_test.go`）
  - **Requirement**: REQ-SIGNINGKEYS-012
  - **Observed Failure**: `sharedconfig_test.go:30: err=<nil>, want a KEY_PROVIDER required error`
  - **Detection Reason**: 変わった計算は `LoadSharedConfig` の検証だけであり、`KEY_PROVIDER` を名指しする設定エラーが記録されることを直接見る。名前を確かめているので、別のキーの不足で偶然エラーになった状態と区別できる。`TestMemoryPersistenceNeedsNoKeyCustodyChoice` が条件の反対側を押さえ、すべての配備で一律に必須化する実装を落とす。
- **Change-Resistance Results**:
  2 つの変異を入れて確認した。(1) 述語を恒真化する（`l.Require("KEY_PROVIDER", true, ...)`）→ `TestPlaintextKeyCustodyIsNeverImplicit` と `TestLoadSharedConfigPostgresRequiresExplicitKeyProvider` の 2 件が落ちた。(2) 条件を反転する（`cfg.Persistence == "memory"`）→ 新規 2 件を含む 13 件が落ち、うち `TestMemoryPersistenceNeedsNoKeyCustodyChoice` が反転そのものを名指しで捉えた。`l.RequiredWhen("KEY_PROVIDER", "PERSISTENCE=postgres")` の行だけを消す変異はテストでは検出されないが、これは設定リファレンスの記載であって拒否の実体ではなく、`mise run check-config-reference` が `CONFIGURATION.md` との不一致として落とす。
- **Verification Results**:
  - `mise run verify` - passed
  - `mise run check-compose` / `mise run check-k8s` / `mise run check-config-reference` - passed（配備アセットへ `KEY_PROVIDER` を足したため個別に確認した）
