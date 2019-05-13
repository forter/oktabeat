// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
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
	"context"
	"fmt"
	"time"
	
	"github.com/antihax/optional"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/mitchellh/mapstructure"
	
	"github.com/forter/oktabeat/config"
	okta "github.com/forter/oktabeat/okta"
)

// Oktabeat configuration.
type Oktabeat struct {
	done     chan struct{}
	config   config.Config
	client   beat.Client
	okta     okta.APIClient
	oktaAuth context.Context
	logger   logp.Logger
}

func GetSystemLogs(o Oktabeat) ([]okta.LogEvent, error) {
	now := time.Now().UTC()
	since := now.Add(o.config.Period)
	options := &okta.GetLogsOpts{
		Since: optional.NewString(since.Format(time.RFC3339)),
		Limit: optional.NewInt32(1000),
	}
	value, r, err := o.okta.LogsApi.GetLogs(o.oktaAuth, options)
	if err != nil {
		o.logger.Error(err)
		return nil, err
	}
	next := r.Header.Get("Link")
	o.logger.Info(next)
	return value, nil
}

func EventLogToCommonMap(event *okta.LogEvent) (common.MapStr, error) {
	var result common.MapStr
	err := mapstructure.Decode(event, &result)
	if err != nil {
		logp.L().Error("Error decoding Okta LogEvent record", err)
		return nil, err
	}
	return result, nil
}

// New creates an instance of oktabeat.
func New(b *beat.Beat, cfg *common.Config) (beat.Beater, error) {
	logger := logp.NewLogger("internal")
	c := config.DefaultConfig
	if err := cfg.Unpack(&c); err != nil {
		return nil, fmt.Errorf("Error reading config file: %v", err)
	}

	oktaConfig := okta.NewConfiguration()
	basePath := fmt.Sprintf("https://%s/api/v1", c.OktaDomain)
	logger.Infof("Okta domain configured %s", basePath)
	o := okta.NewAPIClient(oktaConfig)
	o.ChangeBasePath(basePath)
	auth := context.WithValue(context.Background(), okta.ContextAPIKey, okta.APIKey{
		Key:    c.ApiKey,
		Prefix: "SSWS",
	})
	bt := &Oktabeat{
		done:     make(chan struct{}),
		config:   c,
		okta:     *o,
		oktaAuth: auth,
		logger:   *logger,
	}
	return bt, nil
}

// Run starts oktabeat.
func (bt *Oktabeat) Run(b *beat.Beat) error {
	bt.logger.Info("oktabeat is running! Hit CTRL-C to stop it.")

	var err error
	bt.client, err = b.Publisher.Connect()
	if err != nil {
		return err
	}

	ticker := time.NewTicker(bt.config.Period)
	for {
		select {
		case <-bt.done:
			return nil
		case <-ticker.C:
		}
		logs, err := GetSystemLogs(*bt)
		if err != nil {
			bt.logger.Error(err)
		}
		fields := common.MapStr{}
		fields["type"] = b.Info.Name
		for _, log := range logs {
			values, err := EventLogToCommonMap(&log)
			if err != nil {
				bt.logger.Error("Could not convert okta log event into common map")
				return err
			}
			event := beat.Event{
				Timestamp: time.Now(),
				Fields:    values,
			}
			bt.client.Publish(event)
			bt.logger.Info("Event sent")
		}
	}
}

// Stop stops oktabeat.
func (bt *Oktabeat) Stop() {
	bt.client.Close()
	close(bt.done)
}
