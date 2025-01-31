package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	wots "github.com/NickP005/WOTS-Go"
	mcm "github.com/NickP005/go_mcminterface"
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
		"-src", account.MCMAccountNumber,
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
	/*
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
		fmt.Printf("Secret: %s\n\n", account.WOTSSecretKey)*/

	// Generate keypair from cf1fca58d97b5fbb8b8221e94ec1f91048fb9597303e771a7de45891324bcfa0
	private_seed := [32]byte{0xcf, 0x1f, 0xca, 0x58, 0xd9, 0x7b, 0x5f, 0xbb, 0x8b, 0x82, 0x21, 0xe9, 0x4e, 0xc1, 0xf9, 0x10, 0x48, 0xfb, 0x95, 0x97, 0x30, 0x3e, 0x77, 0x1a, 0x7d, 0xe4, 0x58, 0x91, 0x32, 0x4b, 0xcf, 0xa0}
	keychain, _ := wots.NewKeychain(private_seed)
	keypair := keychain.Next()
	fmt.Printf("Public Key: %x\n", keypair.PublicKey)
	addr := mcm.WotsAddressFromBytes(keypair.PublicKey[:])
	fmt.Println("Public Key: ", hex.EncodeToString(addr.GetAddress()))
}
