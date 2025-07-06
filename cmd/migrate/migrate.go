package migrate

import (
	"fmt"
	"github.com/golang-migrate/migrate/v4"
	"log"
	"strings"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/spf13/cobra"
)

func GetMigrateCmd(dbURL string) *cobra.Command {
	var down bool

	migrateCmd := &cobra.Command{
		Use:   "migrate",
		Short: "Run database migrations",
		Run: func(cmd *cobra.Command, args []string) {
			m, err := migrate.New(
				"file://migrations",
				dbURL,
			)
			if err != nil {
				log.Fatal("❌ Failed to initialize migrations:", err)
			}

			if down {
				err := m.Down()
				if err != nil {
					if err == migrate.ErrNoChange {
						fmt.Println("⚠️ No migrations to rollback.")
						return
					} else if strings.Contains(err.Error(), "dirty") {
						fmt.Println("⚠️ Database is in a dirty state. Forcing version fix...")
						m.Force(0)
						m.Down()
					} else {
						log.Fatal("❌ Failed to apply down migrations:", err)
					}
				} else {
					fmt.Println("✅ Migrations rolled back successfully!")
				}
				return
			}

			err = m.Up()
			if err != nil {
				if err == migrate.ErrNoChange {
					fmt.Println("⚠️ No new migrations to apply.")
					return
				}
				log.Fatal("❌ Failed to apply up migrations:", err)
			}

			fmt.Println("✅ Migrations applied successfully!")
		},
	}

	migrateCmd.Flags().BoolVarP(&down, "down", "d", false, "Rollback migrations")

	return migrateCmd
}
