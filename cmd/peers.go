package cmd

import (
	"fmt"

	"github.com/envsync/envsync/internal/peer"
	"github.com/spf13/cobra"
)

var peersCmd = &cobra.Command{
	Use:   "peers",
	Short: "List team peers",
	Long:  "Shows all known peers in your teams with their trust status.",
	RunE:  runPeers,
}

func runPeers(cmd *cobra.Command, args []string) error {
	registry, err := peer.NewRegistry()
	if err != nil {
		return err
	}

	teams, err := registry.ListTeams()
	if err != nil {
		return err
	}

	if len(teams) == 0 {
		fmt.Println()
		fmt.Println("  No teams found. Start one with:")
		fmt.Println("    envsync invite @teammate")
		fmt.Println()
		return nil
	}

	fmt.Println()
	for _, teamID := range teams {
		team, err := registry.LoadTeam(teamID)
		if err != nil {
			continue
		}

		fmt.Printf("  ✦ Team: %s\n", team.Name)
		fmt.Println()

		peers, err := registry.ListPeers(teamID)
		if err != nil {
			continue
		}

		if len(peers) == 0 {
			fmt.Println("    No peers yet.")
		} else {
			fmt.Print("    ")
			fmt.Printf("%-2s %-20s %-44s %s\n", "", "User", "Fingerprint", "Status")
			fmt.Print("    ")
			fmt.Println("── ──────────────────── ──────────────────────────────────────────── ────────")
			for _, p := range peers {
				username := p.GitHubUsername
				if username == "" {
					username = "(unknown)"
				}
				fingerprint := p.Fingerprint
				if len(fingerprint) > 44 {
					fingerprint = fingerprint[:44]
				}
				fmt.Print("    ")
				fmt.Printf("%s  %-20s %-44s %s\n", p.StatusIcon(), "@"+username, fingerprint, p.Trust)
			}
		}
		fmt.Println()
	}

	return nil
}

func init() {
	rootCmd.AddCommand(peersCmd)
}
