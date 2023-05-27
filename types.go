package logger

type logmessage struct {
	componentFlag int8
	component     string
	logmsg        string
}

type LogConfig struct {
	SrcBaseDir      string `json:"srcBaseDir"`
	FileSize        int    `json:"fileSize"`
	MaxFilesCnt     int    `json:"maxFilesCnt"`
	DefaultLogLevel string `json:"defaultLogLevel"`
}

type loglevel struct {
	str   string
	color string
	wt    uint8
}
