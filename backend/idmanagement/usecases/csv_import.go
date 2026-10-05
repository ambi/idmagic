package usecases

import (
	"context"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	idmports "github.com/ambi/idmagic/backend/idmanagement/ports"
)

// CheckStoredCSVLimits は保存したインポートのファイル全体を、行を計画または確定する前に
// 実効の上限で確かめる。投入はすでに上限で拒否しているので、これが拒否するのは投入の後に
// 上限が下がった場合であり、そのとき先行する行を確定してから超過に気付くことを防ぐ。
func CheckStoredCSVLimits(ctx context.Context, artifacts idmports.CSVArtifactStore, tenantID, ref string, policy idmdomain.CSVTransferPolicy) error {
	reader, _, err := artifacts.OpenCSVArtifact(ctx, tenantID, ref)
	if err != nil {
		return err
	}
	defer func() { _ = reader.Close() }()
	return idmdomain.CheckCSVLimits(reader, policy)
}
