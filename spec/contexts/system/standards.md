# System Standards

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
| GDPR-CONSENT-WITHDRAWAL | required | MUST | ResourceOwner が同意を撤回でき、撤回後の新規発行には利用しない。`Consent` と `ConsentLifecycle` は OAuth2 Context が所有する。 |
| GDPR-ERASURE | required | MUST | 削除要求後は法的保存義務を除く PII を定義済み期間内に消去する。消去は IdManagement の UserLifecycle Purge 遷移と Authentication の資格情報破棄が個別に担う。 |
| GDPR-PROCESSING-RECORDS | required | MUST | セキュリティおよび認可イベントの監査記録を定義済みの期間保持する。保持期間は Audit Context が所有する。 |
