package root

import (
	"fmt"
	"github.com/dinerozz/web-behavior-backend/cmd/migrate"
	"github.com/dinerozz/web-behavior-backend/config"
	"github.com/dinerozz/web-behavior-backend/server"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "web-behavior-backend",
	Short: "Web behavior application",
}

func GetRootCmd(config *config.Config) *cobra.Command {
	dbURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		config.DB.User,
		config.DB.Password,
		config.DB.Host,
		config.DB.Port,
		config.DB.DBName,
		config.DB.SSLMode)

	rootCmd.AddCommand(&cobra.Command{
		Use:   "serve",
		Short: "Start the HTTP server",
		Run: func(cmd *cobra.Command, args []string) {
			server.RunServer(config)
		},
	})

	rootCmd.AddCommand(migrate.GetMigrateCmd(dbURL))

	return rootCmd
}
