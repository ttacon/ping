package ping

import (
	"errors"
	"os"
	"os/exec"
	"regexp"
	"time"
)

// PingStatistics represents the round-trip minimum, average and maximum time
// along with the standard deviation. It also represents the number of packets
// sent and received as well as the percentage packet loss.
type PingStatistics struct {
	// Timing statistics
	RoundTripMin    string
	RoundTripAvg    string
	RoundTripMax    string
	RoundTripStdDev string

	// Packet # statistics
	PacketsSent     string
	PacketsReceived string
	PacketLoss      string
}

// AsMap returns the statistics for a given set of pings as a map of
// string to string.
func (p PingStatistics) AsMap() map[string]string {
	return map[string]string{
		"roundTripMin":    p.RoundTripMin,
		"roundTripAvg":    p.RoundTripAvg,
		"roundTripMax":    p.RoundTripMax,
		"roundTripStdDev": p.RoundTripStdDev,
		"packetsSent":     p.PacketsSent,
		"packetsReceived": p.PacketsReceived,
		"packetLoss":      p.PacketLoss,
	}
}

var emptyPing = PingStatistics{}

// PingExec pings the specifie host for the given number of seconds. It returns
// a the ping statistics for that given time period.
func PingExec(host string, pingDuration int) (PingStatistics, error) {
	cmd := exec.Command("ping", host)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return emptyPing, err
	}

	err = cmd.Start()
	if err != nil {
		return emptyPing, err
	}

	<-time.After(time.Second * time.Duration(pingDuration)) // ugh... ugly hack

	err = cmd.Process.Signal(os.Interrupt)
	if err != nil {
		return emptyPing, err
	}

	<-time.After(time.Second) // ugly hack to ensure we get the statistic line

	// TODO(ttacon): make this dependent on ping "duration"
	buf := make([]byte, 8192) // should be enough for this time period...right?
	n, err := stdout.Read(buf)
	if err != nil {
		return emptyPing, err
	}

	err = cmd.Wait()
	if err != nil {
		return emptyPing, err
	}

	tStats := timingStatsFromPing(string(buf[0:n]))
	pStats := packetStatsFromPing(string(buf[0:n]))

	if tStats == nil || pStats == nil {
		return emptyPing, errors.New("failed to parse ping output")
	}

	return PingStatistics{
		RoundTripMin:    tStats[0],
		RoundTripAvg:    tStats[1],
		RoundTripMax:    tStats[2],
		RoundTripStdDev: tStats[3],
		PacketsSent:     pStats[0],
		PacketsReceived: pStats[1],
		PacketLoss:      pStats[2],
	}, nil
}

////////// Retrieve info from ping output //////////
var (
	statsRegexp      = regexp.MustCompile(`(\d+.\d+)/(\d+.\d+)/(\d+.\d+)/(\d+.\d+)`)
	packetLossRegexp = regexp.MustCompile(
		`(\d+) packets transmitted, (\d+) packets received, (\d+.\d+)% packet loss`)
)

// statsFromPing retrieves the data from the last line of ping output
// e.g. round-trip min/avg/max/stddev = 110.555/112.243/113.908/1.307 ms
func timingStatsFromPing(str string) []string {
	found := statsRegexp.FindAllStringSubmatch(str, -1)
	if len(found) == 0 {
		return nil
	}
	vals := found[0]
	return []string{vals[1], vals[2], vals[3], vals[4]}
}

// packetStatsFromPing retrieves the data from the second to last line
// of output.
// e.g. 5 packets transmitted, 5 packets received, 0.0% packet loss
func packetStatsFromPing(str string) []string {
	found := packetLossRegexp.FindAllStringSubmatch(str, -1)
	if len(found) == 0 {
		return nil
	}
	vals := found[0]
	return []string{vals[1], vals[2], vals[3]}
}
