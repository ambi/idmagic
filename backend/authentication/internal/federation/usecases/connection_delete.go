package usecases

import (
	"context"
	"errors"
)

// ErrConnectionInUse は、利用者との連携が残っている接続を削除しようとしたことを表す。
var ErrConnectionInUse = errors.New("identity provider connection still has linked identities")

// DeleteConnection は外部 IdP 接続を削除する。存在しない接続の削除は成功として扱う。
//
// 連携が残る接続は削除しない。削除すると、連携した利用者はその接続でサインインできなく
// なるうえ、どの接続に連携していたかも失われる。PostgreSQL の外部キーも同じ削除を
// 止めるが、それは 500 にしかならず、メモリの保存先では止まらないため、ここで判定する。
func DeleteConnection(ctx context.Context, deps BrokerDeps, tenantID, providerID string) error {
	connection, err := deps.Connections.Find(ctx, tenantID, providerID)
	if err != nil {
		return err
	}
	if connection == nil {
		return nil
	}
	linked, err := deps.Identities.ExistsForProvider(ctx, tenantID, providerID)
	if err != nil {
		return err
	}
	if linked {
		return ErrConnectionInUse
	}
	return deps.Connections.Delete(ctx, tenantID, providerID)
}
