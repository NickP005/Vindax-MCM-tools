package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type MeshAPIClient struct {
	endpoint string
}

func NewMeshAPIClient(endpoint string) *MeshAPIClient {
	return &MeshAPIClient{endpoint: endpoint}
}

func (c *MeshAPIClient) ResolveTAG(tag_hex string) (error, string, uint64) {
	fmt.Println("Resolving TAG", tag_hex)
	resp, err := http.Post(c.endpoint+"/call", "application/json", bytes.NewBuffer([]byte(fmt.Sprintf(`{
		"network_identifier": {
			"blockchain": "mochimo",
			"network": "mainnet"
		},
		"method": "tag_resolve",
		"parameters": {
			"tag": "0x%s"
		}
	}`, tag_hex))))

	if err != nil {
		return err, "", 0
	}
	defer resp.Body.Close()

	var result struct {
		Result struct {
			Address string `json:"address"`
			Amount  uint64 `json:"amount"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err, "", 0
	}

	if string(result.Result.Address) == "" {
		return fmt.Errorf("TAG not found"), "", 0
	}

	return nil, result.Result.Address, result.Result.Amount
}
