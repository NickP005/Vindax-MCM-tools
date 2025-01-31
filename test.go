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
	cmd := exec.Command("./tool-2/tool-2", "-n", "1")
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

func createTransaction(account *Account, destAddress string, amount uint64, balance uint64) error {
	// Execute tool-3 to create transaction
	cmd := exec.Command("./tool-3/tool-3",
		"-src", account.,
		"-dst", destAddress,
		"-wots-pk", account.WOTSPublicKey,
		"-change-pk", account.WOTSPublicKey, // Using same WOTS-PK for change
		"-balance", fmt.Sprintf("%d", balance),
		"-amount", fmt.Sprintf("%d", amount),
		"-secret", account.WOTSSecretKey,
		"-memo", "Test transaction")

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func main() {
	// Test 1: Generate account
	fmt.Println("=== Generating new account ===")
	account, err := generateAccount()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating account: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Generated Account:\n")
	fmt.Printf("Address: %s\n", account.MCMAccountNumber)
	fmt.Printf("WOTS-PK: %s\n", account.WOTSPublicKey)
	fmt.Printf("Secret: %s\n\n", account.WOTSSecretKey)

	// Test 2: Create transaction
	fmt.Println("=== Creating transaction ===")
	destAddress := "0000000000000000000000000000000000000001" // Example destination
	err = createTransaction(account, destAddress, 1000, 10000)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating transaction: %v\n", err)
		os.Exit(1)
	}
}
