package allfields

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/davecgh/go-spew/spew"
)

type (
	functionFinder struct {
		found  bool
		target string
		*parentRef
	}

	parentRef struct {
		_pkgs  packageMap
		_tree  confNode
		_scope *ast.Scope
	}

	visitor interface {
		pkgs() packageMap
		tree() confNode
		scope() *ast.Scope
		pRef() *parentRef
	}

	confirmVisitor struct {
		*parentRef
		name *ast.Ident
		recv *recvVisitor
		args *recvVisitor
	}

	recvVisitor struct {
		*parentRef
		args      []*argVisitor
		matchRoot bool
	}

	argVisitor struct {
		*parentRef
		lastNode  ast.Node
		names     []string
		matchRoot bool
	}

	compLitVisitor struct {
		*parentRef
	}

	kvVisitor struct {
		name *ast.Ident
		*parentRef
	}

	exprVisitor struct {
		*parentRef
	}

	selectorVisitor struct {
		*parentRef

		on         *selectorVisitor
		targetName string
		target     confNode
	}

	bodyVisitor struct {
		*parentRef
		scope *ast.Scope
	}

	bodyDeclVisitor struct {
		*parentRef
	}

	ifVisitor struct {
		*parentRef
		scope *ast.Scope
	}

	rangeVisitor struct {
		*parentRef
		stmt  *ast.RangeStmt
		scope *ast.Scope
	}

	assignVisitor struct {
		*parentRef
		stmt *ast.AssignStmt
		lhs  []asgPath
		rhs  []rhsItem
	}

	callVisitor struct {
		*parentRef
		fun  *selectorVisitor
		args []*selectorVisitor
	}

	asgPath struct {
		name string
	}

	rhsItem struct {
		sel *selectorVisitor
		fun *ast.FuncLit
	}

	missedReport []string
)

func ConfirmTree(cn confNode, pkgs packageMap, funcName string) missedReport {
	for _, pkg := range pkgs {
		ff := &functionFinder{
			target:    funcName,
			parentRef: &parentRef{_pkgs: pkgs, _tree: cn, _scope: ast.NewScope(nil)},
		}
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

func (rep missedReport) Exempt(forgiven []string) []string {
	realBad := []string{}
	for _, miss := range rep {
		keep := true
		for n, forgive := range forgiven {
			if miss == forgive {
				keep = false
				forgiven[n] = forgiven[len(forgiven)-1]
				forgiven = forgiven[:len(forgiven)-1]
				break
			}
		}
		if keep {
			realBad = append(realBad, miss)
		}
	}
	if len(forgiven) > 0 {
		panic(fmt.Errorf("extra strings passed to Exempt: %v", forgiven))
	}
	return realBad
}

func (pr *parentRef) tree() confNode    { return pr._tree }
func (pr *parentRef) pkgs() packageMap  { return pr._pkgs }
func (pr *parentRef) scope() *ast.Scope { return pr._scope }
func (pr *parentRef) pRef() *parentRef  { return pr }

func (pr *parentRef) pRefNewScope() *parentRef {
	return &parentRef{
		_pkgs:  pr.pkgs(),
		_tree:  pr.tree(),
		_scope: ast.NewScope(pr.scope()),
	}
}

func (v *functionFinder) Visit(n ast.Node) ast.Visitor {
	switch rn := n.(type) {
	default:
		fmt.Printf("Not FuncDecl: %T\n", rn)
	case nil:
	case *ast.GenDecl:
	case *ast.FuncDecl:
		if rn.Name.Name == v.target {
			v.found = true
			return &confirmVisitor{parentRef: v.pRefNewScope()}
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
		v.name = rn
	case *ast.FieldList: //Recv
		v.recv = &recvVisitor{parentRef: v.pRef()}
		return v.recv
	case *ast.FuncType: //Type
		v.args = &recvVisitor{parentRef: v.pRef()}
		ast.Walk(v.args, rn.Params)
	case *ast.BlockStmt: //body
		if (v.recv != nil && v.recv.matchRoot) || (v.args != nil && v.args.matchRoot) { // only if the function's receiver or arg is if the type
			return &bodyVisitor{parentRef: v.pRefNewScope()}
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
		av := &argVisitor{names: []string{}, parentRef: v.pRef()}
		v.args = append(v.args, av)
		return av
	}
	return nil
}

func (v *argVisitor) Visit(n ast.Node) ast.Visitor {
	switch n.(type) {
	default:
		if ident, is := v.lastNode.(*ast.Ident); is {
			v.names = append(v.names, ident.Name)
		}
		v.lastNode = n
	case *ast.CommentGroup, *ast.BasicLit:
	case nil:
		switch rn := v.lastNode.(type) {
		default:
			spew.Printf("unexpected type to argV: %+#v\n", v.lastNode)
		case *ast.Ident, *ast.StarExpr:
			tName := typeName(rn)
			if tName == v.tree().name() {
				v.matchRoot = true
				v.tree().confirm()
				for _, name := range v.names {
					obj := ast.NewObj(ast.Var, name)
					obj.Decl = rn
					obj.Data = v.tree()
					v.scope().Insert(obj)
				}
			}
		case ast.Expr:
			spew.Printf("argV Expr: %#v\n", rn)
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
	case *ast.BranchStmt:
	case *ast.AssignStmt:
		return &assignVisitor{stmt: rn, parentRef: v.pRef()}
	case *ast.IfStmt:
		return &ifVisitor{parentRef: v.pRefNewScope()}
	case *ast.RangeStmt:
		return &rangeVisitor{parentRef: v.pRefNewScope(), stmt: rn}
	case *ast.ExprStmt:
		return &exprVisitor{parentRef: v.pRef()}
	case *ast.DeclStmt:
		return &bodyDeclVisitor{parentRef: v.pRef()}
	case *ast.ReturnStmt:
		return &exprVisitor{parentRef: v.pRef()}
	}
	return nil
}

func (v *rangeVisitor) Visit(n ast.Node) ast.Visitor {
	switch rn := n.(type) {
	default:
		spew.Printf("unexpected type to rangeV: %+#v\n", n)
	case nil:
	case *ast.CommentGroup, *ast.BasicLit:
	case *ast.BlockStmt:
		return &bodyVisitor{parentRef: v.pRef()}
	case ast.Expr:
		if rn.Pos() > v.stmt.TokPos {
			return &exprVisitor{parentRef: v.pRef()}
		}
	}
	return nil
}

func (v *bodyDeclVisitor) Visit(n ast.Node) ast.Visitor {
	switch rn := n.(type) {
	default:
		spew.Printf("unexpected type to bodyDeclV: %+#v\n", n)
	case *ast.CommentGroup, *ast.BasicLit:
	case nil:
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
		return &exprVisitor{parentRef: v.pRef()}
	case *ast.ExprStmt:
		return v
	case *ast.StarExpr:
		return v
	case *ast.SelectorExpr:
		return &selectorVisitor{parentRef: v.pRef()}
	case *ast.CallExpr:
		return &callVisitor{parentRef: v.pRef()}
	case *ast.Ident:
		// in these contexts, not interesting
	case *ast.CompositeLit:
		return &compLitVisitor{parentRef: v.pRef()}
	}
	return nil
}

func (v *compLitVisitor) Visit(n ast.Node) ast.Visitor {
	switch n.(type) {
	default:
		spew.Printf("unexpected type to compLitV: %+#v\n", n)
	case *ast.CommentGroup, *ast.BasicLit:
	case nil:
	case *ast.Ident:
		// just the type of the struct/array
	case *ast.KeyValueExpr:
		return &kvVisitor{parentRef: v.pRef()}
	}
	return nil
}

func (v *kvVisitor) Visit(n ast.Node) ast.Visitor {
	switch n.(type) {
	default:
		spew.Printf("unexpected type to kvV: %+#v\n", n)
	case *ast.CommentGroup, *ast.BasicLit:
	case nil:
	case *ast.Ident:
		return nil //don't care about keys, don't care about `key: localVar`
	case *ast.CallExpr:
		return &callVisitor{parentRef: v.pRef()}
	case *ast.SelectorExpr:
		return &selectorVisitor{parentRef: v.pRef()}
	}
	return nil
}

func (v *selectorVisitor) Visit(n ast.Node) ast.Visitor {
	if v.on == nil {
		switch rn := n.(type) {
		default:
			spew.Printf("unexpected type to selectorV: %+#v\n", n)
		case nil:
			spew.Printf("selector 'complete' before getting any part\n")
		//case ast.Expr: //body
		//		return &exprVisitor{tree: v.tree(), scope: v.scope}
		case *ast.Ident:
			c := findVar(v.scope(), rn.Name)
			if c != nil {
				v.on = &selectorVisitor{targetName: rn.Name, target: c}
			} else {
				v.on = &selectorVisitor{targetName: rn.Name, target: c}
			}
		case *ast.SelectorExpr:
			v.on = &selectorVisitor{parentRef: v.pRef()}
			return v.on
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
			if v.target != nil {
				v.target.confirm()
			}
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
	switch rn := n.(type) {
	default:
		spew.Printf("unexpected type to ifV: %+#v\n", n)
	case *ast.CommentGroup, *ast.BasicLit:
	case nil:
	case *ast.Ident:
		// not interesting in this context
	case *ast.BlockStmt: //body
		return &bodyVisitor{parentRef: v.pRef()}
	case *ast.UnaryExpr, *ast.BinaryExpr:
		return &exprVisitor{parentRef: v.pRef()}
	case *ast.IfStmt:
		return &ifVisitor{parentRef: v.pRefNewScope()}
	case *ast.AssignStmt:
		return &assignVisitor{stmt: rn, parentRef: v.pRef()}
	case *ast.CallExpr:
		return &callVisitor{parentRef: v.pRef()}
	}
	return nil
}

func (v *assignVisitor) Visit(n ast.Node) ast.Visitor {
	switch rn := n.(type) {
	default:
		if rn.Pos() < v.stmt.TokPos {
			v.lhs = append(v.lhs, asgPath{})
		} else {
			v.rhs = append(v.rhs, rhsItem{sel: &selectorVisitor{}})
		}
	case *ast.CommentGroup, *ast.BasicLit:
	case nil:
		//spew.Printf("LHS: %+#v\n", v.lhs)
		for n, r := range v.rhs {
			if r.sel != nil && r.sel.target != nil {
				obj := ast.NewObj(ast.Var, v.lhs[n].name)
				obj.Decl = rn
				obj.Data = r.sel.target
				r.sel.target.confirm()
				v.scope().Insert(obj)
			}
		}
	case *ast.FuncLit:
		v.rhs = append(v.rhs, rhsItem{fun: rn})
	case *ast.CallExpr:
		return &callVisitor{parentRef: v.pRef()}
	case *ast.Ident:
		if rn.Pos() < v.stmt.TokPos {
			v.lhs = append(v.lhs, asgPath{name: rn.Name})
		} else {
			v.rhs = append(v.rhs, rhsItem{sel: &selectorVisitor{target: findVar(v.scope(), rn.Name)}})
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
					ConfirmTree(v.fun.on.target, v.pkgs(), v.fun.targetName)
				}

				for _, a := range v.args {
					if a.target != nil {
						ConfirmTree(a.target, v.pkgs(), v.fun.targetName)
					}
				}
			}

		}
	case *ast.Ident:
		if v.fun == nil {
			v.fun = &selectorVisitor{target: findVar(v.scope(), rn.Name)}
		} else {
			as := &selectorVisitor{target: findVar(v.scope(), rn.Name)}
			v.args = append(v.args, as)
		}
	case *ast.SelectorExpr:
		if v.fun == nil {
			as := &selectorVisitor{parentRef: v.pRef()}
			v.fun = as
			return as
		}
		as := &selectorVisitor{parentRef: v.pRef()}
		v.args = append(v.args, as)
		return as
	case *ast.CallExpr:
		return &callVisitor{parentRef: v.pRef()}
	case *ast.UnaryExpr, *ast.BinaryExpr, *ast.StarExpr:
		return &exprVisitor{parentRef: v.pRef()}
	case *ast.CompositeLit:
		return &compLitVisitor{parentRef: v.pRef()}
	}

	return nil
}
