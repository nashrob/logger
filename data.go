package logger

import (
	"os"
	"sync"
)

// log levels
const DBGRM string = "DBGRM"     // green
const DEBUG string = "DEBUG"     // normal
const INFO string = "INFO"       // normal
const WARNING string = "WARNING" // yellow
const ERROR string = "ERROR"     // red

// buffered channel with size 10.
var chanbuffLog chan logmessage

// log-file file handler.
var pServerLogFile *os.File

var currentLogfileCnt uint8 = 1
var logfileNameList []string
var dummyLogfile string

//var loggerWG sync.WaitGroup

const log_MAX_FILES int8 = 10
const log_FILE_NAME_PREFIX string = "server.log"
const log_FILE_SIZE int64 = 20971520 // 20 MB
var current_LOG_LEVEL string = "DBGRM"

var isInit bool
var isLoggerInstanceInit bool
var srcBaseDir string

const colorNornal string = "\033[0m"
const colorErrorRed string = "\033[31m"
const colorDbgrmGreen string = "\033[32m"
const colorWarnYellow string = "\033[33m"

var loglevelMap = map[string]loglevel{
	"DBGRM": loglevel{
		str:   DBGRM,
		color: colorDbgrmGreen,
		wt:    0,
	},
	"DEBUG": loglevel{
		str:   DEBUG,
		color: colorNornal,
		wt:    1,
	},
	"INFO": loglevel{
		str:   INFO,
		color: colorNornal,
		wt:    2,
	},
	"WARNING": loglevel{
		str:   WARNING,
		color: colorWarnYellow,
		wt:    3,
	},
	"ERROR": loglevel{
		str:   ERROR,
		color: colorErrorRed,
		wt:    4,
	},
}

//var doneChan chan bool
var doneChanFlag bool
var pDoneChanLock *sync.Mutex
