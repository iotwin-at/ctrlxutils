package ctrlxutils

import (
	"context"

	"github.com/boschrexroth/ctrlx-datalayer-golang/v2/pkg/datalayer"
)

type IDataLayerProvider interface {
	Name() string
	Run(ctx context.Context, provider *datalayer.Provider, dlRootPath string) error
}
