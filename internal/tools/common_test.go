package tools

import "testing"

func TestIsWithinRoot(t *testing.T) {
	cases := []struct {
		name     string
		path     string
		root     string
		expected bool
	}{
		{"exact root", "/data", "/data", true},
		{"child file", "/data/file.xlsx", "/data", true},
		{"nested child", "/data/sub/dir/file.xlsx", "/data", true},
		{"outside sibling", "/data-other/file.xlsx", "/data", false},
		{"unrelated path", "/etc/passwd", "/data", false},
		{"parent escape via dotdot", "/data/../etc/passwd", "/data", false},
		{"root with trailing slash", "/data/file.xlsx", "/data/", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := isWithinRoot(c.path, c.root); got != c.expected {
				t.Errorf("isWithinRoot(%q, %q) = %v, want %v", c.path, c.root, got, c.expected)
			}
		})
	}
}

func TestAbsolutePathTestEnforcesAllowedRoot(t *testing.T) {
	t.Setenv("EXCEL_MCP_ALLOWED_ROOT", "/data")

	schema := excelValidateTemplateArgumentsSchema
	type args struct {
		FileAbsolutePath string   `zog:"fileAbsolutePath"`
		RequiredSheets   []string `zog:"requiredSheets"`
	}

	insideRoot := args{}
	issues := schema.Parse(map[string]any{
		"fileAbsolutePath": "/data/template.xlsx",
		"requiredSheets":   []any{"Sheet1"},
	}, &insideRoot)
	if len(issues) != 0 {
		t.Fatalf("expected a path inside the allowed root to pass, got issues: %v", issues)
	}

	outsideRoot := args{}
	issues = schema.Parse(map[string]any{
		"fileAbsolutePath": "/etc/passwd",
		"requiredSheets":   []any{"Sheet1"},
	}, &outsideRoot)
	if len(issues) == 0 {
		t.Fatalf("expected a path outside the allowed root to be rejected")
	}
}
