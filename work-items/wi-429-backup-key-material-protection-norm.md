---
depends_on: []
status: pending
authors: [tn]
risk: medium
created_at: 2026-08-27
priority: p2
change_kind: docs
affected_spec:
  - { path: docs/contexts/signing-keys/scenarios.md, requirement: REQ-SIGNINGKEYS-007 }
---

# バックアップに含まれる鍵素材の保護を、手順ではなく規範として持つ

## Motivation

`docs/runbooks/backup-restore-dr.md` は、`Local` と `Postgres` の鍵提供元では「秘密鍵は `signing_keys.private_jwk` に平文で入り、PostgreSQL のバックアップにそのまま含まれる」と述べ、保存先の暗号化を指示している。指示の内容は正しい。

問題は、それが **runbook にしかない**ことである。`docs/README.md` は「手順はこの平面には置かない。障害時の手順は runbooks にある」と述べており、runbook は事象の最中に読む手順であって、製品が従う規範ではない。したがってこの保護には、対応する規範 ID が無く、テストが無く、`mise run check-security-controls` の対象にもならない。手順が読まれなければ保護は存在しない。

対照的に、可逆なシークレットについては `docs/database.md` がエンベロープ暗号化を規範として定め、`REQ-DATAKEYS-*` が対応するシナリオを持つ。署名鍵だけが、保護の根拠を手順に置いたまま残っている。

これは [docs/threat-model.md](../docs/threat-model.md) の THREAT-062 が指す欠落である。

## Scope

- 平文の鍵素材が永続化される条件と、そのときに要求する保護を規範として書く。owner は `docs/database.md` か `docs/contexts/signing-keys/decisions.md` のいずれかであり、どちらかを決める。
- 鍵提供元ごとに、鍵素材がデータベースに平文で置かれるかどうかを明示する。
- 平文で置かれる提供元を本番で選んだ場合の扱い（拒否、警告、許可）を決める。`REQ-SEEDING-005` が本番で環境変数由来のシークレット提供元を拒否する前例になる。
- 対応する規範シナリオを足す。
- runbook からは規範を参照する形にし、同じ内容を二重に持たない。

## Out of Scope

- バックアップの取得と復旧の手順そのもの。runbook が持ち続ける。
- 保存先（ボリューム、バケット）の暗号化の実装。配備側の責務である。
- `Vault` / `OpenBao` 提供元への移行の強制。

## Design

規範の owner は `docs/contexts/signing-keys/decisions.md` を採る方向である。平文で置かれるかどうかは提供元の性質であり、`SigningKeys` が持つ判断だからである。`docs/database.md` は列型と保持区分とエンベロープ暗号化という横断的な方針を持つ場所であり、提供元ごとの差はそこには収まらない。ただしエンベロープ暗号化の記述と近接するため、相互に参照する。

本番で平文の提供元を選んだ場合の扱いには、拒否と警告の両案がある。`REQ-SEEDING-005` は本番で `env` シークレット提供元を拒否しており、一貫性からは拒否が自然である。一方、`Local` 提供元は開発と単一レプリカの構成で正当に使われており、本番の定義が構成から一意に決まるかを確認する必要がある。実装着手前に確定させる。

## Plan

1. 鍵提供元ごとに、鍵素材がどこにどの形で置かれるかを確認する。
2. 本番の判定が起動時設定から一意に決まるかを確認する。
3. 規範を書き、規範シナリオを足す。
4. runbook を、規範を参照する形へ書き換える。

## Tasks

- [ ] T001 [Baseline] 提供元ごとの鍵素材の置かれ方と、本番判定の可否を確認する。
- [ ] T002 [Spec] 規範の owner を決め、判断とシナリオを書く。
- [ ] T003 [Spec] runbook を規範の参照へ書き換え、内容の重複を除く。
- [ ] T004 [Verify] 平文の提供元を本番構成で選んだときに、決めた扱いが働くことを確認する。

## Verification

- 鍵素材が平文で永続化される条件が、正本文書に書かれている。
- runbook が同じ内容を二重に持たない。
- `mise run check-spec` と `mise run verify`

## Risk Notes

本番の判定を厳しくしすぎると、開発と評価の構成を壊す。`REQ-SEEDING-007` が既に本番でのプロファイル判定を持っているので、判定の基準をそちらに揃える。

規範を書いても、保存先の暗号化そのものは配備側の責務であり、製品が強制できない。規範が保証するのは「どの構成で鍵素材が平文になるかが分かること」までであり、それを超える保証を主張しない。
