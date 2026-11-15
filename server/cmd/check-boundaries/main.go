// Command check-boundaries enforces game module import rules without golangci.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func main() {
	out, err := exec.Command("go", "list", "-f", "{{.ImportPath}}\t{{join .Imports \",\"}}", "./...").CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "go list: %v\n%s\n", err, out)
		os.Exit(1)
	}

	var violations []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		pkg := parts[0]
		var imports []string
		if len(parts) == 2 && parts[1] != "" {
			imports = strings.Split(parts[1], ",")
		}
		for _, imp := range imports {
			if msg := forbidden(pkg, imp); msg != "" {
				violations = append(violations, fmt.Sprintf("%s imports %s (%s)", pkg, imp, msg))
			}
		}
	}

	if len(violations) > 0 {
		fmt.Fprintln(os.Stderr, "game boundary violations:")
		for _, v := range violations {
			fmt.Fprintln(os.Stderr, " -", v)
		}
		os.Exit(1)
	}
	fmt.Println("check-boundaries: ok")
}

func forbidden(pkg, imp string) string {
	const (
		dawugui    = "github.com/example/game/internal/game/dawugui"
		liuzichong = "github.com/example/game/internal/game/liuzichong"
		hdawugui   = "github.com/example/game/internal/pitaya/handlers/dawugui"
		hliuzi     = "github.com/example/game/internal/pitaya/handlers/liuzichong"
	)

	// Composition roots may wire all games.
	if pkg == "github.com/example/game/cmd/game" || pkg == "github.com/example/game/internal/game" {
		return ""
	}
	// Nested rules may only be imported by their parent game package (Go enforces).
	if strings.Contains(pkg, "/internal/game/dawugui") && (imp == liuzichong || strings.HasPrefix(imp, liuzichong+"/") || imp == hliuzi) {
		return "games must not cross-import"
	}
	if strings.Contains(pkg, "/internal/game/liuzichong") && (imp == dawugui || strings.HasPrefix(imp, dawugui+"/") || imp == hdawugui) {
		return "games must not cross-import"
	}
	if strings.Contains(pkg, "/handlers/dawugui") && (imp == liuzichong || strings.HasPrefix(imp, liuzichong+"/") || imp == hliuzi) {
		return "handlers must not cross-import"
	}
	if strings.Contains(pkg, "/handlers/liuzichong") && (imp == dawugui || strings.HasPrefix(imp, dawugui+"/") || imp == hdawugui) {
		return "handlers must not cross-import"
	}

	concreteGame := imp == dawugui || imp == liuzichong ||
		strings.HasPrefix(imp, dawugui+"/") || strings.HasPrefix(imp, liuzichong+"/")

	if concreteGame {
		if strings.Contains(pkg, "/internal/platform/") ||
			strings.Contains(pkg, "/pkg/") ||
			strings.Contains(pkg, "/handlers/room") ||
			strings.Contains(pkg, "/pitaya/bot") ||
			strings.Contains(pkg, "/pitaya/runtime") ||
			strings.Contains(pkg, "/pitaya/commit") {
			return "platform/room/bot must use registry+engine only"
		}
	}

	if strings.Contains(pkg, "/internal/game/") {
		if strings.Contains(imp, "github.com/topfreegames/pitaya") || imp == "github.com/gin-gonic/gin" {
			return "game packages must not import Pitaya/Gin"
		}
	}
	return ""
}
