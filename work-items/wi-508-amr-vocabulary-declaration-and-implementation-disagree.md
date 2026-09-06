---
depends_on: []
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-06
priority: p2
change_kind: bugfix
affected_spec:
  - { path: docs/contexts/authentication/standards.md, requirement: RFC8176-AMR-VOCABULARY }
  - { path: docs/contexts/authentication/scenarios.feature.md, requirement: REQ-AUTHENTICATION-001 }
  - { path: spec/contexts/authentication/models.tsp, symbol: IdMagic.Contract.LoginSession }
---

# `amr` の語彙について、standards.md と scenarios.feature.md が食い違っている

## Motivation

[[wi-501-back-authentication-standards-rows-with-tests]] が `RFC8176-AMR-VOCABULARY` にテストを対応付けようとして見つけた。**この行はいま製品と一致していないので、行を満たすテストが書けない。**

`docs/contexts/authentication/standards.md` の `RFC8176-AMR-VOCABULARY` は、`LoginSession.amr` に許される語彙を `pwd` / `otp` / `webauthn` / `hwk` / `swk` / `rc` / `tdev` の 7 語と宣言している。一方 `docs/contexts/authentication/scenarios.feature.md:17` は「AMR に `federated` を持つ LoginSession を発行する」を規範として書いており、`docs/contexts/authentication/internals.md:13` も同じことを書いている。実装 (`backend/authentication/federation/usecases/broker.go:106`) は後者に従っている。`federated` は RFC 8176 の登録値でもなければ、標準の行が挙げる非 IANA 拡張値でもない。

同じ context の 2 つの正典文書が、同じフィールドについて両立しないことを言っている。どちらが正しいかを決めるのは規範の変更であり、テストの追加ではない。

第 2 の食い違いが同じ場所にある。`spec/contexts/authentication/models.tsp` の `LoginSession.acr` の doc と `internals.md:77` は、復旧コード (`rc`) による第二要素の成立で `acr` が `urn:idmagic:acr:mfa` へ上がると書いている。実装の `mfaAMRValues` (`backend/authentication/usecases/acr_vocabulary.go:17`) は `otp` / `webauthn` / `hwk` / `swk` / `tdev` の 5 語で、`rc` を含まない。復旧コードで第二要素を通したセッションの `acr` は `urn:idmagic:acr:pwd` のままになる。

第 3 に、語彙そのものを強制する場所がどこにも無い。`LoginSession` の zog スキーマは `AMR` について `Min(1)` しか課しておらず、語彙の外の値を書き込む呼び出し元があれば通る。行が「語彙のみを許可する」と書いている以上、許可されない値が拒否されることの観測が要る。

## Scope

- `federated` について、標準の行と規範シナリオのどちらへ寄せるかを決め、寄せた側へ両方を合わせる。
- `rc` が `acr` を `urn:idmagic:acr:mfa` へ上げるのかどうかを決め、TypeSpec の doc、`internals.md`、`mfaAMRValues` を一致させる。
- 語彙の強制を `LoginSession` の境界へ置き、語彙外の値が拒否されることを観測するテストを書く。
- `RFC8176-AMR-VOCABULARY` を `tools/check/standards-coverage-debt.json` から外す。

## Out of Scope

- `docs/contexts/authentication/standards.md` の他の行。[[wi-501-back-authentication-standards-rows-with-tests]] が済ませた。
- 認証要素の追加や削除。
- `acr` の URN そのものの変更。

## Verification

- `RFC8176-AMR-VOCABULARY` が `tools/check/standards-coverage-debt.json` から消えている。
- 語彙の外の値を持つ `LoginSession` が拒否されることを観測するテストがある。
- `federated` と `rc` について、`standards.md`、`scenarios.feature.md`、`internals.md`、TypeSpec の doc、実装が同じことを言っている。
- `mise run check-spec`
- `mise run verify`

## Risk Notes

- **語彙の強制を入れた結果、既存のセッションが解決できなくなる。** 語彙外の値を持つ行が保存済みなら、読み出しの検証は既存セッションを一斉に失効させる。書き込みの側だけに課すか、寄せ先を `federated` を含む語彙にするかで挙動が変わる。
- **`rc` を MFA 充足に含める判断は、サインインポリシーの強度を変える。** 復旧コードだけで「毎回 MFA」を満たせるようになるので、`internals.md:77` が `mfa_enrolled` に復旧コードを数えない理由と併せて読む必要がある。規範をどちらへ寄せるかは、テストではなく判断である。
