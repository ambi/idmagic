package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"slices"
	"strings"
	"time"
)

// CatalogToolList は、ある時点のカタログの一覧と、それが上流の一覧のすべてかである。
type CatalogToolList struct {
	Tools    []CatalogTool `json:"tools"`
	Complete bool          `json:"complete"`
}

// CatalogToolModification は、直前の一覧と最新のカタログの両方にあって定義が異なるツールである。
type CatalogToolModification struct {
	Name string
	// Fields は title、description、arguments のうち値が異なるものを、この順で持つ。
	Fields   []string
	Previous CatalogTool
}

// CatalogToolChanges は、内容の変わった直前の一覧から最新のカタログへの差分である (REQ-OAUTH2-087)。
type ToolCatalogChanges struct {
	ChangedAt time.Time
	Added     []string
	Removed   []string
	Modified  []CatalogToolModification
}

// Fingerprint はカタログの内容を表す。ツールと引数の並びと complete は内容に含めない (REQ-OAUTH2-086)。
// 保存先はこの値を比べて、内容の変わったカタログでだけ直前の一覧を残す。
func (r McpToolCatalog) Fingerprint() string {
	tools := make([]CatalogTool, len(r.Tools))
	for index, tool := range r.Tools {
		tool.Arguments = sortedArguments(tool.Arguments)
		tools[index] = tool
	}
	slices.SortFunc(tools, func(a, b CatalogTool) int { return strings.Compare(a.Name, b.Name) })
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
func (r McpToolCatalog) Changes() *ToolCatalogChanges {
	if r.Previous == nil {
		return nil
	}
	previous := make(map[string]CatalogTool, len(r.Previous.Tools))
	for _, tool := range r.Previous.Tools {
		previous[tool.Name] = tool
	}
	current := make(map[string]bool, len(r.Tools))
	changes := &ToolCatalogChanges{ChangedAt: r.ChangedAt, Added: []string{}, Removed: []string{}, Modified: []CatalogToolModification{}}
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
			changes.Modified = append(changes.Modified, CatalogToolModification{Name: tool.Name, Fields: fields, Previous: before})
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

func differentFields(before, after CatalogTool) []string {
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
