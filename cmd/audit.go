package cmd

import (
	"fmt"

	"github.com/envsync/envsync/internal/audit"
	"github.com/envsync/envsync/internal/ui"
	"github.com/spf13/cobra"
)

var auditCmd = &cobra.Command{
	Use:   "audit",
	Short: "View sync audit log",
	Long:  "Displays recent sync events with timestamps, peers, and details.",
	RunE:  runAudit,
}

var (
	auditLast  int
	auditPeer  string
	auditEvent string
)

func runAudit(cmd *cobra.Command, args []string) error {
	logger, err := audit.NewLogger()
	if err != nil {
		return err
	}

	var entries []audit.Entry

	switch {
	case auditPeer != "":
		entries, err = logger.FilterByPeer(auditPeer, auditLast)
	case auditEvent != "":
		entries, err = logger.FilterByEvent(audit.EventType(auditEvent), auditLast)
	default:
		entries, err = logger.Read(auditLast)
	}

	if err != nil {
		return err
	}

	if len(entries) == 0 {
		ui.Header("Audit Log")
		ui.Line("No events recorded yet.")
		ui.Blank()
		return nil
	}

	ui.Header("Audit Log")

	table := ui.NewTable("Time", "Event", "Peer", "File", "Method", "Details")
	for _, e := range entries {
		ts := e.Timestamp.Format("01-02 15:04")
		peer := e.Peer
		if peer == "" {
			peer = "—"
		}
		file := e.File
		if file == "" {
			file = "—"
		}
		method := e.Method
		if method == "" {
			method = "—"
		}
		details := e.Details
		if details == "" && e.VarsChanged > 0 {
			details = fmt.Sprintf("%d vars", e.VarsChanged)
		}
		if details == "" {
			details = "—"
		}
		table.AddRow(ts, string(e.Event), peer, file, method, details)
	}
	fmt.Print(table.Render())
	ui.Blank()

	return nil
}

func init() {
	auditCmd.Flags().IntVar(&auditLast, "last", 20, "Show last N events")
	auditCmd.Flags().StringVar(&auditPeer, "peer", "", "Filter by peer @username")
	auditCmd.Flags().StringVar(&auditEvent, "event", "", "Filter by event type (push, pull, invite, etc.)")
	rootCmd.AddCommand(auditCmd)
}
