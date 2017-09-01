package project_test

import (
	"testing"

	"github.com/spenczar/project"
)

func TestFunc(t *testing.T) {
	const _ = project.OnlyTestExportedConst

	var _ = project.OnlyTestExportedVar

	type s project.OnlyTestExportedStruct
	type i project.OnlyTestExportedInterface

	var _ = project.OnlyTestExportedFunc
}
