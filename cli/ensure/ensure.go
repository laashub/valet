package ensure

import (
	"context"
	"github.com/solo-io/go-utils/cliutils"
	"github.com/solo-io/go-utils/contextutils"
	"github.com/solo-io/go-utils/errors"
	"github.com/solo-io/valet/cli/config"
	"github.com/solo-io/valet/cli/ensure/cluster"
	"github.com/solo-io/valet/cli/ensure/cluster/gke"
	"github.com/solo-io/valet/cli/ensure/cluster/minikube"
	"github.com/solo-io/valet/cli/ensure/demo"
	"github.com/solo-io/valet/cli/ensure/demo/petclinic"
	"github.com/solo-io/valet/cli/ensure/gloo"
	"github.com/solo-io/valet/cli/ensure/resources"
	"github.com/solo-io/valet/cli/file"
	"github.com/solo-io/valet/cli/options"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"os"
)

var (
	MustProvideFileError = errors.Errorf("Must provide file option or subcommand")
)

func EnsureCmd(opts *options.Options, optionsFunc ...cliutils.OptionsFunc) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ensure",
		Short: "ensures kubernetes cluster is running",
		RunE: func(cmd *cobra.Command, args []string) error {
			return ensure(opts)
		},
	}

	cliutils.ApplyOptions(cmd, optionsFunc)
	cmd.PersistentFlags().StringVarP(&opts.Ensure.File, "file", "f", "", "path to file containing config to ensure")
	cmd.AddCommand(
		cluster.ClusterCmd(opts, optionsFunc...),
		gloo.GlooCmd(opts, optionsFunc...),
		demo.DemoCmd(opts, optionsFunc...),
		resources.ResourcesCmd(opts, optionsFunc...))
	return cmd
}

func ensure(opts *options.Options) error {
	if opts.Ensure.File == "" {
		return MustProvideFileError
	}

	cfg, err := file.LoadConfig(opts.Top.Ctx, opts.Ensure.File)
	if err != nil {
		return err
	}

	if err := loadEnv(opts.Top.Ctx); err != nil {
		return err
	}

	if cfg.Cluster != nil {
		opts.Cluster.Type = cfg.Cluster.Type
		opts.Cluster.GKE = cfg.Cluster.GKE
		opts.Cluster.Minikube = cfg.Cluster.Minikube

		var clusterErr error
		if opts.Cluster.Type == "gke" {
			clusterErr = gke.EnsureGke(opts)
		} else if opts.Cluster.Type == "minikube" {
			clusterErr = minikube.EnsureMinikube(opts)
		} else {
			return errors.Errorf("unknown type", zap.String("type", opts.Cluster.Type))
		}
		if clusterErr != nil {
			return clusterErr
		}
	}

	if cfg.Gloo != nil {
		opts.Gloo = *cfg.Gloo
		err := gloo.EnsureGloo(opts)
		if err != nil {
			return err
		}
	}

	if cfg.Demos != nil {
		if cfg.Demos.Petclinic != nil {
			opts.Demos.Petclinic = cfg.Demos.Petclinic
			err := petclinic.EnsurePetclinicDemo(opts)
			if err != nil {
				return err
			}
		}
	}

	if cfg.Resources != nil {
		opts.Ensure.Resources = cfg.Resources
		err := resources.EnsureResources(opts)
		if err != nil {
			return err
		}
	}

	return nil
}

func loadEnv(ctx context.Context) error {
	globalConfig, err := config.LoadGlobalConfig(ctx)
	if err != nil {
		contextutils.LoggerFrom(ctx).Errorw("Failed to load global config", zap.Error(err))
		return err
	}

	for k, v := range globalConfig.Env {
		val := os.Getenv(k)
		if val == "" {
			err := os.Setenv(k, v)
			if err != nil {
				contextutils.LoggerFrom(ctx).Errorw("Failed to set environment variable", zap.Error(err))
				return err
			}
		}
	}
	return nil
}
