---
status: accepted
authors: [tn]
created_at: 2026-07-15
---

# ADR-110: MFA 強制開始後の未登録オンボーディングは管理者承認の単発 bypass に限定する

## コンテキスト

MFA 必須ポリシーを有効にした時点で factor を持たないユーザーを常に拒否すると、新規追加、
段階導入、factor 喪失からの復旧ができない。一方、password 認証に成功しただけで factor 登録を
許すと、password を知る攻撃者が自分の factor を登録でき、MFA 必須化が実質 1 要素認証になる。
強制開始日時や猶予期間は運用時刻を表すだけで、登録主体を信頼する根拠にはならない。

## 決定

MFA 強制開始前は password session を成立させ通常の step-up 付き account security 画面で事前登録を
促す。強制開始後の未登録ユーザーは、管理者が対象 user に発行した短期・単発のサーバー側承認
`MfaEnrollmentBypass`（平文コードは配布しない）が未消費・未取消・期限内である場合だけ登録専用
flow へ進める — 時刻ではなく、bypass の発行という管理者の能動的判断を trust anchor にする。
bypass は password 成功時に原子的に消費し、同じ `LoginSession` を `pending_purpose=Enrollment` へ
遷移させる。enrollment
pending session は登録専用 API と元の authorization transaction だけに使え、account・admin・
Application 等の他リソースには未認証として扱う。期限切れ・登録不能・取消・重複消費は fail closed
とする。初期 enrollment factor は TOTP とし、WebAuthn は同じ pending/bypass 契約に従う後続 adapter
として追加できるようにする。

現在の設計は [`backend/authentication/ARCHITECTURE.md`](../backend/authentication/ARCHITECTURE.md)
の MFA enrollment bypass セクションにある。

## 却下した代替案

- 未登録なら password 成功後に常に自由登録: password 窃取者が factor を乗っ取れる。
- 強制開始後の猶予期間だけ自由登録: 時刻は trust anchor ではなく、同じ乗っ取りを期間限定にするだけである。
- 未登録なら常に拒否: 新規ユーザーと復旧対象を管理者が安全に救済できない。
- bypass で password-only login を完了: MFA 必須ポリシーを例外的に破り、発行 token の認証強度も曖昧になる。
- 利用者へ bypass secret を配布: secret 配布・本人確認・漏えい対策が別の認証方式となり、初期実装の運用負荷が高い。

## 影響

- `spec/contexts/authentication.yaml` の `models.LoginPendingPurpose`、
  `models.MfaEnrollmentBypass`、`interfaces.StartBrowserMfaEnrollment`、
  `interfaces.ConfirmBrowserMfaEnrollment`、`interfaces.IssueMfaEnrollmentBypass`、
  `interfaces.RevokeMfaEnrollmentBypass` と関連イベント・シナリオに反映する。
- `spec/contexts/application.yaml` の `models.MfaEnrollmentPolicy` と tenant default sign-in policy の
  管理契約に反映する。
- bypass の永続化は tenant/user 境界、単発消費、取消、期限を原子的に保証し、各遷移を監査する。
- WebAuthn enrollment UI/API は後続で同じ contract に追加できるが、初期実装の受け入れ factor は TOTP とする。
