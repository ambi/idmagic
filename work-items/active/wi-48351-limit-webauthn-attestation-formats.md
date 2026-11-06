---
status: pending
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-11-07
priority: p2
depends_on: [wi-47102-contain-go-webauthn-types-in-authentication]
change_kind: feature
affected_spec:
  - { path: docs/modules/authentication/features/webauthn.md, requirement: REQ-AUTHENTICATION-038 }
---

# WebAuthn で受け付ける attestation の形式を決め、評価の範囲を限る

## 動機

[WebAuthn の仕様](../../docs/modules/authentication/features/webauthn.md)は、端末の機種の強制よりプライバシーを優先し、アテステーション `none` を要求する。
機種を強制しないため、TPM、SafetyNet などの attestation を検証しても、結果をどこにも使わない。

それでも、クライアントが tpm などの形式を返せば、go-webauthn は TPM の構造体、X.509 証明書の鎖、SafetyNet の JWS、metadata を解析して検証する。
利用者から届く入力に対する複雑な処理であり、証明書の有効期限の誤りのように、製品として不要な理由で登録が失敗する原因にもなる。
[ライブラリ選定の見直し](../done/wi-90339-review-library-adoption.md)で、この処理に到達させないと決めた。

## 対象範囲

- 受け付ける attestation の形式と、それ以外の形式を受けたときの結果を仕様に書く。
- TPM、SafetyNet、metadata の処理に、登録の入力が到達しないようにする。

## 対象外

- go-webauthn の置き換え。
- 第二要素とステップアップ認証以外の用途（パスワードレス、Discoverable Credential）。

## 設計

W3C の[登録手順](https://www.w3.org/TR/webauthn-3/#sctn-registering-a-new-credential)では、RP は対応する attestation の形式を決めてよく、`none` だけに対応しても手順に沿う。
ただし `none` を要求したときに attestation を `none` に置き換える処理は、クライアントにとって MAY である。
主要なブラウザーは置き換えるが、すべての環境での保証はない。

着手時に次を比較して決める。

| 案 | 結果 |
| --- | --- |
| `none` 以外を拒否する | 最も狭い。置き換えないクライアントで登録できなくなる |
| `none` と `packed` を受け付ける | 自己署名の `packed` を返す環境を救える。`packed` の x5c の検証が残る |
| 形式を問わず attStmt を評価せず `none` として扱う | 互換性が最も高い。attestationObject を CBOR で読み替える処理を所有する |

実現の手段は、go-webauthn の設定で形式を制限できるかを先に確かめ、できなければライブラリへ渡す前に入力を判定する。
`protocol` パッケージは TPM などの処理を登録するため、依存のコードはビルドから消えない。
目的は入力の到達範囲を限ることであり、依存の削減ではないと記録する。

## 計画

1. 主要なブラウザーと認証器が `none` を要求したときに返す形式を確かめる。
2. 上の案を選び、仕様の差分を書いてから `spec-change` で更新する。
3. 主要ユースケースの証拠を計画し、RED から実装する。

## タスク

- [ ] T001 [Research] 実際に返る attestation の形式を確かめる。
- [ ] T002 [Spec] 受け付ける形式を仕様に書く。
- [ ] T003 [Acceptance] 受け付けない形式の登録を観測する境界で RED を確かめる。
- [ ] T004 [App] 形式の判定を実装する。
- [ ] T005 [Verify] 変更を検証する。

## 検証

- `mise run check-spec` で仕様を確かめる。
- `mise run test-go-package -- ./backend/authentication/...` で登録と認証を確かめる。

## リスク

形式を狭めすぎると、一部の環境で認証器を登録できなくなる。
実際の形式を確かめてから案を選ぶ。
