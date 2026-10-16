package usecases

import authndomain "github.com/ambi/idmagic/backend/authentication/domain"

// acr の語彙は authentication/domain が定義する。ここでは Authentication の内部が同じ名前で参照できるように別名を置く。
const (
	ACRPassword = authndomain.ACRPassword
	ACRMFA      = authndomain.ACRMFA
)
