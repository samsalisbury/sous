package allfields

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"

	"github.com/davecgh/go-spew/spew"
)

type (
	confTreeVisitor struct {
		structs map[string]*structNode
		aliases map[string]struct{}
		needs   []fieldNeed
	}

	typeVisitor struct {
		ctv *confTreeVisitor
	}

	typeSpecVisitor struct {
		ctv   *confTreeVisitor
		tName string
	}

	fieldNeed struct {
		typeName, fieldName string
		node                *structNode
	}

	packageMap map[string]*ast.Package
)

func (ctv *confTreeVisitor) Visit(n ast.Node) ast.Visitor {
	switch rn := n.(type) {
	default:
		fmt.Printf("Not GenDecl: %T\n", rn)
	case *ast.FuncDecl:
	case *ast.GenDecl:
		switch rn.Tok {
		default:
			spew.Printf("Not TYPE: %v\n", rn.Tok)
		case token.IMPORT, token.CONST, token.VAR:
		case token.TYPE:
			return &typeVisitor{ctv: ctv}
		}
	}
	return nil
}

func (v *typeVisitor) Visit(n ast.Node) ast.Visitor {
	switch ts := n.(type) {
	case *ast.TypeSpec:
		return &typeSpecVisitor{ctv: v.ctv, tName: ts.Name.Name}
	}
	return nil
}

func (tv *typeSpecVisitor) Visit(n ast.Node) ast.Visitor {
	ctv := tv.ctv

	switch ts := n.(type) {
	default:
		ctv.aliases[tv.tName] = struct{}{}
	case *ast.StructType:
		sName := tv.tName
		snode, has := ctv.structs[sName]
		if !has {
			snode = newStructNode(sName)
			ctv.structs[sName] = snode
		}

		for _, f := range ts.Fields.List {
			for _, n := range f.Names {
				if simpleType(f.Type) {
					snode.kids[n.Name] = &confirmation{typeName: typeName(f.Type)}
				} else {
					snode.needs = append(snode.needs, fieldNeed{typeName: typeName(f.Type), fieldName: n.Name})
				}
			}
		}
	}

	return nil
}

func (ctv *confTreeVisitor) treeFor(name string) confNode {
	first, has := ctv.structs[name]
	if !has {
		panic(fmt.Errorf("no struct type named %q", name))
	}
	must := []*structNode{first}

	for len(must) > 0 {
		var thumb *structNode
		thumb, must = must[0], must[1:]

		for len(thumb.needs) > 0 {
			var nd fieldNeed
			nd, thumb.needs = thumb.needs[0], thumb.needs[1:]

			if s, has := ctv.structs[nd.typeName]; has {
				thumb.kids[nd.fieldName] = s
				must = append(must, s)
			} else if _, has := ctv.aliases[nd.typeName]; has {
				thumb.kids[nd.fieldName] = &confirmation{typeName: nd.typeName}
			} else {
				panic(fmt.Errorf("unfulfilled need: %v", nd))
			}
		}
	}

	return first
}

func typeName(fType ast.Expr) string {
	switch tx := fType.(type) {
	default:
		spew.Printf("default %+#v\n", tx)
	case *ast.Ident:
		return tx.Name
	case *ast.ArrayType:
		switch ex := tx.Elt.(type) {
		default:
			spew.Println("array default", tx, ex)
		case *ast.Ident:
			return ex.Name
		}
	case *ast.SelectorExpr:
		return typeName(tx.X) + "." + tx.Sel.Name
	case *ast.StarExpr:
		switch sx := tx.X.(type) {
		default:
			spew.Println("star default", tx, sx)
		case *ast.Ident:
			return sx.Name
		}
	}
	return "dunno!"
}

// This is kind of misnomer, and means more like "followable": is the type a struct in this package?
// Either this behavior is wrong and should change, or it's right and the name should be "local struct"
func simpleType(fType ast.Expr) bool {
	switch tx := fType.(type) {
	default:
		spew.Printf("default case, simpleType: %+#v\n", tx)
		return false
	case *ast.Ident:
		switch tx.Name {
		default:
			return false
		case "bool", "byte", "complex64", "complex128", "error", "float32",
			"float64", "int", "int8", "int16", "int32", "int64", "rune", "string",
			"uint", "uint8", "uint16", "uint32", "uint64", "uintptr":
			return true
		}

	// This and ArrayType maybe should deplend on the kind of their element types,
	// but can't really be sure that each element is confirmed anyway...
	case *ast.MapType:
		return true
	case *ast.ArrayType:
		//return simpleType(tx.Elt)
		return true
	case *ast.SelectorExpr:
		return true
	case *ast.ChanType:
		return true
	case *ast.FuncType:
		return true
	case *ast.StarExpr:
		return simpleType(tx.X)
	}
}

func ParseDir(dir string) packageMap {
	fset := &token.FileSet{}
	inclAll := func(os.FileInfo) bool { return true }

	pkgs, err := parser.ParseDir(fset, dir, inclAll, parser.DeclarationErrors)
	if err != nil {
		panic(err)
	}

	return pkgs
}

func ExtractTree(pkgs packageMap, structName string) confNode {
	ctv := &confTreeVisitor{structs: map[string]*structNode{}, aliases: map[string]struct{}{}}

	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				ast.Walk(ctv, decl)
			}
		}
	}
	spew.Dump("done visiting")

	return ctv.treeFor(structName)
}
