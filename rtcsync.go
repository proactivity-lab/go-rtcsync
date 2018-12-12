// Author  Raido Pahtma
// License MIT

package rtcsync

import "fmt"
import "time"

import "github.com/proactivity-lab/go-loggers"
import "github.com/proactivity-lab/go-moteconnection"
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

type ClockTimePeriod struct {
	Start int
	End   int
}

type SyncSender struct {
	loggers.DIWEloggers

	conn moteconnection.MoteConnection
	dsp  moteconnection.Dispatcher

	host string

	quiet []ClockTimePeriod

	Exit chan bool
}

func NewSyncSender(conn moteconnection.MoteConnection, source moteconnection.AMAddr, group moteconnection.AMGroup, host string) *SyncSender {
	ss := new(SyncSender)
	ss.InitLoggers()

	ss.dsp = moteconnection.NewMessageDispatcher(moteconnection.NewMessage(group, source))

	ss.conn = conn
	ss.conn.AddDispatcher(ss.dsp)

	ss.host = host

	ss.Exit = make(chan bool)

	return ss
}

func (self *SyncSender) AddQuietPeriod(qp ClockTimePeriod) {
	self.Debug.Printf("Quiet period %d-%d\n", qp.Start, qp.End)
	self.quiet = append(self.quiet, qp)
}

func (self *SyncSender) AnnounceTime(destination moteconnection.AMAddr, ntpr *ntp.Response, offset int64) {
	m := new(RTCSyncMsg)
	m.Header = TIME_ANNOUNCEMENT_UNIX
	m.Stratum = ntpr.Stratum + 1
	m.Nxtime = ntpr.Time.Unix() + offset

	msg := self.dsp.NewPacket().(*moteconnection.Message)
	msg.SetDestination(destination)
	msg.SetType(AMID_RTC)
	msg.Payload = moteconnection.SerializePacket(m)

	self.Info.Printf("Announce %s->%s %d(%d)\n", msg.Source(), destination, m.Nxtime, offset)
	self.conn.Send(msg)
}

func (self *SyncSender) QuietPeriod(t time.Time) bool {
	tu := t.UTC()
	tss := tu.Hour()*3600 + tu.Minute()*60 + tu.Second()
	self.Debug.Printf("Quiet period check: %d\n", tss)
	for _, qp := range self.quiet {
		if qp.Start <= tss && tss <= qp.End {
			return true
		}
	}
	return false
}

func (self *SyncSender) Run(destination moteconnection.AMAddr, period time.Duration, offset int64) {
	self.Debug.Printf("run\n")
	announcePeriod := 1 * time.Second
	for {
		select {
		case <-self.Exit:
			self.Debug.Printf("Exit.\n")
			self.conn.Disconnect()
		case <-time.After(announcePeriod):
			announcePeriod = period
			if ntpr, err := ntp.Query(self.host); err == nil {
				if err := ntpr.Validate(); err == nil {
					self.Debug.Printf("NTP %d stratum %d RTT %s offset %s", ntpr.Time.Unix(), ntpr.Stratum, ntpr.RTT, ntpr.ClockOffset)
					if self.QuietPeriod(ntpr.Time) {
						self.Info.Printf("Quiet period, not sending time sync.\n")
					} else {
						self.AnnounceTime(destination, ntpr, offset)
					}
				} else {
					self.Warning.Printf("NTP response validation failed %s", err)
				}
			} else {
				self.Warning.Printf("NTP query failed %s", err)
			}
		}
	}
}
