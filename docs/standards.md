# Standards

製品全体が従う外部規範を宣言する。ここに置くのは、二つ以上の Context が同じ従い方をしなければならず、Context ごとに違う従い方をすることが選択ではなく欠陥であるものだけである。1 つの Context が単独で満たす規範 — OAuth 2.0 と OIDC、SAML 2.0、SCIM 2.0、WS-Federation、SSF — は、その Context の `standards.md` が持つ。

`Statement` は製品が何をするかを書き、標準の側の義務を要約しない。各行は、規範 ID をテスト名に含めた対応するテストを持つ。

## Web Content Accessibility Guidelines 2.2

W3C Recommendation — https://www.w3.org/TR/WCAG22/

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| WCAG22-KEYBOARD | required | MUST | すべての認証操作をキーボードだけで完了可能にする。 |
| WCAG22-FOCUS | required | MUST | フォーカスを視認可能にし重要な要素が完全に隠れないようにする。 |
| WCAG22-LABELS-ERRORS | required | MUST | 入力にラベルを付け、エラーをテキストで識別して修正方法を示す。 |
| WCAG22-STATUS | required | MUST | 認証結果や送信エラーをフォーカス移動なしに支援技術へ通知する。 |

## General Data Protection Regulation

Regulation (EU) 2016/679 — https://eur-lex.europa.eu/eli/reg/2016/679/oj

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| GDPR-CONSENT-WITHDRAWAL | required | MUST | ResourceOwner が同意を撤回でき、撤回後の新規発行には利用しない。`Consent` と `ConsentLifecycle` は OAuth2 Context が担う。 |
| GDPR-ERASURE | required | MUST | 削除要求後は法的保存義務を除く PII を定義済み期間内に消去する。消去は IdManagement の UserLifecycle Purge 遷移と Authentication の資格情報破棄が個別に担う。 |
| GDPR-PROCESSING-RECORDS | required | MUST | セキュリティおよび認可イベントの監査記録を定義済みの期間保持する。保持期間は Audit Context が定める。 |
