package log

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

func Init(file string) {
	logrus.SetFormatter(&prefixed.TextFormatter{
		DisableSorting: true,
		FullTimestamp: true,
		ForceFormatting: true,
	})
	logrus.SetLevel(logrus.DebugLevel)
	// logrus.SetReportCaller(true)
	f, err := os.OpenFile(file, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		logrus.SetOutput(f)
	} else {
		logrus.Warnf("log %v, using default stderr", err)
	}
}

func Debugf(format string, args ...interface{}) {
	logrus.Debugf(format, args...)
}

func Infof(format string, args ...interface{}) {
	fmt.Printf(format + "\n", args...)
	logrus.Infof(format, args...)
}

func Warnf(format string, args ...interface{}) {
	fmt.Printf(format + "\n", args...)
	logrus.Warnf(format, args...)
}

func Errorf(format string, args ...interface{}) {
	fmt.Printf(format + "\n", args...)
	logrus.Errorf(format, args...)
}


func Fatalf(format string, args ...interface{}) {
	fmt.Printf(format, args...)
	logrus.Fatalf(format, args...)
}

func Debug(args ...interface{}) {
	logrus.Debugln(args...)
}

func Info(args ...interface{}) {
	fmt.Println(args...)
	logrus.Infoln(args...)
}

func Warn(args ...interface{}) {
	fmt.Println(args...)
	logrus.Warnln(args...)
}

func Error(args ...interface{}) {
	fmt.Println(args...)
	logrus.Errorln(args...)
}

func Fatal(args ...interface{}) {
	fmt.Println(args...)
	logrus.Fatalln(args...)
}