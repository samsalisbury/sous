package allfields

import (
	"fmt"
	"go/ast"

	"github.com/davecgh/go-spew/spew"
)

type (
	functionFinder struct {
		target string
		tree   confNode
		scope  *ast.Scope
	}

	confirmVisitor struct {
		tree  confNode
		scope *ast.Scope
	}

	recvVisitor struct {
		tree  confNode
		scope *ast.Scope
	}

	argVisitor struct {
		names []string
		tree  confNode
		scope *ast.Scope
	}

	exprVisitor struct {
		tree  confNode
		scope *ast.Scope
	}

	selectorVisitor struct {
		tree  confNode
		scope *ast.Scope

		on     *selectorVisitor
		target confNode
	}

	bodyVisitor struct {
		tree  confNode
		scope *ast.Scope
	}

	ifVisitor struct {
		tree  confNode
		scope *ast.Scope
	}

	assignVisitor struct {
		stmt  *ast.AssignStmt
		tree  confNode
		scope *ast.Scope
	}
)

func (v *functionFinder) Visit(n ast.Node) ast.Visitor {
	switch rn := n.(type) {
	default:
		fmt.Printf("Not FuncDecl: %T\n", rn)
	case nil:
	case *ast.GenDecl:
	case *ast.FuncDecl:
		if rn.Name.Name == v.target {
			return &confirmVisitor{tree: v.tree, scope: ast.NewScope(v.scope)}
		}
	}
	return nil
}

func (v *confirmVisitor) Visit(n ast.Node) ast.Visitor {
	switch rn := n.(type) {
	default:
		spew.Printf("unexpected type to cV: %+#v\n", n)
	case nil:
	case *ast.CommentGroup:
	case *ast.Ident: //Name
	case *ast.FieldList: //Recv
		return &recvVisitor{tree: v.tree, scope: v.scope}
	case *ast.FuncType: //Type
		rv := &recvVisitor{tree: v.tree, scope: v.scope}
		ast.Walk(rv, rn.Params)
	case *ast.BlockStmt: //body
		if v.tree.selfConfirmed() { // only if the function's receiver or arg is if the type
			spew.Dump(v.scope)
			return &bodyVisitor{tree: v.tree, scope: v.scope}
		}
	}
	return nil
}

func (v *recvVisitor) Visit(n ast.Node) ast.Visitor {
	switch n.(type) {
	default:
		spew.Printf("unexpected type to recvV: %+#v\n", n)
	case nil:
	case *ast.CommentGroup:
	case *ast.FieldList:
		return v
	case *ast.Field:
		return &argVisitor{names: []string{}, scope: v.scope, tree: v.tree}
	}
	return nil
}

func (v *argVisitor) Visit(n ast.Node) ast.Visitor {
	switch rn := n.(type) {
	default:
		spew.Printf("unexpected type to argV: %+#v\n", n)
	case nil:
	case *ast.CommentGroup, *ast.BasicLit:
	case *ast.Ident:
		v.names = append(v.names, rn.Name)
	case ast.Expr:
		tName := typeName(rn)
		if tName == v.tree.name() {
			v.tree.confirm()
			for _, name := range v.names {
				obj := ast.NewObj(ast.Var, name)
				obj.Decl = rn
				obj.Data = v.tree
				v.scope.Insert(obj)
			}
		}
	}
	return nil
}

func (v *bodyVisitor) Visit(n ast.Node) ast.Visitor {
	switch rn := n.(type) {
	default:
		spew.Printf("unexpected type to bodyV: %+#v\n", n)
	case *ast.CommentGroup, *ast.BasicLit:
	case nil:
	case *ast.AssignStmt:
		return &assignVisitor{stmt: rn, tree: v.tree, scope: v.scope}
	case *ast.IfStmt:
		return &ifVisitor{tree: v.tree, scope: ast.NewScope(v.scope)}
	case *ast.ExprStmt:
		return &exprVisitor{tree: v.tree, scope: v.scope}
	}
	return nil
}

func (v *exprVisitor) Visit(n ast.Node) ast.Visitor {
	switch n.(type) {
	default:
		spew.Printf("unexpected type to exprV: %+#v\n", n)
	case *ast.CommentGroup, *ast.BasicLit:
	case nil:
	case *ast.UnaryExpr, *ast.BinaryExpr:
		return &exprVisitor{tree: v.tree, scope: v.scope}
	case *ast.ExprStmt:
		return v
	case *ast.SelectorExpr:
		return &selectorVisitor{tree: v.tree, scope: v.scope}
	}
	return nil
}

var dummyOn = &selectorVisitor{target: nil}

func (v *selectorVisitor) Visit(n ast.Node) ast.Visitor {
	if v.on == nil {
		switch rn := n.(type) {
		default:
			spew.Printf("unexpected type to selectorV: %+#v\n", n)
		case nil:
		//case ast.Expr: //body
		//		return &exprVisitor{tree: v.tree, scope: v.scope}
		case *ast.Ident:
			c := findVar(v.scope, rn.Name)
			if c != nil {
				v.on = &selectorVisitor{target: c}
			} else {
				spew.Printf("selector Couldn't find %q in scope.", rn.Name)
				v.on = dummyOn
			}
		case *ast.SelectorExpr:
			sv := &selectorVisitor{scope: v.scope, tree: v.tree}
			return sv
		}
		return nil
	}

	switch rn := n.(type) {
	default:
		spew.Printf("unexpected type to selectorV: %+#v\n", n)
	case nil:
	case *ast.Ident:
		if stgt, ok := v.on.target.(confTreeNode); ok {
			v.target = stgt.child(rn.Name)
			spew.Printf("Found selector target: %#v\n", v.target)
			if v.target != nil {
				v.target.confirm()
			}
		} else {
			spew.Printf("selector v.on.target was not a confTreeNode: %#v", v.on.target)
		}
	}
	return nil
}

func findVar(s *ast.Scope, name string) confNode {
	if s == nil {
		return nil
	}
	if obj := s.Lookup(name); obj != nil {
		return obj.Data.(confNode)
	}
	return findVar(s.Outer, name)
}

func (v *ifVisitor) Visit(n ast.Node) ast.Visitor {
	switch n.(type) {
	default:
		spew.Printf("unexpected type to ifV: %+#v\n", n)
	case *ast.CommentGroup, *ast.BasicLit:
	case nil:
	case *ast.BlockStmt: //body
		return &bodyVisitor{tree: v.tree, scope: v.scope}
	case *ast.UnaryExpr, *ast.BinaryExpr:
		return &exprVisitor{tree: v.tree, scope: v.scope}
	}
	return nil
}

func (v *assignVisitor) Visit(n ast.Node) ast.Visitor {
	switch rn := n.(type) {
	default:
		if rn.Pos() < v.stmt.TokPos {
			spew.Printf("LHS assignV: %+#v\n", n)
		} else {
			spew.Printf("RHS assignV: %+#v\n", n)
		}
	case *ast.CommentGroup, *ast.BasicLit:
	case nil:
	}
	return nil
}

func ConfirmTree(cn confNode, pkgs packageMap, funcName string) bool {
	for _, pkg := range pkgs {
		ff := &functionFinder{target: funcName, tree: cn, scope: ast.NewScope(nil)}
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				ast.Walk(ff, decl)
			}
		}
		if cn.confirmed() {
			return true
		}
	}

	return cn.confirmed()
}
