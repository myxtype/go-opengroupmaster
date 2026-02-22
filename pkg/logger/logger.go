package logger

import (
	"log"
	"os"
)

func New() *log.Logger {
	return log.New(os.Stdout, "[groupmaster] ", log.LstdFlags|log.Lshortfile)
}
