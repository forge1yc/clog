# clog
A powerful log module for go

## Features 
* detached file for warning/fatal level
* support rotate by year/month/day/hour
* support delete out of date log file By *DeleteCycle* 
* json conf file
* record file name and line number
* ...

## init.go

```go
func initLog(log_file string) error {
	if err := logger.SetupLogWithConf(log_file); err != nil {
        err = fmt.Errof("err=%s || failed to init logger",err.Error())
        return err
    }
    return nil
}
```
## log.json
```json
{
    "LogLevel":"info",

    "FileWriter":{
        "On" : true,
        "DeleteCycle" : 2592000,

        "LogPath": "./log/service.log.info",
        "RotateLogPath": "./log/service.log.info.%Y%M%D%H",
        "WfLogPath": "./log/service.log.wf",
        "RotateWfLogPath": "./log/service.log.wf.%Y%M%D%H",
        "PublicLogPath": "./log/public.log",
        "RotatePublicLogPath": "./log/public.log.%Y%M%D%H"
    },
    
    "ConsoleWriter" :{
        "On" : false
    }
}
```


