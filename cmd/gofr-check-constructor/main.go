package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/neo532/gokit/errorx"
)

func main() {
	prefix := flag.String("prefix", "New", "function name prefix to match")
	exclude := flag.String("exclude", "", "regex to exclude type names")
	flag.Parse()

	if flag.NArg() == 0 {
		fmt.Fprintf(os.Stderr, "usage: gofr-check-constructor [-prefix New] [-exclude regex] ./path...\n")
		os.Exit(2)
	}

	var excludeRe *regexp.Regexp
	if *exclude != "" {
		var err error
		excludeRe, err = regexp.Compile(*exclude)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid -exclude regex: %v\n", err)
			os.Exit(2)
		}
	}

	fset := token.NewFileSet()
	pkgs := make(packageDir) // dir -> struct name -> struct type
	exit := 0

	for _, root := range flag.Args() {
		root = strings.TrimRight(root, "/")
		root = strings.TrimSuffix(root, "/...")

		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return errorx.Wrap(err)
			}
			if info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}

			f, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
			if err != nil {
				return nil
			}

			dir := filepath.Dir(path)
			structs := pkgs.structs(dir)

			// Collect struct definitions in this file
			for _, decl := range f.Decls {
				gen, ok := decl.(*ast.GenDecl)
				if !ok || gen.Tok != token.TYPE {
					continue
				}
				for _, spec := range gen.Specs {
					ts, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}
					st, ok := ts.Type.(*ast.StructType)
					if !ok {
						continue
					}
					structs[ts.Name.Name] = st
				}
			}

			// Check constructor functions in this file
			for _, decl := range f.Decls {
				fd, ok := decl.(*ast.FuncDecl)
				if !ok || fd.Body == nil || fd.Type.Results == nil {
					continue
				}
				name := fd.Name.Name
				if !strings.HasPrefix(name, *prefix) {
					continue
				}
				if excludeRe != nil && excludeRe.MatchString(name) {
					continue
				}
				if missing := checkConstructor(fd, structs); len(missing) > 0 {
					pos := fset.Position(fd.Pos())
					fmt.Fprintf(os.Stderr, "%s:%d: %s is missing fields %s\n",
						pos.Filename, pos.Line, name, strings.Join(missing, ", "))
					exit = 1
				}
			}
			return nil
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "error walking %s: %v\n", root, err)
			exit = 1
		}
	}
	os.Exit(exit)
}

// packageDir maps directory paths to their struct type definitions.
type packageDir map[string]map[string]*ast.StructType

func (p packageDir) structs(dir string) map[string]*ast.StructType {
	if p[dir] == nil {
		p[dir] = make(map[string]*ast.StructType)
	}
	return p[dir]
}
