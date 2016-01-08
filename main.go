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
	"strings"
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
		return hd + "[" + strings.Join(args, ",") + "]"
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
