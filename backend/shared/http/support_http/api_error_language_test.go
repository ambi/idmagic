package support_http_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"unicode"
)

func TestAPIErrorLiteralTextIsEnglish(t *testing.T) {
	t.Parallel()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve current test file")
	}
	backendRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", ".."))

	errorTextArgument := map[string]int{
		"NewMutationError":           1,
		"NewOAuthError":              1,
		"NewScimError":               1,
		"OAuthErrorBody":             1,
		"WriteBrowserError":          3,
		"authorizationErrorURL":      3,
		"newFilterError":             0,
		"newMutationError":           1,
		"newPaginationError":         0,
		"redirectAuthorizationError": 3,
	}

	var files []string
	err := filepath.WalkDir(backendRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") ||
			strings.HasSuffix(entry.Name(), "_test.go") {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		t.Fatalf("walk backend Go files: %v", err)
	}

	fileSet := token.NewFileSet()
	for _, path := range files {
		file, parseErr := parser.ParseFile(fileSet, path, nil, 0)
		if parseErr != nil {
			t.Errorf("parse %s: %v", path, parseErr)
			continue
		}
		ast.Inspect(file, func(node ast.Node) bool {
			composite, ok := node.(*ast.CompositeLit)
			if ok && calledFunctionName(composite.Type) == "SignInOutcome" {
				for _, element := range composite.Elts {
					keyValue, keyed := element.(*ast.KeyValueExpr)
					key, identified := keyValue.Key.(*ast.Ident)
					if keyed && identified && key.Name == "Message" {
						reportJapaneseLiterals(t, fileSet, "SignInOutcome.Message", keyValue.Value)
					}
				}
			}
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			name := calledFunctionName(call.Fun)
			argumentIndex, tracked := errorTextArgument[name]
			if !tracked || len(call.Args) <= argumentIndex {
				return true
			}
			reportJapaneseLiterals(t, fileSet, name, call.Args[argumentIndex])
			return true
		})
	}
}

func reportJapaneseLiterals(t *testing.T, fileSet *token.FileSet, sink string, node ast.Node) {
	t.Helper()

	ast.Inspect(node, func(argumentNode ast.Node) bool {
		literal, ok := argumentNode.(*ast.BasicLit)
		if !ok || literal.Kind != token.STRING {
			return true
		}
		value, unquoteErr := strconv.Unquote(literal.Value)
		if unquoteErr == nil && containsJapaneseScript(value) {
			position := fileSet.Position(literal.Pos())
			t.Errorf("%s:%d: %s error text must be English: %q",
				position.Filename, position.Line, sink, value)
		}
		return true
	})
}

func calledFunctionName(expression ast.Expr) string {
	switch expression := expression.(type) {
	case *ast.Ident:
		return expression.Name
	case *ast.SelectorExpr:
		return expression.Sel.Name
	default:
		return ""
	}
}

func containsJapaneseScript(value string) bool {
	return strings.IndexFunc(value, func(r rune) bool {
		return unicode.In(r, unicode.Han, unicode.Hiragana, unicode.Katakana)
	}) >= 0
}
