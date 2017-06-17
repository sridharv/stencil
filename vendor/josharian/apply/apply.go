// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package apply

import (
	"fmt"
	"go/ast"
	"reflect"
)

// An ApplyFunc is invoked by Apply for each node n, even if n is nil,
// before and/or after the node's children.
//
// The return value of ApplyFunc controls the syntax tree traversal.
// See Apply for details.
type ApplyFunc func(cursor ApplyCursor) bool

// Apply traverses a syntax tree recursively, starting with root,
// and calling pre and post for each node as described below. The
// result is the (possibly modified) syntax tree.
//
// If pre is not nil, it is called for each node before its children
// are traversed (pre-order). If the result of calling pre is false,
// no children are traversed, and post is not called for that node.
//
// If post is not nil, it is called for each node after its children
// were traversed (post-order). If the result of calling post is false,
// traversal is terminated and Apply returns immediately.
//
// Only fields that refer to AST nodes are considered children.
// Children are traversed in the order in which they appear in the
// respective node's struct definition.
func Apply(root ast.Node, pre, post ApplyFunc) ast.Node {
	defer func() {
		if r := recover(); r != nil && r != abort {
			panic(r)
		}
	}()
	a := &application{Node: root, pre: pre, post: post}
	a.apply(a, "Node", -1, a.Node)
	return a.Node
}

// An ApplyCursor describes a node encountered during Apply.
// Information about the node and its parent is available
// via the Node, Parent, Name, and Index methods.
//
// Roughly speaking, the following invariants hold:
//
//   Parent().Name()          == Node()  if !HasIndex()
//   Parent().Name()[Index()] == Node()  if HasIndex()
//
// The methods Replace, Delete, InsertBefore, and InsertAfter
// can be used to change the AST without disrupting Apply.
type ApplyCursor struct {
	node   ast.Node
	parent ast.Node
	name   string
	index  *int
	incr   *int // increment to index done after this ApplyFunc is completed
}

// Node returns the current Node.
func (c ApplyCursor) Node() ast.Node { return c.node }

// Parent returns the parent of the current Node.
func (c ApplyCursor) Parent() ast.Node { return c.parent }

// Name returns the name of the parent Node field that contains the current Node.
// If the parent is a Package and the current Node is a File,
// it returns the filename for the current Node.
func (c ApplyCursor) Name() string { return c.name }

// HasIndex reports whether the current Node is part of a slice of Nodes.
func (c ApplyCursor) HasIndex() bool { return c.index != nil }

// Index reports the index of the current Node in the slice of Nodes that contains it.
// Index panics if the current Node is not part of a slice.
func (c ApplyCursor) Index() int {
	if !c.HasIndex() {
		panic("ApplyCursor has no index")
	}
	return *c.index
}

// IsFile reports whether the current Node is a *File that is part of a *Package map of *Files.
func (c ApplyCursor) IsFile() bool {
	_, isfile := c.pkgfile()
	return isfile
}

// pkgfile reports whether the current Node is *File that is part of a *Package File map.
// If so, it returns the parent *Package.
func (c ApplyCursor) pkgfile() (pkg *ast.Package, ok bool) {
	pkg, ispkg := c.parent.(*ast.Package)
	if !ispkg {
		return nil, false
	}
	_, isfile := c.node.(*ast.File)
	if !isfile {
		return nil, false
	}
	return pkg, true
}

// Replace replaces the current Node with n.
// The replacement node is not walked by Apply.
func (c ApplyCursor) Replace(n ast.Node) {
	if pkg, ispkg := c.pkgfile(); ispkg {
		file, ok := n.(*ast.File)
		if !ok {
			panic("attempt to replace *File with non-*File")
		}
		pkg.Files[c.name] = file
		return
	}
	v := reflect.Indirect(reflect.ValueOf(c.parent)).FieldByName(c.name)
	if c.HasIndex() {
		v = v.Index(*c.index)
	}
	v.Set(reflect.ValueOf(n))
}

// Delete deletes the current Node from its containing slice.
// If the current Node is not part of a slice, Delete panics.
func (c ApplyCursor) Delete() {
	if !c.HasIndex() {
		panic("Delete node not contained in slice")
	}
	v := reflect.Indirect(reflect.ValueOf(c.parent)).FieldByName(c.name)
	last := v.Len()
	reflect.Copy(v.Slice(*c.index, last), v.Slice(*c.index+1, last))
	v.Index(last - 1).Set(reflect.Zero(v.Type().Elem()))
	v.SetLen(last - 1)
	*c.incr--
}

// InsertAfter inserts n after the current Node in its containing slice.
// If the current Node is not part of a slice, InsertAfter panics.
// Apply will walk n.
func (c ApplyCursor) InsertAfter(n ast.Node) {
	if !c.HasIndex() {
		panic("InsertAfter node not contained in slice")
	}
	v := reflect.Indirect(reflect.ValueOf(c.parent)).FieldByName(c.name)
	v.Set(reflect.Append(v, reflect.Zero(v.Type().Elem())))
	last := v.Len()
	reflect.Copy(v.Slice(*c.index+2, last), v.Slice(*c.index+1, last))
	v.Index(*c.index + 1).Set(reflect.ValueOf(n))
}

// InsertBefore inserts n before the current Node in its containing slice.
// If the current Node is not part of a slice, InsertBefore panics.
// Apply will not walk n.
func (c ApplyCursor) InsertBefore(n ast.Node) {
	if !c.HasIndex() {
		panic("InsertBefore node not contained in slice")
	}
	v := reflect.Indirect(reflect.ValueOf(c.parent)).FieldByName(c.name)
	v.Set(reflect.Append(v, reflect.Zero(v.Type().Elem())))
	last := v.Len()
	reflect.Copy(v.Slice(*c.index+1, last), v.Slice(*c.index, last))
	v.Index(*c.index).Set(reflect.ValueOf(n))
	*c.index++
}

type application struct {
	ast.Node
	pre, post ApplyFunc
}

func (a *application) apply(parent ast.Node, name string, index int, n ast.Node) (newindex, incr int) {
	incr = 1
	cursor := ApplyCursor{
		parent: parent,
		node:   n,
		name:   name,
	}
	if index >= 0 {
		cursor.index = &index
	}
	cursor.incr = &incr
	if a.pre != nil && !a.pre(cursor) {
		return index, incr
	}

	// walk children
	// (the order of the cases matches the order
	// of the corresponding node types in ast.go)
	switch n := n.(type) {
	case nil:
		// nothing to do

	// Comments and fields
	case *ast.Comment:
		// nothing to do

	case *ast.CommentGroup:
		if n != nil {
			a.applyList(n, "List")
		}

	case *ast.Field:
		a.apply(n, "Doc", -1, n.Doc)
		a.applyList(n, "Names")
		a.apply(n, "Type", -1, n.Type)
		a.apply(n, "Tag", -1, n.Tag)
		a.apply(n, "Comment", -1, n.Comment)

	case *ast.FieldList:
		if n != nil {
			a.applyList(n, "List")
		}

	// Expressions
	case *ast.BadExpr, *ast.Ident, *ast.BasicLit:
		// nothing to do

	case *ast.Ellipsis:
		a.apply(n, "Elt", -1, n.Elt)

	case *ast.FuncLit:
		a.apply(n, "Type", -1, n.Type)
		a.apply(n, "Body", -1, n.Body)

	case *ast.CompositeLit:
		a.apply(n, "Type", -1, n.Type)
		a.applyList(n, "Elts")

	case *ast.ParenExpr:
		a.apply(n, "X", -1, n.X)

	case *ast.SelectorExpr:
		a.apply(n, "X", -1, n.X)
		a.apply(n, "Sel", -1, n.Sel)

	case *ast.IndexExpr:
		a.apply(n, "X", -1, n.X)
		a.apply(n, "Index", -1, n.Index)

	case *ast.SliceExpr:
		a.apply(n, "X", -1, n.X)
		a.apply(n, "Low", -1, n.Low)
		a.apply(n, "High", -1, n.High)
		a.apply(n, "Max", -1, n.Max)

	case *ast.TypeAssertExpr:
		a.apply(n, "X", -1, n.X)
		a.apply(n, "Type", -1, n.Type)

	case *ast.CallExpr:
		a.apply(n, "Fun", -1, n.Fun)
		a.applyList(n, "Args")

	case *ast.StarExpr:
		a.apply(n, "X", -1, n.X)

	case *ast.UnaryExpr:
		a.apply(n, "X", -1, n.X)

	case *ast.BinaryExpr:
		a.apply(n, "X", -1, n.X)
		a.apply(n, "Y", -1, n.Y)

	case *ast.KeyValueExpr:
		a.apply(n, "Key", -1, n.Key)
		a.apply(n, "Value", -1, n.Value)

	// Types
	case *ast.ArrayType:
		a.apply(n, "Len", -1, n.Len)
		a.apply(n, "Elt", -1, n.Elt)

	case *ast.StructType:
		a.apply(n, "Fields", -1, n.Fields)

	case *ast.FuncType:
		a.apply(n, "Params", -1, n.Params)
		a.apply(n, "Results", -1, n.Results)

	case *ast.InterfaceType:
		a.apply(n, "Methods", -1, n.Methods)

	case *ast.MapType:
		a.apply(n, "Key", -1, n.Key)
		a.apply(n, "Value", -1, n.Value)

	case *ast.ChanType:
		a.apply(n, "Value", -1, n.Value)

	// Statements
	case *ast.BadStmt:
		// nothing to do

	case *ast.DeclStmt:
		a.apply(n, "Decl", -1, n.Decl)

	case *ast.EmptyStmt:
		// nothing to do

	case *ast.LabeledStmt:
		a.apply(n, "Label", -1, n.Label)
		a.apply(n, "Stmt", -1, n.Stmt)

	case *ast.ExprStmt:
		a.apply(n, "X", -1, n.X)

	case *ast.SendStmt:
		a.apply(n, "Chan", -1, n.Chan)
		a.apply(n, "Value", -1, n.Value)

	case *ast.IncDecStmt:
		a.apply(n, "X", -1, n.X)

	case *ast.AssignStmt:
		a.applyList(n, "Lhs")
		a.applyList(n, "Rhs")

	case *ast.GoStmt:
		a.apply(n, "Call", -1, n.Call)

	case *ast.DeferStmt:
		a.apply(n, "Call", -1, n.Call)

	case *ast.ReturnStmt:
		a.applyList(n, "Results")

	case *ast.BranchStmt:
		a.apply(n, "Label", -1, n.Label)

	case *ast.BlockStmt:
		a.applyList(n, "List")

	case *ast.IfStmt:
		a.apply(n, "Init", -1, n.Init)
		a.apply(n, "Cond", -1, n.Cond)
		a.apply(n, "Body", -1, n.Body)
		a.apply(n, "Else", -1, n.Else)

	case *ast.CaseClause:
		a.applyList(n, "List")
		a.applyList(n, "Body")

	case *ast.SwitchStmt:
		a.apply(n, "Init", -1, n.Init)
		a.apply(n, "Tag", -1, n.Tag)
		a.apply(n, "Body", -1, n.Body)

	case *ast.TypeSwitchStmt:
		a.apply(n, "Init", -1, n.Init)
		a.apply(n, "Assign", -1, n.Assign)
		a.apply(n, "Body", -1, n.Body)

	case *ast.CommClause:
		a.apply(n, "Comm", -1, n.Comm)
		a.applyList(n, "Body")

	case *ast.SelectStmt:
		a.apply(n, "Body", -1, n.Body)

	case *ast.ForStmt:
		a.apply(n, "Init", -1, n.Init)
		a.apply(n, "Cond", -1, n.Cond)
		a.apply(n, "Post", -1, n.Post)
		a.apply(n, "Body", -1, n.Body)

	case *ast.RangeStmt:
		a.apply(n, "Key", -1, n.Key)
		a.apply(n, "Value", -1, n.Value)
		a.apply(n, "X", -1, n.X)
		a.apply(n, "Body", -1, n.Body)

	// Declarations
	case *ast.ImportSpec:
		a.apply(n, "Doc", -1, n.Doc)
		a.apply(n, "Name", -1, n.Name)
		a.apply(n, "Path", -1, n.Path)
		a.apply(n, "Comment", -1, n.Comment)

	case *ast.ValueSpec:
		a.apply(n, "Doc", -1, n.Doc)
		a.applyList(n, "Names")
		a.apply(n, "Type", -1, n.Type)
		a.applyList(n, "Values")
		a.apply(n, "Comment", -1, n.Comment)

	case *ast.TypeSpec:
		a.apply(n, "Doc", -1, n.Doc)
		a.apply(n, "Name", -1, n.Name)
		a.apply(n, "Type", -1, n.Type)
		a.apply(n, "Comment", -1, n.Comment)

	case *ast.BadDecl:
		// nothing to do

	case *ast.GenDecl:
		a.apply(n, "Doc", -1, n.Doc)
		a.applyList(n, "Specs")

	case *ast.FuncDecl:
		a.apply(n, "Doc", -1, n.Doc)
		a.apply(n, "Recv", -1, n.Recv)
		a.apply(n, "Name", -1, n.Name)
		a.apply(n, "Type", -1, n.Type)
		a.apply(n, "Body", -1, n.Body)

	// Files and packages
	case *ast.File:
		a.apply(n, "Doc", -1, n.Doc)
		a.apply(n, "Name", -1, n.Name)
		a.applyList(n, "Decls")
		// don't walk n.Comments - they have been
		// visited already through the individual
		// nodes

	case *ast.Package:
		for name, f := range n.Files {
			a.apply(n, name, -1, f)
		}

	default:
		panic(fmt.Sprintf("ast.Apply: unexpected node type %T", n))
	}

	if a.post != nil && !a.post(cursor) {
		panic(abort)
	}

	return index, incr
}

var abort = new(int) // singleton, to signal abortion of Apply

func (a *application) applyList(parent ast.Node, name string) {
	index := 0
	for {
		// Must reload parent.name each time, since cursor modifications might change it.
		v := reflect.Indirect(reflect.ValueOf(parent)).FieldByName(name)
		if index >= v.Len() {
			break
		}
		x := v.Index(index).Interface().(ast.Node)
		var incr int
		index, incr = a.apply(parent, name, index, x)
		index += incr
	}
}
