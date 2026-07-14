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

- MFA 強制開始前は password session を成立させ、通常の step-up 付き account security 画面で
  事前登録を促す。
- 強制開始後の未登録ユーザーは、ポリシーが管理者承認オンボーディングを許可し、猶予期限内で、
  対象ユーザーに未消費・未取消・期限内の `MfaEnrollmentBypass` がある場合だけ登録専用 flow へ
  進める。
- bypass は管理者が対象 user に発行する短期・単発のサーバー側承認とする。password 成功時に
  原子的に消費し、同じ `LoginSession` を `pending_purpose=Enrollment` に遷移させる。平文コードを
  配布せず、MFA 必須ログインそのものを免除しない。
- enrollment pending session は通常の account、admin、Application resource には未認証として扱い、
  登録専用 API と元の authorization transaction だけに利用できる。
- factor の所持証明が成功した後だけ同じ session に第二要素 AMR を追加し、MFA 済みとして元の
  transaction を継続する。期限切れ、登録不能、取消、重複消費は fail closed とする。
- 初期 enrollment factor は TOTP とする。WebAuthn は同じ pending/bypass 契約に従う後続 adapter とし、
  ポリシーや session 状態を増やさず追加できるようにする。

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
