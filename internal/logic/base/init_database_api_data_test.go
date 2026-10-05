package base

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestAPICatalogMatchesRegisteredRoutes(t *testing.T) {
	source, err := os.ReadFile("../../../../api/internal/handler/routes.go")
	if err != nil {
		t.Fatal(err)
	}
	routes := regexp.MustCompile(`Method:\s+http\.Method(\w+),\s+Path:\s+"([^"]+)"`).FindAllStringSubmatch(string(source), -1)
	want := make(map[string]bool)
	for _, route := range routes {
		want[strings.ToUpper(route[1])+" "+route[2]] = true
	}
	if len(want) == 0 {
		t.Fatal("no registered API routes found")
	}
	for _, endpoint := range apiEndpoints {
		key := endpoint.method + " " + endpoint.path
		if !want[key] {
			t.Fatalf("stale or duplicate API catalog entry: %s", key)
		}
		delete(want, key)
	}
	if len(want) != 0 {
		t.Fatalf("registered routes missing from bootstrap catalog: %v", want)
	}
}
