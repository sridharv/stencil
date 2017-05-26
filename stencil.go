package stencil

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"

	"bytes"
	"go/printer"

	"path/filepath"

	"os"

	"io/ioutil"

	"go/build"

	"github.com/pkg/errors"
	"golang.org/x/tools/go/loader"
	"golang.org/x/tools/imports"
)

type posMap map[token.Pos]string

func (m posMap) Visit(node ast.Node) (w ast.Visitor) {
	if node == nil {
		return nil
	}
	if _, ok := m[node.Pos()]; !ok {
		return m
	}
	lit, ok := node.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return m
	}
	m[node.Pos()] = lit.Value
	return m
}

type file struct {
	data []byte
	path string
}

func makeConfig() *loader.Config {
	return &loader.Config{
		ParserMode:  parser.AllErrors | parser.ParseComments,
		Fset:        token.NewFileSet(),
		AllowErrors: true,
		TypeChecker: types.Config{
			IgnoreFuncBodies: true,
			Error:            func(err error) {},
		},
	}
}

// Process process paths, generating vendored, specialized code for any stencil import paths
func Process(paths []string) error {
	return process(paths, os.MkdirAll, ioutil.WriteFile)
}

func listPackages(paths []string) (map[string]map[string]struct{}, error) {
	dirs := build.Default.SrcDirs()

	pkgs := map[string]map[string]struct{}{}
	for _, path := range paths {
		var trimmed string
		for _, dir := range dirs {
			if strings.HasPrefix(path, dir) {
				trimmed = strings.TrimPrefix(path, dir)
				break
			}
		}
		var name string
		if strings.HasSuffix(trimmed, ".go") {
			name = filepath.Base(name)
			trimmed = filepath.Dir(trimmed)
		}
		if trimmed == "" {
			return nil, errors.Errorf("%s: not in GOPATH", path)
		}
		pkg := trimmed[1:]
		existing := pkgs[pkg]
		switch {
		case name == "" || name == ".":
			pkgs[pkg] = map[string]struct{}{}
		case len(existing) == 0 && existing != nil:
		case existing != nil:
			existing[name] = struct{}{}
		default:
			pkgs[pkg] = map[string]struct{}{name: {}}
		}
	}
	return pkgs, nil
}

func process(paths []string, mkdir func(string, os.FileMode) error, write func(string, []byte, os.FileMode) error) error {
	pkgMap, err := listPackages(paths)
	if err != nil {
		return err
	}
	cfg := makeConfig()
	for pkg, _ := range pkgMap {
		cfg.Import(pkg)
	}
	p, err := cfg.Load()
	if err != nil {
		return errors.WithStack(err)
	}
	pkgs := p.InitialPackages()

	var files []file
	filer := func(f file) { files = append(files, f) }
	for _, pkg := range pkgs {
		if err := processPackage(p, pkg, filer, pkgMap[pkg.Pkg.Path()]); err != nil {
			return err
		}
	}

	for _, f := range files {
		dir := filepath.Dir(f.path)
		if err := mkdir(dir, 0755); err != nil {
			return errors.WithStack(err)
		}
		if err := write(f.path, f.data, 0644); err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

type replacer map[string]string

func (r replacer) Visit(node ast.Node) (w ast.Visitor) {
	if node == nil {
		return nil
	}
	switch t := node.(type) {
	case *ast.Ident:
		if s, ok := r[t.Name]; ok {
			t.Name = s
		}

	}
	return r
}

func (r replacer) filterTypeDecls(name string) bool {
	_, present := r[name]
	return !present
}

func processPackage(p *loader.Program, pkg *loader.PackageInfo, add func(file), files map[string]struct{}) error {
	pos := posMap{}
	for _, err := range pkg.Errors {
		if terr, ok := err.(types.Error); ok && strings.Contains(terr.Error(), "could not import") {
			pos[terr.Pos] = ""
		}
	}

	for _, f := range pkg.Files {
		if _, ok := files[f.Name.Name+".go"]; ok || len(files) == 0 {
			ast.Walk(pos, f)
		}
	}

	src := p.Fset.File(pkg.Files[0].Pos()).Name()
	base := filepath.Join(filepath.Dir(src), "vendor")
	for _, str := range pos {
		if str == "" {
			continue
		}
		l := len(str)
		if l < 3 || str[0] != '"' || str[l-1] != '"' {
			return errors.Errorf("found invalid import path: %s", str)
		}
		path := str[1 : l-1]
		if err := substituteIfNeeded(base, path, add); err != nil {
			return err
		}
	}
	return nil
}

func substituteIfNeeded(base, path string, add func(file)) error {
	parts := strings.Split(path, "/")
	// See if we can form a substitution pattern from the parts here
	r := replacer{}
	var origin *loader.Program

	for origin == nil && len(parts) > 2 {
		l := len(parts)
		// A path looks like github.com/foo/bar/Parameter/Specialization
		parameter, specialization := parts[l-2], parts[l-1]
		if _, ok := r[parameter]; ok {
			return errors.Errorf("%s specialized twice", parameter)
		}
		parts = parts[:l-2]
		cfg := makeConfig()
		cfg.Import(strings.Join(parts, "/"))
		var err error
		if origin, err = cfg.Load(); err != nil {
			return errors.WithStack(err)
		}
		r[parameter] = specialization
	}

	if origin == nil {
		return errors.Errorf("%s: failed to find stencil", path)
	}
	var pcfg printer.Config

	root := filepath.Join(append([]string{base}, strings.Split(path, "/")...)...)
	for _, f := range origin.InitialPackages()[0].Files {
		target := filepath.Join(root, f.Name.Name+".go")
		// Remove type declaration for types being replaced
		if !ast.FilterFile(f, r.filterTypeDecls) {
			continue
		}
		// Rename types
		ast.Walk(r, f)
		var b bytes.Buffer
		if err := pcfg.Fprint(&b, origin.Fset, f); err != nil {
			return errors.Errorf("%s: code generation failed", f.Name)
		}
		out, err := imports.Process(target, b.Bytes(), nil)
		if err != nil {
			return errors.WithStack(err)
		}
		add(file{path: target, data: out})
	}

	return nil
}
