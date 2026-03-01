// Copyright (c) EnvSync Contributors. SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/envsync/envsync/internal/relay"
	"github.com/envsync/envsync/internal/ui"
	"github.com/spf13/cobra"
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade to a paid plan",
	Long:  "Shows current tier, usage, and opens Stripe Checkout in your browser.",
	RunE:  runUpgrade,
}

var upgradePlan string

func runUpgrade(cmd *cobra.Command, args []string) error {
	kp, err := loadIdentity()
	if err != nil {
		return err
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	teamID := generateTeamID(kp.Fingerprint)
	client := relay.NewClient(cfg.Relay.URL, kp)

	// Get current status
	status, err := client.GetTierStatus(teamID)
	if err != nil {
		ui.RenderError(ui.ErrRelayUnavailable(err.Error()))
		return err
	}

	ui.Header("EnvSync Plan")

	// Current tier
	table := ui.NewTable("", "Current", "Team ($29/mo)", "Enterprise ($199/mo)")
	table.AddRow("Members", fmtUsage(status.Usage.Members, status.Limits.Members), "unlimited", "unlimited")
	table.AddRow("Relay syncs/day", fmtUsage(status.Usage.BlobsToday, status.Limits.BlobsPerDay), "unlimited", "unlimited")
	table.AddRow("History", fmt.Sprintf("%dd", status.Limits.HistoryDays), "30 days", "365 days")
	table.AddRow("Cloud audit", "—", "✓", "✓")
	table.AddRow("SSO", "—", "—", "✓")
	fmt.Print(table.Render())
	ui.Blank()

	ui.Line(fmt.Sprintf("Current tier: %s", ui.StyleBold.Render(status.Tier)))
	ui.Blank()

	if status.Tier != "free" {
		ui.Success("You're already on a paid plan!")
		return nil
	}

	// Ask to upgrade
	plan := upgradePlan
	if plan == "" {
		plan = "team"
	}

	if !ui.ConfirmAction(fmt.Sprintf("Upgrade to %s plan?", plan), true) {
		ui.Line("Cancelled.")
		return nil
	}

	// Create checkout
	checkoutURL, err := client.CreateCheckout(teamID, plan)
	if err != nil {
		return fmt.Errorf("creating checkout: %w", err)
	}

	ui.Blank()
	ui.Line("Opening checkout in your browser...")
	openBrowser(checkoutURL)
	ui.Blank()
	ui.Code(fmt.Sprintf("  %s", checkoutURL))
	ui.Blank()
	ui.Line("Complete payment in your browser, then run:")
	ui.Code("  envsync upgrade")
	ui.Line("to verify your upgrade.")

	return nil
}

func fmtUsage(current, limit int) string {
	if limit < 0 {
		return fmt.Sprintf("%d / ∞", current)
	}
	return fmt.Sprintf("%d / %d", current, limit)
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}

func init() {
	upgradeCmd.Flags().StringVar(&upgradePlan, "plan", "team", "Plan to upgrade to (team or enterprise)")
	rootCmd.AddCommand(upgradeCmd)
}
