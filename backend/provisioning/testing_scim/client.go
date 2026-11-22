// Package testing_scim はローカルの相手へ実際の SCIM 表現を送る受け入れテスト用の生成器を提供する。
package testing_scim

import (
	"net/http"

	"github.com/ambi/idmagic/backend/provisioning"
	"github.com/ambi/idmagic/backend/provisioning/internal/client_scim"
	provisioningdomain "github.com/ambi/idmagic/backend/provisioning/internal/domain"
	provisioningports "github.com/ambi/idmagic/backend/provisioning/internal/ports"
)

// BearerClientFactory は通信先の検査と TLS の信頼をテスト用の HTTP クライアントへ任せる。
// 本文の変換と保存された資格情報の送信は本番の SCIM クライアントを使う。
func BearerClientFactory(client *http.Client) provisioning.TargetClientFactory {
	return func(conn *provisioningdomain.ProvisioningConnection, secret string) (provisioningports.ProvisioningTargetClient, error) {
		return client_scim.NewBearerTokenClient(client, conn.BaseURL, secret), nil
	}
}
