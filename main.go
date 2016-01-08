package main

import (
	_ "github.com/k0kubun/pp"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	//_ "llvm.org/llvm/bindings/go/llvm"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/k0kubun/pp"
	"strconv"
	"strings"
	"sync"
)

type MExpr interface {
	Head() MExpr
	Length() int
	String() string
}

type MExprBase struct {
	Position token.Pos
}

type MExprComment struct {
	MExprBase
	Value string
}

type MExprString struct {
	MExprBase
	Value string
}

type MExprInteger struct {
	MExprBase
	Value int
}

type MExprReal struct {
	MExprBase
	Value float64
}

type MExprSymbol struct {
	MExprBase
	Context string
	Name    string
}

type MExprNormal struct {
	MExprBase
	Hd        MExpr
	Arguments []MExpr
}
type Generator struct {
	Program chan MExpr
}

func (this *MExprNormal) Head() MExpr {
	return this.Hd
}
func (this *MExprNormal) Length() int {
	return len(this.Arguments)
}
func (this *MExprNormal) String() string {
	args := make([]string, len(this.Arguments))
	for ii, elem := range this.Arguments {
		args[ii] = elem.String()
	}
	hd := this.Hd.String()
	if hd == "CompoundExpression" {
		return strings.Join(args, ";\n")
	} else {
		return hd + "[" + strings.Join(args, ", ") + "]"
	}
}

func (*MExprSymbol) Head() MExpr {
	return &MExprSymbol{
		Context: "System",
		Name:    "Symbol",
	}
}
func (*MExprSymbol) Length() int {
	return 0
}
func (this *MExprSymbol) String() string {
	if this.Context == "System" {
		return this.Name
	}
	return this.Context + "`" + this.Name
}
func (*MExprString) Head() MExpr {
	return &MExprSymbol{
		Context: "System",
		Name:    "String",
	}
}
func (*MExprString) Length() int {
	return 0
}
func (this *MExprString) String() string {
	return "\"" + this.Value + "\""
}

func (*MExprInteger) Head() MExpr {
	return &MExprSymbol{
		Context: "System",
		Name:    "Integer",
	}
}
func (*MExprInteger) Length() int {
	return 0
}
func (this *MExprInteger) String() string {
	return strconv.Itoa(this.Value)
}

func (*MExprReal) Head() MExpr {
	return &MExprSymbol{
		Context: "System",
		Name:    "Real",
	}
}
func (*MExprReal) Length() int {
	return 0
}
func (this *MExprReal) String() string {
	return fmt.Sprint(this.Value)
}

func walkIdentList(v ast.Visitor, list []*ast.Ident) {
	for _, x := range list {
		ast.Walk(v, x)
	}
}

func walkExprList(v ast.Visitor, list []ast.Expr) {
	for _, x := range list {
		ast.Walk(v, x)
	}
}

func walkStmtList(v ast.Visitor, list []ast.Stmt) {
	for _, x := range list {
		ast.Walk(v, x)
	}
}

func walkDeclList(v ast.Visitor, list []ast.Decl) {
	for _, x := range list {
		ast.Walk(v, x)
	}
}

func (this *Generator) Visit(anode ast.Node) (w ast.Visitor) {
	switch node := anode.(type) {
	//case *ast.Comment:
	//	pp.Println(node.Text)
	//case *ast.FuncDecl:
	//	fmt.Println(node.Name.Name)
	//case *ast.CommentGroup:
	//	pp.Println(node.Text())
	//case *ast.Package:
	//	pp.Println(node)
	case *ast.DeclStmt:
		this.Visit(node.Decl)
	case *ast.SelectorExpr:
		gen := &Generator{
			Program: make(chan MExpr),
		}
		defer close(gen.Program)
		go func() {
			gen.Visit(node.X)
			gen.Visit(node.Sel)
		}()
		x := <-gen.Program
		sel := <-gen.Program
		if x.String() == "C" {
			this.Program <- &MExprNormal{
				MExprBase: MExprBase{
					Position: node.Pos(),
				},
				Hd: &MExprSymbol{
					MExprBase: MExprBase{
						Position: node.Pos(),
					},
					Context: "Rasta",
					Name:    "C",
				},
				Arguments: []MExpr{sel},
			}
		} else {
			this.Program <- &MExprNormal{
				MExprBase: MExprBase{
					Position: node.Pos(),
				},
				Hd: &MExprSymbol{
					MExprBase: MExprBase{
						Position: node.Pos(),
					},
					Context: "Rasta",
					Name:    "GetField",
				},
				Arguments: []MExpr{x, sel},
			}
		}
	case *ast.Ident:
		this.Program <- &MExprSymbol{
			MExprBase: MExprBase{
				Position: node.Pos(),
			},
			Context: "System",
			Name:    node.Name,
		}
	case *ast.StarExpr:
		gen := &Generator{
			Program: make(chan MExpr),
		}
		defer close(gen.Program)
		go func() {
			gen.Visit(node.X)
		}()
		this.Program <- &MExprNormal{
			MExprBase: MExprBase{
				Position: node.Pos(),
			},
			Hd: &MExprSymbol{
				MExprBase: MExprBase{
					Position: node.Pos(),
				},
				Context: "Rasta",
				Name:    "Reference",
			},
			Arguments: []MExpr{
				<-gen.Program,
			},
		}
	case *ast.TypeSpec:
		gen := &Generator{
			Program: make(chan MExpr),
		}
		defer close(gen.Program)
		go func() {
			gen.Visit(node.Name)
			gen.Visit(node.Type)
		}()

		this.Program <- &MExprNormal{
			MExprBase: MExprBase{
				Position: node.Pos(),
			},
			Hd: &MExprSymbol{
				MExprBase: MExprBase{
					Position: node.Pos(),
				},
				Context: "Rasta",
				Name:    "Type",
			},
			Arguments: []MExpr{
				<-gen.Program,
				<-gen.Program,
			},
		}
	case *ast.BlockStmt:
		stmts := []MExpr{}
		gen := &Generator{
			Program: make(chan MExpr),
		}
		defer close(gen.Program)
		go func() {
			for _, stmt := range node.List {
				gen.Visit(stmt)
			}
		}()
		for range node.List {
			stmts = append(stmts, <-gen.Program)
		}
		this.Program <- &MExprNormal{
			MExprBase: MExprBase{
				Position: node.Pos(),
			},
			Hd: &MExprSymbol{
				MExprBase: MExprBase{
					Position: node.Pos(),
				},
				Context: "System",
				Name:    "CompoundExpression",
			},
			Arguments: stmts,
		}
	case *ast.FuncType:

		this.Program <- &MExprNormal{
			MExprBase: MExprBase{
				Position: node.Pos(),
			},
			Hd: &MExprSymbol{
				MExprBase: MExprBase{
					Position: node.Pos(),
				},
				Context: "Rasta",
				Name:    "List",
			},
			Arguments: []MExpr{},
		}
	case *ast.FuncDecl:

		gen := &Generator{
			Program: make(chan MExpr),
		}
		defer close(gen.Program)
		go func() {
			gen.Visit(node.Name)
			gen.Visit(node.Type)
			gen.Visit(node.Body)
		}()
		this.Program <- &MExprNormal{
			MExprBase: MExprBase{
				Position: node.Pos(),
			},
			Hd: &MExprSymbol{
				MExprBase: MExprBase{
					Position: node.Pos(),
				},
				Context: "Rasta",
				Name:    "Function",
			},
			Arguments: []MExpr{
				<-gen.Program,
				<-gen.Program,
				<-gen.Program,
			},
		}
	case *ast.ValueSpec:

		gen := &Generator{
			Program: make(chan MExpr),
		}
		defer close(gen.Program)
		go func() {
			if len(node.Names) > 1 {
				panic("Unexpected number of indentifiers for value spec")
			}
			for _, ident := range node.Names {
				gen.Visit(ident)
			}
			gen.Visit(node.Type)
		}()
		args := []MExpr{
			<-gen.Program,
			<-gen.Program,
		}
		this.Program <- &MExprNormal{
			MExprBase: MExprBase{
				Position: node.Pos(),
			},
			Hd: &MExprSymbol{
				MExprBase: MExprBase{
					Position: node.Pos(),
				},
				Context: "Rasta",
				Name:    "Value",
			},
			Arguments: args,
		}
	case *ast.ImportSpec:
		var nm MExpr
		if node.Name == nil && node.Path == nil {
			nm = &MExprString{
				MExprBase: MExprBase{
					Position: node.Pos(),
				},
				Value: "Empty",
			}
		} else if node.Path != nil {
			nm = &MExprString{
				MExprBase: MExprBase{
					Position: node.Path.ValuePos,
				},
				Value: strings.Trim(node.Path.Value, "\""),
			}
		} else {
			nm = &MExprString{
				MExprBase: MExprBase{
					Position: node.Pos(),
				},
				Value: node.Name.Name,
			}
		}
		this.Program <- &MExprNormal{
			MExprBase: MExprBase{
				Position: node.Pos(),
			},
			Hd: &MExprSymbol{
				MExprBase: MExprBase{
					Position: node.Pos(),
				},
				Context: "Rasta",
				Name:    "Import",
			},
			Arguments: []MExpr{
				nm,
			},
		}
	case *ast.GenDecl:

		gen := &Generator{
			Program: make(chan MExpr),
		}
		defer close(gen.Program)
		go func() {
			for _, spec := range node.Specs {
				gen.Visit(spec)
			}
		}()
		for range node.Specs {

			expr := <-gen.Program
			name := "Declare"
			if node.Tok == token.CONST {
				name = "DeclareConstant"
			} else if node.Tok == token.TYPE {
				name = "DeclareType"
			}
			if node.Tok == token.IMPORT {
				this.Program <- expr
			} else {
				this.Program <- &MExprNormal{
					MExprBase: MExprBase{
						Position: node.Pos(),
					},
					Hd: &MExprSymbol{
						MExprBase: MExprBase{
							Position: node.Pos(),
						},
						Context: "Rasta",
						Name:    name,
					},
					Arguments: []MExpr{
						expr,
					},
				}
			}
		}
	case *ast.File:
		this.Program <- &MExprNormal{
			MExprBase: MExprBase{
				Position: node.Pos(),
			},
			Hd: &MExprSymbol{
				MExprBase: MExprBase{
					Position: node.Pos(),
				},
				Context: "System",
				Name:    "BeginPackage",
			},
			Arguments: []MExpr{
				&MExprString{
					MExprBase: MExprBase{
						Position: node.Pos(),
					},
					Value: node.Name.Name,
				},
			},
		}
		for _, decl := range node.Decls {
			this.Visit(decl)
		}
		this.Program <- &MExprNormal{
			MExprBase: MExprBase{
				Position: node.Pos(),
			},
			Hd: &MExprSymbol{
				MExprBase: MExprBase{
					Position: node.Pos(),
				},
				Context: "System",
				Name:    "EndPackage",
			},
			Arguments: []MExpr{},
		}
		return nil
	case *ast.DeferStmt:
		gen := &Generator{
			Program: make(chan MExpr),
		}
		defer close(gen.Program)
		go func() {
			gen.Visit(node.Call)
		}()
		args := []MExpr{
			<-gen.Program,
		}
		this.Program <- &MExprNormal{
			MExprBase: MExprBase{
				Position: node.Pos(),
			},
			Hd: &MExprSymbol{
				MExprBase: MExprBase{
					Position: node.Pos(),
				},
				Context: "Rasta",
				Name:    "Defer",
			},
			Arguments: args,
		}
	case *ast.CallExpr:
		gen := &Generator{
			Program: make(chan MExpr),
		}
		defer close(gen.Program)
		go func() {
			gen.Visit(node.Fun)
			for _, arg := range node.Args {
				gen.Visit(arg)
			}
		}()
		name := <-gen.Program
		args := []MExpr{}
		for range node.Args {
			args = append(args, <-gen.Program)
		}
		this.Program <- &MExprNormal{
			MExprBase: MExprBase{
				Position: node.Pos(),
			},
			Hd:        name,
			Arguments: args,
		}
	case *ast.AssignStmt:
		genLhs := &Generator{
			Program: make(chan MExpr),
		}
		genRhs := &Generator{
			Program: make(chan MExpr),
		}
		defer close(genLhs.Program)
		defer close(genRhs.Program)
		go func() {
			for _, nd := range node.Lhs {
				genLhs.Visit(nd)
			}
		}()
		go func() {
			for _, nd := range node.Rhs {
				genRhs.Visit(nd)
			}
		}()
		lhs := []MExpr{}
		rhs := []MExpr{}
		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			for range node.Lhs {
				lhs = append(lhs, <-genLhs.Program)
			}
		}()
		go func() {
			defer wg.Done()
			for range node.Rhs {
				rhs = append(rhs, <-genRhs.Program)
			}
		}()
		wg.Wait()
		if len(lhs) == 1 {
			this.Program <- &MExprNormal{
				MExprBase: MExprBase{
					Position: node.Pos(),
				},
				Hd: &MExprSymbol{
					MExprBase: MExprBase{
						Position: node.Pos(),
					},
					Context: "Rasta",
					Name:    "Set",
				},
				Arguments: append(lhs, rhs...),
			}
		} else {
			this.Program <- &MExprNormal{
				MExprBase: MExprBase{
					Position: node.Pos(),
				},
				Hd: &MExprSymbol{
					MExprBase: MExprBase{
						Position: node.Pos(),
					},
					Context: "Rasta",
					Name:    "Set",
				},
				Arguments: []MExpr{
					&MExprNormal{
						Hd: &MExprSymbol{
							MExprBase: MExprBase{
								Position: node.Pos(),
							},
							Context: "System",
							Name:    "List",
						},
						Arguments: lhs,
					},
					&MExprNormal{
						Hd: &MExprSymbol{
							MExprBase: MExprBase{
								Position: node.Pos(),
							},
							Context: "System",
							Name:    "List",
						},
						Arguments: rhs,
					},
				},
			}
		}
	case *ast.BinaryExpr:
		gen := &Generator{
			Program: make(chan MExpr),
		}
		defer close(gen.Program)
		go func() {
			gen.Program <- &MExprString{
				MExprBase: MExprBase{
					Position: node.OpPos,
				},
				Value: node.Op.String(),
			}
			gen.Visit(node.X)
			gen.Visit(node.Y)
		}()

		args := []MExpr{
			<-gen.Program,
			<-gen.Program,
			<-gen.Program,
		}
		this.Program <- &MExprNormal{
			MExprBase: MExprBase{
				Position: node.Pos(),
			},
			Hd: &MExprSymbol{
				MExprBase: MExprBase{
					Position: node.Pos(),
				},
				Context: "Rasta",
				Name:    "BinaryExpr",
			},
			Arguments: args,
		}
	case *ast.UnaryExpr:
		gen := &Generator{
			Program: make(chan MExpr),
		}
		defer close(gen.Program)
		go func() {
			gen.Visit(node.X)
		}()
		args := []MExpr{
			&MExprString{
				MExprBase: MExprBase{
					Position: node.OpPos,
				},
				Value: node.Op.String(),
			},
			<-gen.Program,
		}
		this.Program <- &MExprNormal{
			MExprBase: MExprBase{
				Position: node.Pos(),
			},
			Hd: &MExprSymbol{
				MExprBase: MExprBase{
					Position: node.Pos(),
				},
				Context: "Rasta",
				Name:    "UnaryOperation",
			},
			Arguments: args,
		}
	case *ast.IfStmt:
		gen := &Generator{
			Program: make(chan MExpr),
		}
		defer close(gen.Program)
		go func() {
			gen.Visit(node.Cond)
			gen.Visit(node.Body)
			if node.Else != nil {
				gen.Visit(node.Else)
			} else {
				gen.Program <- &MExprSymbol{
					MExprBase: MExprBase{
						Position: node.Pos(),
					},
					Context: "System",
					Name:    "Null",
				}
			}
		}()

		args := []MExpr{
			<-gen.Program,
			<-gen.Program,
			<-gen.Program,
		}
		this.Program <- &MExprNormal{
			MExprBase: MExprBase{
				Position: node.Pos(),
			},
			Hd: &MExprSymbol{
				MExprBase: MExprBase{
					Position: node.Pos(),
				},
				Context: "System",
				Name:    "If",
			},
			Arguments: args,
		}
	case *ast.ExprStmt:
		this.Program <- &MExprSymbol{
			MExprBase: MExprBase{
				Position: node.Pos(),
			},
			Context: "Rasta",
			Name:    "ExprStmt",
		}
	case *ast.ReturnStmt:
		var args []MExpr
		gen := &Generator{
			Program: make(chan MExpr),
		}
		defer close(gen.Program)
		go func() {
			for _, res := range node.Results {
				gen.Visit(res)
			}
		}()
		if len(node.Results) == 1 {
			args = []MExpr{
				<-gen.Program,
			}
		} else {
			for range node.Results {
				args = append(args, <-gen.Program)
			}
			args = []MExpr{
				&MExprNormal{
					MExprBase: MExprBase{
						Position: node.Pos(),
					},
					Hd: &MExprSymbol{
						MExprBase: MExprBase{
							Position: node.Pos(),
						},
						Context: "System",
						Name:    "List",
					},
					Arguments: args,
				},
			}
		}
		this.Program <- &MExprNormal{
			MExprBase: MExprBase{
				Position: node.Pos(),
			},
			Hd: &MExprSymbol{
				MExprBase: MExprBase{
					Position: node.Pos(),
				},
				Context: "Rasta",
				Name:    "BinaryExpr",
			},
			Arguments: args,
		}
	case *ast.BasicLit:
		if node.Kind == token.INT {
			ii, err := strconv.Atoi(node.Value)
			if err != nil {
				panic(spew.Sdump("Cannot parse integer value ", node.Value))
			}
			this.Program <- &MExprInteger{
				MExprBase: MExprBase{
					Position: node.Pos(),
				},
				Value: ii,
			}
		} else {
			this.Program <- &MExprString{
				MExprBase: MExprBase{
					Position: node.Pos(),
				},
				Value: "Unhandeled BasicLit",
			}
		}
	case *ast.CompositeLit:
		this.Program <- &MExprSymbol{
			MExprBase: MExprBase{
				Position: node.Pos(),
			},
			Context: "Rasta",
			Name:    "CompositeLit",
		}
	default:
		pp.Println(node)
		panic(node)

	}
	//pp.Println(node)
	return this
}

const code = `
package main

import "C"
import "errors"

type VerifierFailureAction C.LLVMVerifierFailureAction

const (
	// verifier will print to stderr and abort()
	AbortProcessAction VerifierFailureAction = C.LLVMAbortProcessAction
	AbortProcessAction VerifierFailureAction = C.LLVMAbortProcessAction
	)
func ParseBitcodeFile(name string) (Module, error) {
	var buf C.LLVMMemoryBufferRef
	var errmsg *C.char
	var cfilename *C.char = C.CString(name)
	defer C.free(unsafe.Pointer(cfilename))
	result := C.LLVMCreateMemoryBufferWithContentsOfFile(cfilename, &buf, &errmsg)
	if result != 0 {
		err := errors.New(C.GoString(errmsg))
		C.free(unsafe.Pointer(errmsg))
		return Module{}, err
	}

	defer C.LLVMDisposeMemoryBuffer(buf)

	var m Module
	if C.LLVMParseBitcode2(buf, &m.C) == 0 {
		return m, nil
	}

	err := errors.New(C.GoString(errmsg))
	C.free(unsafe.Pointer(errmsg))
	return Module{}, err
}
`

func main() {

	const pth = `/Users/abduld/Code/go/src/llvm.org/llvm/bindings/go/llvm/analysis.go`
	_, err := ioutil.ReadFile(pth)
	if err != nil {
		panic(err)
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, pth, string(code), 0)
	if err != nil {
		panic(err)
	}
	done := make(chan bool, 1)
	gen := &Generator{
		Program: make(chan MExpr),
	}

	go func() {
		gen.Visit(f)
		done <- true

	}()
	if false {
		spew.Dump("dummy")
		pp.Println("dummy")
	}
	for {
		select {
		case mexpr := <-gen.Program:
			fmt.Println(mexpr)
		case <-done:
			//fmt.Println("Done")
			return
		}
	}

}
