package server_http

import (
	"context"

	auditports "github.com/ambi/idmagic/backend/audit/ports"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
)

// auditUsernameLookup は Audit の要求する ID と存在の有無へ User を変換する。
// 未配線の照会は、従来と同じく username に一致する記録を返さない。
func auditUsernameLookup(users userports.UserRepository) auditports.UsernameLookup {
	if users == nil {
		return nil
	}
	return func(ctx context.Context, tenantID, username string) (string, bool, error) {
		user, err := users.FindByUsername(ctx, tenantID, username)
		if err != nil || user == nil {
			return "", false, err
		}
		return user.ID, true, nil
	}
}
