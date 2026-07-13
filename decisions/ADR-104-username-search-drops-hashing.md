---
status: accepted
authors: [tn]
created_at: 2026-07-14
supersedes: ADR-046 (username 条項のみ)
---

# ADR-104: 監査検索の username は平文で扱う (ADR-046 の username 条項を撤回)

## コンテキスト

[[ADR-046]] は認証イベントの username を「tenant salt 付き SHA-256 hash が first-class、平文は
失敗イベント限定で 7 日だけ」という方針にしていた。この方針は、監査検索を使う運用（wi-147 での
`ConsentGranted` 等 OAuth2 フロー系イベントへの相関検索拡張）を通じて見直したところ、次の問題が
判明した。

- 相関先の事故確率に対して、tenant salt 用の新しい依存 (`TenantSaltProvider`) を各 usecase の
  Deps に配線し直す実装コストが不釣り合いに大きい。
- 実アカウントが常に確定しているイベント (`UserAuthenticated`、OAuth2 フロー系イベント全般) は
  そもそも `user_id` で一意に相関できるため、username の hash を持たせる必要自体がなかった。
  hash はむしろ「username 変更後に過去イベントと不整合が起きる」新しい問題を生んでいた。
- 7 日後に平文を null 化して hash だけ残す sweep は、hash 化のための複雑さを増やす一方で、
  実際の脅威モデル (この監査ログの想定読者は tenant 内の admin に限定される) に対して
  過剰な防御だった。

## 決定

1. **username の hash 化・tenant salt 化をやめる。** `AuditSearchRegistry` の `actor.username`
   は `raw_storable: true` / `transform: none` の平文属性として扱う。
2. **実アカウントが確定しているイベント (`UserAuthenticated` および OAuth2 フロー系イベント:
   `ConsentGranted` / `AuthorizationCodeIssued` / `AuthorizationCodeRedeemed` /
   `AccessTokenIssued` / `RefreshTokenIssued`) は username 自体を payload に持たない。**
   管理 UI が username で検索したい場合は、検索時に `UserRepo.FindByUsername` で `user_id` に
   解決してから、既存の `user_id` / `actor.id` 経路でフィルタする (`AuditEventQuery.username`)。
   該当ユーザーが存在しない場合は 0 件を返す。
3. **実アカウントが確定しない可能性があるイベント (`AuthenticationFailed`) は、引き続き
   平文 username を payload に持つ。** ただし hash 化・7 日後の redaction sweep は廃止し、
   他の failure イベントと同じ保持期間 (`FailDays`, 既定 30 日) でそのまま保持する。
4. **IP / 位置情報 / device fingerprint の取り扱い ([[ADR-046]] の該当項目) は変更しない。**
   本 ADR は username の取り扱いのみを対象とする。

## 影響

- `backend/authentication/usecases/retention.go` の `FailureUsernamePlaintextDays` /
  `AuthenticationFailureUsernameRedactor` は削除する。
- `UserAuthenticated` / `ConsentGranted` / `AuthorizationCodeIssued` /
  `AuthorizationCodeRedeemed` / `AccessTokenIssued` / `RefreshTokenIssued` の
  `usernameHash` payload フィールドは削除する。
- `AuthenticationFailed` の `usernameHash` payload フィールドは削除する (`username` 平文のみ残す)。
- `LoginThrottled` / `AuthenticationEventAggregated` の `keyHash` (throttle bucket key、
  監査検索の registry には出ていない別用途) は本 ADR の対象外で変更しない。

## 却下した代替案

- **ADR-046 を全面破棄し IP / 位置情報 / device fingerprint の hash 化もやめる**: username 以外の
  取り扱いについて具体的な問題提起がなく、スコープが不必要に広がるため見送った。
