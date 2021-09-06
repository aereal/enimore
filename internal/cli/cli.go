package cli

import (
	"context"
	"encoding/json"
	"flag"
	"io"
	"os"

	"github.com/aereal/enimore"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"golang.org/x/sync/errgroup"
)

func New() *App {
	return &App{
		outStream: os.Stdout,
		errStream: os.Stderr,
	}
}

type App struct {
	outStream io.Writer
	errStream io.Writer

	arns   ArnList
	region string
}

func (a *App) Run(argv []string) int {
	fs := flag.NewFlagSet(argv[0], flag.ContinueOnError)
	fs.Var(&a.arns, "arn", "ARNs")
	fs.StringVar(&a.region, "region", "", "default region")
	err := fs.Parse(argv[1:])
	if err == flag.ErrHelp {
		return 0
	}
	if err != nil {
		return 1
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var optFns []func(*config.LoadOptions) error
	if a.region != "" {
		optFns = append(optFns, config.WithRegion(a.region))
	}
	cfg, err := config.LoadDefaultConfig(ctx, optFns...)
	if err != nil {
		return 1
	}

	ecsAccum := enimore.NewECSServiceAccumulator(ecs.NewFromConfig(cfg), a.arns)
	lambdaAccum := enimore.NewLambdaFunctionAccumulator(lambda.NewFromConfig(cfg), a.arns)
	populator := enimore.NewENIPopulator(ec2.NewFromConfig(cfg))

	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return ecsAccum.Accumulate(ctx, populator)
	})
	eg.Go(func() error {
		return lambdaAccum.Accumulate(ctx, populator)
	})
	if err := eg.Wait(); err != nil {
		return 1
	}
	res, err := populator.Run(ctx)
	if err != nil {
		return 1
	}

	if err := json.NewEncoder(a.outStream).Encode(res); err != nil {
		return 1
	}
	return 0
}
