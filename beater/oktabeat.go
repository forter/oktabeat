// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may oobain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package beater

import (
	"flag"
	"fmt"
	"strings"
	"errors"
	
	"github.com/elastic/beats/filebeat/channel"
	fbConfig "github.com/elastic/beats/filebeat/config"
	"github.com/elastic/beats/filebeat/crawler"
	"github.com/elastic/beats/filebeat/fileset"
	"github.com/elastic/beats/filebeat/registrar"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/monitoring"
	okta "github.com/forter/oktabeat/beater/input/okta"
)

var (
	once = flag.Bool("once", false, "Run filebeat only once until all harvesters reach EOF")
)

// Oktabeat configuration.
type Oktabeat struct {
	done     chan struct{}
	client beat.Client
	config   fbConfig.Config
	logger   logp.Logger
}

// New creates an instance of oktabeat.
func New(b *beat.Beat, cfg *common.Config) (beat.Beater, error) {
	logger := logp.NewLogger("oktabeat")
	c := fbConfig.DefaultConfig
	if err := cfg.Unpack(&c); err != nil {
		return nil, fmt.Errorf("Error reading config file: %v", err)
	}
	moduleRegistry, err := fileset.NewModuleRegistry(c.Modules, b.Info.Version, true)
	if err != nil {
		return nil, err
	}
	if !moduleRegistry.Empty() {
		logp.Info("Enabled modules/filesets: %s", moduleRegistry.InfoString())
	}
	
	moduleInputs, err := moduleRegistry.GetInputConfigs()
	if err != nil {
		return nil, err
	}
	if err := c.FetchConfigs(); err != nil {
		return nil, err
	}
	
	// Add inputs created by the modules
	c.Inputs = append(c.Inputs, moduleInputs...)
	
	enabledInputs := c.ListEnabledInputs()
	var haveEnabledInputs bool
	if len(enabledInputs) > 0 {
		haveEnabledInputs = true
	}
	
	if !c.ConfigInput.Enabled() && !c.ConfigModules.Enabled() && !haveEnabledInputs && c.Autodiscover == nil && !b.ConfigManager.Enabled() {
		if !b.InSetupCmd {
			return nil, errors.New("no modules or inputs enabled and configuration reloading disabled. What files do you want me to watch?")
		}
		
		// in the `setup` command, log this only as a warning
		logp.Warn("Setup called, but no modules enabled.")
	}
	
	if *once && c.ConfigInput.Enabled() && c.ConfigModules.Enabled() {
		return nil, errors.New("input configs and -once cannot be used together")
	}
	
	if c.IsInputEnabled("stdin") && len(enabledInputs) > 1 {
		return nil, fmt.Errorf("stdin requires to be run in exclusive mode, configured inputs: %s", strings.Join(enabledInputs, ", "))
	}
	ob := &Oktabeat{
		done:     make(chan struct{}),
		config:   c,
		logger:   *logger,
	}
	return ob, nil
}

// Run starts oktabeat.
func (ob *Oktabeat) Run(b *beat.Beat) error {
	var err error
	config := ob.config
	waitFinished := newSignalWait()
	waitEvents := newSignalWait()
	
	wgEvents := &eventCounter{
		count: monitoring.NewInt(nil, "oktabeat.events.active"),
		added: monitoring.NewUint(nil, "oktabeat.events.added"),
		done:  monitoring.NewUint(nil, "oktabeat.events.done"),
	}
	finishedLogger := newFinishedLogger(wgEvents)
	
	// Setup reg to persist state
	reg, err := registrar.New(ob.config.Registry, finishedLogger)
	if err != nil {
		ob.logger.Error("Could not init reg: %v", err)
		return err
	}
	// Make sure all events that were published in
	registrarChannel := newRegistrarLogger(reg)
	
	err = b.Publisher.SetACKHandler(beat.PipelineACKHandler{
		ACKEvents: newEventACKer(finishedLogger, registrarChannel).ackEvents,
	})
	
	if err != nil {
		ob.logger.Errorf("Failed to install the registry with the publisher pipeline: %v", err)
		return err
	}
	
	outDone := make(chan struct{})
	crawl, err := crawler.New(
		channel.NewOutletFactory(outDone, wgEvents).Create,
		config.Inputs,
		b.Info.Version,
		ob.done,
		*once)
	if err != nil {
		logp.Err("Could not init crawler: %v", err)
		return err
	}
	
	// Start the reg
	err = reg.Start()
	if err != nil {
		return fmt.Errorf("Could not start reg: %v", err)
	}
	// Stopping reg will write last state
	
	defer reg.Stop()
	
	// Stopping publisher (might potentially drop items)
	defer func() {
		// Closes first the reg logger to make sure not more events arrive at the reg
		// registrarChannel must be closed first to potentially unblock (pretty unlikely) the publisher
		registrarChannel.Close()
		close(outDone) // finally close all active connections to publisher pipeline
	}()

	// Wait for all events to be processed or timeout
	defer waitEvents.Wait()
	// Create a ES connection factory for dynamic modules pipeline loading
	var pipelineLoaderFactory fileset.PipelineLoaderFactory
	
	if config.OverwritePipelines {
		logp.L().Debug("modules", "Existing Ingest pipelines will be updated")
	}
	err = crawl.Start(b.Publisher, reg, config.ConfigInput, config.ConfigModules, pipelineLoaderFactory, config.OverwritePipelines)
	if err != nil {
		crawl.Stop()
		return err
	}
	// If run once, add crawler completion check as alternative to done signal
	if *once {
		runOnce := func() {
			logp.Info("Running filebeat once. Waiting for completion ...")
			crawl.WaitForCompletion()
			logp.Info("All data collection completed. Shutting down.")
		}
		waitFinished.Add(runOnce)
	}
	
	
	// Add done channel to wait for shutdown signal
	waitFinished.AddChan(ob.done)
	waitFinished.Wait()
	
	crawl.Stop()
	
	timeout := ob.config.ShutdownTimeout
	// Checks if on shutdown it should wait for all events to be published
	waitPublished := ob.config.ShutdownTimeout > 0 || *once
	if waitPublished {
		// Wait for reg to finish writing registry
		waitEvents.Add(withLog(wgEvents.Wait,
			"Continue shutdown: All enqueued events being published."))
		// Wait for either timeout or all events having been ACKed by outputs.
		if ob.config.ShutdownTimeout > 0 {
			logp.Info("Shutdown output timer started. Waiting for max %v.", timeout)
			waitEvents.Add(withLog(waitDuration(timeout),
				"Continue shutdown: Time out waiting for events being published."))
		} else {
			waitEvents.AddChan(ob.done)
		}
	}
	return nil
}

// Stop stops oktabeat.
func (ob *Oktabeat) Stop() {
	logp.Info("Stopping oktabeat")
	close(ob.done)
}
