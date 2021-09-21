package main

import (
	"errors"
	"log"
	"os"
	"path/filepath"

	"github.com/creachadair/command"
	"github.com/creachadair/misctools/ffs/config"
)

var (
	configFile = "$HOME/.config/ffs/config.yml"
	storeAddr  string

	root = &command.C{
		Name:  filepath.Base(os.Args[0]),
		Usage: `<command> [arguments]`,
		Help:  `A command-line tool to manage FFS file trees.`,

		Init: func(env *command.Env) error {
			cfg, err := config.Load(os.ExpandEnv(configFile))
			if err != nil {
				return err
			}
			if storeAddr != "" {
				cfg.StoreAddress = storeAddr
			}
			return nil
		},

		Commands: nil,
	}
)

func init() {
	if cf, ok := os.LookupEnv("FFS_CONFIG"); ok && cf != "" {
		configFile = cf
	}
	root.Flags.StringVar(&configFile, "config", configFile, "Configuration file path")
	root.Flags.StringVar(&storeAddr, "store", storeAddr, "Store service address (overrides config)")
}

func main() {
	if err := command.Execute(root.NewEnv(nil), os.Args[1:]); err != nil {
		if errors.Is(err, command.ErrUsage) {
			os.Exit(2)
		}
		log.Printf("Error: %v", err)
		os.Exit(1)
	}
}
