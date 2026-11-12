package db_memory_test

import (
	"testing"

	"github.com/ambi/idmagic/backend/sourcing/internal/directory/db_memory"
	"github.com/ambi/idmagic/backend/sourcing/internal/directory/testing_contract"
)

//spec:covers REQ-SOURCING-026: テナント、Connector、世代と期限の両側を検査し、単回配信時に暗号文を消す。
func TestPasswordChallengePersistenceContract(t *testing.T) {
	testing_contract.RunPasswordChallenges(t, testing_contract.PasswordFixture{Store: db_memory.NewPasswordChallengeStore(), TenantID: "tenant-a", OtherTenantID: "tenant-b", ConnectorID: "connector-a", OtherConnectorID: "connector-b"})
}
