package logger

import (
	"log"
	"os"
)

var (
	Debug   *log.Logger
	Info    *log.Logger
	Warning *log.Logger
	Error   *log.Logger
	HTTP    *log.Logger
)

func Setup() {
	HTTP = log.New(os.Stdout, "[HTTP]\t", log.Ldate|log.Ltime)
	Info = log.New(os.Stdout, "[INFO]\t", log.Ldate|log.Ltime)
	Warning = log.New(os.Stdout, "[WARNING]\t", log.Ldate|log.Ltime|log.Lshortfile)
	Debug = log.New(os.Stdout, "[DEBUG]\t", log.Ldate|log.Ltime|log.Lshortfile)
	Error = log.New(os.Stdout, "[ERROR]\t", log.Ldate|log.Ltime|log.Lshortfile)
}
