package dotimporter

import . "github.com/spenczar/project"

const _ = UsedExportedConst

var _ = UsedExportedVar

type s UsedExportedStruct
type i UsedExportedInterface

var _ = UsedExportedFunc

func init() {
	v := UsedExportedStruct{}
	v.UsedExportedMethod()
}
