// Author  Raido Pahtma
// License MIT

package main

import "fmt"
import "time"
import "os"
import "os/signal"

import "github.com/proactivity-lab/go-loggers"
import "github.com/proactivity-lab/go-sfconnection"
import "github.com/proactivity-lab/go-rtcsync"

import "github.com/jessevdk/go-flags"

const ApplicationVersionMajor = 0
const ApplicationVersionMinor = 1
const ApplicationVersionPatch = 0

var ApplicationBuildDate string
var ApplicationBuildDistro string

type Options struct {
	Positional struct {
		Host string
		Port uint16
	} `positional-args:"yes" required:"yes"  description:"Host and port"`

	Destination sfconnection.AMAddr `short:"d" long:"destination" default:"FFFF" description:"Sync destination"`

	Address sfconnection.AMAddr  `short:"a" long:"address" default:"0015" description:"Local address"`
	Group   sfconnection.AMGroup `short:"g" long:"group"   default:"22"   description:"Packet AM Group (hex)"`

	Offset int64 `short:"o" long:"offset" default:"0"   description:"Time offset"`
	Period int   `short:"p" long:"period" default:"900" description:"Announcement period"`

	Debug       []bool `short:"D" long:"debug"   description:"Debug mode, print raw packets"`
	ShowVersion func() `short:"V" long:"version" description:"Show application version"`
}

func mainfunction() int {
	var opts Options
	opts.ShowVersion = func() {
		fmt.Printf("sendrtc %d.%d.%d (%s %s)\n", ApplicationVersionMajor, ApplicationVersionMinor, ApplicationVersionPatch, ApplicationBuildDate, ApplicationBuildDistro)
		os.Exit(0)
	}

	_, err := flags.Parse(&opts)
	if err != nil {
		flagserr := err.(*flags.Error)
		if flagserr.Type != flags.ErrHelp {
			if len(opts.Debug) > 0 {
				fmt.Printf("Argument parser error: %s\n", err)
			}
			return 1
		}
		return 0
	}

	signals := make(chan os.Signal)
	signal.Notify(signals, os.Interrupt, os.Kill)

	sfc := sfconnection.NewSfConnection()
	ss := rtcsync.NewSyncSender(sfc, opts.Address, opts.Group)

	logger := loggers.BasicLogSetup(len(opts.Debug))
	if len(opts.Debug) > 0 {
		sfc.SetLoggers(logger)
	}
	ss.SetLoggers(logger)

	sfc.Autoconnect(opts.Positional.Host, opts.Positional.Port, 30*time.Second)

	time.Sleep(time.Second)

	go ss.Run(opts.Destination, time.Duration(opts.Period)*time.Second, opts.Offset)

	for interrupted := false; interrupted == false; {
		select {
		case sig := <-signals:
			signal.Stop(signals)
			logger.Debug.Printf("signal %s\n", sig)
			ss.Exit <- true
			interrupted = true
		}
	}

	time.Sleep(100 * time.Millisecond)
	return 0
}

func main() {
	os.Exit(mainfunction())
}
