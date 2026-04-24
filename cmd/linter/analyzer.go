package main

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

const analyzerName = "nofatalpanic"

type forbiddenCall struct {
	pkg  string
	name string
	msg  string
}

var forbiddenCalls = []forbiddenCall{
	{"log", "Fatal", "do not use log.Fatal outside main function of main package"},
	{"log", "Fatalf", "do not use log.Fatalf outside main function of main package"},
	{"log", "Fatalln", "do not use log.Fatalln outside main function of main package"},
	{"os", "Exit", "do not use os.Exit outside main function of main package"},
}

var noFatalPanicAnalyzer = &analysis.Analyzer{
	Name: analyzerName,
	Doc:  "checks panic, log.Fatal and os.Exit usage",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}

			inspectFunc(pass, fn)
		}
	}

	return nil, nil
}

func inspectFunc(pass *analysis.Pass, fn *ast.FuncDecl) {
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}

		checkCall(pass, call, fn)
		return true
	})
}

func checkCall(pass *analysis.Pass, call *ast.CallExpr, fn *ast.FuncDecl) {
	if isPanicCall(pass, call) {
		pass.Reportf(call.Pos(), "do not use panic")
		return
	}

	if isAllowedFatalOrExit(pass, fn) {
		return
	}

	for _, item := range forbiddenCalls {
		if isPackageFunctionCall(pass, call, item.pkg, item.name) {
			pass.Reportf(call.Pos(), "%s", item.msg)
			return
		}
	}
}

func isPanicCall(pass *analysis.Pass, call *ast.CallExpr) bool {
	ident, ok := call.Fun.(*ast.Ident)
	if !ok || ident.Name != "panic" {
		return false
	}

	obj := pass.TypesInfo.Uses[ident]
	builtin, ok := obj.(*types.Builtin)

	return ok && builtin.Name() == "panic"
}

func isPackageFunctionCall(pass *analysis.Pass, call *ast.CallExpr, packagePath string, functionName string) bool {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != functionName {
		return false
	}

	ident, ok := selector.X.(*ast.Ident)
	if !ok {
		return false
	}

	obj := pass.TypesInfo.Uses[ident]
	pkgName, ok := obj.(*types.PkgName)
	if !ok {
		return false
	}

	return pkgName.Imported().Path() == packagePath
}

func isAllowedFatalOrExit(pass *analysis.Pass, fn *ast.FuncDecl) bool {
	if pass.Pkg == nil || pass.Pkg.Name() != "main" {
		return false
	}

	return fn.Name != nil && fn.Name.Name == "main"
}
