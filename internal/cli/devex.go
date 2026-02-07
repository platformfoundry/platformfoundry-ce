package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/platformfoundry/pf-ce/internal/devex"
	"github.com/spf13/cobra"
)

var devexManager *devex.Manager

func init() {
	devexManager = devex.NewManager()
}

var devexCmd = &cobra.Command{
	Use:   "devex",
	Short: "Developer experience analytics",
	Long: `Track and analyze developer experience metrics.

Includes DORA metrics, platform adoption, friction point analysis,
and developer journey tracking.`,
}

var devexDORACmd = &cobra.Command{
	Use:   "dora [team]",
	Short: "Show DORA metrics",
	Long:  `Display DORA (DevOps Research and Assessment) metrics for a team or organization.`,
	RunE:  runDevexDORA,
}

var devexAdoptionCmd = &cobra.Command{
	Use:   "adoption",
	Short: "Show platform adoption metrics",
	RunE:  runDevexAdoption,
}

var devexFrictionCmd = &cobra.Command{
	Use:   "friction",
	Short: "Show developer friction points",
	RunE:  runDevexFriction,
}

var devexReportCmd = &cobra.Command{
	Use:   "report [team]",
	Short: "Generate developer experience report",
	RunE:  runDevexReport,
}

var devexTeamsCmd = &cobra.Command{
	Use:   "teams",
	Short: "List teams with metrics",
	RunE:  runDevexTeams,
}

var devexScoreCmd = &cobra.Command{
	Use:   "score [team]",
	Short: "Show developer experience score",
	RunE:  runDevexScore,
}

func init() {
	devexCmd.AddCommand(devexDORACmd)
	devexCmd.AddCommand(devexAdoptionCmd)
	devexCmd.AddCommand(devexFrictionCmd)
	devexCmd.AddCommand(devexReportCmd)
	devexCmd.AddCommand(devexTeamsCmd)
	devexCmd.AddCommand(devexScoreCmd)

	devexReportCmd.Flags().String("format", "text", "Output format (text, json)")
}

func runDevexDORA(cmd *cobra.Command, args []string) error {
	team := ""
	if len(args) > 0 {
		team = args[0]
	}

	dora, err := devexManager.GetDORAMetrics(team)
	if err != nil {
		return err
	}

	if team != "" {
		fmt.Printf("DORA Metrics - %s\n", team)
	} else {
		fmt.Println("DORA Metrics - Organization")
	}
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("\nOverall Rating: %s\n\n", formatRating(dora.Rating))

	// Deployment Frequency
	if dora.DeploymentFrequency != nil {
		df := dora.DeploymentFrequency
		fmt.Println("Deployment Frequency")
		fmt.Println(strings.Repeat("-", 40))
		fmt.Printf("  Frequency:    %.1f deploys/day\n", df.Value)
		fmt.Printf("  Total:        %d deployments (30d)\n", df.TotalDeployments)
		fmt.Printf("  Rating:       %s\n", formatRating(df.Rating))
		if len(df.ByEnvironment) > 0 {
			fmt.Println("  By Environment:")
			for env, count := range df.ByEnvironment {
				fmt.Printf("    %-12s %d\n", env+":", count)
			}
		}
		fmt.Println()
	}

	// Lead Time
	if dora.LeadTime != nil {
		lt := dora.LeadTime
		fmt.Println("Lead Time for Changes")
		fmt.Println(strings.Repeat("-", 40))
		fmt.Printf("  Average:      %s\n", formatDevexDuration(lt.Value))
		fmt.Printf("  P50:          %s\n", formatDevexDuration(lt.P50))
		fmt.Printf("  P90:          %s\n", formatDevexDuration(lt.P90))
		fmt.Printf("  P95:          %s\n", formatDevexDuration(lt.P95))
		fmt.Printf("  Rating:       %s\n", formatRating(lt.Rating))
		if lt.Breakdown != nil {
			fmt.Println("  Time Breakdown:")
			fmt.Printf("    Code Review: %s\n", formatDevexDuration(lt.Breakdown.CodeReview))
			fmt.Printf("    Build:       %s\n", formatDevexDuration(lt.Breakdown.Build))
			fmt.Printf("    Test:        %s\n", formatDevexDuration(lt.Breakdown.Test))
			fmt.Printf("    Staging:     %s\n", formatDevexDuration(lt.Breakdown.Staging))
			fmt.Printf("    Approval:    %s\n", formatDevexDuration(lt.Breakdown.Approval))
			fmt.Printf("    Production:  %s\n", formatDevexDuration(lt.Breakdown.Production))
		}
		fmt.Println()
	}

	// Change Failure Rate
	if dora.ChangeFailureRate != nil {
		cfr := dora.ChangeFailureRate
		fmt.Println("Change Failure Rate")
		fmt.Println(strings.Repeat("-", 40))
		fmt.Printf("  Rate:         %.1f%%\n", cfr.Value)
		fmt.Printf("  Total:        %d changes\n", cfr.TotalChanges)
		fmt.Printf("  Failed:       %d changes\n", cfr.FailedChanges)
		fmt.Printf("  Rating:       %s\n", formatRating(cfr.Rating))
		if len(cfr.TopFailures) > 0 {
			fmt.Println("  Top Failure Categories:")
			for _, f := range cfr.TopFailures {
				fmt.Printf("    %-15s %d (%.1f%%)\n", f.Category+":", f.Count, f.Percentage)
			}
		}
		fmt.Println()
	}

	// Time to Restore
	if dora.TimeToRestore != nil {
		ttr := dora.TimeToRestore
		fmt.Println("Time to Restore Service")
		fmt.Println(strings.Repeat("-", 40))
		fmt.Printf("  Average:      %s\n", formatDevexDuration(ttr.Value))
		fmt.Printf("  P50:          %s\n", formatDevexDuration(ttr.P50))
		fmt.Printf("  P90:          %s\n", formatDevexDuration(ttr.P90))
		fmt.Printf("  Incidents:    %d\n", ttr.Incidents)
		fmt.Printf("  Rating:       %s\n", formatRating(ttr.Rating))
	}

	return nil
}

func runDevexAdoption(cmd *cobra.Command, args []string) error {
	adoption := devexManager.GetPlatformAdoption()

	fmt.Println("Platform Adoption Metrics")
	fmt.Println(strings.Repeat("=", 60))

	fmt.Printf("\nSelf-Service Ratio:      %.1f%%\n", adoption.SelfServiceRatio)
	fmt.Printf("Golden Path Adoption:    %.1f%%\n", adoption.GoldenPathAdoption)
	fmt.Printf("Automated Deployments:   %.1f%%\n", adoption.AutomatedDeployments)
	fmt.Println()
	fmt.Printf("Active Users:            %d\n", adoption.ActiveUsers)
	fmt.Printf("Total Applications:      %d\n", adoption.TotalApplications)

	if len(adoption.FeatureUsage) > 0 {
		fmt.Println("\nFeature Usage:")
		for feature, count := range adoption.FeatureUsage {
			pct := float64(count) / float64(adoption.TotalApplications) * 100
			bar := strings.Repeat("█", int(pct/5))
			fmt.Printf("  %-15s %2d apps (%5.1f%%) %s\n", feature+":", count, pct, bar)
		}
	}

	return nil
}

func runDevexFriction(cmd *cobra.Command, args []string) error {
	frictionPoints := devexManager.GetFrictionPoints()

	if len(frictionPoints) == 0 {
		fmt.Println("No friction points identified")
		return nil
	}

	fmt.Println("Developer Friction Points")
	fmt.Println(strings.Repeat("=", 60))

	for i, fp := range frictionPoints {
		impact := formatImpact(fp.Impact)
		fmt.Printf("\n%d. [%s] %s\n", i+1, impact, fp.Category)
		fmt.Printf("   %s\n", fp.Description)
		fmt.Printf("   Occurrences: %d\n", fp.Occurrences)
		if fp.Suggestion != "" {
			fmt.Printf("   Suggestion: %s\n", fp.Suggestion)
		}
		fmt.Printf("   Detected: %s\n", fp.DetectedAt.Format("2006-01-02"))
	}

	return nil
}

func runDevexReport(cmd *cobra.Command, args []string) error {
	team := ""
	if len(args) > 0 {
		team = args[0]
	}

	report, err := devexManager.GenerateReport(cmd.Context(), team)
	if err != nil {
		return err
	}

	fmt.Println("╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║            DEVELOPER EXPERIENCE REPORT                         ║")
	fmt.Println("╠════════════════════════════════════════════════════════════════╣")
	if team != "" {
		fmt.Printf("║  Team: %-55s ║\n", team)
	} else {
		fmt.Printf("║  Scope: %-54s ║\n", "Organization-wide")
	}
	fmt.Printf("║  Period: %-53s ║\n", report.Period)
	fmt.Printf("║  Generated: %-50s ║\n", report.GeneratedAt.Format(time.RFC3339))
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")

	// Overall Score
	if report.Score != nil {
		fmt.Println("\nOVERALL SCORE")
		fmt.Println(strings.Repeat("-", 50))
		scoreBar := strings.Repeat("█", int(report.Score.Overall))
		fmt.Printf("  Score: %.1f/5.0 %s\n", report.Score.Overall, scoreBar)
		fmt.Println("\n  Category Breakdown:")
		for cat, score := range report.Score.Categories {
			bar := strings.Repeat("█", int(score))
			fmt.Printf("    %-15s %.1f/5.0 %s\n", cat+":", score, bar)
		}
	}

	// DORA Summary
	if report.DORA != nil {
		fmt.Println("\nDORA METRICS")
		fmt.Println(strings.Repeat("-", 50))
		fmt.Printf("  Overall Rating: %s\n", formatRating(report.DORA.Rating))
		if report.DORA.DeploymentFrequency != nil {
			fmt.Printf("  Deployment Frequency: %.1f/day (%s)\n",
				report.DORA.DeploymentFrequency.Value,
				report.DORA.DeploymentFrequency.Rating)
		}
		if report.DORA.LeadTime != nil {
			fmt.Printf("  Lead Time: %s (%s)\n",
				formatDevexDuration(report.DORA.LeadTime.Value),
				report.DORA.LeadTime.Rating)
		}
		if report.DORA.ChangeFailureRate != nil {
			fmt.Printf("  Change Failure Rate: %.1f%% (%s)\n",
				report.DORA.ChangeFailureRate.Value,
				report.DORA.ChangeFailureRate.Rating)
		}
		if report.DORA.TimeToRestore != nil {
			fmt.Printf("  Time to Restore: %s (%s)\n",
				formatDevexDuration(report.DORA.TimeToRestore.Value),
				report.DORA.TimeToRestore.Rating)
		}
	}

	// Adoption
	if report.Adoption != nil {
		fmt.Println("\nPLATFORM ADOPTION")
		fmt.Println(strings.Repeat("-", 50))
		fmt.Printf("  Self-Service:    %.1f%%\n", report.Adoption.SelfServiceRatio)
		fmt.Printf("  Golden Paths:    %.1f%%\n", report.Adoption.GoldenPathAdoption)
		fmt.Printf("  Automated:       %.1f%%\n", report.Adoption.AutomatedDeployments)
	}

	// Top Friction Points
	if len(report.FrictionPoints) > 0 {
		fmt.Println("\nTOP FRICTION POINTS")
		fmt.Println(strings.Repeat("-", 50))
		limit := 3
		if len(report.FrictionPoints) < limit {
			limit = len(report.FrictionPoints)
		}
		for i := 0; i < limit; i++ {
			fp := report.FrictionPoints[i]
			fmt.Printf("  %d. [%s] %s: %s\n", i+1, fp.Impact, fp.Category, fp.Description)
		}
	}

	// Recommendations
	if len(report.Recommendations) > 0 {
		fmt.Println("\nRECOMMENDATIONS")
		fmt.Println(strings.Repeat("-", 50))
		for _, rec := range report.Recommendations {
			fmt.Printf("  %d. %s\n", rec.Priority, rec.Title)
			fmt.Printf("     %s\n", rec.Description)
			fmt.Printf("     Impact: %s | Effort: %s\n", rec.Impact, rec.Effort)
		}
	}

	return nil
}

func runDevexTeams(cmd *cobra.Command, args []string) error {
	teams := devexManager.ListTeams()

	if len(teams) == 0 {
		fmt.Println("No teams with metrics found")
		return nil
	}

	fmt.Printf("%-20s %-12s %-12s %-12s\n", "TEAM", "DEPLOYS", "LEAD TIME", "RATING")
	fmt.Println(strings.Repeat("-", 60))

	for _, team := range teams {
		dora, _ := devexManager.GetDORAMetrics(team)
		deploys := "-"
		leadTime := "-"
		rating := "-"

		if dora != nil {
			if dora.DeploymentFrequency != nil {
				deploys = fmt.Sprintf("%.1f/day", dora.DeploymentFrequency.Value)
			}
			if dora.LeadTime != nil {
				leadTime = formatDevexDuration(dora.LeadTime.Value)
			}
			rating = dora.Rating
		}

		fmt.Printf("%-20s %-12s %-12s %-12s\n", team, deploys, leadTime, rating)
	}

	return nil
}

func runDevexScore(cmd *cobra.Command, args []string) error {
	team := ""
	if len(args) > 0 {
		team = args[0]
	}

	report, err := devexManager.GenerateReport(cmd.Context(), team)
	if err != nil {
		return err
	}

	if report.Score == nil {
		fmt.Println("No score data available")
		return nil
	}

	title := "Organization"
	if team != "" {
		title = team
	}

	fmt.Printf("Developer Experience Score - %s\n", title)
	fmt.Println(strings.Repeat("=", 50))

	// Visual score representation
	scoreBar := strings.Repeat("█", int(report.Score.Overall*4))
	emptyBar := strings.Repeat("░", 20-len(scoreBar))
	fmt.Printf("\n  Overall: %.1f/5.0 [%s%s]\n\n", report.Score.Overall, scoreBar, emptyBar)

	fmt.Println("Category Scores:")
	for cat, score := range report.Score.Categories {
		catBar := strings.Repeat("█", int(score*4))
		catEmpty := strings.Repeat("░", 20-len(catBar))
		fmt.Printf("  %-15s %.1f/5.0 [%s%s]\n", cat+":", score, catBar, catEmpty)
	}

	// Interpretation
	fmt.Println()
	if report.Score.Overall >= 4.5 {
		fmt.Println("Rating: EXCELLENT - Elite developer experience")
	} else if report.Score.Overall >= 4.0 {
		fmt.Println("Rating: GOOD - High-performing developer experience")
	} else if report.Score.Overall >= 3.0 {
		fmt.Println("Rating: AVERAGE - Room for improvement")
	} else {
		fmt.Println("Rating: NEEDS ATTENTION - Significant improvements needed")
	}

	return nil
}

// Helper functions

func formatRating(rating string) string {
	switch rating {
	case "elite":
		return "Elite"
	case "high":
		return "High"
	case "medium":
		return "Medium"
	case "low":
		return "Low"
	default:
		return rating
	}
}

func formatDevexDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	} else if d < time.Hour {
		return fmt.Sprintf("%.0fm", d.Minutes())
	} else if d < 24*time.Hour {
		return fmt.Sprintf("%.1fh", d.Hours())
	}
	return fmt.Sprintf("%.1fd", d.Hours()/24)
}

func formatImpact(impact string) string {
	switch impact {
	case "high":
		return "HIGH"
	case "medium":
		return "MED"
	case "low":
		return "LOW"
	default:
		return impact
	}
}

// GetDevexCmd returns the devex command for registration
func GetDevexCmd() *cobra.Command {
	return devexCmd
}
