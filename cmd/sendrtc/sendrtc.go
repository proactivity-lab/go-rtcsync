// Author  Raido Pahtma
// License MIT

// sendrtc executable
package main

import "fmt"
import "time"
import "os"
import "os/signal"
import "strconv"
import "strings"

import "github.com/proactivity-lab/go-loggers"
import "github.com/proactivity-lab/go-moteconnection"
import "github.com/proactivity-lab/go-rtcsync"

import "github.com/jessevdk/go-flags"

const ApplicationVersionMajor = 0
const ApplicationVersionMinor = 4
const ApplicationVersionPatch = 0

var ApplicationBuildDate string
var ApplicationBuildDistro string

type Options struct {
	Positional struct {
		ConnectionString string `description:"Connectionstring sf@HOST:PORT or serial@PORT:BAUD"`
	} `positional-args:"yes"`

	Group       moteconnection.AMGroup `short:"g" long:"group" default:"22" description:"Packet AM Group (hex)"`
	Address     moteconnection.AMAddr  `short:"a" long:"address" default:"5678" description:"Source AM address (hex)"`
	Destination moteconnection.AMAddr  `short:"d" long:"destination" default:"0" description:"Destination AM address (hex)"`

	NtpHost string `short:"n" long:"ntp-host" default:"0.pool.ntp.org" description:"NTP server address"`

	Offset int64 `short:"o" long:"offset" default:"0"   description:"Time offset"`
	Period int   `short:"p" long:"period" default:"900" description:"Announcement period"`

	Quiet []string `long:"quiet" description:"Quiet periods HH:MM-HH:MM"`

	Debug       []bool `short:"D" long:"debug"   description:"Debug mode, print raw packets"`
	ShowVersion func() `short:"V" long:"version" description:"Show application version"`
}

func mainfunction() int {

	var opts Options
	opts.ShowVersion = func() {
		if ApplicationBuildDate == "" {
			ApplicationBuildDate = "YYYY-mm-dd_HH:MM:SS"
		}
		if ApplicationBuildDistro == "" {
			ApplicationBuildDistro = "unknown"
		}
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

	conn, cs, err := moteconnection.CreateConnection(opts.Positional.ConnectionString)
	if err != nil {
		fmt.Printf("ERROR: %s\n", err)
		os.Exit(1)
	}

	signals := make(chan os.Signal)
	signal.Notify(signals, os.Interrupt, os.Kill)

	ss := rtcsync.NewSyncSender(conn, opts.Address, opts.Group, opts.NtpHost)

	logger := loggers.BasicLogSetup(len(opts.Debug))
	if len(opts.Debug) > 0 {
		conn.SetLoggers(logger)
	}
	ss.SetLoggers(logger)

	for _, p := range opts.Quiet {
		times := strings.Split(p, "-")
		if len(times) != 2 {
			fmt.Printf("Invalid quiet period %s\n", p)
			os.Exit(1)
		}

		st := strings.Split(times[0], ":")
		et := strings.Split(times[1], ":")
		if len(st) != 2 || len(et) != 2 {
			fmt.Printf("Invalid start or end time for quiet period %s\n", p)
			os.Exit(1)
		}

		sth, err1 := strconv.Atoi(st[0])
		stm, err2 := strconv.Atoi(st[1])
		eth, err3 := strconv.Atoi(et[0])
		etm, err4 := strconv.Atoi(et[1])

		if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
			fmt.Printf("Invalid start or end time for quiet period %s\n", p)
			os.Exit(1)
		}

		ss.AddQuietPeriod(rtcsync.ClockTimePeriod{sth*3600 + stm*60, eth*3600 + etm*60})
	}

	conn.Autoconnect(30 * time.Second)

	time.Sleep(time.Second)
	if conn.Connected() {
		logger.Info.Printf("Connected with %s\n", cs)
	}

	go ss.Run(opts.Destination, time.Duration(opts.Period)*time.Second, opts.Offset)

	for interrupted := false; interrupted == false; {
		select {
		case sig := <-signals:
			signal.Stop(signals)
			logger.Debug.Printf("signal %s\n", sig)
			conn.Disconnect()
			interrupted = true
			ss.Exit <- true
		}
	}

	time.Sleep(100 * time.Millisecond)
	return 0
}

func main() {
	os.Exit(mainfunction())
}
