package zeusmaster

import (
	"os"
	"os/signal"
	"syscall"

	slog "github.com/burke/zeus/go/shinylog"
	"github.com/burke/zeus/go/unixsocket"
)

var exitNow chan int
var exitStatus int = 0

func ExitNow(code int) {
	exitNow <- code
}

func Run(color bool) {
	if !color {
		slog.DisableColor()
		DisableErrorColor()
	}
	slog.StartingZeus()

	var tree *ProcessTree = BuildProcessTree()

	exitNow = make(chan int)

	localMasterFile, remoteMasterFile, err := unixsocket.Socketpair(syscall.SOCK_DGRAM)
	if err != nil {
		panic(err)
	}

	localMasterSocket, err := unixsocket.NewUsockFromFile(localMasterFile)
	if err != nil {
		panic(err)
	}

	quit1 := make(chan bool)
	quit2 := make(chan bool)
	quit3 := make(chan bool)

	go StartSlaveMonitor(tree, localMasterSocket, remoteMasterFile, quit1)
	go StartClientHandler(tree, quit2)
	go StartFileMonitor(tree, quit3)

	quit := make(chan bool, 1)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		// FIXME: Unprecedented levels of jank, right here.
		terminateComponents(quit1, quit2, quit3, quit)
	}()

	go func() {
		exitStatus = <-exitNow
		terminateComponents(quit1, quit2, quit3, quit)
	}()

	<-quit
	<-quit
	<-quit

	os.Exit(exitStatus)
}

func terminateComponents(quit1, quit2, quit3, quit chan bool) {
	SuppressErrors()
	slog.Suppress()
	go func() {
		quit1 <- true
		<-quit1
		quit <- true
	}()
	go func() {
		quit2 <- true
		<-quit2
		quit <- true
	}()
	go func() {
		quit3 <- true
		<-quit3
		quit <- true
	}()
}
