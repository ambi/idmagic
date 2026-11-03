package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"slices"
	"strings"
	"time"
)

// ReportedToolList は、ある時点の報告の一覧と、それが上流の一覧のすべてかである。
type ReportedToolList struct {
	Tools    []ReportedTool `json:"tools"`
	Complete bool           `json:"complete"`
}

// ModifiedTool は、直前の一覧と最新の報告の両方にあって定義が異なるツールである。
type ModifiedTool struct {
	Name string
	// Fields は title、description、arguments のうち値が異なるものを、この順で持つ。
	Fields   []string
	Previous ReportedTool
}

// ReportedToolChanges は、内容の変わった直前の一覧から最新の報告への差分である (REQ-OAUTH2-087)。
type ReportedToolChanges struct {
	ChangedAt time.Time
	Added     []string
	Removed   []string
	Modified  []ModifiedTool
}

// Fingerprint は報告の内容を表す。ツールと引数の並びと complete は内容に含めない (REQ-OAUTH2-086)。
// 保存先はこの値を比べて、内容の変わった報告でだけ直前の一覧を残す。
func (r McpToolReport) Fingerprint() string {
	tools := make([]ReportedTool, len(r.Tools))
	for index, tool := range r.Tools {
		tool.Arguments = sortedArguments(tool.Arguments)
		tools[index] = tool
	}
	slices.SortFunc(tools, func(a, b ReportedTool) int { return strings.Compare(a.Name, b.Name) })
	// JSON は文字列を引用して区切るため、項目の境界をまたいで同じ表現にならない。
	canonical, err := json.Marshal(tools)
	if err != nil {
		panic(err) // 文字列と文字列の配列だけの値は必ず符号化できる。
	}
	sum := sha256.Sum256(canonical)
	return hex.EncodeToString(sum[:])
}

// Changes は直前の一覧からの差分を返す。直前の一覧がなければ nil を返す。
// 完全でない一覧は取得できなかったツールを持たないので、その側から追加と削除を推測しない。
func (r McpToolReport) Changes() *ReportedToolChanges {
	if r.Previous == nil {
		return nil
	}
	previous := make(map[string]ReportedTool, len(r.Previous.Tools))
	for _, tool := range r.Previous.Tools {
		previous[tool.Name] = tool
	}
	current := make(map[string]bool, len(r.Tools))
	changes := &ReportedToolChanges{ChangedAt: r.ChangedAt, Added: []string{}, Removed: []string{}, Modified: []ModifiedTool{}}
	for _, tool := range r.Tools {
		current[tool.Name] = true
		before, existed := previous[tool.Name]
		if !existed {
			if r.Previous.Complete {
				changes.Added = append(changes.Added, tool.Name)
			}
			continue
		}
		if fields := differentFields(before, tool); len(fields) > 0 {
			changes.Modified = append(changes.Modified, ModifiedTool{Name: tool.Name, Fields: fields, Previous: before})
		}
	}
	if r.Complete {
		for _, tool := range r.Previous.Tools {
			if !current[tool.Name] {
				changes.Removed = append(changes.Removed, tool.Name)
			}
		}
	}
	return changes
}

func differentFields(before, after ReportedTool) []string {
	var fields []string
	if before.Title != after.Title {
		fields = append(fields, "title")
	}
	if before.Description != after.Description {
		fields = append(fields, "description")
	}
	if !slices.Equal(sortedArguments(before.Arguments), sortedArguments(after.Arguments)) {
		fields = append(fields, "arguments")
	}
	return fields
}

func sortedArguments(arguments []string) []string {
	sorted := slices.Clone(arguments)
	slices.Sort(sorted)
	return sorted
}
