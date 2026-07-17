---
status: accepted
authors: [tn]
created_at: 2026-07-14
supersedes: [ADR-046]  # username / IP / User-Agent / device fingerprint 条項のみ
---

# ADR-104: 監査イベントの username と接続元属性は平文で扱う

## コンテキスト

[[ADR-046]] は認証イベントの username、IP、User-Agent、device fingerprint を hash 化または
truncate し、username の平文は失敗イベント限定で 7 日だけ保持する方針にしていた。この方針は、
監査検索を使う運用（wi-147 での `ConsentGranted` 等 OAuth2 フロー系イベントへの相関検索拡張）を
通じて見直したところ、次の問題が判明した。

- 相関先の事故確率に対して、tenant salt 用の新しい依存 (`TenantSaltProvider`) を各 usecase の
  Deps に配線し直す実装コストが不釣り合いに大きい。
- 実アカウントが常に確定しているイベント (`UserAuthenticated`、OAuth2 フロー系イベント全般) は
  そもそも `user_id` で一意に相関できるため、username の hash を持たせる必要自体がなかった。
  hash はむしろ「username 変更後に過去イベントと不整合が起きる」新しい問題を生んでいた。
- 7 日後に平文を null 化して hash だけ残す sweep は、hash 化のための複雑さを増やす一方で、
  実際の脅威モデル (この監査ログの想定読者は tenant 内の admin に限定される) に対して
  過剰な防御だった。
- IP、User-Agent、device fingerprint も同じ監査イベントの保持期間とアクセス制御下にあり、
  hash / truncate した値ではインシデント調査時の照合・説明が難しくなる一方、変換処理と salt 管理を
  増やしていた。

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
4. **IP / User-Agent / device fingerprint の hash 化・truncate もやめる。** 監査イベントの保持期間中は
   平文の `ip` / `userAgent` / `deviceFingerprint` として保存し、監査検索属性も平文一致で扱う。
   位置情報は引き続き country code のみを保存し、詳細な位置情報は保持しない。

## 影響

- `backend/authentication/usecases/retention.go` の `FailureUsernamePlaintextDays` /
  `AuthenticationFailureUsernameRedactor` は削除する。
- `UserAuthenticated` / `ConsentGranted` / `AuthorizationCodeIssued` /
  `AuthorizationCodeRedeemed` / `AccessTokenIssued` / `RefreshTokenIssued` の
  `usernameHash` payload フィールドは削除する。
- `AuthenticationFailed` の `usernameHash` payload フィールドは削除する (`username` 平文のみ残す)。
- `UserAuthenticated` / `AuthenticationFailed` / `SessionStarted` の `ipHash` / `ipTruncated` /
  `uaHash` / `deviceFingerprintHash` payload フィールドは削除し、平文の `ip` / `userAgent` /
  `deviceFingerprint` に置き換える。
- `LoginThrottled` / `AuthenticationEventAggregated` の `keyHash` (throttle bucket key、
  監査検索の registry には出ていない別用途) は本 ADR の対象外で変更しない。

## 却下した代替案

- **username だけを平文化し、IP / User-Agent / device fingerprint の hash / truncate を残す**:
  同じ監査アクセス境界の接続元属性に別々の保存規則と変換依存を残し、調査時の照合性も損なうため却下した。
- **位置情報の詳細化**: 国コードを超える位置情報は本決定の検索要件に不要で、プライバシー影響が増えるため
  引き続き採用しない。
