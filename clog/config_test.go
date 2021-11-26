package clog

import (
	"fmt"
	"testing"
)

func TestSetupLogWithConf(t *testing.T) {

	err := SetupLogWithConf("hello world")

	if err != nil {
		fmt.Printf("%+v\n",err)
	}

}
