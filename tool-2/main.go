package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	wots "github.com/NickP005/WOTS-Go"
)

type Account struct {
	MCMAccountNumber string `json:"mcmAccountNumber"`
	WOTSPublicKey    string `json:"wotsPublicKey"`
	WOTSSecretKey    string `json:"wotsSecretKey"`
}

type Output struct {
	Accounts []Account `json:"accounts"`
}

type Components struct {
	PrivateSeed [32]byte
	PublicSeed  [32]byte
	AddrSeed    [32]byte
}

func mochimoHash(data []byte) [32]byte {
	hash := sha256.Sum256(data)
	return hash
}

/*
 * ComponentsGenerator derives three different seeds from an initial WOTS seed
 *
 * Parameters:
 * - wotsSeed: byte array of 32 bytes used as the initial seed
 *
 * Returns:
 * - Components: struct containing:
 *   1. PrivateSeed: 32 bytes used for WOTS secret key generation
 *   2. PublicSeed: 32 bytes used for WOTS public key generation
 *   3. AddrSeed: 32 bytes used for MCM address generation
 *
 * The function appends different strings to the seed ("seed", "publ", "addr")
 * and hashes each combination to generate the components
 */
func componentsGenerator(wotsSeed []byte) Components {
	seedAscii := string(wotsSeed)
	privateSeed := mochimoHash([]byte(seedAscii + "seed"))
	publicSeed := mochimoHash([]byte(seedAscii + "publ"))
	addrSeed := mochimoHash([]byte(seedAscii + "addr"))

	return Components{
		PrivateSeed: privateSeed,
		PublicSeed:  publicSeed,
		AddrSeed:    addrSeed,
	}
}

/*
 * GenerateAccount creates a new MCM 3.0 account using WOTS signatures
 *
 * Parameters:
 * - seed: byte array of exactly 32 bytes used as the initial seed
 * - index: uint64 used to generate unique addresses for multiple accounts
 *
 * Returns:
 * - *Account: contains MCM account number (20 bytes hex), WOTS public key (2208 bytes hex),
 *            and WOTS secret key (32 bytes hex)
 * - error: if seed length is invalid or if generation fails
 *
 * The function uses the seed to generate three components via componentsGenerator:
 * 1. Private seed - used for WOTS secret key
 * 2. Public seed - used for WOTS public key generation
 * 3. Address seed - used for MCM account number
 */
func generateAccount(seed []byte, index uint64) (*Account, error) {
	if len(seed) != 32 {
		return nil, fmt.Errorf("seed must be exactly 32 bytes, got %d", len(seed))
	}
	var privateKey [32]byte
	copy(privateKey[:], seed)

	keypair, err := wots.Keygen(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to generate WOTS keypair: %v", err)
	}

	var public_key [2208]byte
	copy(public_key[:], keypair.PublicKey[:])
	copy(public_key[2144:], keypair.Components.PublicSeed[:])
	copy(public_key[2144+32:], keypair.Components.AddrSeed[:])

	// Set the last 12 bytes of public key to default tag
	copy(public_key[2208-12:], []byte{66, 0, 0, 0, 14, 0, 0, 0, 1, 0, 0, 0})

	return &Account{
		MCMAccountNumber: fmt.Sprintf("%020x", index),
		WOTSPublicKey:    hex.EncodeToString(public_key[:]),
		WOTSSecretKey:    hex.EncodeToString(seed),
	}, nil
}

/*
 * Main function for the MCM 3.0 WOTS keypair generator tool
 *
 * Command line flags:
 * -n uint: number of accounts to generate (default: 1)
 *
 * For each account:
 * 1. Generates a random 32-byte seed
 * 2. Derives WOTS components (private, public, address seeds)
 * 3. Generates WOTS keypair and MCM account number
 *
 * Outputs JSON containing array of accounts with:
 * - mcmAccountNumber: 20 bytes hex (index based)
 * - wotsPublicKey: 2208 bytes hex
 * - wotsSecretKey: 32 bytes hex
 */
func main() {
	numAccounts := flag.Uint64("n", 1, "number of accounts to generate")
	flag.Parse()

	output := Output{
		Accounts: make([]Account, 0, *numAccounts),
	}

	for i := uint64(0); i < *numAccounts; i++ {
		// Generate random seed for each account
		seed := make([]byte, 32)
		if _, err := rand.Read(seed); err != nil {
			fmt.Fprintf(os.Stderr, "Error generating random seed: %v\n", err)
			os.Exit(1)
		}

		account, err := generateAccount(seed, i)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating account %d: %v\n", i, err)
			os.Exit(1)
		}
		output.Accounts = append(output.Accounts, *account)
	}

	// Output JSON
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(output); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
		os.Exit(1)
	}
}
