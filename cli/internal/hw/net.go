package hw

import (
	"fmt"

	"github.com/anibalnet/blackbeard/cli/internal/ui"
	psnet "github.com/shirou/gopsutil/v4/net"
)

// RunNet shows network I/O counters per interface.
func RunNet(p *ui.Printer) error {
	p.Header("Network I/O")

	counters, err := psnet.IOCounters(true)
	if err != nil {
		return fmt.Errorf("reading network counters: %w", err)
	}

	table := ui.NewTable(p.Out, "INTERFACE", "RX", "TX", "RX PKT", "TX PKT", "RX ERR", "TX ERR")

	for _, c := range counters {
		if c.Name == "lo" {
			continue
		}
		table.Row(
			c.Name,
			formatBytesHW(c.BytesRecv),
			formatBytesHW(c.BytesSent),
			fmt.Sprintf("%d", c.PacketsRecv),
			fmt.Sprintf("%d", c.PacketsSent),
			fmt.Sprintf("%d", c.Errin),
			fmt.Sprintf("%d", c.Errout),
		)
	}
	table.Flush()
	return nil
}
