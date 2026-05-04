package ctrlxutils

import (
	"context"
	"log/slog"
	"time"

	"github.com/boschrexroth/ctrlx-datalayer-golang/v2/pkg/datalayer"
)

type DataLayerProviderManager struct {
	ctx        context.Context
	providers  []IDataLayerProvider
	dlConnStr  string
	dlRootPath string

	retryInterval time.Duration
}

func WithProvider(p IDataLayerProvider) func(*DataLayerProviderManager) {
	return func(dlsm *DataLayerProviderManager) {
		dlsm.providers = append(dlsm.providers, p)
	}
}

func NewDataLayerProviderManager(ctx context.Context, dlConnStr string, dlRootPath string, opts ...func(*DataLayerProviderManager)) *DataLayerProviderManager {
	d := &DataLayerProviderManager{
		ctx:           ctx,
		dlConnStr:     dlConnStr,
		dlRootPath:    dlRootPath,
		retryInterval: 30 * time.Second,
	}
	for _, o := range opts {
		o(d)
	}
	// Run Providers
	go func() {
		// Create Datalayer system
		slog.Debug("Creating DatalayerSystem...")
		dlSys := datalayer.NewSystem("")
		dlSys.Start(false)
		// Create Datalayer provider
		slog.Debug("Creating Datalayer Providers...", slog.String("address", dlConnStr))
		dlProvider := dlSys.Factory().CreateProvider(dlConnStr)
		dlProvider.Start()
		// Run Providers
		for _, p := range d.providers {
			// Spawn new go routine for all providers
			slog.Debug("Creating DatalayerProvider...", slog.String("provider", p.Name()))
			go func() {
				for {
					err := p.Run(ctx, dlProvider, dlRootPath)
					if err == nil {
						// Provider terminated successfully
						return
					}
					// Provider terminated with error
					slog.Error("Datalayer Provider terminated with error", slog.String("provider", p.Name()), slog.Any("error", err))
					time.Sleep(d.retryInterval)
				}
			}()
		}
		<-ctx.Done()
		dlProvider.Stop()
		datalayer.DeleteProvider(dlProvider)
		dlSys.Stop(false)
		datalayer.DeleteSystem(dlSys)
	}()
	return d
}
