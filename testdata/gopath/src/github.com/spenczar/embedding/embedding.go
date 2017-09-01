package embedding

import "github.com/spenczar/project"

const _ = project.UsedExportedConst

var _ = project.UsedExportedVar

type s struct {
	project.UsedExportedStruct
	project.UsedExportedInterface
}

var _ = project.UsedExportedFunc

func init() {
	v := s{}
	v.UsedExportedMethod()
}
