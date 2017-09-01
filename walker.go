// Copyright 2017 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may not
// use this file except in compliance with the License. A copy of the License is
// located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed on
// an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing
// permissions and limitations under the License.

package main

import (
	"go/ast"
	"go/token"
	"go/types"
	"strconv"
	"strings"

	"golang.org/x/tools/go/loader"
)

// queryTarget holds intermediate state about a query while it runs
type queryTarget struct {
	all     []types.Object
	pkg     *types.Package
	unfound map[types.Object]bool
}

func newTarget(objs []types.Object) queryTarget {
	tgt := queryTarget{
		all:     objs,
		pkg:     objs[0].Pkg(),
		unfound: make(map[types.Object]bool, len(objs)),
	}
	for _, o := range objs {
		tgt.unfound[o] = true
	}
	return tgt
}

func (t queryTarget) allFound() bool {
	return len(t.unfound) == 0
}

// check sees if o is one of t's targets. It returns true if so.
func (t queryTarget) isTarget(o types.Object) bool {
	for _, targetO := range t.all {
		if o == targetO {
			return true
		}
	}
	return false
}

// discover marks o as found, if it's in t's set of unfound objects. It returns
// true if it was looking for o.
func (t queryTarget) discover(o types.Object) bool {
	if o == nil {
		return false
	}
	for lookingFor := range t.unfound {
		debugf("(%v) == (%v): %t", o, lookingFor, o == lookingFor)
		if o == lookingFor {
			debugf("found %v", o.Name())
			delete(t.unfound, o)
			return true
		}
		if tn, ok := lookingFor.(*types.TypeName); ok {
			debugf("type name: (%v) (%T) == (%v): %t", o.Type(), o.Type(), tn.Type(), o.Type() == tn.Type())
			if tn.Type() == o.Type() {
				delete(t.unfound, lookingFor)
				return true
			}
			// Dereference pointers, too, if required.
			ot := o.Type()
			for true {
				ptr, ok := ot.(*types.Pointer)
				if !ok {
					break
				}
				ot = ptr.Elem()
				if tn.Type() == ot {
					delete(t.unfound, lookingFor)
					return true
				}
			}
			if _, ok := o.Type().(*types.Pointer); ok {

			}
		}
	}
	return false
}

// useFinder implements ast.Visitor. It walks an AST, checking to see whether
// its query targets are used. All query targets should be from the same
// package.
type useFinder struct {
	target queryTarget

	// queryPkgLocalName is the name of the queryPkg in the scope of the file
	// currently being visited.
	queryPkgLocalName string

	// Should we skip vendor paths?
	skipVendor bool

	// Stored type checker info
	typeInfo types.Info

	position func(token.Pos) token.Position
}

func findUnused(prog *loader.Program, targets []types.Object) (unusedObjs []types.Object) {
	finder := &useFinder{
		target:     newTarget(targets),
		skipVendor: true,
		position:   prog.Fset.Position,
	}

	for _, pkgInfo := range prog.AllPackages {
		// Skip the original package that contained the targeted objects.
		if pkgInfo.Pkg == finder.target.pkg {
			continue
		}
		// Skip standard library packages.
		if !strings.Contains(pkgInfo.Pkg.Path(), ".") {
			continue
		}
		// Skip vendor directories, if told to
		if finder.skipVendor && strings.Contains(pkgInfo.Pkg.Path(), "vendor") {
			continue
		}
		debugf("checking in %v for %d targets", pkgInfo, len(targets))
		finder.typeInfo = pkgInfo.Info
		// Walk the AST of each file in the package.
		for _, file := range pkgInfo.Files {
			ast.Walk(finder, file)
			// If all query objects have been found, we're done, and can return.
			debugf("after walking %v: %v", file, finder.target.unfound)
			if finder.target.allFound() {
				return
			}
		}
		if finder.target.allFound() {
			return
		}
	}

	for o := range finder.target.unfound {
		unusedObjs = append(unusedObjs, o)
	}
	return unusedObjs
}

func (f *useFinder) Visit(node ast.Node) ast.Visitor {
	if node != nil {
		debugf("node=%v\ttype=%T", f.position(node.Pos()), node)
	}
	switch v := node.(type) {
	case *ast.File:
		return f.visitFile(v)
	case nil:
		return nil
	case *ast.ImportSpec:
		return f.visitImportSpec(v)
	case *ast.SelectorExpr:
		return f.visitSelector(v)
	case *ast.Ident:
		return f.visitIdent(v)

	case *ast.Field, *ast.FieldList:
		// Stuff we want to go into, but have no special behavior for
		return f

	case *ast.Comment, *ast.CommentGroup, *ast.BadExpr:
		// Skippable nodes
		return nil

	default:
		return f
	}
}

// visitFile enters a file if it imports the package that the query is targeting
func (f *useFinder) visitFile(file *ast.File) ast.Visitor {
	f.queryPkgLocalName = ""

	for _, importSpec := range file.Imports {
		path, _ := strconv.Unquote(importSpec.Path.Value)
		if path == f.target.pkg.Path() {
			debugf("file %v imports pkg %v, entering", f.position(file.Name.Pos()), f.target.pkg.Path())
			return f
		}
	}
	return nil
}

// visitImportSpec sets f.queryPkgLocalName if spec refers to f.queryPkg.
// Otherwise, it does nothing. It always returns nil.
func (f *useFinder) visitImportSpec(spec *ast.ImportSpec) ast.Visitor {
	path, _ := strconv.Unquote(spec.Path.Value)
	if path == f.target.pkg.Path() {
		debugf("found import of pkg %v", path)
		if spec.Name != nil {
			debugf("pkg alias used: %v", spec.Name.Name)
			f.queryPkgLocalName = spec.Name.Name
		} else {
			debugf("no pkg alias used: %v", f.target.pkg.Name())
			f.queryPkgLocalName = f.target.pkg.Name()
		}
	}
	return nil
}

// visitSelector visits a selector expression. This is one of the main chances
// we have to identify uses of the query targets.
func (f *useFinder) visitSelector(sel *ast.SelectorExpr) ast.Visitor {
	debugf("pos(sel): %v", f.position(sel.Pos()))
	debugf("sel.X: %v", sel.X)
	debugf("sel.Sel: %v", sel.Sel)

	rcvrIdent, ok := sel.X.(*ast.Ident)
	if !ok {
		debugf("nil rcvrIdent, type=%T", sel.X)
		return nil
	}

	rcvrObj := f.typeInfo.ObjectOf(rcvrIdent)
	switch rcvr := rcvrObj.(type) {
	case *types.PkgName:
		// The receiver is a package. Is it the package we're querying for?
		if rcvr.Imported() == f.target.pkg {
			// Yes! Check if the selection is one of the objects we're hunting for.
			selObj := f.typeInfo.ObjectOf(sel.Sel)
			discovered := f.target.discover(selObj)
			debugf("discovered: %v", discovered)
			return f
		} else {
			debugf("wrong pkg")
			return nil
		}
	case *types.Var:
		debugf("rcvrObj is a var")
		// The receiver is a var. Is it one of the objects we're looking for?
		found := f.target.discover(rcvrObj)
		if !found {

		}
	default:
		debugf("unexpected rcvrObj.(type): %T", rcvrObj)
	}

	// Now time to look at the selected value.
	selObj := f.typeInfo.ObjectOf(sel.Sel)
	debugf("sel obj=%v", selObj)
	f.target.discover(selObj)

	return nil
}

// visitIdent visits an identifier. If the current file imported queried package
// with a dot import, then we're interested, and will check the name of the
// identifier against the queryObjects set. Otherwise, we don't care about this
// node.
func (f *useFinder) visitIdent(ident *ast.Ident) ast.Visitor {
	if f.queryPkgLocalName != "." {
		return nil
	}
	obj := f.typeInfo.ObjectOf(ident)
	f.target.discover(obj)
	return nil
}
