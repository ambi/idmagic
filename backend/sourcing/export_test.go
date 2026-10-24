package sourcing

import "context"

// EnabledComputerBySID と EnabledComputerByPrincipal は、受け入れテストが保存したコンピューターを
// 外部の不変 ID で確かめるための入口である。見つからなければ空を返す。
func (m Module) EnabledComputerBySID(ctx context.Context, tenantID, sid string) (string, error) {
	found, err := m.computers.FindEnabledBySID(ctx, tenantID, sid)
	if err != nil || found == nil {
		return "", err
	}
	return found.ExternalID, nil
}

func (m Module) EnabledComputerByPrincipal(ctx context.Context, tenantID, samAccountName, realm string) (string, error) {
	found, err := m.computers.FindEnabledByPrincipal(ctx, tenantID, samAccountName, realm)
	if err != nil || found == nil {
		return "", err
	}
	return found.ExternalID, nil
}
