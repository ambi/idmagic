---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-04
priority: p2
depends_on: [wi-35451-rewrite-remaining-context-specifications-in-the-requirement-format]
change_kind: bugfix
affected_spec:
  - { path: docs/domain/oauth2/consent/README.md, requirement: REQ-OAUTH2-008 }
  - { path: docs/domain/oauth2/authorization/README.md, requirement: REQ-OAUTH2-005 }
---

# 認可リクエストを、同意の要否に応じて ConsentPending と Consented の段に進める

## 動機

AuthorizationCodeFlow の状態遷移表は、同意が要る要求を `Authenticated` から `ConsentPending`、`Consented` へ進めると定め、EX-OAUTH2-008 もその状態を観測する。
実装は、同意の判定と保存を `Received` のまま行い、ログインの完了で `Received` から `CodeIssued` までを一度に進めるので、同意の段の状態が記録に現れない。
同意の画面で拒否した要求も `Received` から `Rejected` へ進み、表の `ConsentPending` からの拒否を通らない。

## 対象範囲

- 認可の処理を、認証の後に同意の要否を判定し、要る場合は `ConsentPending`、同意の後に `Consented` へ進める形にする。
- 同意の画面での拒否を `ConsentPending` からの拒否にする。

## 対象外

- 同意の画面の表示と文言。

## 設計

状態の更新はすでに `spec.TransitionAuthorizationCodeFlow` を通るので、不正な遷移は保存で拒否される。
認可のトランザクションの Cookie が `Received` の要求だけを受け付ける判定を、同意の段の状態も受け付けるよう改める。

## タスク

- [ ] T001 [Acceptance] 同意の段の状態が記録に現れないことを RED で確認する。
- [ ] T002 [App] GREEN にする。
- [ ] T003 [Verify] 変更を検証する。

## 検証

- `mise run verify`

## リスク

- 認可のトランザクションの状態を前提にする画面の継続処理が多い。既存の認可フローのテストを先に通しておき、段ごとに確かめる。
