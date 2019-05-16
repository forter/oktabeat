package okta

import (
	"time"
	
	"github.com/antihax/optional"
	"github.com/elastic/beats/filebeat/harvester"
	"github.com/elastic/beats/filebeat/util"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/gofrs/uuid"
	"github.com/mitchellh/mapstructure"
	
	"github.com/forter/oktabeat/beater/pkg"
	okta "github.com/forter/oktabeat/okta"
)

type Harvester struct {
	id        uuid.UUID
	done      chan struct{}
	oktaAPI  pkg.OktaAPI
	forwarder *harvester.Forwarder
}

func NewHarvester(api pkg.OktaAPI) (*Harvester, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	
	return &Harvester{
		id:   id,
		done: make(chan struct{}),
		oktaAPI: api,
	}, nil
}


// Run starts a new redis harvester
func (h *Harvester) Run() error {
	
	select {
	case <-h.done:
		return nil
	default:
	}
	now := time.Now().UTC()
	since := now.Add(h.oktaAPI.Period)
	options := &okta.GetLogsOpts{
		Since: optional.NewString(since.Format(time.RFC3339)),
		Limit: optional.NewInt32(1000),
	}
	logs, r, err := h.oktaAPI.Client.LogsApi.GetLogs(h.oktaAPI.Auth, options)
	if err != nil {
		logp.L().Error(err)
		return err
	}
	next := r.Header.Get("Link")
	logp.L().Info(next)
	data := util.NewData()
	for _, log := range logs {
		values, err := EventLogToCommonMap(&log)
		if err != nil {
			logp.L().Error("Could not convert okta log event into common map")
			return err
		}
		data.Event = beat.Event{
			Timestamp: time.Now().UTC(),
			Fields: values,
		}
		
		h.forwarder.Send(data)
		
	}
	
		

	return nil
}

// Stop stops the harvester
func (h *Harvester) Stop() {
	close(h.done)
}

func (h *Harvester) ID() uuid.UUID {
	return h.id
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
