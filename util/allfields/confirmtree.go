package allfields

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/davecgh/go-spew/spew"
)

type (
	functionFinder struct {
		pkgs   packageMap
		target string
		tree   confNode
		scope  *ast.Scope
	}

	confirmVisitor struct {
		pkgs  packageMap
		tree  confNode
		scope *ast.Scope
		recv  *recvVisitor
		args  *recvVisitor
	}

	recvVisitor struct {
		tree      confNode
		scope     *ast.Scope
		args      []*argVisitor
		matchRoot bool
	}

	argVisitor struct {
		names     []string
		matchRoot bool
		tree      confNode
		scope     *ast.Scope
	}

	exprVisitor struct {
		pkgs  packageMap
		tree  confNode
		scope *ast.Scope
	}

	selectorVisitor struct {
		tree  confNode
		scope *ast.Scope

		on         *selectorVisitor
		targetName string
		target     confNode
	}

	bodyVisitor struct {
		pkgs  packageMap
		tree  confNode
		scope *ast.Scope
	}

	bodyDeclVisitor struct {
		tree  confNode
		scope *ast.Scope
	}

	ifVisitor struct {
		pkgs  packageMap
		tree  confNode
		scope *ast.Scope
	}

	assignVisitor struct {
		pkgs  packageMap
		stmt  *ast.AssignStmt
		tree  confNode
		scope *ast.Scope
		lhs   []asgPath
		rhs   []rhsItem
	}

	asgPath struct {
		name string
	}

	rhsItem struct {
		sel *selectorVisitor
		fun *ast.FuncLit
	}

	callVisitor struct {
		pkgs  packageMap
		fun   *selectorVisitor
		args  []*selectorVisitor
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
			return &confirmVisitor{pkgs: v.pkgs, tree: v.tree, scope: ast.NewScope(v.scope)}
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
		v.recv = &recvVisitor{tree: v.tree, scope: v.scope}
		return v.recv
	case *ast.FuncType: //Type
		v.args = &recvVisitor{tree: v.tree, scope: v.scope}
		ast.Walk(v.args, rn.Params)
	case *ast.BlockStmt: //body
		if (v.recv != nil && v.recv.matchRoot) || (v.args != nil && v.args.matchRoot) { // only if the function's receiver or arg is if the type
			spew.Println("start body")
			spew.Dump(v.scope)
			return &bodyVisitor{pkgs: v.pkgs, tree: v.tree, scope: v.scope}
		}
	}
	return nil
}

func (v *recvVisitor) Visit(n ast.Node) ast.Visitor {
	switch n.(type) {
	default:
		spew.Printf("unexpected type to recvV: %+#v\n", n)
	case nil:
		for _, a := range v.args {
			if a.matchRoot {
				v.matchRoot = true
				break
			}
		}
	case *ast.CommentGroup:
	case *ast.FieldList:
		return v
	case *ast.Field:
		av := &argVisitor{names: []string{}, scope: v.scope, tree: v.tree}
		v.args = append(v.args, av)
		return av
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
	case *ast.StarExpr:
		tName := typeName(rn)
		//spew.Printf("arg type: %v (vs. root type %v)\n", tName, v.tree.name())
		if tName == v.tree.name() {
			v.matchRoot = true
			v.tree.confirm()
			for _, name := range v.names {
				obj := ast.NewObj(ast.Var, name)
				obj.Decl = rn
				obj.Data = v.tree
				v.scope.Insert(obj)
			}
		}
	case ast.Expr:
		spew.Printf("argV Expr: %#v\n", rn)
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
		return &assignVisitor{pkgs: v.pkgs, stmt: rn, tree: v.tree, scope: v.scope}
	case *ast.IfStmt:
		return &ifVisitor{pkgs: v.pkgs, tree: v.tree, scope: ast.NewScope(v.scope)}
	case *ast.ExprStmt:
		return &exprVisitor{pkgs: v.pkgs, tree: v.tree, scope: v.scope}
	case *ast.DeclStmt:
		return &bodyDeclVisitor{tree: v.tree, scope: v.scope}
	case *ast.ReturnStmt:
		return &exprVisitor{pkgs: v.pkgs, tree: v.tree, scope: v.scope}
	}
	return nil
}

func (v *bodyDeclVisitor) Visit(n ast.Node) ast.Visitor {
	switch rn := n.(type) {
	default:
		spew.Printf("unexpected type to bodyDeclV: %+#v\n", n)
	case *ast.CommentGroup, *ast.BasicLit:
	case nil:
		spew.Println("end body")
	case *ast.GenDecl:
		switch rn.Tok {
		default:
			spew.Printf("unexpected Tok: %+#v\n", n)
		case token.VAR:
			//spew.Printf("Variable declaration.\n")
		}

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
		return &exprVisitor{pkgs: v.pkgs, tree: v.tree, scope: v.scope}
	case *ast.ExprStmt:
		return v
	case *ast.SelectorExpr:
		return &selectorVisitor{tree: v.tree, scope: v.scope}
	case *ast.CallExpr:
		return &callVisitor{pkgs: v.pkgs, tree: v.tree, scope: v.scope}
	case *ast.Ident:
		// in these contexts, not interesting
	}
	return nil
}

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
				v.on = &selectorVisitor{targetName: rn.Name, target: c}
			} else {
				spew.Printf("selector Couldn't find %q in scope.\n", rn.Name)
				v.on = &selectorVisitor{targetName: rn.Name, target: c}
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
		//spew.Printf("selector complete: %#v\n", v)
	case *ast.Ident:
		v.targetName = rn.Name
		if stgt, ok := v.on.target.(confTreeNode); ok {
			v.target = stgt.child(rn.Name)
			//spew.Printf("Found selector target %q: %#v\n", rn.Name, v.target)
			if v.target != nil {
				v.target.confirm()
			}
		} else {
			spew.Printf("selector v.on.target was not a confTreeNode: %#v\n", v.on.target)
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
		return &bodyVisitor{pkgs: v.pkgs, tree: v.tree, scope: v.scope}
	case *ast.UnaryExpr, *ast.BinaryExpr:
		return &exprVisitor{pkgs: v.pkgs, tree: v.tree, scope: v.scope}
	}
	return nil
}

func (v *assignVisitor) Visit(n ast.Node) ast.Visitor {
	switch rn := n.(type) {
	default:
		if rn.Pos() < v.stmt.TokPos {
			spew.Printf("LHS assignV: %+#v\n", n)
			v.lhs = append(v.lhs, asgPath{})
		} else {
			spew.Printf("RHS assignV: %+#v\n", n)
			v.rhs = append(v.rhs, rhsItem{sel: &selectorVisitor{}})
		}
	case *ast.CommentGroup, *ast.BasicLit:
	case nil:
		//spew.Printf("LHS: %+#v\n", v.lhs)
		for n, r := range v.rhs {
			if r.sel != nil {
				obj := ast.NewObj(ast.Var, v.lhs[n].name)
				obj.Decl = rn
				obj.Data = r.sel.target
				if obj.Data != nil {
					r.sel.target.confirm()
				}
				v.scope.Insert(obj)
			}
		}
	case *ast.FuncLit:
		v.rhs = append(v.rhs, rhsItem{fun: rn})
	case *ast.CallExpr:
		return &callVisitor{pkgs: v.pkgs, tree: v.tree, scope: v.scope}
	case *ast.Ident:
		if rn.Pos() < v.stmt.TokPos {
			v.lhs = append(v.lhs, asgPath{name: rn.Name})
		} else {
			v.rhs = append(v.rhs, rhsItem{sel: &selectorVisitor{target: findVar(v.scope, rn.Name)}})
		}
	}
	return nil
}

func (v *callVisitor) Visit(n ast.Node) ast.Visitor {
	switch rn := n.(type) {
	default:
		spew.Printf("unexpected type to callV: %+#v\n", n)
	case *ast.CommentGroup, *ast.BasicLit:
	case nil:
		if v.fun == nil {
			spew.Printf("callV complete without understanding a Fun\n")
		} else {
			if v.fun.on != nil {
				if v.fun.on.target != nil {
					ConfirmTree(v.fun.on.target, v.pkgs, v.fun.targetName)
				}

				for _, a := range v.args {
					if a.target != nil {
						ConfirmTree(a.target, v.pkgs, v.fun.targetName)
					}
				}
			}

		}
	case *ast.Ident:
		if v.fun == nil {
			spew.Printf("Call of %q\n", rn.Name)
			v.fun = &selectorVisitor{target: findVar(v.scope, rn.Name)}
		} else {
			as := &selectorVisitor{target: findVar(v.scope, rn.Name)}
			v.args = append(v.args, as)
		}
	case *ast.SelectorExpr:
		if v.fun == nil {
			spew.Printf("Call of %v\n", rn)
			as := &selectorVisitor{tree: v.tree, scope: v.scope}
			v.fun = as
			return as
		}
		as := &selectorVisitor{tree: v.tree, scope: v.scope}
		v.args = append(v.args, as)
		return as
	case *ast.CallExpr:
		return &callVisitor{pkgs: v.pkgs, tree: v.tree, scope: v.scope}
	case *ast.UnaryExpr, *ast.BinaryExpr:
		return &exprVisitor{pkgs: v.pkgs, tree: v.tree, scope: v.scope}
	}

	return nil
}

func ConfirmTree(cn confNode, pkgs packageMap, funcName string) []string {
	spew.Printf("Confirming %q touches fields of %q\n", funcName, cn.name())
	for _, pkg := range pkgs {
		ff := &functionFinder{pkgs: pkgs, target: funcName, tree: cn, scope: ast.NewScope(nil)}
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				ast.Walk(ff, decl)
			}
		}
		if cn.confirmed() {
			return []string{}
		}
	}

	missed := []string{}
	tName := cn.name()
	for _, f := range cn.missed() {
		missed = append(missed, tName+f)
	}
	return missed

}
