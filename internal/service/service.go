package service

import (
	"flag"
	"os"

	log "github.com/sirupsen/logrus"
)

var logFileHandle *os.File

// InitService initialize the microservice
// It does the following:
//   - initialize config file (passed in as command line arg)
//   - Setup Logging
//   - Connect to DB & setup ORM
func InitService() (string, int, bool, bool) {
	currentWorkingDir, err := os.Getwd()
	if err != nil {
		log.Panicf("unable to get working directory: %s", err)
	}

	var rootDir string
	var logFile string
	var port int
	var allowUpload bool
	var isCli bool
	flag.StringVar(&rootDir, "root", currentWorkingDir, "root directory to serve files")
	flag.IntVar(&port, "port", 8000, "port to listen on")
	flag.BoolVar(&isCli, "cli", false, "run service in CLI")
	flag.BoolVar(&allowUpload, "allowUpload", false, "allow upload to server")
	flag.StringVar(&logFile, "logFile", "", "log file")
	flag.Parse()

	if logFile != "" {
		log.SetFormatter(&log.JSONFormatter{})
		logFileHandle, err := os.OpenFile(logFile,
			os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			log.WithFields(log.Fields{
				"file":  logFile,
				"error": err,
			}).Fatal("unable to open file")
		}
		log.WithFields(log.Fields{
			"file": logFile,
		}).Info("switching log output to file")
		log.SetOutput(logFileHandle)
	}

	log.SetLevel(log.DebugLevel)

	log.Infof("using port: %d", port)
	log.Infof("CLI: %t", isCli)
	// // set log level
	// switch strings.ToLower(c.AppConf.Log.Level) {
	// case "trace":
	// 	log.SetLevel(log.TraceLevel)
	// case "debug":
	// 	log.SetLevel(log.DebugLevel)
	// case "info":
	// 	log.SetLevel(log.InfoLevel)
	// case "warn":
	// 	log.SetLevel(log.WarnLevel)
	// case "error":
	// 	log.SetLevel(log.ErrorLevel)
	// case "fatal":
	// 	log.SetLevel(log.FatalLevel)
	// case "panic":
	// 	log.SetLevel(log.PanicLevel)
	// }

	return rootDir, port, allowUpload, !isCli
}

// Shutdown closes any open files or pipes the microservice started
// It does the following:
//   - Disconnect to DB
func Shutdown() {

	if logFileHandle != nil {
		// Revert logging back to StdOut
		log.SetOutput(os.Stdout)
		logFileHandle.Close()
	}
}
