package pkg

import (
	"time"
	"context"
	
	okta "github.com/forter/oktabeat/okta"
)

type OktaAPI struct {
	Client okta.APIClient
	Auth context.Context
	Period time.Duration
}
