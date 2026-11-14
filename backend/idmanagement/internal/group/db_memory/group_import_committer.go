package db_memory

// メモリ実装の行確定ポート。PostgreSQL 実装がトランザクションで与える「行の
// 途中経過を残さない」性質を、メモリでは失敗時に先行した書き込みを巻き戻して
// 再現する。テストとローカル組み立てが同じポートを使えるようにするためである。

import (
	"context"
	"errors"

	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

type GroupImportRowCommitter struct {
	repo  *GroupRepository
	quota tenantports.QuotaRepository
}

// NewGroupImportRowCommitter は、quota でグループの使用量を加減算する committer を返す。
// quota が nil なら使用量を扱わない。使用量を見ないテストのための省略である。
func NewGroupImportRowCommitter(repo *GroupRepository, quota tenantports.QuotaRepository) *GroupImportRowCommitter {
	return &GroupImportRowCommitter{repo: repo, quota: quota}
}

func (c *GroupImportRowCommitter) CommitGroupImportRow(ctx context.Context, mutation groupports.GroupImportRowMutation) error {
	if c.repo == nil {
		return errors.New("group import committer is not wired to a repository")
	}
	if mutation.Delete {
		if mutation.Before == nil {
			return errors.New("a delete mutation must name the group it removes")
		}
		for _, userID := range mutation.RemovedMemberships {
			if _, err := c.repo.RemoveMember(ctx, mutation.Before.TenantID, mutation.Before.ID, userID); err != nil {
				return err
			}
		}
		if err := c.repo.Delete(ctx, mutation.Before.TenantID, mutation.Before.ID); err != nil {
			return err
		}
		if mutation.ReleasesGroupQuota && c.quota != nil {
			return c.quota.Decrement(ctx, mutation.Before.TenantID, tenancydomain.ResourceGroups, 1)
		}
		return nil
	}
	if mutation.After == nil {
		return errors.New("an upsert mutation must carry the resulting group")
	}
	consumesQuota := mutation.ConsumesGroupQuota && c.quota != nil
	if consumesQuota {
		if err := c.quota.CheckAndIncrement(ctx, mutation.After.TenantID, tenancydomain.ResourceGroups, 1); err != nil {
			return err
		}
	}
	// 行が途中で失敗したら、加算した使用量も戻す。
	release := func() {
		if consumesQuota {
			_ = c.quota.Decrement(ctx, mutation.After.TenantID, tenancydomain.ResourceGroups, 1)
		}
	}
	if err := c.repo.Save(ctx, mutation.After); err != nil {
		release()
		return err
	}
	if mutation.Rule != nil {
		if err := c.repo.SaveDynamicRule(ctx, mutation.Rule); err != nil {
			// 規則が保存できない行は、Group 本体の変更も残さない。
			if mutation.Before != nil {
				_ = c.repo.Save(ctx, mutation.Before)
			} else {
				_ = c.repo.Delete(ctx, mutation.After.TenantID, mutation.After.ID)
			}
			release()
			return err
		}
	}
	return nil
}

var _ groupports.GroupImportRowCommitter = (*GroupImportRowCommitter)(nil)
