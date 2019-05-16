package okta

import (
	"context"
	"fmt"
	"time"
	
	"github.com/elastic/beats/filebeat/channel"
	"github.com/elastic/beats/filebeat/harvester"
	"github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/filebeat/input/file"
	fbConfig "github.com/elastic/beats/filebeat/config"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/logp"
	
	"github.com/forter/oktabeat/beater/pkg"
	okta "github.com/forter/oktabeat/okta"
)

func init() {
	logp.L().Info("Registering okta")
	err := input.Register("okta", NewInput)
	if err != nil {
		panic(err)
	}
}

// Input is a input for okta
type Input struct {
	started  bool
	outlet   channel.Outleter
	config   config
	cfg      *common.Config
	registry *harvester.Registry
}

// NewInput creates a new redis input
func NewInput(cfg *common.Config, outletFactory channel.Connector, context input.Context) (input.Input, error) {
	cfgwarn.Experimental("Okta syslog input is enabled.")
	
	config := defaultConfig
	
	err := cfg.Unpack(&config)
	if err != nil {
		return nil, err
	}
	
	outlet, err := outletFactory(cfg, context.DynamicFields)
	if err != nil {
		return nil, err
	}
	
	p := &Input{
		started:  false,
		outlet:   outlet,
		config:   config,
		cfg:      cfg,
		registry: harvester.NewRegistry(),
	}
	
	return p, nil
}

// LoadStates loads the states
func (p *Input) LoadStates(states []file.State) error {
	return nil
}

// Run runs the input
func (p *Input) Run() {
	logp.L().Debug("okta", "Run redis input with domain: %s", p.config.OktaDomain)
	
	forwarder := harvester.NewForwarder(p.outlet)
	api := CreateOktaClient(p.config.OktaDomain, p.config.ApiKey, p.config.Period)
	h, err := NewHarvester(*api)
	if err != nil {
		logp.Err("Failed to create harvester: %v", err)
	}
	h.forwarder = forwarder
	
	if err := p.registry.Start(h); err != nil {
		logp.Err("Harvester start failed: %s", err)
	}
}

// Stop stops the input and all its harvesters
func (p *Input) Stop() {
	p.registry.Stop()
	p.outlet.Close()
}

// Wait waits for the input to be completed. Not implemented.
func (p *Input) Wait() {}

func CreateOktaClient(oktaDomain, token string, period time.Duration) *pkg.OktaAPI {
	oktaConfig := okta.NewConfiguration()
	basePath := fmt.Sprintf("https://%s/api/v1", oktaDomain)
	o := okta.NewAPIClient(oktaConfig)
	o.ChangeBasePath(basePath)
	auth := context.WithValue(context.Background(), okta.ContextAPIKey, okta.APIKey{
		Key:    token,
		Prefix: "SSWS",
	})
	return &pkg.OktaAPI{
		Client: *o,
		Auth: auth,
		Period: period,
	}
}
