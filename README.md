# Go ctrlX Data Layer Utilities

This Go library provides utilities to simplify interaction with the Bosch Rexroth ctrlX Data Layer. It offers a high-level manager to handle the lifecycle of data layer providers and includes a generic node provider for easily exposing values.

### Core Components

*   **`DataLayerProviderManager`**: Manages the data layer connection and runs one or more providers in the background. It handles the creation and teardown of the `datalayer.System` and `datalayer.Provider` instances.
*   **`IDataLayerProvider`**: An interface that allows for the creation of modular, custom providers. Any struct that implements the `Name()` and `Run()` methods can be managed by the `DataLayerProviderManager`.
*   **`DataLayerProviderNode`**: A generic provider for exposing a single value of any type (`T`) as a node in the data layer. It automatically handles data layer events like read, write, create, and remove requests for the specified path.

### Installation

To use this library, add it to your Go project.

```bash
go github.com/iotwin-at/ctrlxutils
```

### Example Usage

The following example demonstrates how to create a custom `DataLayerProviderNode` and configure it using settings, that are stored to ctrlX appdata. Logs are written to jounald, such that they will be visible in CtrlX-UI's `Logbook`.

```go
package main

import (
	"context"
	"os/signal"
	"syscall"
	"time"

	"github.com/boschrexroth/ctrlx-datalayer-golang/v2/pkg/datalayer"
	"github.com/iotwin-at/ctrlxutils"
	"github.com/uoul/go-common/log"
	"github.com/uoul/go-common/serialization"
)

// ---------------------------------------------------------------------------------
// Example AppConfig
// ---------------------------------------------------------------------------------
type AppConfig struct {
	UpdateRate int
}

// ---------------------------------------------------------------------------------
// Example Datalayer Provider implementation
// ---------------------------------------------------------------------------------

type CounterProvider struct {
	updateRate int
}

// Name implements ctrlxutils.IDataLayerProvider.
func (c *CounterProvider) Name() string {
	return "CounterProvider"
}

// Run implements ctrlxutils.IDataLayerProvider.
func (c *CounterProvider) Run(ctx context.Context, provider *datalayer.Provider, dlRootPath string) error {
	// Create nodes (here just one for demonstration purpose - in general a provider can host multiple nodes)
	counterNode := ctrlxutils.NewDataLayerProviderNode[int](ctx, provider, ctrlxutils.DataLayerProviderNodeConfig{
		Path:    "Counter", // Will be visible at Datalayer under <dlRootPath>/<ProviderName>/Counter
		Unit:    "inc.",
		Desc:    "Incrementing counter example",
		DescUrl: "",
	})
	// Init. value
	counterNode.SetValue(0)
	// Run Provider
	for {
		select {
		case <-time.Tick(time.Duration(c.updateRate) * time.Second):
			// Increment counter by 1
			counterNode.SetValue(counterNode.GetValue() + 1)
		case <-ctx.Done():
			return nil
		}
	}
}

// Use constructor to ensure provider implements interface
func NewCounterProvider(updateRate int) ctrlxutils.IDataLayerProvider {
	return &CounterProvider{
		updateRate: updateRate,
	}
}

// ---------------------------------------------------------------------------------
// Setup Application
// ---------------------------------------------------------------------------------

func main() {
	// Create Application Context
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGABRT)
	defer cancel()

	// Load AppConfig from file (you need to make sure, that the corresponding entries in snapcraft.yaml and manifest.json exists!)
	appConfig, _ := ctrlxutils.UnmarshalAppdata[AppConfig]( // Ignore Errors - do not do this in your code!
		serialization.NewJSONSerializer(),
		"properties.json",
	)

	// Create Logger for Ctrlx Logbook
	logger, _ := ctrlxutils.NewJournaldLogger(ctx, "", log.INFO)

	// Create provider manager
	ctrlxutils.NewDataLayerProviderManager(
		ctx,
		logger,
		"ipc://",     // Datalayer interface (for local deployment as snap always ips://)
		"MyFancyApp", // Datalayer RootPath
		ctrlxutils.WithProvider(NewCounterProvider(appConfig.UpdateRate)), // Register our CounterProvider from above (you can register multiple providers here)
	)

	// Wait for termination
	<-ctx.Done()
}
```

