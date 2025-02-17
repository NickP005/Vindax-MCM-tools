package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
)

// Account matches the structure from tool-2
type Account struct {
	MCMAccountNumber string `json:"mcmAccountNumber"`
	WOTSPublicKey    string `json:"wotsPublicKey"`
	WOTSSecretKey    string `json:"wotsSecretKey"`
}

type Output struct {
	Accounts []Account `json:"accounts"`
}

func generateAccount() (*Account, error) {
	// Execute tool-2 to generate one account
	cmd := exec.Command("./tool-2", "-n", "1")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute tool-2: %v", err)
	}

	// Parse the JSON output
	var result Output
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %v", err)
	}

	if len(result.Accounts) == 0 {
		return nil, fmt.Errorf("no accounts generated")
	}

	return &result.Accounts[0], nil
}

func createTransaction(sourceAddress string, sourcePublicKey string, sourceSecret string, sourceBalance uint64,
	changePublicKey string, destAddress string, amount uint64) error {
	//fmt.Println("Source address:", sourceAddress)
	//fmt.Println("Source secret:", sourceSecret)
	//fmt.Println("Change address:", changeAddress)
	//fmt.Println("Destination address:", destAddress)

	// Execute tool-3 to create transaction with updated parameters
	cmd := exec.Command("./tool-3",
		"-src", sourceAddress,
		"-source-pk", sourcePublicKey,
		"-dst", destAddress,
		"-change-pk", changePublicKey,
		"-balance", fmt.Sprintf("%d", sourceBalance),
		"-amount", fmt.Sprintf("%d", amount),
		"-secret", sourceSecret,
		"-memo", "TEST",
		"-fee", "500")

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func main() {
	// Read and parse cache.json
	data, err := os.ReadFile("cache.json")
	if err != nil {
		fmt.Printf("Failed to read cache.json: %v\n", err)
		return
	}

	var output Output
	if err := json.Unmarshal(data, &output); err != nil {
		fmt.Printf("Failed to parse JSON: %v\n", err)
		return
	}

	// Print the account numbers
	/*
		for i, account := range output.Accounts {
			fmt.Printf("Account %d: %s\n", i+1, account.WOTSSecretKey)
		}*/

	// Get the addresses from tool-1 by giving the full account WOTSPublicKey
	var addresses []string
	for _, account := range output.Accounts {
		cmd := exec.Command("./tool-1", "-wots", account.WOTSPublicKey)
		addressOutput, err := cmd.Output()
		if err != nil {
			fmt.Printf("Failed to get address for account: %v\n", err)
			continue
		}
		// remove \n newline
		addressOutput = addressOutput[:len(addressOutput)-1]
		addresses = append(addresses, string(addressOutput))
	}

	// Print the addresses
	meshClient := NewMeshAPIClient("http://localhost:8080")
	for i, address := range addresses {
		//fmt.Printf("Address %d: %s\n", i+1, address)
		err, full_address, amount := meshClient.ResolveTAG(address)
		if err != nil {
			fmt.Printf("Failed to resolve TAG %s: %v\n", address, err)
			continue
		}
		fmt.Printf("Resolved TAG %s to address %s (%d) with amount %d\n", address, full_address, i, amount)
	}

	// Send transaction
	if len(output.Accounts) < 3 {
		fmt.Println("Need at least 2 accounts to send a transaction")
		return
	}

	sourceAccount := output.Accounts[1]
	changeAccount := output.Accounts[0]
	destAddress := addresses[2]

	// Resolve TAG of source address
	err, address, amount := meshClient.ResolveTAG(addresses[0])
	if err != nil {
		fmt.Printf("Failed to resolve TAG: %v\n", err)
		return
	}
	//fmt.Printf("Resolved TAG %s to address %s with amount %d\n", addresses[1], address, amount)

	if err := createTransaction(address[2:], sourceAccount.WOTSPublicKey, sourceAccount.WOTSSecretKey, amount, changeAccount.WOTSPublicKey, destAddress, 5); err != nil {
		fmt.Printf("Failed to create transaction: %v\n", err)
		return
	}

	//fmt.Println("Transaction created successfully")

}
