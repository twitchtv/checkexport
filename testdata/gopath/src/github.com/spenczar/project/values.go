package project

// Exported values which are imported by other packages:
const UsedExportedConst = 1

var UsedExportedVar = 2

type UsedExportedInterface interface{}
type UsedExportedStruct struct{}

func (UsedExportedStruct) UsedExportedMethod()   {}
func (UsedExportedStruct) UnusedExportedMethod() {}

func UsedExportedFunc() {}

// Exported values which are not imported by other packages:
const UnusedExportedConst = 1

var UnusedExportedVar = 2

type UnusedExportedStruct struct{}
type UnusedExportedInterface interface{}

func UnusedExportedFunc() {}

// Exported values which are not imported by other packages, but are used in
// tests in this package:
const OnlyTestExportedConst = 1

var OnlyTestExportedVar = 2

type OnlyTestExportedStruct struct{}
type OnlyTestExportedInterface interface{}

func OnlyTestExportedFunc() {}

// Unexported values
const unexportedConst = 3

var unexportedVar = 4
var unexportedStruct struct{}
var unexportedInterface interface{}

func unexportedFunc() {}

// methods on private objects
type s struct{}

func (s) MethodOnPrivateStruct() {}

type i interface {
	MethodOnPrivateInterface()
}
