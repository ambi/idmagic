package db_memory

// メモリ実装の行確定ポート。PostgreSQL 実装がトランザクションで与える「行の
// 途中経過を残さない」性質を、メモリでは 1 回の書き込みで再現する。テストと
// ローカル組み立てが同じポートを使えるようにするためである。

import (
	"context"
	"errors"

	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
)

type GroupMembershipImportRowCommitter struct {
	repo *GroupRepository
}

func NewGroupMembershipImportRowCommitter(repo *GroupRepository) *GroupMembershipImportRowCommitter {
	return &GroupMembershipImportRowCommitter{repo: repo}
}

func (c *GroupMembershipImportRowCommitter) CommitGroupMembershipImportRow(
	ctx context.Context, mutation groupports.GroupMembershipImportRowMutation,
) error {
	if c.repo == nil {
		return errors.New("group membership import committer is not wired to a repository")
	}
	if mutation.Release {
		_, err := c.repo.RemoveMember(ctx, mutation.TenantID, mutation.GroupID, mutation.UserID)
		return err
	}
	if mutation.Member == nil {
		return errors.New("an add mutation must carry the membership it creates")
	}
	member := *mutation.Member
	if member.Source == "" {
		member.Source = groupdomain.MembershipSourceManual
	}
	_, err := c.repo.AddMember(ctx, &member)
	return err
}

var _ groupports.GroupMembershipImportRowCommitter = (*GroupMembershipImportRowCommitter)(nil)
