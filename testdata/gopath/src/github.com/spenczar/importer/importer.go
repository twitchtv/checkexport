package importer

import "github.com/spenczar/project"

const _ = project.UsedExportedConst

var _ = project.UsedExportedVar

type s project.UsedExportedStruct
type i project.UsedExportedInterface

var _ = project.UsedExportedFunc

func init() {
	v := project.UsedExportedStruct{}
	v.UsedExportedMethod()
}
