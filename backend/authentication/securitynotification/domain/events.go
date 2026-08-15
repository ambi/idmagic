package domain

import "time"

// AccountSecurityNotificationSent はセキュリティ通知を 1 通送った (または送ろうとして
// 失敗した) ことを表す。本文も宛先アドレスも payload に含めない。何を知らせたかだけを
// 監査に残す。この種別はカタログ (triggers) に無いので、ディスパッチャーが自分の発行を
// 拾って通知を連鎖させることはない。
type AccountSecurityNotificationSent struct {
	At       time.Time `json:"-"`
	TenantID string    `json:"tenantId"`
	// UserID は通知を受け取る本人の sub。なりすましでは操作した admin ではなく対象本人。
	UserID   string   `json:"userId"`
	Category Category `json:"category"`
	// TriggerEventType は通知の契機になったイベントの種別名。
	TriggerEventType string `json:"triggerEventType"`
	// Delivered は EmailSender が受け付けたかどうか。最大限努力なので false でも
	// 元の操作は成立している。
	Delivered bool `json:"delivered"`
}

func (e *AccountSecurityNotificationSent) EventType() string {
	return "AccountSecurityNotificationSent"
}

func (e *AccountSecurityNotificationSent) OccurredAt() time.Time { return e.At }
