package ports

import (
	"context"

	appdomain "github.com/ambi/idmagic/backend/application/domain"
	policydomain "github.com/ambi/idmagic/backend/signinpolicy/domain"
)

// ApplicationGate はデバイスコードの承認前に、割り当てとサインインポリシーを評価する。
type ApplicationGate interface {
	EvaluateApplicationAccess(ctx context.Context, tenantID string, bindingType appdomain.ApplicationProtocolType,
		bindingKey, sub string, authn policydomain.SignInAuthentication, clientIP string) (appdomain.ApplicationAccessDecision, error)
}
