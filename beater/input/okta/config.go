package okta

import (
	"time"
	
	"github.com/elastic/beats/filebeat/harvester"
)
const apiKeyFiller = "FILLER"

var defaultConfig = config{
	ForwarderConfig: harvester.ForwarderConfig{
		Type: "redis",
	},
	ApiKey:          apiKeyFiller,
	OktaDomain:      apiKeyFiller,
	Period:          60 * time.Second,
}

type config struct {
	harvester.ForwarderConfig `config:",inline"`
	OktaDomain      string          `config:"okta_domain"`
	ApiKey          string          `config:"api_key"`
	Period          time.Duration   `config:"period"`
}
