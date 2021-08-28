package enimore

import (
	"context"
)

type Accumulator interface {
	Accumulate(ctx context.Context, populator *ENIPopulator) error
}
