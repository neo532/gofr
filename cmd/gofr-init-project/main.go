package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: gofr-init-project <module-path>\n\n")
		fmt.Fprintf(os.Stderr, "Clones github.com/neo532/gofr_layout and renames the module.\n\n")
		fmt.Fprintf(os.Stderr, "Example:\n")
		fmt.Fprintf(os.Stderr, "  gofr-init-project github.com/myorg/myapp\n")
		os.Exit(2)
	}
	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
	}
	modPath := flag.Arg(0)
	oldPath := "github.com/neo532/gofr_layout"

	// Derive project name from module path (last segment).
	parts := strings.Split(modPath, "/")
	projectName := parts[len(parts)-1]
	if projectName == "" {
		fmt.Fprintf(os.Stderr, "invalid module path: %s\n", modPath)
		os.Exit(1)
	}

	// Clone the template.
	fmt.Printf("cloning github.com/neo532/gofr_layout into ./%s ...\n", projectName)
	cmd := exec.Command("git", "clone", "https://github.com/neo532/gofr_layout.git", projectName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		os.Exit(1)
	}

	// Remove .git history so the new project starts fresh.
	gitDir := filepath.Join(projectName, ".git")
	os.RemoveAll(gitDir)

	// Walk the project and replace the old module path.
	fmt.Println("rewriting module path ...")
	replaceCount := 0
	filepath.Walk(projectName, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		// Skip binary files by extension.
		ext := filepath.Ext(path)
		skipExt := map[string]bool{
			".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".ico": true,
			".svg": true, ".woff": true, ".woff2": true, ".ttf": true, ".eot": true,
			".zip": true, ".tar": true, ".gz": true, ".bz2": true,
			".bin": true, ".exe": true, ".dll": true, ".so": true, ".dylib": true,
		}
		if skipExt[ext] {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		content := string(data)
		if !strings.Contains(content, oldPath) {
			return nil
		}

		newContent := strings.ReplaceAll(content, oldPath, modPath)
		if err := os.WriteFile(path, []byte(newContent), info.Mode()); err != nil {
			return err
		}
		replaceCount++
		return nil
	})

	fmt.Printf("\ndone! %d files updated.\n", replaceCount)
	fmt.Printf("\nnext steps:\n")
	fmt.Printf("  cd %s\n", projectName)
	fmt.Printf("  go mod tidy\n")
}
