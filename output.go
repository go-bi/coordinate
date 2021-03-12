package main

import (
	"fmt"
	"log"
	"os"
	"time"

	. "github.com/logrusorgru/aurora"
)

var (
	logger  *log.Logger
	tabs    string
)

func InitLogger() {
	logger = log.New(os.Stdout, "", 0)
}

func Time(t time.Time) string {
	return t.Format("03:04:05 PM")
}

func Tabber(tabnum int) {
	tabs = ""
	for i := 0; i < tabnum; i++ {
		tabs += "\t"
	}
}

func Alert(a ...interface{}) {
    logger.Printf("%s%s %s", tabs, BrightMagenta("[ALERT]"), fmt.Sprintln(a...))
}

func Stdout(a ...interface{}) {
    logger.Printf("%s%s %s", tabs, BrightCyan("[STDOUT]"), fmt.Sprintln(a...))
}

func Stderr(a ...interface{}) {
    logger.Printf("%s%s %s", tabs, BrightMagenta("[STDERR]"), fmt.Sprintln(a...))
}

func Crit(i Instance, m Module, s Script, a ...interface{}) {
    logger.Printf("%s%s:%s%s %s", tabs, Red("[CRIT"), Summary(i, m, s), Red("]"), fmt.Sprintln(a...))
}

func Err(a ...interface{}) {
	logger.Printf("%s%s %s", tabs, BrightRed("[ERROR]"), fmt.Sprintln(a...))
}

func Fatal(a ...interface{}) {
	logger.Printf("%s%s %s", tabs, BrightRed("[FATAL]"), fmt.Sprintln(a...))
    os.Exit(1)
}

func PrintRed(i Instance, m Module, s Script, a ...interface{}) {
    logger.Printf("%s%s:%s%s %s", tabs, BrightRed("[!"), Summary(i, m, s), BrightCyan("]"), fmt.Sprintln(a...))
}

func PrintGreen(i Instance, m Module, s Script, a ...interface{}) {
    logger.Printf("%s%s:%s%s %s", tabs, BrightGreen("[+"), Summary(i, m, s), BrightCyan("]"), fmt.Sprintln(a...))
}

func Warning(a ...interface{}) {
	logger.Printf("%s%s %s", tabs, Yellow("[WARN]"), fmt.Sprintln(a...))
}

func Info(a ...interface{}) {
    logger.Printf("%s%s %s", tabs, BrightCyan("[INFO]"), fmt.Sprintln(a...))
}

func InfoExtra(i Instance, m Module, s Script, a ...interface{}) {
    logger.Printf("%s%s:%s%s %s", tabs, BrightCyan("[INFO"), Summary(i, m, s), BrightCyan("]"), fmt.Sprintln(a...))
}

func Debug(a ...interface{}) {
    if c.Verbose {
        logger.Printf("%s%s %s", tabs, Cyan("[DEBUG]"), fmt.Sprintln(a...))
    }
}

func Summary(i Instance, m Module, s Script) string {
    return fmt.Sprintf("%s:%s:%s/%s", Blue(i.Id), BrightRed(i.Ip), BrightGreen(m.Name), BrightBlue(s.Name))
}

func Notice(a ...interface{}) {
	logger.Printf("%s%s %s", tabs, BrightCyan("[NOTICE]"), fmt.Sprintln(a...))
}

func Positive(a ...interface{}) {
	logger.Printf("%s%s %s", tabs, Green("[+]"), fmt.Sprintln(a...))
}

