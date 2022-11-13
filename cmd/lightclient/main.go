/*
   Copyright 2022 Erigon-Lightclient contributors
   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at
       http://www.apache.org/licenses/LICENSE-2.0
   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package main

import (
	"context"
	"fmt"
	"os"

	clcore "github.com/ledgerwatch/erigon/cmd/erigon-cl/cl-core"
	"github.com/ledgerwatch/erigon/cmd/lightclient/lightclient"
	lcCli "github.com/ledgerwatch/erigon/cmd/sentinel/cli"
	"github.com/ledgerwatch/erigon/cmd/sentinel/cli/flags"
	"github.com/ledgerwatch/erigon/cmd/sentinel/sentinel"
	"github.com/ledgerwatch/erigon/cmd/sentinel/sentinel/service"
	lightclientapp "github.com/ledgerwatch/erigon/turbo/app"
	"github.com/ledgerwatch/log/v3"
	"github.com/urfave/cli"
)

func main() {
	app := lightclientapp.MakeApp(runLightClientNode, flags.ConsensusLayerDefaultFlags)
	if err := app.Run(os.Args); err != nil {
		_, printErr := fmt.Fprintln(os.Stderr, err)
		if printErr != nil {
			log.Warn("Fprintln error", "err", printErr)
		}
		os.Exit(1)
	}
}

func runLightClientNode(cliCtx *cli.Context) {
	ctx := context.Background()
	lcCfg, err := lcCli.SetUpConsensusLayerCfg(cliCtx)
	if err != nil {
		log.Error("[Lightclient] Could not initialize lightclient", "err", err)
	}
	log.Root().SetHandler(log.LvlFilterHandler(log.Lvl(lcCfg.LogLvl), log.StderrHandler))
	log.Info("[ConsensusLayer]", "chain", cliCtx.GlobalString(flags.ConsensusLayerChain.Name))
	log.Info("[ConsensusLayer] Running lightclient", "cfg", lcCfg)
	sentinel, err := service.StartSentinelService(&sentinel.SentinelConfig{
		IpAddr:        lcCfg.Addr,
		Port:          int(lcCfg.Port),
		TCPPort:       lcCfg.ServerTcpPort,
		GenesisConfig: lcCfg.GenesisCfg,
		NetworkConfig: lcCfg.NetworkCfg,
		BeaconConfig:  lcCfg.BeaconCfg,
		NoDiscovery:   lcCfg.NoDiscovery,
	}, &service.ServerConfig{Network: lcCfg.ServerProtocol, Addr: lcCfg.ServerAddr}, nil)
	if err != nil {
		log.Error("Could not start sentinel", "err", err)
	}
	log.Info("Sentinel started", "addr", lcCfg.ServerAddr)

	bs, err := clcore.RetrieveBeaconState(ctx, lcCfg.CheckpointUri)

	if err != nil {
		log.Error("[Checkpoint Sync] Failed", "reason", err)
		return
	}
	log.Info("Finalized Checkpoint", "Epoch", bs.FinalizedCheckpoint.Epoch)
	lc, err := lightclient.NewLightClient(ctx, lcCfg.GenesisCfg, lcCfg.BeaconCfg, nil, sentinel, 0, true)
	if err != nil {
		log.Error("Could not make Lightclient", "err", err)
	}
	if err := lc.BootstrapCheckpoint(ctx, bs.FinalizedCheckpoint.Root); err != nil {
		log.Error("[Bootstrap] failed to bootstrap", "err", err)
		return
	}
	lc.Start()
}
