package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"github.com/masteralanlab/free-proxy/internal/platform"
	"github.com/masteralanlab/free-proxy/internal/providers/vpngate"
	"github.com/masteralanlab/free-proxy/internal/security"
	"github.com/masteralanlab/free-proxy/internal/services"
	"github.com/masteralanlab/free-proxy/internal/store"
	"github.com/spf13/cobra"
)

// The command bodies are filled in across the refactor phases (see
// docs/refactor-go.md). At P0 they parse config and report pending status so the
// binary builds and `--help` works end to end.

func serveCmd() *cobra.Command {
	var host string
	var port int
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Run the web/API control plane, proxy gateway, and background workers",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer stop()
			return runServe(ctx, cfg, host, port)
		},
	}
	cmd.Flags().StringVar(&host, "host", "", "Web bind address override")
	cmd.Flags().IntVar(&port, "port", 0, "Web bind port override")
	return cmd
}

func discoverCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "discover",
		Short: "Fetch nodes from the configured provider and store them",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			if err := cfg.EnsureDirectories(); err != nil {
				return err
			}
			db, err := store.Open(cfg.DatabaseURL)
			if err != nil {
				return err
			}
			defer db.Close()
			if err := store.Migrate(db); err != nil {
				return err
			}
			repos := store.NewRepos(db)
			provider := vpngate.NewProvider(cfg)
			svc := services.NewDiscoveryService(provider, repos.Nodes)
			res, err := svc.Discover(cmd.Context())
			if err != nil {
				return err
			}
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(res)
		},
	}
}

func credentialsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "credentials",
		Short: "Print the current web management address and credentials",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			if err := cfg.EnsureDirectories(); err != nil {
				return err
			}
			adminStore, err := security.NewAdminConfigStore(cfg)
			if err != nil {
				return err
			}
			c := adminStore.Config()
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "URL: http://%s:%d/%s/\n", c.Host, c.Port, c.SecretPath)
			fmt.Fprintf(out, "Username: %s\n", c.Username)
			if pw := adminStore.BootstrapPassword(); pw != "" {
				fmt.Fprintf(out, "Password: %s\n", pw)
			} else {
				fmt.Fprintln(out, "Password: [configured; cannot be recovered]")
			}
			return nil
		},
	}
}

func statusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Print local configuration and database schema status",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			if err := cfg.EnsureDirectories(); err != nil {
				return err
			}
			db, err := store.Open(cfg.DatabaseURL)
			if err != nil {
				return err
			}
			defer db.Close()
			tables, _ := store.SchemaTables(context.Background(), db)
			payload := map[string]any{
				"environment":  cfg.Environment,
				"data_dir":     cfg.DataDir,
				"database_url": cfg.DatabaseURL,
				"web":          fmt.Sprintf("%s:%d", cfg.WebHost, cfg.WebPort),
				"proxy":        fmt.Sprintf("%s:%d", cfg.ProxyHost, cfg.ProxyPort),
				"tables":       tables,
			}
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(payload)
		},
	}
}

func preflightCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "preflight",
		Short: "Run startup environment checks",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			if err := cfg.EnsureDirectories(); err != nil {
				return err
			}
			diag := services.NewDiagnosticsService(cfg, nil)
			result := diag.StartupPreflight(cmd.Context())
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(result)
		},
	}
}

func logsCmd() *cobra.Command {
	var lines int
	cmd := &cobra.Command{
		Use:   "logs",
		Short: "Print the most recent structured application log entries",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			entries, err := os.ReadDir(cfg.LogsDir())
			if err != nil {
				return nil
			}
			var files []string
			for _, e := range entries {
				if strings.HasSuffix(e.Name(), ".json") {
					files = append(files, e.Name())
				}
			}
			if len(files) == 0 {
				return nil
			}
			sort.Strings(files)
			data, err := os.ReadFile(filepath.Join(cfg.LogsDir(), files[len(files)-1]))
			if err != nil {
				return err
			}
			all := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
			if lines < len(all) {
				all = all[len(all)-lines:]
			}
			fmt.Fprintln(cmd.OutOrStdout(), strings.Join(all, "\n"))
			return nil
		},
	}
	cmd.Flags().IntVar(&lines, "lines", 100, "Number of log lines to print")
	return cmd
}

func adminConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "admin-config",
		Short: "Update web administration credentials and listener settings",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			if err := cfg.EnsureDirectories(); err != nil {
				return err
			}
			adminStore, err := security.NewAdminConfigStore(cfg)
			if err != nil {
				return err
			}
			prev := adminStore.Config()
			updated := prev
			f := cmd.Flags()
			if v, _ := f.GetString("username"); v != "" {
				updated.Username = v
			}
			if v, _ := f.GetString("password"); v != "" {
				if updated.PasswordHash, err = security.HashPassword(v); err != nil {
					return err
				}
			}
			if v, _ := f.GetString("secret-path"); v != "" {
				updated.SecretPath = v
			}
			if v, _ := f.GetString("host"); v != "" {
				updated.Host = v
			}
			if v, _ := f.GetInt("port"); v != 0 {
				updated.Port = v
			}
			if v, _ := f.GetString("proxy-host"); v != "" {
				updated.ProxyHost = v
			}
			if v, _ := f.GetInt("proxy-port"); v != 0 {
				updated.ProxyPort = v
			}
			if err := adminStore.Update(updated); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Administration configuration updated; restart the service if the listener changed")
			return nil
		},
	}
	cmd.Flags().String("username", "", "Administrator username")
	cmd.Flags().String("password", "", "Administrator password")
	cmd.Flags().String("secret-path", "", "Secret URL path segment")
	cmd.Flags().String("host", "", "Web bind address")
	cmd.Flags().Int("port", 0, "Web bind port")
	cmd.Flags().String("proxy-host", "", "Proxy bind address")
	cmd.Flags().Int("proxy-port", 0, "Proxy bind port")
	return cmd
}

func databaseUpgradeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "database-upgrade",
		Short: "Upgrade the application database to the latest migration",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			if err := cfg.EnsureDirectories(); err != nil {
				return err
			}
			db, err := store.Open(cfg.DatabaseURL)
			if err != nil {
				return err
			}
			defer db.Close()
			if err := store.Migrate(db); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Database upgraded to head")
			return nil
		},
	}
}

func doctorCmd() *cobra.Command {
	var fix bool
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check (and optionally install) required system dependencies",
		RunE: func(cmd *cobra.Command, _ []string) error {
			checks := platform.RunChecks()
			out := cmd.OutOrStdout()
			for _, c := range checks {
				mark := "OK  "
				if !c.OK {
					mark = "FAIL"
				}
				fmt.Fprintf(out, "[%s] %-12s %s\n", mark, c.Name, c.Detail)
			}
			missing := platform.MissingPackages(checks)
			if fix && len(missing) > 0 {
				if os.Geteuid() != 0 {
					return fmt.Errorf("--fix requires root")
				}
				if err := platform.Install(cmd.Context(), missing); err != nil {
					return err
				}
				fmt.Fprintln(out, "Dependencies installed; re-run `doctor` to verify.")
				return nil
			}
			if platform.CriticalMissing(checks) {
				return fmt.Errorf("critical dependencies missing (run `free-proxy install-deps` as root)")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&fix, "fix", false, "Install missing dependencies")
	return cmd
}

func installDepsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install-deps",
		Short: "Install required system dependencies (openvpn, iproute2, ...)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if os.Geteuid() != 0 {
				return fmt.Errorf("install-deps requires root")
			}
			return platform.Install(cmd.Context(), platform.RecommendedPackages)
		},
	}
}

func installCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "Install Free Proxy on this machine: binary, dependencies, env file, service",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := platform.RequireInstallSupport(); err != nil {
				return err
			}
			ctx := cmd.Context()
			out := cmd.OutOrStdout()
			if err := platform.InstallSelf(); err != nil {
				return fmt.Errorf("install binary: %w", err)
			}
			fmt.Fprintf(out, "Binary installed at %s\n", platform.BinPath)
			if missing := platform.MissingPackages(platform.RunChecks()); len(missing) > 0 {
				if err := platform.Install(ctx, missing); err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(),
						"warning: dependency install failed: %v\nFix the package manager and re-run `free-proxy doctor --fix`.\n", err)
				}
			}
			if err := platform.WriteDefaultEnv(); err != nil {
				return fmt.Errorf("write environment: %w", err)
			}
			if err := platform.InstallService(ctx); err != nil {
				return fmt.Errorf("install service: %w", err)
			}
			fmt.Fprintln(out, "Free Proxy installed and started.")
			fmt.Fprintln(out, "Run `free-proxy credentials` to print the management URL and initial password.")
			return nil
		},
	}
}

func uninstallCmd() *cobra.Command {
	var purge bool
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Stop the service and remove Free Proxy from this machine",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := platform.RequireInstallSupport(); err != nil {
				return err
			}
			if err := platform.Uninstall(cmd.Context(), purge); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Free Proxy removed.")
			return nil
		},
	}
	cmd.Flags().BoolVar(&purge, "purge-data", false, "Also delete "+platform.DataDir+" (database, logs, configs)")
	return cmd
}

