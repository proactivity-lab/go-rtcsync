// Author  Raido Pahtma
// License MIT

package rtcsync

import "fmt"
import "time"

import "github.com/proactivity-lab/go-loggers"
import "github.com/proactivity-lab/go-sfconnection"

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

	Exit chan bool
}

func NewSyncSender(sfc *sfconnection.SfConnection, source sfconnection.AMAddr, group sfconnection.AMGroup) *SyncSender {
	ss := new(SyncSender)
	ss.InitLoggers()

	ss.dsp = sfconnection.NewMessageDispatcher(sfconnection.NewMessage(group, source))

	ss.sfc = sfc
	ss.sfc.AddDispatcher(ss.dsp)

	ss.Exit = make(chan bool)

	return ss
}

func (self *SyncSender) AnnounceTime(destination sfconnection.AMAddr, offset int64) {
	m := new(RTCSyncMsg)
	m.Header = TIME_ANNOUNCEMENT_UNIX
	m.Stratum = 5 // Random
	m.Nxtime = time.Now().Unix() + offset

	msg := self.dsp.NewMessage()
	msg.SetDestination(destination)
	msg.SetType(AMID_RTC)
	msg.Payload = sfconnection.SerializePacket(m)

	self.Info.Printf("Announce %s->%s %d(%d)\n", msg.Source(), destination, m.Nxtime, offset)
	self.sfc.Send(msg)
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
