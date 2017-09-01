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
	"go/build"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration(t *testing.T) {
	buildCtx := build.Default
	wd, err := os.Getwd()
	require.NoError(t, err, "unable to look up working directory")

	buildCtx.GOPATH = filepath.Join(wd, "testdata", "gopath")

	testcase := func(importer string) func(*testing.T) {
		return func(t *testing.T) {
			res, err := unused(&buildCtx, "github.com/spenczar/project", []string{importer})
			require.NoError(t, err)

			wantNames := []string{
				`UnusedExportedConst`,
				`UnusedExportedVar`,
				`UnusedExportedStruct`,
				`UnusedExportedMethod`,
				`UnusedExportedInterface`,
				`UnusedExportedFunc`,
			}
			assert.Len(t, res.unusedObjs, len(wantNames))

		outer:
			for _, name := range wantNames {
				for _, obj := range res.unusedObjs {
					if obj.Name() == name {
						continue outer
					}
				}
				assert.Fail(t, "missing expected result", "expected to report %q as unused, couldnt find it in result", name)
			}
			if t.Failed() {
				t.Log("result set:")
				for _, obj := range res.unusedObjs {
					t.Logf("%q", obj.Name())
				}
			}
		}
	}

	t.Run("simple", testcase("github.com/spenczar/importer"))
	t.Run("aliased import", testcase("github.com/spenczar/alias_importer"))
	t.Run("dot import", testcase("github.com/spenczar/dotimporter"))
	t.Run("embedding", testcase("github.com/spenczar/embedding"))
	t.Run("pointer_embedding", testcase("github.com/spenczar/pointer_embedding"))
}
