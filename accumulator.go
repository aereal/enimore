package enimore

import (
	"context"

	"github.com/aereal/enimore/enipopulator"
)

type Accumulator interface {
	Accumulate(ctx context.Context, populator *enipopulator.ENIPopulator) error
}
