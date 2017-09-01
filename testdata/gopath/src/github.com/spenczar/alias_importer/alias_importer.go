package alias_importer

import p "github.com/spenczar/project"

const _ = p.UsedExportedConst

var _ = p.UsedExportedVar

type s p.UsedExportedStruct
type i p.UsedExportedInterface

var _ = p.UsedExportedFunc

func init() {
	v := p.UsedExportedStruct{}
	v.UsedExportedMethod()
}
