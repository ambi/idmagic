package usecases_test

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	groupusecases "github.com/ambi/idmagic/backend/idmanagement/group/usecases"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

//go:embed testdata/normalization.examples.json
var groupNormalizationExamples []byte

type groupNormalizationCase struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Given struct {
		ExistingName string `json:"existing_name"`
	} `json:"given"`
	Input struct {
		Name        string  `json:"name"`
		Email       *string `json:"email"`
		Description *string `json:"description"`
	} `json:"input"`
	Expected struct {
		Error       string  `json:"error"`
		Name        *string `json:"name"`
		Email       *string `json:"email"`
		Description *string `json:"description"`
	} `json:"expected"`
}

//spec:covers REQ-IDMANAGEMENT-060, EX-IDMANAGEMENT-060-01, EX-IDMANAGEMENT-060-02, EX-IDMANAGEMENT-060-03, EX-IDMANAGEMENT-060-04, EX-IDMANAGEMENT-060-05, EX-IDMANAGEMENT-060-06, EX-IDMANAGEMENT-060-07, EX-IDMANAGEMENT-060-08, EX-IDMANAGEMENT-060-09: 同じ一次情報の入力と期待結果を作成と更新へ実行し、保存値と拒否後の無作用を検査する。
func TestGroupNormalizationExamples(t *testing.T) {
	var cases []groupNormalizationCase
	decoder := json.NewDecoder(bytes.NewReader(groupNormalizationExamples))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&cases); err != nil || len(cases) == 0 {
		t.Fatalf("具体例を読めないか空である: %v", err)
	}
	for _, tc := range cases {
		for _, operation := range []string{"create", "update"} {
			t.Run(tc.ID+"/"+operation, func(t *testing.T) {
				deps, events := newGroupDeps(t)
				notifier := &recordingGroupNotifier{}
				deps.ProvisioningNotifier = notifier
				ctx := context.Background()
				if tc.Given.ExistingName != "" {
					if _, err := groupusecases.CreateGroup(ctx, deps, groupusecases.CreateGroupInput{ActorUserID: "operator", Name: tc.Given.ExistingName, Now: groupRulesNow}); err != nil {
						t.Fatal(err)
					}
				}
				var targetID string
				if operation == "update" {
					group, err := groupusecases.CreateGroup(ctx, deps, groupusecases.CreateGroupInput{ActorUserID: "operator", Name: "before", Now: groupRulesNow})
					if err != nil {
						t.Fatal(err)
					}
					targetID = group.ID
				}
				before, err := deps.GroupRepo.ListAll(ctx, tenancydomain.DefaultTenantID)
				if err != nil {
					t.Fatal(err)
				}
				// リポジトリが同じポインターを返しても、後の書き換えで比較対象を失わない。
				beforeJSON, err := json.Marshal(before)
				if err != nil {
					t.Fatal(err)
				}
				*events, notifier.calls = nil, 0
				var resultID string
				if operation == "create" {
					group, createErr := groupusecases.CreateGroup(ctx, deps, groupusecases.CreateGroupInput{ActorUserID: "operator", Name: tc.Input.Name, Email: tc.Input.Email, Description: tc.Input.Description, Now: groupRulesNow})
					err = createErr
					if group != nil {
						resultID = group.ID
					}
				} else {
					group, updateErr := groupusecases.UpdateGroup(ctx, deps, groupusecases.UpdateGroupInput{ActorUserID: "operator", ID: targetID, Name: &tc.Input.Name, Email: tc.Input.Email, Description: tc.Input.Description, Now: groupRulesNow.Add(time.Minute)})
					err = updateErr
					if group != nil {
						resultID = group.ID
					}
				}
				expectedErrors := map[string]error{"group_name_required": groupusecases.ErrGroupNameEmpty, "group_name_conflict": groupusecases.ErrGroupNameConflict, "invalid_email": groupusecases.ErrInvalidEmail}
				if tc.Expected.Error != "" {
					want, known := expectedErrors[tc.Expected.Error]
					if !known || !errors.Is(err, want) {
						t.Fatalf("err=%v, want %s", err, tc.Expected.Error)
					}
					after, readErr := deps.GroupRepo.ListAll(ctx, tenancydomain.DefaultTenantID)
					afterJSON, marshalErr := json.Marshal(after)
					if readErr != nil || marshalErr != nil || !bytes.Equal(beforeJSON, afterJSON) || len(*events) != 0 || notifier.calls != 0 {
						t.Fatalf("拒否した操作が作用を残した: state=%s events=%v notifications=%d read=%v marshal=%v", afterJSON, *events, notifier.calls, readErr, marshalErr)
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				stored := storedGroupByID(t, deps, resultID)
				if tc.Expected.Name == nil || stored.Name != *tc.Expected.Name || !reflect.DeepEqual(stored.Email, tc.Expected.Email) || !reflect.DeepEqual(stored.Description, tc.Expected.Description) {
					t.Fatalf("保存値=%+v, want %+v", stored, tc.Expected)
				}
				if len(*events) != 1 || notifier.calls != 1 {
					t.Fatalf("成功の作用: events=%v notifications=%d, want one each", *events, notifier.calls)
				}
				after, readErr := deps.GroupRepo.ListAll(ctx, tenancydomain.DefaultTenantID)
				wantCount, wantEvent := len(before), "GroupUpdated"
				if operation == "create" {
					wantCount++
					wantEvent = "GroupCreated"
				} else if resultID != targetID {
					t.Fatalf("更新対象の識別子が変わった: got %s want %s", resultID, targetID)
				}
				if readErr != nil || len(after) != wantCount || eventTypes(*events)[0] != wantEvent {
					t.Fatalf("成功の保存件数とイベント: groups=%d events=%v read=%v", len(after), *events, readErr)
				}
			})
		}
	}
}
