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
	"flag"
	"fmt"
	"go/build"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"github.com/pkg/errors"

	"golang.org/x/tools/go/buildutil"
	"golang.org/x/tools/go/loader"
)

func main() {
	scope := flag.String("scope", "repo root", "scope of search")
	flag.BoolVar(&loggerEnabled, "v", loggerEnabled, "include log output")
	flag.BoolVar(&debugEnabled, "debug", debugEnabled, "include tons of log output (implies -v)")
	flag.Parse()

	args := flag.Args()
	if len(args) != 1 {
		flag.Usage()
		os.Exit(1)
	}
	targetPkg := args[0]

	if *scope == "repo root" {
		root, err := repoRoot(targetPkg)
		if err != nil {
			fatalf("unable to set up scope: %s", err)
		}
		*scope = path.Join(root, "...")
		logf("scope=%v", *scope)
	}

	result, err := unused(&build.Default, targetPkg, []string{*scope})
	if err != nil {
		fatalf(err.Error())
	}
	sort.Slice(result.unusedObjs, func(i, j int) bool {
		return result.unusedObjs[i].Pos() < result.unusedObjs[j].Pos()
	})
	var output []string
	for _, obj := range result.unusedObjs {
		output = append(output, printUnused(result.fset.Position, obj, *scope))
	}
	for _, o := range output {
		fmt.Println(o)
	}
}

type unusedSearchResult struct {
	fset       *token.FileSet
	unusedObjs []types.Object
}

func printUnused(finder func(token.Pos) token.Position, obj types.Object, scope string) string {
	var name string
	switch v := obj.(type) {
	case *types.Func:
		sig, ok := v.Type().(*types.Signature)
		if ok && (sig.Recv() != nil) {
			var recvName string
			recvType, ok := sig.Recv().Type().(*types.Named)
			if ok {
				recvName = recvType.Obj().Name()
			} else {
				recvName = "<unknown>"
			}
			name = fmt.Sprintf("method %s.%s", recvName, obj.Name())
		} else {
			name = fmt.Sprintf("func %s", obj.Name())
		}
	case *types.TypeName:
		name = fmt.Sprintf("type %s", obj.Name())
	case *types.Const:
		name = fmt.Sprintf("const %s", obj.Name())
	case *types.Var:
		name = fmt.Sprintf("var %s", obj.Name())
	default:
		name = obj.Name()
	}
	return fmt.Sprintf("%v: %v is exported but not used anywhere else in %v",
		finder(obj.Pos()).String(), name, scope)
}

// filterExceptions removes objects from the query target set if they have
// special exemptions, like being needed for the sort interface.
func filterExceptions(objs []types.Object, finder func(pos token.Pos) token.Position) []types.Object {
	var out []types.Object
	for _, o := range objs {
		if shouldCheckForUsage(o, finder) {
			out = append(out, o)
		} else {
			debugf("filtering out %v", o)
		}
	}
	return out
}

func shouldCheckForUsage(o types.Object, finder func(pos token.Pos) token.Position) bool {
	switch v := o.Type().(type) {
	case *types.Signature:
		recvr := v.Recv()
		if recvr != nil { // this is a method
			// Less, Swap, and Len are methods used for sorting.
			if o.Name() == "Less" || o.Name() == "Swap" || o.Name() == "Len" {
				return false
			}

			recvrType, ok := recvr.Type().(*types.Named)
			if !ok {
				// might not be possible?
				return false
			}
			if !recvrType.Obj().Exported() {
				return false
			}
		}
		pos := finder(o.Pos())
		if strings.HasSuffix(pos.Filename, "_test.go") {
			return false
		}
	}
	return true
}

func unused(buildCtx *build.Context, target string, scope []string) (*unusedSearchResult, error) {
	fset := token.NewFileSet()
	conf := loader.Config{Fset: fset, Build: buildCtx}
	allowErrors(&conf)

	pkgs := buildutil.ExpandPatterns(buildCtx, scope)
	conf.ImportPkgs = pkgs

	logf("importing all packages in scope (%d packages)", len(pkgs))
	conf.ImportWithTests(target)

	logf("loading all packages")
	// Load/parse/type-check the query package.
	prog, err := conf.Load()
	if err != nil {
		return nil, err
	}

	logf("finding exported members of package")
	targetPkg := prog.Package(target)
	if targetPkg == nil {
		fatalf("target pkg %q not found in scope", target)
	}
	targetObjs := exportedMembers(targetPkg)
	targetObjs = filterExceptions(targetObjs, fset.Position)
	if len(targetObjs) == 0 {
		logf("no exported members in package")
		return &unusedSearchResult{fset: fset}, nil
	}

	for _, o := range targetObjs {
		debugf("checking for %v", o)
	}
	logf("%d exported members found", len(targetObjs))
	logf("finding unused exported objects")
	unusedObjs := findUnused(prog, targetObjs)
	logf("%d unused exported members found", len(unusedObjs))
	return &unusedSearchResult{
		unusedObjs: unusedObjs,
		fset:       fset,
	}, nil
}

func exportedMembers(pkg *loader.PackageInfo) []types.Object {
	var out []types.Object
	outSet := make(map[string]types.Object)
	for _, obj := range pkg.Defs {
		if obj == nil || obj.Pkg() == nil {
			continue
		}
		switch v := obj.(type) {
		case *types.Const:
		case *types.Var:
			if v.IsField() {
				// Ignore fields because sometimes they have to be public for
				// marshaling/unmarshaling.
				continue
			}
		case *types.Func:
		case *types.TypeName:
		default:
			continue
		}
		if obj.Pkg() != pkg.Pkg {
			continue
		}
		if strings.HasPrefix(obj.Name(), "Test") {
			continue
		}
		if obj.Exported() {
			outSet[obj.Id()] = obj
		}
	}

	for _, obj := range outSet {
		out = append(out, obj)
	}

	return out
}

// allowErrors causes type errors to be silently ignored.
// (Not suitable if SSA construction follows.)
func allowErrors(lconf *loader.Config) {
	ctxt := *lconf.Build // copy
	ctxt.CgoEnabled = false
	lconf.Build = &ctxt
	lconf.AllowErrors = true
	// AllErrors makes the parser always return an AST instead of
	// bailing out after 10 errors and returning an empty ast.File.
	lconf.ParserMode = parser.AllErrors
	lconf.TypeChecker.Error = func(err error) {}
}

func repoRoot(pkg string) (string, error) {
	pkgPath := filepath.Join(os.Getenv("GOPATH"), "src", pkg)
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = pkgPath
	stdout, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitStatus := exitErr.Sys().(syscall.WaitStatus).ExitStatus()
			if exitStatus == 128 { // not in a repository
				return "", nil
			}
		}
		return "", errors.Wrap(err, "failed to invoke git")
	}
	repoFileRoot := strings.TrimSpace(string(stdout))
	repoRelFileRoot, err := filepath.Rel(filepath.Join(os.Getenv("GOPATH"), "src"), repoFileRoot)
	if err != nil {
		return "", err
	}
	return repoRelFileRoot, nil
}
