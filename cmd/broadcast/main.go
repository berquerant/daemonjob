// Copyright 2026 berquerant.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"flag"
	"os"

	"github.com/berquerant/daemonjob/internal/broadcast"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func main() {
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))
	log := ctrl.Log.WithName("broadcast-main")

	cfg, err := broadcast.LoadConfigFromEnv()
	if err != nil {
		broadcast.WriteTerminationLog(err.Error())
		os.Exit(1)
	}

	runner, err := broadcast.NewRunner(cfg)
	if err != nil {
		broadcast.WriteTerminationLog(err.Error())
		os.Exit(1)
	}

	ctx := context.Background()
	if err := runner.Run(ctx); err != nil {
		broadcast.WriteTerminationLog(err.Error())
		os.Exit(1)
	}

	log.Info("Broadcast job completed successfully")
}
