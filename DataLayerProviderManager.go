package ctrlxutils

import (
	"context"
	"time"

	"github.com/boschrexroth/ctrlx-datalayer-golang/v2/pkg/datalayer"
	"github.com/uoul/go-common/log"
)

type DataLayerProviderManager struct {
	ctx        context.Context
	logger     log.ILogger
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

func NewDataLayerProviderManager(ctx context.Context, logger log.ILogger, dlConnStr string, dlRootPath string, opts ...func(*DataLayerProviderManager)) *DataLayerProviderManager {
	d := &DataLayerProviderManager{
		ctx:           ctx,
		logger:        logger,
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
		logger.Debug("Creating DatalayerSystem...")
		dlSys := datalayer.NewSystem("")
		dlSys.Start(false)
		// Create Datalayer provider
		logger.Debugf("Creating Datalayer Providers for %s...", dlConnStr)
		dlProvider := dlSys.Factory().CreateProvider(dlConnStr)
		dlProvider.Start()
		// Run Providers
		for _, p := range d.providers {
			// Spawn new go routine for all providers
			logger.Debugf("Creating DatalayerProvider(%s)...", p.Name())
			go func() {
				for {
					err := p.Run(ctx, dlProvider, dlRootPath)
					if err == nil {
						// Provider terminated successfully
						return
					}
					// Provider terminated with error
					d.logger.Error(err.Error())
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
