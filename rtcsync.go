// Author  Raido Pahtma
// License MIT

package rtcsync

import "fmt"
import "time"

import "github.com/proactivity-lab/go-loggers"
import "github.com/proactivity-lab/go-sfconnection"
import "github.com/beevik/ntp"

const AMID_RTC = 0x80

const TIME_ANNOUNCEMENT_UNIX = 1
const STARTUP_DELAY = 3 * time.Second

type RTCSyncMsg struct {
	Header  uint8
	Stratum uint8
	Nxtime  int64
}

func (self *RTCSyncMsg) String() string {
	return fmt.Sprintf("ta %d", self.Nxtime)
}

type SyncSender struct {
	loggers.DIWEloggers

	sfc *sfconnection.SfConnection
	dsp *sfconnection.MessageDispatcher

	host string

	Exit chan bool
}

func NewSyncSender(sfc *sfconnection.SfConnection, source sfconnection.AMAddr, group sfconnection.AMGroup, host string) *SyncSender {
	ss := new(SyncSender)
	ss.InitLoggers()

	ss.dsp = sfconnection.NewMessageDispatcher(sfconnection.NewMessage(group, source))

	ss.sfc = sfc
	ss.sfc.AddDispatcher(ss.dsp)

	ss.host = host

	ss.Exit = make(chan bool)

	return ss
}

func (self *SyncSender) AnnounceTime(destination sfconnection.AMAddr, offset int64) {
	if ntpr, err := ntp.Query(self.host, 4); err == nil {
		self.Debug.Printf("NTP %d stratum %d RTT %s offset %s", ntpr.Time.Unix(), ntpr.Stratum, ntpr.RTT, ntpr.ClockOffset)
		m := new(RTCSyncMsg)
		m.Header = TIME_ANNOUNCEMENT_UNIX
		m.Stratum = ntpr.Stratum + 1
		m.Nxtime = ntpr.Time.Unix() + offset

		msg := self.dsp.NewMessage()
		msg.SetDestination(destination)
		msg.SetType(AMID_RTC)
		msg.Payload = sfconnection.SerializePacket(m)

		self.Info.Printf("Announce %s->%s %d(%d)\n", msg.Source(), destination, m.Nxtime, offset)
		self.sfc.Send(msg)
	} else {
		self.Warning.Printf("NTP query failed %s", err)
	}
}

func (self *SyncSender) Run(destination sfconnection.AMAddr, period time.Duration, offset int64) {
	self.Debug.Printf("run\n")
	self.AnnounceTime(destination, offset)
	for {
		select {
		case <-self.Exit:
			self.Debug.Printf("Exit.\n")
			self.sfc.Disconnect()
		case <-time.After(period):
			self.AnnounceTime(destination, offset)
		}
	}
}
