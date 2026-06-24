package codebase

import (
	"go/ast"
	"go/token"
	"strings"
)

type visitor struct {
	fset    *token.FileSet
	f       *ast.File
	pkgPath string
	file    string
	data    []byte
	idx     *Index
}

func (v *visitor) walk() {
	for _, decl := range v.f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}

		f := v.buildFunction(fn)

		v.idx.Functions[f.ID] = f
		v.idx.ByFile[v.file] = append(v.idx.ByFile[v.file], f.ID)

		if fn.Recv == nil {
			v.idx.Roots = append(v.idx.Roots, f.ID)
		}
	}

	for id, fn := range v.idx.Functions {
		if fn.File == v.file {
			v.resolveCalls(&fn)
			v.idx.Functions[id] = fn
		}
	}
}

func (v *visitor) buildFunction(fn *ast.FuncDecl) Function {
	var recv string
	if fn.Recv != nil && len(fn.Recv.List) > 0 {
		recv = exprString(fn.Recv.List[0].Type)
	}

	id := fn.Name.Name
	if recv != "" {
		id = "(" + recv + ")." + id
	}
	fullID := v.pkgPath + "." + id

	start := fn.Pos()
	end := fn.End()
	body := string(v.data[start-1 : end-1])

	sig := functionSignature(fn)

	var doc string
	if fn.Doc != nil {
		doc = fn.Doc.Text()
	}

	return Function{
		ID:         fullID,
		PkgPath:    v.pkgPath,
		Name:       fn.Name.Name,
		Receiver:   recv,
		File:       v.file,
		Line:       v.fset.Position(fn.Pos()).Line,
		EndLine:    v.fset.Position(fn.End()).Line,
		Signature:  sig,
		Body:       body,
		IsExported: fn.Name.IsExported(),
		Doc:        doc,
	}
}

func (v *visitor) resolveCalls(fn *Function) {
	for id := range v.idx.Functions {
		if id == fn.ID {
			continue
		}
		shortName := id[strings.LastIndex(id, ".")+1:]
		if strings.Contains(fn.Body, shortName+"(") {
			fn.Calls = append(fn.Calls, id)
		}
	}
}

func functionSignature(fn *ast.FuncDecl) string {
	var b strings.Builder
	b.WriteString("func ")
	if fn.Recv != nil && len(fn.Recv.List) > 0 {
		b.WriteString("(")
		b.WriteString(exprString(fn.Recv.List[0].Type))
		b.WriteString(") ")
	}
	b.WriteString(fn.Name.Name)
	b.WriteString("(")
	for i, p := range fn.Type.Params.List {
		if i > 0 {
			b.WriteString(", ")
		}
		for j, name := range p.Names {
			if j > 0 {
				b.WriteString(", ")
			}
			b.WriteString(name.Name)
		}
		if len(p.Names) > 0 {
			b.WriteString(" ")
		}
		b.WriteString(exprString(p.Type))
	}
	b.WriteString(")")
	if fn.Type.Results != nil && len(fn.Type.Results.List) > 0 {
		b.WriteString(" ")
		if len(fn.Type.Results.List) == 1 && len(fn.Type.Results.List[0].Names) == 0 {
			b.WriteString(exprString(fn.Type.Results.List[0].Type))
		} else {
			b.WriteString("(")
			for i, r := range fn.Type.Results.List {
				if i > 0 {
					b.WriteString(", ")
				}
				for j, name := range r.Names {
					if j > 0 {
						b.WriteString(", ")
					}
					b.WriteString(name.Name)
				}
				if len(r.Names) > 0 {
					b.WriteString(" ")
				}
				b.WriteString(exprString(r.Type))
			}
			b.WriteString(")")
		}
	}
	return b.String()
}

func exprString(e ast.Expr) string {
	switch t := e.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + exprString(t.X)
	case *ast.SelectorExpr:
		return exprString(t.X) + "." + t.Sel.Name
	case *ast.ArrayType:
		return "[]" + exprString(t.Elt)
	case *ast.MapType:
		return "map[" + exprString(t.Key) + "]" + exprString(t.Value)
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.Ellipsis:
		return "..." + exprString(t.Elt)
	default:
		return "..."
	}
}
