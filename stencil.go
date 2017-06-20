// Package stencil generates specialized versions of Go packages by replacing types.
package stencil

import (
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"strings"

	"bytes"

	"path/filepath"

	"os"

	"io/ioutil"

	"go/build"

	"josharian/apply"

	"github.com/pkg/errors"
	"golang.org/x/tools/imports"
)

type file struct {
	data []byte
	path string
}

// Process process paths, generating vendored, specialized code for any stencil import paths.
// If format is true any go files in paths are processed using goimports.
//
// For detailed documentation consult the docs for "github.com/sridharv/stencil/cmd/stencil"
func Process(paths []string, format bool) error {
	files, err := processStencil(paths)
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

func (r replacer) preReplace(c apply.ApplyCursor) bool {
	switch t := c.Node().(type) {
	case *ast.GenDecl:
		// Delete named type specifications that will be replaced.
		if len(t.Specs) == 0 {
			return true
		}
		spec, ok := t.Specs[0].(*ast.TypeSpec)
		if !ok {
			return true
		}

		if _, ok = r[spec.Name.Name]; !ok {
			return true
		}
		c.Delete()
	case *ast.Ident:
		if t == nil {
			return true
		}
		if s, ok := r[t.Name]; ok {
			t.Name = s
		}
	case *ast.InterfaceType:
		rep, ok := r["interface"]
		if !ok {
			return true
		}
		if _, isType := c.Parent().(*ast.TypeSpec); isType {
			return true
		}
		c.Replace(&ast.Ident{
			Name:    rep,
			NamePos: t.Pos(),
		})
	}
	return true
}

func listPackages(paths []string) (map[string][]string, error) {
	if len(paths) == 0 {
		paths = append(paths, ".")
	}
	dirs := map[string][]string{}
	for _, arg := range paths {
		c, err := filepath.Abs(arg)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		if strings.HasSuffix(c, ".go") {
			dir := filepath.Dir(c)
			dirs[dir] = append(dirs[dir], c)
			continue
		}
		infos, err := ioutil.ReadDir(c)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		var files []string
		for _, i := range infos {
			n := i.Name()
			if strings.HasSuffix(n, ".go") && !strings.HasSuffix(n, "_test.go") {
				files = append(files, filepath.Join(c, n))
			}
		}
		dirs[c] = files
	}
	return dirs, nil
}

func packageExists(roots []string, pkg string) (string, bool) {
	for _, r := range roots {
		// Rough heuristic to check if a package exists.
		dir := filepath.Join(r, pkg)
		if s, err := os.Stat(dir); err == nil && s.IsDir() {
			return dir, true
		}
	}
	return "", false
}

func replacements(roots []string, pkg string) (string, replacer) {
	parts, path := strings.Split(pkg, "/"), pkg
	// See if we can form a substitution pattern from the parts here
	r := replacer{}
	dir, found := packageExists(roots, path)
	for !found && len(parts) > 2 {
		l := len(parts)
		// A path looks like github.com/foo/bar/Parameter/Specialization
		// r[originalType] = replacementType
		r[parts[l-2]] = parts[l-1]
		parts = parts[:l-2]
		path = strings.Join(parts, "/")
		dir, found = packageExists(roots, path)
	}
	if !found || len(r) == 0 {
		return "", nil
	}
	return dir, r
}

func makeStencilled(stencil, stencilled string, r replacer, res *[]file) error {
	fs := token.NewFileSet()
	pkgs, err := parser.ParseDir(fs, stencil, func(s os.FileInfo) bool {
		return !strings.HasSuffix(s.Name(), "_test.go")
	}, parser.AllErrors|parser.ParseComments)
	if err != nil {
		return errors.Wrapf(err, "%s: errors parsing", stencil)
	}
	if len(pkgs) != 1 {
		return errors.Errorf("%d: expected 1 package, got %d", stencil, len(pkgs))
	}
	var files map[string]*ast.File
	for _, p := range pkgs {
		files = p.Files
		break
	}
	for path, f := range files {
		target := filepath.Join(stencilled, filepath.Base(path))
		apply.Apply(f, r.preReplace, nil)
		var b bytes.Buffer
		if err := format.Node(&b, fs, f); err != nil {
			return errors.Errorf("%s:%s: code generation failed", stencil, f.Name)
		}
		out, err := imports.Process(target, b.Bytes(), nil)
		if err != nil {
			return errors.WithStack(err)
		}
		*res = append(*res, file{path: target, data: out})
	}
	return nil
}

func srcRoot(dir string) (string, error) {
	srcs := build.Default.SrcDirs()
	for _, src := range srcs {
		if strings.HasPrefix(dir, src) {
			return src, nil
		}
	}

	var candidates []os.FileInfo
	for d := dir; d != filepath.Dir(d); d = filepath.Dir(d) {
		if filepath.Base(d) != "src" {
			continue
		}
		info, err := os.Stat(d)
		if err != nil {
			return "", errors.Wrapf(err, "failed to stat parent dir: %s", d)
		}
		candidates = append(candidates, info)
	}

	for _, src := range srcs {
		si, err := os.Stat(src)
		if err != nil {
			return "", errors.Wrapf(err, "couldn't stat Go src folder: %s", src)
		}
		for _, ci := range candidates {
			if os.SameFile(ci, si) {
				return src, nil
			}
		}
	}

	return "", errors.Errorf("%s: not in GOPATH", dir)
}

func processDir(dir string, files []string, res *[]file) error {
	// Read files
	fs := token.NewFileSet()
	srcs, err := srcRoot(dir)
	if err != nil {
		return err
	}

	vendor := filepath.Join(dir, "vendor")
	for d := dir; d != srcs; d = filepath.Dir(d) {
		v := filepath.Join(d, "vendor")
		st, err := os.Stat(d)
		if err == nil && st.IsDir() {
			vendor = v
			break
		}
	}
	roots := append(build.Default.SrcDirs(), vendor)

	for _, fl := range files {
		f, err := parser.ParseFile(fs, fl, nil, parser.ImportsOnly)
		if err != nil {
			return errors.Wrapf(err, "%s: parse failed", fl)
		}
		for _, imp := range f.Imports {
			path := imp.Path.Value
			path = path[1 : len(path)-1]
			stencil, r := replacements(roots, path)
			if stencil == "" {
				continue
			}
			if err = makeStencilled(stencil, filepath.Join(vendor, path), r, res); err != nil {
				return err
			}
		}
	}
	return nil
}

func processStencil(paths []string) ([]file, error) {
	dirs, err := listPackages(paths)
	if err != nil {
		return nil, err
	}
	var res []file
	for dir, files := range dirs {
		if err := processDir(dir, files, &res); err != nil {
			return nil, err
		}
	}
	return res, nil
}
