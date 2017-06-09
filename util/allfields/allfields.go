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
		structs map[string]*structReq
	}

	typeVisitor struct {
		ctv *confTreeVisitor
	}

	typeSpecVisitor struct {
		ctv   *confTreeVisitor
		tName string
	}

	fieldVisitor struct {
		ctv   *confTreeVisitor
		snode *structDef
	}

	structDef struct {
		typeName                string
		simpleFields            map[string]string
		namedFields, anonFields map[string]*structReq
	}

	structReq struct {
		typeName string
		sat      *structDef
		alias    bool
	}

	packageMap map[string]*ast.Package
)

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
	ctv := &confTreeVisitor{structs: map[string]*structReq{}}

	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				ast.Walk(ctv, decl)
			}
		}
	}

	return ctv.treeFor(structName)
}

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

func (ctv *confTreeVisitor) getStructReq(name string) *structReq {
	req, has := ctv.structs[name]
	if has {
		return req
	}
	req = &structReq{typeName: name}
	ctv.structs[name] = req
	return req
}

func (v *typeVisitor) Visit(n ast.Node) ast.Visitor {
	switch ts := n.(type) {
	default:
		spew.Printf("tV: not typespec: %v\n", ts)
	case nil:
	case *ast.TypeSpec:
		return &typeSpecVisitor{ctv: v.ctv, tName: ts.Name.Name}
	}
	return nil
}

func (tv *typeSpecVisitor) Visit(n ast.Node) ast.Visitor {
	ctv := tv.ctv

	switch rn := n.(type) {
	default:
		spew.Printf("typeSpec default (->alias): %+#v\n", n)
	case nil:
	case *ast.CommentGroup:
	case *ast.Ident:
		if tv.tName == "" {
			tv.tName = rn.Name
		} else {
			alias := ctv.getStructReq(tv.tName)
			alias.alias = true
		}
	case *ast.ArrayType, *ast.MapType, *ast.ChanType:
		alias := ctv.getStructReq(tv.tName)
		alias.alias = true
	case *ast.InterfaceType, *ast.FuncType: // not really, but end of confirmable paths
		alias := ctv.getStructReq(tv.tName)
		alias.alias = true
	case *ast.StructType:
		sName := tv.tName
		snode := ctv.getStructReq(sName)
		if snode.sat != nil {
			panic("redefinition of " + sName)
		}
		snode.sat = &structDef{
			typeName:     tv.tName,
			namedFields:  map[string]*structReq{},
			anonFields:   map[string]*structReq{},
			simpleFields: map[string]string{},
		}

		return &fieldVisitor{ctv: ctv, snode: snode.sat}
	}

	return nil
}

func (v *fieldVisitor) Visit(n ast.Node) ast.Visitor {
	switch f := n.(type) {
	default:
		spew.Printf("fieldVisitor default: %v\n", f)
	case nil:
	case *ast.FieldList:
		return v
	case *ast.Field:
		switch {
		default:
			for _, n := range f.Names {
				v.snode.namedFields[n.Name] = v.ctv.getStructReq(typeName(f.Type))
			}
		case len(f.Names) == 0:
			tn := typeName(f.Type)
			v.snode.anonFields[tn] = v.ctv.getStructReq(tn)
		case simpleType(f.Type):
			for _, n := range f.Names {
				v.snode.simpleFields[n.Name] = typeName(f.Type)
			}
		}
	}
	return nil
}

// XXX Cycles in the struct def tree will not be caught!
// Linked lists are a no go!
func (ctv *confTreeVisitor) treeFor(name string) confNode {
	first, has := ctv.structs[name]
	if !has {
		panic(fmt.Errorf("no requirement recorded for struct type named %q", name))
	}

	return first.confNode(name)
}

func (sr *structReq) confNode(name string) confNode {
	switch {
	default:
		panic(fmt.Errorf("no satisfying def for `%s %s`", name, sr.typeName))
	case sr.sat != nil:
		return sr.sat.confNode()
	case sr.alias:
		return newConfirmation(sr.typeName)
	}
}

func (sd *structDef) confNode() confNode {
	node := newStructNode(sd.typeName)

	for name, typeName := range sd.simpleFields {
		node.kids[name] = newConfirmation(typeName)
	}

	for name, req := range sd.namedFields {
		node.kids[name] = req.confNode(name)
	}

	for name, req := range sd.anonFields {
		field := req.confNode(name)
		node.kids[name] = field
		if aStruct, is := field.(*structNode); is {
			for n, subField := range aStruct.kids {
				if _, has := node.kids[n]; !has {
					node.kids[n] = subField
				}
			}
		}
	}
	return node
}

func typeName(fType ast.Node) string {
	switch tx := fType.(type) {
	default:
		//spew.Printf("typeName default %+#v\n", tx)
	case *ast.Ident:
		return tx.Name
	case *ast.MapType:
		return "map[?]?"
	case *ast.ArrayType:
		switch ex := tx.Elt.(type) {
		default:
			//spew.Println("array default", tx, ex)
		case *ast.Ident:
			return "[]" + ex.Name
		}
	case *ast.SelectorExpr:
		return typeName(tx.X) + "." + tx.Sel.Name
	case *ast.StarExpr:
		switch sx := tx.X.(type) {
		default:
			//spew.Println("star default", tx, sx)
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
		//spew.Printf("default case, simpleType: %+#v\n", tx)
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
