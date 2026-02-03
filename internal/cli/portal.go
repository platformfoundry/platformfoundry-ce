package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/platformfoundry/pf-ce/internal/portal/server"
	"github.com/spf13/cobra"
)

var portalCmd = &cobra.Command{
	Use:   "portal",
	Short: "Web-based developer portal",
	Long:  `Start and manage the PlatformFoundry web portal.`,
}

var portalServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the portal server",
	RunE:  runPortalServe,
}

var (
	portalPort int
	portalCORS bool
)

func init() {
	portalCmd.AddCommand(portalServeCmd)

	portalServeCmd.Flags().IntVarP(&portalPort, "port", "p", 3000, "Port to listen on")
	portalServeCmd.Flags().BoolVar(&portalCORS, "cors", false, "Enable CORS for development")
}

func runPortalServe(cmd *cobra.Command, args []string) error {
	fmt.Printf("Starting PlatformFoundry Portal on http://localhost:%d\n", portalPort)

	srv := server.New(server.Config{
		Port:       portalPort,
		EnableCORS: portalCORS,
	})

	// Handle shutdown gracefully
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		fmt.Println("\nShutting down portal...")
		srv.Shutdown(ctx)
	}()

	if err := srv.Start(); err != nil {
		return fmt.Errorf("portal server error: %w", err)
	}

	return nil
}

func GetPortalCmd() *cobra.Command {
	return portalCmd
}
