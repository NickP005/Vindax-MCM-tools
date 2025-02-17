package main

/*
 * MCM 3.0 Transaction Submission Tool
 *
 * This tool creates and submits transactions to the Mochimo network via the Mesh API.
 *
 * Required inputs:
 * -src: Source account address (20 bytes hex)
 * -dst: Destination account address (20 bytes hex)
 * -wots-pk: Source WOTS public key (2208 bytes hex)
 * -change-pk: Change WOTS public key (2208 bytes hex)
 * -balance: Source balance in nanoMCM
 * -amount: Amount to send in nanoMCM
 * -secret: Secret key for signing (32 bytes hex)
 * -memo: Optional transaction memo
 * -fee: Transaction fee in nanoMCM (default: 500)
 * -api: Mesh API endpoint (default: http://localhost:8080)
 */

import (
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	wots "github.com/NickP005/WOTS-Go"
	mcm "github.com/NickP005/go_mcminterface"
)

// MeshAPISubmitRequest represents the request body for /construction/submit

/*
 * MeshAPISubmitRequest represents the JSON structure for submitting
 * a signed transaction to the Mochimo Mesh API
 *
 * Fields:
 * - NetworkIdentifier: Identifies the blockchain network
 *   - Blockchain: Name of the blockchain (always "mochimo")
 *   - Network: Network name (e.g. "mainnet")
 * - SignedTransaction: Hex-encoded signed transaction data
 */
type MeshAPISubmitRequest struct {
	NetworkIdentifier struct {
		Blockchain string `json:"blockchain"`
		Network    string `json:"network"`
	} `json:"network_identifier"`
	SignedTransaction string `json:"signed_transaction"`
}

// Add new type for parse request
type ConstructionParseRequest struct {
	NetworkIdentifier NetworkIdentifier `json:"network_identifier"`
	Signed            bool              `json:"signed"`
	Transaction       string            `json:"transaction"`
}

type NetworkIdentifier struct {
	Blockchain string `json:"blockchain"`
	Network    string `json:"network"`
}

/*
 * main is the entry point for the MCM transaction submission tool
 *
 * The function:
 * 1. Parses and validates command line arguments
 * 2. Creates a new transaction using the MCM interface
 * 3. Sets transaction parameters (addresses, amounts, fee)
 * 4. Generates transaction components from the secret key
 * 5. Signs the transaction using WOTS
 * 6. Creates a Mesh API submission request
 * 7. Outputs the request as formatted JSON
 *
 * Required flags:
 * -src: Source account address
 * -source-pk: Source WOTS public key
 * -change-pk: Change WOTS public key
 * -balance: Source balance in nanoMCM
 * -dst: Destination account address
 * -amount: Amount to send in nanoMCM
 * -secret: Secret key for signing
 * -memo: Transaction memo
 * -fee: Transaction fee (default: 500 nanoMCM)
 *
 * Optional flags:
 * -memo: Transaction memo
 * -fee: Transaction fee (default: 500 nanoMCM)
 */
func main() {
	// Define command line flags
	sourceTag := flag.String("src", "", "Source account address (20 bytes hex)")
	sourcePk := flag.String("source-pk", "", "Source WOTS public key (2208 bytes hex)")
	changePk := flag.String("change-pk", "", "Change WOTS public key (2208 bytes hex)")
	sourceBalance := flag.Uint64("balance", 0, "Source balance in nanoMCM")
	dstAddress := flag.String("dst", "", "Destination account address (20 bytes hex)")
	amount_int := flag.Int64("amount", -1, "Amount to send in nanoMCM")
	secret := flag.String("secret", "", "Secret key for signing (32 bytes hex)")
	memo := flag.String("memo", "", "Optional transaction memo")
	fee := flag.Uint64("fee", 500, "Transaction fee in nanoMCM")
	//api := flag.String("api", "http://localhost:8080", "Mesh API endpoint")

	flag.Parse()

	// Validate inputs
	if *sourceTag == "" && len(*sourceTag) != 40 {
		fmt.Fprintln(os.Stderr, "Error: Source account address is required")
		os.Exit(1)
	} else if *sourcePk == "" && len(*sourcePk) != 2208*2 {
		fmt.Fprintln(os.Stderr, "Error: Source WOTS public key is required")
		os.Exit(1)
	} else if *changePk == "" && len(*changePk) != 2208*2 {
		fmt.Fprintln(os.Stderr, "Error: Change WOTS public key is required")
		os.Exit(1)
	} else if *sourceBalance == 0 {
		fmt.Fprintln(os.Stderr, "Error: Source balance is required")
		os.Exit(1)
	} else if *dstAddress == "" && len(*dstAddress) != 40 {
		fmt.Fprintln(os.Stderr, "Error: Destination address is required")
		os.Exit(1)
	} else if *amount_int < 0 {
		fmt.Fprintln(os.Stderr, "Error: Amount to send is required")
		os.Exit(1)
	} else if *secret == "" {
		fmt.Fprintln(os.Stderr, "Error: Secret key is required")
		os.Exit(1)
	}

	// Convert amount to uint64
	amount_uint := uint64(*amount_int)
	amount := &amount_uint

	tag, err := hex.DecodeString(*sourceTag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding source tag: %v\n", err)
		os.Exit(1)
	}

	// Source balance must be greater than amount + fee
	if *sourceBalance < *amount+*fee {
		fmt.Fprintln(os.Stderr, "Error: Insufficient balance to send amount and fee")
		os.Exit(1)
	}

	// Create transaction using mcminterface
	tx := mcm.NewTXENTRY()

	// Set source and change addresses
	srcAddr := mcm.WotsAddressFromHex((*sourcePk)[:2208*2-64*2]) // Remove last 64 bytes (public seed and addrss) leaving just the public key
	srcAddr.SetTAG(tag)
	chgAddr := mcm.WotsAddressFromHex((*changePk)[:2208*2-64*2])
	chgAddr.SetTAG(tag)
	tx.SetSourceAddress(srcAddr)
	tx.SetChangeAddress(chgAddr)

	// Set amounts
	tx.SetSendTotal(*amount)
	tx.SetChangeTotal(*sourceBalance - *amount - *fee)
	tx.SetFee(*fee)

	// Add destination
	dstEntry := mcm.NewDSTFromString(*dstAddress, *memo, *amount)
	if !dstEntry.ValidateReference() {
		fmt.Fprintln(os.Stderr, "Error: Invalid memo")
		os.Exit(1)
	}
	tx.AddDestination(dstEntry)
	tx.SetDestinationCount(1)

	// Generate transaction hash
	var message [32]byte = tx.GetMessageToSign()

	// Sign transaction
	secretBytes, err := hex.DecodeString(*secret)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding secret key: %v\n", err)
		os.Exit(1)
	}
	var private_key [32]byte
	copy(private_key[:], secretBytes)
	signing_keypair, _ := wots.Keygen(private_key)

	// Check that public key matches source address
	derived_address := mcm.WotsAddressFromBytes(signing_keypair.PublicKey[:])
	derived_address.SetTAG(tag)
	if derived_address.Address != srcAddr.Address {
		fmt.Println("wots from priv", mcm.WotsAddressFromBytes(signing_keypair.PublicKey[:]).Address)
		fmt.Println("given wots", srcAddr.Address)
		fmt.Fprintln(os.Stderr, "Error: Public key does not match source address")
		os.Exit(1)
	}

	// Sign with fixed length inputs
	var signature [2144]byte = signing_keypair.Sign(message)
	tx.SetWotsSignature(signature[:])

	var addr_seed_default_tag [32]byte
	copy(addr_seed_default_tag[:], signing_keypair.Components.AddrSeed[:20])
	copy(addr_seed_default_tag[20:], []byte{0x42, 0x00, 0x00, 0x00, 0x0e, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00})
	tx.SetWotsSigAddresses(addr_seed_default_tag[:])
	tx.SetWotsSigPubSeed(signing_keypair.Components.PublicSeed)

	tx.SetSignatureScheme("wotsp")

	tx.SetBlockToLive(0)

	/*
			// Create parse request
		request := ConstructionParseRequest{
			NetworkIdentifier: struct {
				Blockchain string `json:"blockchain"`
				Network    string `json:"network"`
			}{
				Blockchain: "mochimo",
				Network:    "mainnet",
			},
			Signed:      true,
			Transaction: tx.String(),
		}*/

	// Create submit request
	request := MeshAPISubmitRequest{
		NetworkIdentifier: struct {
			Blockchain string `json:"blockchain"`
			Network    string `json:"network"`
		}{
			Blockchain: "mochimo",
			Network:    "mainnet",
		},
		SignedTransaction: tx.String(),
	}

	// Output JSON
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(request); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
		os.Exit(1)
	}
}
