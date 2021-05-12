package clog

import (
	"encoding/json"
	"io/ioutil"
)

type ConFileWriter struct {
	On                  bool   `json:"On"`
	DeleteCycle         uint64 `json:"DeleteCycle"` // 秒为单位
	LogPath             string `json:"logPath"`
	RotateLogPath       string `json:"RotateLogPath"`
	WfLogPath           string `json:"WfLogPath"`
	RotateWfLogPath     string `json:"RotateWfLogPath"`
	PublicLogPath       string `json:"PublicLogPath"`
	RotatePublicLogPath string `json:"RotatePublicLogPath"`
	Root                string `json:"root"`
}

type ConfConsoleWriter struct {
	On    bool `json:"On"`
	Color bool `json:"Color"`
}

type LogConfig struct {
	Level string            `json:"LogLevel"`
	FW    ConFileWriter     `json:"FileWriter"`
	CW    ConfConsoleWriter `json:"ConsoleWriter"`
}



func SetupLogWithConf(file string) (err error) {
	var lc LogConfig

	cnt, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}
	if err = json.Unmarshal(cnt, &lc); err != nil {
		return
	}

	if lc.FW.On {
		if len(lc.FW.LogPath) > 0 {
			w := NewFileWriter()
			w.SetFileName(lc.FW.LogPath)
			w.SetPathPattern(lc.FW.RotateLogPath) // if err occur just fix
			// then rerun
			w.SetLogLevelFloor(TRACE)
			w.SetLogDeleteCycle(lc.FW.DeleteCycle)
			w.SetLogRoot(lc.FW.Root)
			if len(lc.FW.WfLogPath) > 0 {
				w.SetLogLevelCeil(INFO)
			} else {
				w.SetLogLevelCeil(ERROR)
			}
			Register(w)
		}

		if len(lc.FW.WfLogPath) > 0 {
			wfw := NewFileWriter()
			wfw.SetFileName(lc.FW.WfLogPath)
			wfw.SetPathPattern(lc.FW.RotateWfLogPath)
			wfw.SetLogDeleteCycle(lc.FW.DeleteCycle)
			wfw.SetLogLevelFloor(WARNING)
			wfw.SetLogLevelCeil(ERROR)
			wfw.SetLogRoot(lc.FW.Root)
			Register(wfw)
		}

		if len(lc.FW.PublicLogPath) > 0 {
			pw := NewFileWriter()
			pw.SetFileName(lc.FW.PublicLogPath)
			pw.SetPathPattern(lc.FW.RotatePublicLogPath)
			pw.SetLogDeleteCycle(lc.FW.DeleteCycle)
			pw.SetLogLevelFloor(PUBLIC)
			pw.SetLogLevelCeil(PUBLIC)
			pw.SetLogRoot(lc.FW.Root)
			Register(pw)
		}
	}

	if lc.CW.On {
		w := NewConsoleWriter()
		Register(w)
	}

	switch lc.Level {

	case "trace":
		SetLevel(TRACE)

	case "debug":
		SetLevel(DEBUG)

	case "info":
		SetLevel(INFO)

	case "warning":
		SetLevel(WARNING)

	case "error":
		SetLevel(ERROR)

	case "fatal":
		SetLevel(FATAL)

	case "public":
		SetLevel(PUBLIC)
	}

	return

}
