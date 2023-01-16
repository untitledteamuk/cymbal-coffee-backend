// Handles all communication to and from the Bond Service
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

// TODO: STORE IN SECRETS MANAGER
const defaultBondURL = "https://bond-service-l5xebjflvq-ew.a.run.app"

var bondCfg bondConfig

type bondConfig struct {
	BondURL string
}

func initBond() {
	url := os.Getenv("BOND_SERVICE_URL")
	if url == "" {
		url = defaultBondURL
	}

	bondCfg = bondConfig{
		BondURL: url,
	}

}

// Sends a JSON request as a POST body to bond and returns the raw bytes from the response
func sendJson(ctx context.Context, endpoint string, body any) (b []byte, err error) {
	// Marshall
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return b, err
	}
	url := bondCfg.BondURL + endpoint
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return b, err
	}
	req.Header.Set("Content-Type", "application/json")
	client := http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return b, err
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return b, fmt.Errorf("expected 200 response")
	}

	b, err = io.ReadAll(res.Body)
	if err != nil {
		return b, err
	}
	return b, nil
}
