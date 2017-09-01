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
	"fmt"
	"log"
	"os"
)

var (
	logger        = log.New(os.Stderr, "checkexport: ", 0)
	loggerEnabled = false

	fatalLogger = log.New(os.Stderr, "checkexport: FATAL: ", log.Lshortfile)

	debugLogger  = log.New(os.Stderr, "debug: ", log.Lshortfile)
	debugEnabled = false
)

func debugf(msg string, args ...interface{}) {
	if debugEnabled {
		_ = debugLogger.Output(2, fmt.Sprintf(msg, args...))
	}
}

func logf(msg string, args ...interface{}) {
	if loggerEnabled {
		_ = logger.Output(2, fmt.Sprintf(msg, args...))
	}
}

func fatalf(msg string, args ...interface{}) {
	_ = fatalLogger.Output(2, fmt.Sprintf(msg, args...))
	os.Exit(1)
}
