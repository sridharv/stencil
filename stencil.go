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
	"golang.org/x/tools/go/vcs"
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
func Process(paths []string, format bool) error {
	files, err := process(paths)
	if err != nil {
		return err
	}

	for _, f := range files {
		dir := filepath.Dir(f.path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return errors.WithStack(err)
		}
		if err := ioutil.WriteFile(f.path, f.data, 0644); err != nil {
			return errors.WithStack(err)
		}
	}
	if !format {
		return nil
	}
	return doImports(paths)
}

func loadConfig(paths []string) (*loader.Program, error) {
	dirs := build.Default.SrcDirs()
	getPackage := func(p string) string {
		d := filepath.Dir(p)
		for _, dir := range dirs {
			if strings.HasPrefix(d, dir) {
				return strings.TrimPrefix(d, dir)[1:]
			}
		}
		return ""
	}

	f, m := []string{}, map[string]bool{}

	if len(paths) == 0 {
		f = append(f, ".")
	}
	cfg := makeConfig()
	for _, p := range paths {
		if strings.HasSuffix(p, ".go") {
			pkg := getPackage(p)
			if pkg == "" {
				continue
			}
			cfg.Import(pkg)
			continue
		}
		if !m[p] {
			f, m[p] = append(f, p), true
		}
	}

	if len(f) > 0 {
		if _, err := cfg.FromArgs(f, true); err != nil {
			return nil, errors.WithStack(err)
		}
	}
	p, err := cfg.Load()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return p, nil
}

func process(paths []string) ([]file, error) {
	p, err := loadConfig(paths)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	pkgs := p.InitialPackages()

	var files []file
	filer := func(f file) { files = append(files, f) }
	for _, pkg := range pkgs {
		if err := processPackage(pkg, filer, getVendor(p, pkg)); err != nil {
			return nil, err
		}
	}
	return files, nil
}

func doImports(paths []string) error {
	for _, p := range paths {
		s, err := os.Stat(p)
		if err != nil {
			return errors.WithStack(err)
		}
		if s.IsDir() {
			continue
		}
		b, err := ioutil.ReadFile(p)
		if err != nil {
			return errors.Wrapf(err, "%s", p)
		}
		if b, err = imports.Process(p, b, nil); err != nil {
			return errors.Wrapf(err, "%s", p)
		}
		if err = ioutil.WriteFile(p, b, s.Mode()); err != nil {
			return errors.Wrapf(err, "failed to write %s", p)
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

func getVendor(p *loader.Program, pkg *loader.PackageInfo) string {
	src := p.Fset.File(pkg.Files[0].Pos()).Name()
	base := filepath.Clean(filepath.Dir(src))

	r, err := vcs.RepoRootForImportPath(pkg.Pkg.Path(), false)
	if err != nil {
		return filepath.Join(base, "vendor")
	}
	root := filepath.Clean(r.Root)
	for dir := base; dir != root; dir = filepath.Dir(dir) {
		v := filepath.Join(dir, "vendor")
		if s, err := os.Stat(v); err == nil && s.IsDir() {
			return v
		}
	}
	return filepath.Join(base, "vendor")
}

func processPackage(pkg *loader.PackageInfo, add func(file), vendor string) error {
	pos := posMap{}
	for _, err := range pkg.Errors {
		if terr, ok := err.(types.Error); ok && strings.Contains(terr.Error(), "could not import") {
			pos[terr.Pos] = ""
		}
	}

	for _, f := range pkg.Files {
		ast.Walk(pos, f)
	}

	for _, str := range pos {
		if str == "" {
			continue
		}
		l := len(str)
		if l < 3 || str[0] != '"' || str[l-1] != '"' {
			return errors.Errorf("found invalid import path: %s", str)
		}
		path := str[1 : l-1]
		if err := substituteIfNeeded(vendor, path, add); err != nil {
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
