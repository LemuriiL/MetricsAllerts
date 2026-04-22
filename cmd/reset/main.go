package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

const marker = "generate:reset"

type resetStruct struct {
	Name   string
	Fields []resetField
}

type resetField struct {
	Name string
	Type ast.Expr
}

func main() {
	if err := run("."); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(root string) error {
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !entry.IsDir() {
			return nil
		}

		name := entry.Name()
		if name == ".git" || name == "vendor" || name == "testdata" {
			return filepath.SkipDir
		}

		return processDir(path)
	})
}

func processDir(dir string) error {
	files, err := collectGoFiles(dir)
	if err != nil {
		return err
	}

	if len(files) == 0 {
		return nil
	}

	fset := token.NewFileSet()
	parsedFiles := make([]*ast.File, 0, len(files))
	pkgName := ""

	for _, filePath := range files {
		file, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
		if err != nil {
			return err
		}

		if pkgName == "" {
			pkgName = file.Name.Name
		}

		if file.Name.Name != pkgName {
			continue
		}

		parsedFiles = append(parsedFiles, file)
	}

	structs := collectResetStructs(parsedFiles)
	if len(structs) == 0 {
		return nil
	}

	src, err := generateFile(fset, pkgName, structs)
	if err != nil {
		return err
	}

	out := filepath.Join(dir, "reset.gen.go")
	return os.WriteFile(out, src, 0o644)
}

func collectGoFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	files := make([]string, 0)

	for _, entry := range entries {
		name := entry.Name()

		if entry.IsDir() {
			continue
		}

		if !strings.HasSuffix(name, ".go") {
			continue
		}

		if strings.HasSuffix(name, "_test.go") {
			continue
		}

		if name == "reset.gen.go" {
			continue
		}

		files = append(files, filepath.Join(dir, name))
	}

	return files, nil
}

func collectResetStructs(files []*ast.File) []resetStruct {
	var result []resetStruct

	for _, file := range files {
		for _, decl := range file.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.TYPE {
				continue
			}

			for _, spec := range genDecl.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}

				structType, ok := typeSpec.Type.(*ast.StructType)
				if !ok {
					continue
				}

				if !hasMarker(genDecl.Doc) && !hasMarker(typeSpec.Doc) {
					continue
				}

				result = append(result, resetStruct{
					Name:   typeSpec.Name.Name,
					Fields: collectFields(structType),
				})
			}
		}
	}

	return result
}

func collectFields(structType *ast.StructType) []resetField {
	var fields []resetField

	if structType.Fields == nil {
		return fields
	}

	for _, field := range structType.Fields.List {
		for _, name := range field.Names {
			fields = append(fields, resetField{
				Name: name.Name,
				Type: field.Type,
			})
		}
	}

	return fields
}

func hasMarker(group *ast.CommentGroup) bool {
	if group == nil {
		return false
	}

	for _, comment := range group.List {
		text := strings.TrimSpace(strings.TrimPrefix(comment.Text, "//"))
		text = strings.TrimSpace(strings.TrimPrefix(text, "/*"))
		text = strings.TrimSpace(strings.TrimSuffix(text, "*/"))

		if text == marker {
			return true
		}
	}

	return false
}

func generateFile(fset *token.FileSet, pkgName string, structs []resetStruct) ([]byte, error) {
	var out bytes.Buffer

	fmt.Fprintf(&out, "package %s\n\n", pkgName)

	for _, item := range structs {
		fmt.Fprintf(&out, "func (v *%s) Reset() {\n", item.Name)
		fmt.Fprintln(&out, "if v == nil {")
		fmt.Fprintln(&out, "return")
		fmt.Fprintln(&out, "}")

		for _, field := range item.Fields {
			writeResetStatement(&out, fset, "v."+field.Name, field.Type)
		}

		fmt.Fprintln(&out, "}")
		fmt.Fprintln(&out)
	}

	return format.Source(out.Bytes())
}

func writeResetStatement(out *bytes.Buffer, fset *token.FileSet, target string, expr ast.Expr) {
	switch typ := expr.(type) {
	case *ast.ArrayType:
		if typ.Len == nil {
			fmt.Fprintf(out, "%s = %s[:0]\n", target, target)
			return
		}

		writeZeroAssignment(out, fset, target, expr)
	case *ast.MapType:
		fmt.Fprintf(out, "clear(%s)\n", target)
	case *ast.StarExpr:
		writePointerReset(out, fset, target, typ.X)
	default:
		writeResetOrZero(out, fset, target, expr)
	}
}

func writePointerReset(out *bytes.Buffer, fset *token.FileSet, target string, expr ast.Expr) {
	fmt.Fprintf(out, "if %s != nil {\n", target)

	switch typ := expr.(type) {
	case *ast.ArrayType:
		if typ.Len == nil {
			fmt.Fprintf(out, "*%s = (*%s)[:0]\n", target, target)
		} else {
			writeZeroAssignment(out, fset, "*"+target, expr)
		}
	case *ast.MapType:
		fmt.Fprintf(out, "clear(*%s)\n", target)
	case *ast.StarExpr:
		writePointerReset(out, fset, "*"+target, typ.X)
	default:
		writeResetPointerOrZero(out, fset, target, expr)
	}

	fmt.Fprintln(out, "}")
}

func writeResetOrZero(out *bytes.Buffer, fset *token.FileSet, target string, expr ast.Expr) {
	fmt.Fprintf(out, "if resetter, ok := any(&%s).(interface{ Reset() }); ok {\n", target)
	fmt.Fprintln(out, "resetter.Reset()")
	fmt.Fprintln(out, "} else {")
	writeZeroAssignment(out, fset, target, expr)
	fmt.Fprintln(out, "}")
}

func writeResetPointerOrZero(out *bytes.Buffer, fset *token.FileSet, target string, expr ast.Expr) {
	fmt.Fprintf(out, "if resetter, ok := any(%s).(interface{ Reset() }); ok {\n", target)
	fmt.Fprintln(out, "resetter.Reset()")
	fmt.Fprintln(out, "} else {")
	writeZeroAssignment(out, fset, "*"+target, expr)
	fmt.Fprintln(out, "}")
}

func writeZeroAssignment(out *bytes.Buffer, fset *token.FileSet, target string, expr ast.Expr) {
	typeName := exprString(fset, expr)

	fmt.Fprintln(out, "{")
	fmt.Fprintf(out, "var zero %s\n", typeName)
	fmt.Fprintf(out, "%s = zero\n", target)
	fmt.Fprintln(out, "}")
}

func exprString(fset *token.FileSet, expr ast.Expr) string {
	var out bytes.Buffer
	_ = printer.Fprint(&out, fset, expr)

	return out.String()
}
