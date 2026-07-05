package main

import "go/ast"

// checkConstructor checks that all fields of the return type struct
// are initialized in composite literals within the function body.
func checkConstructor(fd *ast.FuncDecl, structs map[string]*ast.StructType) []string {
	results := fd.Type.Results
	if len(results.List) != 1 {
		return nil
	}
	star, ok := results.List[0].Type.(*ast.StarExpr)
	if !ok {
		return nil
	}
	ident, ok := star.X.(*ast.Ident)
	if !ok {
		return nil
	}

	st, ok := structs[ident.Name]
	if !ok {
		return nil
	}

	fields := fieldNames(st)
	if len(fields) == 0 {
		return nil
	}

	// Find composite literals of this struct type in the function body.
	missing := make(map[string]bool)
	ast.Inspect(fd.Body, func(n ast.Node) bool {
		cl, ok := n.(*ast.CompositeLit)
		if !ok {
			return true
		}
		cid, ok := cl.Type.(*ast.Ident)
		if !ok || cid.Name != ident.Name {
			return true
		}

		present := make(map[string]bool)
		for _, elt := range cl.Elts {
			kv, ok := elt.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			key, ok := kv.Key.(*ast.Ident)
			if !ok {
				continue
			}
			present[key.Name] = true
		}

		for _, f := range fields {
			if !present[f] {
				missing[f] = true
			}
		}
		return true
	})

	return keys(missing)
}

func fieldNames(st *ast.StructType) []string {
	if st.Fields == nil {
		return nil
	}
	var names []string
	for _, f := range st.Fields.List {
		for _, n := range f.Names {
			names = append(names, n.Name)
		}
	}
	return names
}

func keys(m map[string]bool) []string {
	var s []string
	for k := range m {
		s = append(s, k)
	}
	return s
}
