package midtrans

import (
	"github.com/midtrans/midtrans-go"
	"github.com/midtrans/midtrans-go/coreapi"
	"github.com/midtrans/midtrans-go/snap"
)

type Config struct {
	ServerKey   string
	ClientKey   string
	Environment string // "sandbox" or "production"
}

type MidtransClient struct {
	Snap      snap.Client
	CoreAPI   coreapi.Client
	ClientKey string
}

func Setup(cfg *Config) *MidtransClient {
	env := midtrans.Sandbox
	if cfg.Environment == "production" {
		env = midtrans.Production
	}

	var snapClient snap.Client
	snapClient.New(cfg.ServerKey, env)

	var coreAPIClient coreapi.Client
	coreAPIClient.New(cfg.ServerKey, env)

	return &MidtransClient{
		Snap:      snapClient,
		CoreAPI:   coreAPIClient,
		ClientKey: cfg.ClientKey,
	}
}

func (m *MidtransClient) SnapBaseURL() string {
	if m.Snap.Env == midtrans.Production {
		return "https://app.midtrans.com/snap/snap.js"
	}
	return "https://app.sandbox.midtrans.com/snap/snap.js"
}
