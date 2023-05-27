package logger

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
)

func getFilePath(sourceFilePath string, defaultPath string) (bool, string) {
	filePath := ""
	if len(defaultPath) > len(sourceFilePath) {
		return false, filePath
	}

	fmt.Printf("dbgrm::  sourceFilePath: %s,  defaultPath: %s\n", sourceFilePath, defaultPath)

	length := len(sourceFilePath) - len(defaultPath)
	var i int
	for i = 0; i < length; i++ {
		if sourceFilePath[i] == defaultPath[0] {
			if sourceFilePath[i:i+len(defaultPath)] == defaultPath {
				break
			}
		}
	}

	filePath = sourceFilePath[i+len(defaultPath) : len(sourceFilePath)]
	fmt.Printf("dbgrm::  filePath: %s\n", filePath)
	return true, filePath
}

func Log(strcomponent string, loglevelStr string, msg string, args ...interface{}) {
	defer func() { // chanbuffLog has been closed.
		if recoverVal := recover(); recoverVal != nil {
			fmt.Println("[WARNING]::  Log(): recover value:", recoverVal)
		}
	}()

	currentLoglevel := loglevelMap[current_LOG_LEVEL] // 0: DBGRM, 1: DEBUG, 2: INFO, 3: WARNING, 4: ERROR
	msgLoglevel, isOK := loglevelMap[loglevelStr]
	if !isOK {
		return
	}
	if (msgLoglevel.wt < currentLoglevel.wt) && (currentLoglevel.wt != 1) { // silently slips through a DBGRM message when currentLoglevel.wt is 1(DEBUG)
		return
	}

	t := time.Now()
	zonename, _ := t.In(time.Local).Zone()
	msgTimeStamp := fmt.Sprintf("%02d-%02d-%d:%02d%02d%02d-%06d-%s", t.Day(), t.Month(), t.Year(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), zonename)
	pc, fn, line, _ := runtime.Caller(1)

	//gwd, _ := os.Getwd()
	//fmt.Printf("dbgrm::  gwd: %s\n", gwd)
	////_, filePath := getFilePath(fn, srcBaseDir)
	////srcFile1 := strings.Split(str1, str2)
	//filePath := strings.Split(fn, srcBaseDir)
	//srcFile := srcBaseDir + filePath[len(filePath) - 1]

	tmp1 := strings.Split((runtime.FuncForPC(pc).Name()), ".")
	pkgname := tmp1[0]
	srcFile := pkgname + "/" + path.Base(fn)
	funcName := tmp1[1]

	msgPrefix := ""
	if loglevelStr == "DBGRM" {
		msgPrefix = "#### "
	}

	////logMsg := fmt.Sprintf("[%s] [%s] [%s] [%s: %d] [%s]:\n", strcomponent, msgTimeStamp, loglevelStr, filepath.Base(fn), line, runtime.FuncForPC(pc).Name())
	////logMsg := fmt.Sprintf("[%s] [%s] [%s] [%s: %d] [%s]:\n",
	////strcomponent, msgTimeStamp, loglevelStr, filePath[len(filePath) - 1], line, runtime.FuncForPC(pc).Name())
	//logMsg := fmt.Sprintf("[%s] [%s] [%s] [%s: %d] [%s]:\n", strcomponent, msgTimeStamp, loglevelStr, srcFile, line, runtime.FuncForPC(pc).Name())
	logMsg := fmt.Sprintf("[%s] [%s] [%s] [%s +%d]@[%s]:\n", strcomponent, msgTimeStamp, loglevelStr, srcFile, line, funcName)
	logMsg = fmt.Sprintf(logMsg+msg, args...)
	logMsg = msgPrefix + logMsg + "\n"

	if !isLoggerInstanceInit {
		//logMsg = msgLoglevel.color + msgPrefix + logMsg + colorNornal + "\n"
		logMsg = msgLoglevel.color + logMsg + colorNornal
		fmt.Printf(logMsg)
		return
	}

	logMessage := logmessage{
		component: strcomponent,
		logmsg:    logMsg,
	}

	pDoneChanLock.Lock()
	if doneChanFlag == false {
		chanbuffLog <- logMessage
	}
	pDoneChanLock.Unlock()
}

func LogDispatcher(ploggerWG *sync.WaitGroup, doneChan chan bool) {
	defer func() {
		fmt.Println("logger exiting.")
		ploggerWG.Done()
	}()

	/* for {
		select {
			case logMsg := <-chanbuffLog: // pushes dummy logmessage onto the channel
				dumpServerLog(logMsg.logmsg)
		}
	} */

	runFlag := true
	for runFlag {
		select {
		case logMsg, isOK := <-chanbuffLog: // pushes dummy logmessage onto the channel
			if !isOK {
				runFlag = false
				break
			}
			dumpServerLog(logMsg.logmsg)
			break

		case <-doneChan: // chanbuffLog has been closed. pull all the logs from the channel and dump them to file-system.
			pDoneChanLock.Lock()
			doneChanFlag = true
			runFlag = false
			dumpServerLog("[WARNING]:: logger exiting. breaking out on closed log message-queue.\nstarting to flush all the blocked logs.\n")
			time.Sleep(10 * time.Second)
			close(chanbuffLog)
			for logMsg := range chanbuffLog {
				dumpServerLog(logMsg.logmsg)
			}
			pDoneChanLock.Unlock()
			break
		}
	}

	/* for runFlag {
		select {
			case <-doneChan:  // chanbuffLog needs to be closed. pull all the logs from the channel and dump them to file-system.
				runFlag = false
				dumpServerLog("[WARNING]:: logger exiting. breaking out on closed log message-queue.\nstarting to flush all the blocked logs.\n")
				close(chanbuffLog)
				for logMsg := range chanbuffLog {
					dumpServerLog(logMsg.logmsg)
				}
				break
			default:
				break
		}
		select {
			case logMsg, isOK := <-chanbuffLog: // pushes dummy logmessage onto the channel
				if !isOK {
					runFlag = false
					break
				}

				dumpServerLog(logMsg.logmsg)
				break
			default:
				break
		}
	} */
}

func dumpServerLog(logMsg string) {
	if pServerLogFile == nil {
		fmt.Printf("error-5\n") // nil file handler
		os.Exit(1)
	}

	if logMsg == "" {
		return
	}

	pServerLogFile.WriteString(logMsg)
	//fmt.Printf(logMsg) // TODO-REM: remove this fmp.Printf() call later

	fi, err := pServerLogFile.Stat()
	if err != nil {
		fmt.Printf("error-6: %s\n", err.Error()) // Couldn't obtain stat
		return
	}

	fileSize := fi.Size()
	if fileSize >= log_FILE_SIZE {
		pServerLogFile.Close()
		pServerLogFile = nil
		err = os.Rename(logfileNameList[0], dummyLogfile)
		if err != nil {
			fmt.Printf("error-7: %s\n", err.Error()) // mv %s to %s, error: %s\n", logfileNameList[0], dummyLogfile, err.Error())
			pServerLogFile, err = os.OpenFile(logfileNameList[0], os.O_RDWR|os.O_APPEND|os.O_CREATE, 0644)
			return
		}

		pServerLogFile, err = os.OpenFile(logfileNameList[0], os.O_RDWR|os.O_APPEND|os.O_CREATE, 0644)
		if err != nil {
			fmt.Printf("error-8: %s\n", err.Error()) // recreating logfile: %s,  error: %s\n", logfileNameList[0], err.Error())
			return
		}

		if currentLogfileCnt < 10 {
			currentLogfileCnt = currentLogfileCnt + 1
		}

		go handleLogRotate()
	}
}

func handleLogRotate() {
	for i := currentLogfileCnt; i > 2; i-- {
		err := os.Rename(logfileNameList[i-2], logfileNameList[i-1])
		if err != nil {
			// mv %s to %s. error: %s\n", logfileNameList[i-2], logfileNameList[i-1], err.Error())
			fmt.Printf("error-10: %s\n", err.Error())
			return
		}
	}

	err := os.Rename(dummyLogfile, logfileNameList[1])
	if err != nil {
		// while mv %s to %s. error: %s\n", dummyLogfile, logfileNameList[1], err.Error())
		fmt.Printf("error-11: %s\n", err.Error())
		return
	}
}

func Init(isLoggerInit bool, tmpSrcBaseDir string, logBaseDir string, logLevel string) bool {
	if isInit {
		return true
	}

	var err error

	if tmpSrcBaseDir = strings.TrimSpace(tmpSrcBaseDir); tmpSrcBaseDir == "" {
		fmt.Printf("Error-1: %s\nSource code BaseDir: %s\n", err.Error(), tmpSrcBaseDir) // Error: abs path: %s\n", err.Error())
		return false
	}
	tmpSrcBaseDir = strings.TrimLeft(tmpSrcBaseDir, "/")
	tmpSrcBaseDir = strings.TrimRight(tmpSrcBaseDir, "/")
	srcBaseDir = "/" + tmpSrcBaseDir

	if logBaseDir = strings.TrimSpace(logBaseDir); logBaseDir == "" {
		if logBaseDir, err = filepath.Abs(filepath.Dir(os.Args[0])); err != nil {
			fmt.Printf("Error-1: %s\nlogBaseDir: %s\n", err.Error(), logBaseDir) // Error: abs path: %s\n", err.Error())
			return false
		}
	}

	logLevel = strings.ToUpper(strings.TrimSpace(logLevel))
	if (logLevel != DBGRM) && (logLevel != DEBUG) && (logLevel != INFO) && (logLevel != WARNING) && (logLevel != ERROR) { // covers logLevel == ""
		fmt.Printf("Error-2: Incorrect log-level. Possible values are: DEBUG, INFO, WARNING, ERROR\n")
		return false
	}

	//logdir := filepath.Join(logBaseDir, filepath.Join("logs", filepath.Join("server")))
	logdir := filepath.Join(logBaseDir, "logs")
	if err := os.MkdirAll(logdir, os.ModePerm); err != nil {
		fmt.Printf("error-3: %s\n", err.Error()) // Error: while creating logenv: %s\n", err.Error())
		return false
	}

	logfileNameList = make([]string, log_MAX_FILES)

	chanbuffLog = make(chan logmessage, 10)

	logFile := filepath.Join(logdir, log_FILE_NAME_PREFIX) + ".1"
	tmplogFile := filepath.Join(logdir, log_FILE_NAME_PREFIX)
	dummyLogfile = logFile + ".dummy"

	pServerLogFile, err = os.OpenFile(logFile, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		fmt.Printf("error-4: %s\n", err.Error()) //Error: while creating logfile: %s, error: %s\n", logFile, err.Error())
		return false
	}

	for i := int8(0); i < log_MAX_FILES; i++ {
		logfileNameList[i] = fmt.Sprintf("%s.%d", tmplogFile, i+1)
	}

	// if isLoggerInit == true: closes stderr so that error and panic logs can be captured in the logfile itself.
	if isLoggerInit {
		isLoggerInstanceInit = true
		if errDup2 := syscall.Dup2(int(pServerLogFile.Fd()), syscall.Stderr); errDup2 != nil {
			fmt.Printf("Error: Failed to reuse STDERR.\n")
		} else {
			fmt.Printf("Debug: Reused STDERR.\n")
		}

		if errDup2 := syscall.Dup2(int(pServerLogFile.Fd()), syscall.Stdout); errDup2 != nil {
			fmt.Printf("Error: Failed to reuse STDOUT.\n")
		} else {
			fmt.Printf("Debug: Reused STDOUT.\n")
		}
	}

	pDoneChanLock = &sync.Mutex{}

	isInit = true
	return true
}
