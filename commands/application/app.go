package application

import (
	"context"

	"github.com/spf13/cobra"
)

type (
	RunE        func(cmd *cobra.Command, args []string) error
	RunEAdaptor func(ctx context.Context, cmd *cobra.Command, app *Application) error
)

// Application holds configuration used by commands
type Application struct {
	client Client
	log    Log

	// TODO manage configuration file
	// ConfigurationFile string // Path to the configuration file to use
}

func New(ctx context.Context, cmd *cobra.Command) *Application {
	// application's context
	app := &Application{}
	// app.PersistentFlags().StringVar(&app.ConfigurationFile, "use-configuration", app.ConfigurationFile, "Specifies the configuration to use")
	AddLogFlags(ctx, cmd, app)
	return app
}

func (app *Application) Client() *Client {
	return &app.client
}

func (app *Application) Log() *Log {
	return &app.log
}

func ChainRunEFunctions(prev RunE, fn RunEAdaptor, ctx context.Context, cmd *cobra.Command, app *Application) RunE {
	if prev == nil {
		return func(cmd *cobra.Command, args []string) error {
			return fn(ctx, cmd, app)
		}
	}
	return func(cmd *cobra.Command, args []string) error {
		if prev != nil {
			err := prev(cmd, args)
			if err != nil {
				return err
			}
		}
		return fn(ctx, cmd, app)
	}
}
