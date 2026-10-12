package handlers_http_test

import (
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	userusecases "github.com/ambi/idmagic/backend/idmanagement/user/usecases"
)

// scimUserLifecycle は、組み立ての地点と同じく IdManagement の操作を SCIM のポートへ渡す。
func scimUserLifecycle(users userports.UserRepository) userusecases.UserLifecycleCommands {
	return userusecases.UserLifecycleCommands{Deps: userusecases.AdminUserDeps{UserRepo: users}, Actor: "scim"}
}
