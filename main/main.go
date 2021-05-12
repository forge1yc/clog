package main

import (
	"fmt"
	logger "github.com/forge1yc/clog/clog"
	"time"
)

func main() {

	if err:= logger.SetupLogWithConf("./defaultSettings/default_settings." +
		"json"); err != nil {
		fmt.Println(err)

	}
	defer logger.Close()

	logger.ListError([]string{"hello"},[]interface{}{"world"})
	logger.ListInfo([]string{"hello","hello"}, []interface{}{"world","world"})
	logger.ListPublic([]string{"hello","hello"}, []interface{}{"world",
		"world"})

	time.Sleep(time.Second*10000000)
}
