package main

import (
	"bytes"
	"crypto/rand"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	wots "github.com/NickP005/WOTS-Go"
	mcm "github.com/NickP005/go_mcminterface"
	"github.com/btcsuite/btcutil/base58"
	"github.com/sigurn/crc16"
)

const (
	MESH_API_URL           = "http://api-aus.mochimo.org:8080" // Changed to match the example URL
	MAX_INDEX_SEARCH       = 1000
	CHECK_MEMPOOL_INTERVAL = 5 // seconds
)

// Types for wallet cache
type WalletCache struct {
	SecretKey     string `json:"secretKey"`
	Index         uint64 `json:"index"`
	RefillAddress string `json:"refillAddress,omitempty"`
}

// Types for entries
type SendEntry struct {
	Address      string
	AddressBin   []byte
	AmountToSend uint64
	Balance      uint64
	Memo         string // Added memo field
}

// Types for API responses
type NetworkStatus struct {
	CurrentBlockIdentifier struct {
		Index uint64 `json:"index"`
		Hash  string `json:"hash"`
	} `json:"current_block_identifier"`
}

type AccountBalance struct {
	BlockIdentifier struct {
		Index uint64 `json:"index"`
		Hash  string `json:"hash"`
	} `json:"block_identifier"`
	Balances []struct {
		Value    string `json:"value"`
		Currency struct {
			Symbol   string `json:"symbol"`
			Decimals int    `json:"decimals"`
		} `json:"currency"`
	} `json:"balances"`
}

type TagResolveResponse struct {
	Result struct {
		Address string `json:"address"`
		Amount  uint64 `json:"amount"`
	} `json:"result"`
}

type MempoolResponse struct {
	TransactionIdentifiers []struct {
		Hash string `json:"hash"`
	} `json:"transaction_identifiers"`
}

type MeshAPISubmitRequest struct {
	NetworkIdentifier struct {
		Blockchain string `json:"blockchain"`
		Network    string `json:"network"`
	} `json:"network_identifier"`
	SignedTransaction string `json:"signed_transaction"`
}

type MeshAPISubmitResponse struct {
	TransactionIdentifier struct {
		Hash string `json:"hash"`
	} `json:"transaction_identifier"`
}

// BlockResponse is the response from the /block endpoint
type BlockResponse struct {
	Block struct {
		BlockIdentifier struct {
			Index uint64 `json:"index"`
			Hash  string `json:"hash"`
		} `json:"block_identifier"`
		Transactions []struct {
			TransactionIdentifier struct {
				Hash string `json:"hash"`
			} `json:"transaction_identifier"`
		} `json:"transactions"`
	} `json:"block"`
}

// ValidateBase58Address verifies that an address is valid base58 and has correct CRC16
func ValidateBase58Address(addr string) (bool, []byte) {
	// Check length
	if len(addr) > 255 {
		return false, nil
	}

	// Decode base58
	decoded := base58.Decode(addr)
	if len(decoded) != 22 {
		return false, nil
	}

	// Extract tag and stored checksum (little-endian)
	tag := decoded[:20]
	storedCsum := uint16(decoded[21])<<8 | uint16(decoded[20])

	// Calculate CRC on tag portion using XMODEM
	table := crc16.MakeTable(crc16.CRC16_XMODEM)
	actualCrc := crc16.Checksum(tag, table)

	return storedCsum == actualCrc, tag
}

// GetAccountBalance retrieves balance for an address from Mesh API
func GetAccountBalance(address []byte) (uint64, error) {
	addrHex := hex.EncodeToString(address)

	// Create request body
	reqBody := map[string]interface{}{
		"network_identifier": map[string]string{
			"blockchain": "mochimo",
			"network":    "mainnet",
		},
		"account_identifier": map[string]string{
			"address": "0x" + addrHex,
		},
	}

	reqJSON, _ := json.Marshal(reqBody)

	// Make request
	resp, err := http.Post(
		MESH_API_URL+"/account/balance",
		"application/json",
		strings.NewReader(string(reqJSON)),
	)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return 0, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	// Parse response
	var balanceResp AccountBalance
	err = json.NewDecoder(resp.Body).Decode(&balanceResp)
	if err != nil {
		return 0, err
	}

	// Check if balances exist
	if len(balanceResp.Balances) == 0 {
		return 0, nil
	}

	// Parse balance
	balance, err := strconv.ParseUint(balanceResp.Balances[0].Value, 10, 64)
	if err != nil {
		return 0, err
	}

	return balance, nil
}

// ReadEntriesCSV reads and validates entries from a CSV file
func ReadEntriesCSV(filename string) ([]SendEntry, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = ' ' // Space-separated

	lines, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	entries := make([]SendEntry, 0, len(lines))

	fmt.Println("Validating entries:")
	fmt.Println("-------------------")

	for i, line := range lines {
		// Accept 2 or 3 fields (address, amount, [optional memo])
		if len(line) < 2 || len(line) > 3 {
			return nil, fmt.Errorf("line %d: expected 2 or 3 fields (address, amount, [memo]), got %d", i+1, len(line))
		}

		address := strings.TrimSpace(line[0])
		amountStr := strings.TrimSpace(line[1])

		// Optional memo field
		memo := ""
		if len(line) == 3 {
			memo = strings.TrimSpace(line[2])
		}

		// Validate address
		valid, addressBin := ValidateBase58Address(address)
		if !valid {
			return nil, fmt.Errorf("line %d: invalid address format or checksum", i+1)
		}

		// Parse amount
		amount, err := strconv.ParseUint(amountStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("line %d: invalid amount format - %v", i+1, err)
		}

		// Validate memo if provided
		if memo != "" {
			dstEntry := mcm.NewDSTFromString(hex.EncodeToString(addressBin), memo, amount)
			if !dstEntry.ValidateReference() {
				return nil, fmt.Errorf("line %d: invalid memo format", i+1)
			}
		}

		// Check balance
		balance, err := GetAccountBalance(addressBin)
		if err != nil {
			return nil, fmt.Errorf("line %d: failed to check balance - %v", i+1, err)
		}

		entry := SendEntry{
			Address:      address,
			AddressBin:   addressBin,
			AmountToSend: amount,
			Balance:      balance,
			Memo:         memo,
		}

		// Log validation result
		if memo != "" {
			fmt.Printf("%s (balance: %d nMCM) → sending %d nMCM (memo: %s)\n", address, balance, amount, memo)
		} else {
			fmt.Printf("%s (balance: %d nMCM) → sending %d nMCM\n", address, balance, amount)
		}

		entries = append(entries, entry)
	}

	fmt.Println("-------------------")
	return entries, nil
}

// GetRefillAddress gets the base58 address for refilling (always using index 0)
func GetRefillAddress(secretKey string) (string, error) {
	// Decode secret key
	secretBytes, err := hex.DecodeString(secretKey)
	if err != nil {
		return "", err
	}

	// Create keychain with seed
	var seed [32]byte
	copy(seed[:], secretBytes)
	keychain, err := wots.NewKeychain(seed)
	if err != nil {
		return "", err
	}

	// Always use index 0 for refill address
	keychain.Index = 0
	keypair := keychain.Next()

	// Extract the public key without the last 64 bytes (32 bytes public seed + 32 bytes addr seed)
	publicKeyBytes := keypair.PublicKey[:2144]

	// Use go_mcminterface to get the tag (address) from the WOTS public key
	mcmAddr := mcm.WotsAddressFromBytes(publicKeyBytes)
	tag := mcmAddr.GetAddress()

	// Convert to base58
	return AddrToBase58(tag), nil
}

// ReadWalletCache reads the wallet cache from file or creates a new one
func ReadWalletCache(filename string) (*WalletCache, error) {
	data, err := ioutil.ReadFile(filename)

	// If file doesn't exist or is empty, create new wallet cache
	if os.IsNotExist(err) || len(data) == 0 {
		fmt.Println("Creating new wallet cache...")

		// Generate random seed
		var seed [32]byte
		_, err = rand.Read(seed[:])
		if err != nil {
			return nil, fmt.Errorf("failed to generate random seed: %v", err)
		}

		// Create new wallet cache
		secretKeyHex := hex.EncodeToString(seed[:])

		// Get the refill address (index 0)
		refillAddr, err := GetRefillAddress(secretKeyHex)
		if err != nil {
			return nil, fmt.Errorf("failed to generate refill address: %v", err)
		}

		cache := &WalletCache{
			SecretKey:     secretKeyHex,
			Index:         0,
			RefillAddress: refillAddr,
		}

		// Save to file
		saveErr := SaveWalletCache(filename, cache)
		if saveErr != nil {
			return nil, saveErr
		}

		return cache, nil
	}

	if err != nil {
		return nil, err
	}

	// Parse existing wallet cache
	var cache WalletCache
	err = json.Unmarshal(data, &cache)
	if err != nil {
		return nil, err
	}

	// If the refill address isn't set in an existing wallet cache, set it now
	if cache.RefillAddress == "" {
		refillAddr, err := GetRefillAddress(cache.SecretKey)
		if err != nil {
			return nil, fmt.Errorf("failed to generate refill address: %v", err)
		}
		cache.RefillAddress = refillAddr

		// Save updated cache
		saveErr := SaveWalletCache(filename, &cache)
		if saveErr != nil {
			return nil, saveErr
		}
	}

	return &cache, nil
}

// SaveWalletCache writes the wallet cache to file
func SaveWalletCache(filename string, cache *WalletCache) error {
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filename, data, 0600)
}

// ResolveTag uses Mesh API to resolve an address tag
func ResolveTag(tag []byte) (string, uint64, error) {
	tagHex := hex.EncodeToString(tag)

	// Create request body
	reqBody := map[string]interface{}{
		"network_identifier": map[string]string{
			"blockchain": "mochimo",
			"network":    "mainnet",
		},
		"method": "tag_resolve",
		"parameters": map[string]string{
			"tag": "0x" + tagHex,
		},
	}

	reqJSON, _ := json.Marshal(reqBody)

	// Make request
	resp, err := http.Post(
		MESH_API_URL+"/call",
		"application/json",
		strings.NewReader(string(reqJSON)),
	)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", 0, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	// Parse response
	var tagResp TagResolveResponse
	err = json.NewDecoder(resp.Body).Decode(&tagResp)
	if err != nil {
		return "", 0, err
	}

	return tagResp.Result.Address, tagResp.Result.Amount, nil
}

// GetNetworkStatus retrieves current network status from Mesh API
func GetNetworkStatus() (*NetworkStatus, error) {
	// Create request body
	reqBody := map[string]interface{}{
		"network_identifier": map[string]string{
			"blockchain": "mochimo",
			"network":    "mainnet",
		},
	}

	reqJSON, _ := json.Marshal(reqBody)

	// Make request
	resp, err := http.Post(
		MESH_API_URL+"/network/status",
		"application/json",
		strings.NewReader(string(reqJSON)),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	// Parse response
	var status NetworkStatus
	err = json.NewDecoder(resp.Body).Decode(&status)
	if err != nil {
		return nil, err
	}

	return &status, nil
}

// CheckMempool checks if a transaction is in the mempool
func CheckMempool(txID string) (bool, error) {
	// Create request body
	reqBody := map[string]interface{}{
		"network_identifier": map[string]string{
			"blockchain": "mochimo",
			"network":    "mainnet",
		},
	}

	reqJSON, _ := json.Marshal(reqBody)

	// Make request
	resp, err := http.Post(
		MESH_API_URL+"/mempool",
		"application/json",
		strings.NewReader(string(reqJSON)),
	)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return false, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	// Parse response
	var mempoolResp MempoolResponse
	err = json.NewDecoder(resp.Body).Decode(&mempoolResp)
	if err != nil {
		return false, err
	}

	// Check if txID is in mempool
	for _, tx := range mempoolResp.TransactionIdentifiers {
		if tx.Hash == txID {
			return true, nil
		}
	}

	return false, nil
}

// SubmitTransaction submits a transaction to Mesh API
func SubmitTransaction(signedTx string) (string, error) {
	// Create request body
	reqBody := MeshAPISubmitRequest{
		NetworkIdentifier: struct {
			Blockchain string `json:"blockchain"`
			Network    string `json:"network"`
		}{
			Blockchain: "mochimo",
			Network:    "mainnet",
		},
		SignedTransaction: signedTx,
	}

	reqJSON, _ := json.Marshal(reqBody)

	// Make request
	resp, err := http.Post(
		MESH_API_URL+"/construction/submit",
		"application/json",
		strings.NewReader(string(reqJSON)),
	)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse response
	var submitResp MeshAPISubmitResponse
	err = json.NewDecoder(resp.Body).Decode(&submitResp)
	if err != nil {
		return "", err
	}

	return submitResp.TransactionIdentifier.Hash, nil
}

// VerifyTransactionInBlock checks if a transaction exists in a specific block
func VerifyTransactionInBlock(blockHash string, txID string) (bool, error) {
	// Normalize txID by removing 0x prefix if present for consistent comparison
	txID = strings.TrimPrefix(txID, "0x")

	// Create request body
	reqBody := map[string]interface{}{
		"network_identifier": map[string]string{
			"blockchain": "mochimo",
			"network":    "mainnet",
		},
		"block_identifier": map[string]interface{}{
			"hash": blockHash,
		},
	}

	reqJSON, _ := json.Marshal(reqBody)

	// Make request
	resp, err := http.Post(
		MESH_API_URL+"/block",
		"application/json",
		strings.NewReader(string(reqJSON)),
	)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	// Read response body for debugging
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	if resp.StatusCode != 200 {
		return false, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	// Parse response from saved body
	var blockResp BlockResponse
	err = json.Unmarshal(respBody, &blockResp)
	if err != nil {
		fmt.Printf("Error parsing block response: %v\n", err)
		return false, err
	}

	fmt.Printf("Searching for transaction %s in block with %d transactions\n",
		txID, len(blockResp.Block.Transactions))

	// Check if txID is in block transactions (with normalization)
	for _, tx := range blockResp.Block.Transactions {
		// Normalize comparison by removing 0x prefix if present
		txHashInBlock := strings.TrimPrefix(tx.TransactionIdentifier.Hash, "0x")
		fmt.Printf("Comparing tx: %s with expected: %s\n", txHashInBlock, txID)

		if txHashInBlock == txID {
			return true, nil
		}
	}

	// As a fallback, check directly in the JSON string for the transaction ID
	// This is in case our struct parsing is somehow missing the transaction
	if strings.Contains(string(respBody), txID) {
		fmt.Printf("Transaction %s found in block JSON but not detected by our parser!\n", txID)
		return true, nil
	}

	return false, nil
}

// DirectlyCheckTransaction checks if a transaction exists in the blockchain directly
func DirectlyCheckTransaction(txID string) (bool, error) {
	// Normalize txID by removing 0x prefix if present
	txID = strings.TrimPrefix(txID, "0x")

	// Create request body for block/transaction endpoint
	reqBody := map[string]interface{}{
		"network_identifier": map[string]string{
			"blockchain": "mochimo",
			"network":    "mainnet",
		},
		"transaction_identifier": map[string]interface{}{
			"hash": "0x" + txID,
		},
	}

	reqJSON, _ := json.Marshal(reqBody)

	// Make request to the /block/transaction endpoint
	resp, err := http.Post(
		MESH_API_URL+"/block/transaction",
		"application/json",
		strings.NewReader(string(reqJSON)),
	)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	// Check for 200 status - if we get it, the transaction exists
	if resp.StatusCode == 200 {
		fmt.Println("✅ Transaction found via direct check!")
		return true, nil
	}

	return false, nil
}

// VerifyCurrentIndex verifies the correct index for the wallet chain
func VerifyCurrentIndex(secretKey string, startIndex uint64) (uint64, []byte, uint64, error) {
	// Decode secret key
	secretBytes, err := hex.DecodeString(secretKey)
	if err != nil {
		return 0, nil, 0, err
	}

	// Create keychain
	var seed [32]byte
	copy(seed[:], secretBytes)
	keychain, err := wots.NewKeychain(seed)
	if err != nil {
		return 0, nil, 0, err
	}

	fmt.Printf("Starting wallet address search from index %d...\n", startIndex)

	// First try the requested start index
	keychain.Index = 0
	keypair := keychain.Next()

	// Properly extract the tag using go_mcminterface
	mcmAddr := mcm.WotsAddressFromBytes(keypair.PublicKey[:2144])
	tag := mcmAddr.GetAddress()

	// Resolve tag to check balance
	resolved_tag, amount, err := ResolveTag(tag)
	if err != nil {
		fmt.Printf("Using index %d with 0 nMCM (please refill this address: %s)\n", 0, hex.EncodeToString(tag))
		return 0, tag, 0, fmt.Errorf("Error: Wallet address at index %d has no balance", startIndex)
	}
	fmt.Println("Resolved tag:", resolved_tag)

	// tagged_address_hash is last 20 bytes of resolved_tag (40 bytes)
	resolved_tag_bytes, _ := hex.DecodeString(resolved_tag[2:])
	tagged_address_hash := resolved_tag_bytes[len(resolved_tag_bytes)-20:]

	// Check if startIndex gives the right tag
	keychain.Index = startIndex
	test_keypair := keychain.Next()

	// Properly extract the tag using go_mcminterface
	test_mcmAddr := mcm.WotsAddressFromBytes(test_keypair.PublicKey[:2144])
	test_add_hash := test_mcmAddr.GetAddress()

	if bytes.Equal(tagged_address_hash, test_add_hash) {
		fmt.Printf("Found correct wallet address at index %d\n", startIndex)
		return startIndex, tag, amount, nil
	}

	// If startIndex is wrong, search for the correct index
	for i := uint64(0); i < MAX_INDEX_SEARCH; i++ {
		keychain.Index = i
		test_keypair := keychain.Next()

		// Properly extract the tag using go_mcminterface
		test_mcmAddr := mcm.WotsAddressFromBytes(test_keypair.PublicKey[:2144])
		test_add_hash := test_mcmAddr.GetAddress()

		if bytes.Equal(tagged_address_hash, test_add_hash) {
			fmt.Printf("Found correct wallet address at index %d\n", i)
			return i, tag, amount, nil
		}
	}

	return 0, nil, 0, fmt.Errorf("Error: Wallet address not found in first %d indices", MAX_INDEX_SEARCH)
}

// Debug functions to help diagnose issues
func DumpTxnInfo(tx *mcm.TXENTRY) {
	fmt.Println("--- Transaction Debug Info ---")
	fmt.Printf("Send Total: %d\n", tx.GetSendTotal())
	fmt.Printf("Change Total: %d\n", tx.GetChangeTotal())
	fmt.Printf("Fee: %d\n", tx.GetFee())
	fmt.Printf("Destination Count: %d\n", tx.GetDestinationCount())
	fmt.Printf("Signature Scheme: %s\n", tx.GetSignatureScheme())
	fmt.Printf("Block To Live: %d\n", tx.GetBlockToLive())
	fmt.Println("---------------------------")
}

// Helper function to explicitly check current block before comparing
func IsBlockChanged(prevBlock uint64) (bool, uint64, string, error) {
	status, err := GetNetworkStatus()
	if err != nil {
		return false, prevBlock, "", err
	}

	currentBlock := status.CurrentBlockIdentifier.Index
	currentHash := status.CurrentBlockIdentifier.Hash

	if currentBlock > prevBlock {
		fmt.Printf("Block changed: %d -> %d (hash: %s)\n",
			prevBlock, currentBlock, currentHash)
		return true, currentBlock, currentHash, nil
	}

	return false, currentBlock, currentHash, nil
}

// AddrToBase58 converts a tag to base58 format with checksum
func AddrToBase58(tag []byte) string {
	if len(tag) != 20 {
		return "invalid-tag-length"
	}

	combined := make([]byte, 22)
	copy(combined, tag)

	// Calculate CRC using XMODEM
	table := crc16.MakeTable(crc16.CRC16_XMODEM)
	crc := crc16.Checksum(tag, table)

	// Append in little-endian
	combined[20] = byte(crc & 0xFF)
	combined[21] = byte((crc >> 8) & 0xFF)

	return base58.Encode(combined)
}

func main() {
	csvFile := flag.String("csv", "entries.csv", "CSV file with addresses and amounts")
	walletCacheFile := flag.String("wallet", "wallet-cache.json", "Wallet cache file")
	fee := flag.Uint64("fee", 500, "Transaction fee in nanoMCM")
	flag.Parse()

	// Read entries CSV
	entries, err := ReadEntriesCSV(*csvFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading entries: %v\n", err)
		os.Exit(1)
	}

	if len(entries) == 0 {
		fmt.Println("No valid entries found in CSV. Exiting.")
		os.Exit(0)
	}

	// Read/create wallet cache
	cache, err := ReadWalletCache(*walletCacheFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error with wallet cache: %v\n", err)
		os.Exit(1)
	}

	// Verify current index
	currentIndex, tag, balance, err := VerifyCurrentIndex(cache.SecretKey, cache.Index)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error verifying wallet index: %v\n", err)
		os.Exit(1)
	}

	// Update index in cache
	cache.Index = currentIndex + 1
	err = SaveWalletCache(*walletCacheFile, cache)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error saving wallet cache: %v\n", err)
		os.Exit(1)
	}

	// Check if wallet has sufficient balance
	totalToSend := uint64(0)
	for _, entry := range entries {
		totalToSend += entry.AmountToSend
	}

	// Add fee
	totalNeeded := totalToSend + *fee

	// Use the cached refill address
	if balance < totalNeeded {
		fmt.Fprintf(os.Stderr, "Error: Insufficient balance in wallet. Have %d nMCM, need %d nMCM\n",
			balance, totalNeeded)
		fmt.Fprintf(os.Stderr, "Please refill this address: %s\n", cache.RefillAddress)
		os.Exit(1)
	}

	fmt.Printf("Wallet balance: %d nMCM, sending total: %d nMCM (including %d nMCM fee)\n",
		balance, totalNeeded, *fee)
	fmt.Printf("Using wallet address: %s\n", cache.RefillAddress)

	// Create transaction using mcminterface - following tool-3 approach
	tx := mcm.NewTXENTRY()

	// Decode secret key
	secretBytes, _ := hex.DecodeString(cache.SecretKey)
	var privateKey [32]byte
	copy(privateKey[:], secretBytes)

	// Create keypairs for current and next indices
	keychain, _ := wots.NewKeychain(privateKey)
	keychain.Index = currentIndex
	fmt.Println("Using index", currentIndex)
	currentKeyPair := keychain.Next()
	nextKeyPair := keychain.Next()

	// Get proper public keys for source and change
	srcPubKey := currentKeyPair.PublicKey[:2144]
	chgPubKey := nextKeyPair.PublicKey[:2144]

	// Set source and change addresses exactly as in tool-3
	srcAddr := mcm.WotsAddressFromBytes(srcPubKey)
	srcAddr.SetTAG(tag)

	chgAddr := mcm.WotsAddressFromBytes(chgPubKey)
	chgAddr.SetTAG(tag)

	tx.SetSourceAddress(srcAddr)
	tx.SetChangeAddress(chgAddr)

	// Set amounts
	tx.SetSendTotal(totalToSend)
	tx.SetChangeTotal(balance - totalToSend - *fee)
	tx.SetFee(*fee)

	// Add destinations using exact same approach as tool-3, but now with memos
	for _, entry := range entries {
		dstHex := hex.EncodeToString(entry.AddressBin)
		dstEntry := mcm.NewDSTFromString(dstHex, entry.Memo, entry.AmountToSend)
		tx.AddDestination(dstEntry)
	}
	tx.SetDestinationCount(uint8(len(entries)))

	// Generate transaction hash
	var message [32]byte = tx.GetMessageToSign()

	// Sign transaction
	var signature [2144]byte = currentKeyPair.Sign(message)
	tx.SetWotsSignature(signature[:])

	// Set address components exactly as tool-3
	var addr_seed_default_tag [32]byte
	copy(addr_seed_default_tag[:], currentKeyPair.Components.AddrSeed[:20])
	copy(addr_seed_default_tag[20:], []byte{0x42, 0x00, 0x00, 0x00, 0x0e, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00})

	tx.SetWotsSigAddresses(addr_seed_default_tag[:])
	tx.SetWotsSigPubSeed(currentKeyPair.Components.PublicSeed)

	tx.SetSignatureScheme("wotsp")
	tx.SetBlockToLive(0)

	// Debug output before submitting
	DumpTxnInfo(&tx)

	// Submit transaction
	fmt.Println("Submitting transaction...")
	txID, err := SubmitTransaction(tx.String())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error submitting transaction: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Transaction submitted! TX ID: %s\n", txID)
	fmt.Println("Monitoring mempool for transaction...")

	// More reliable block change detection
	status, err := GetNetworkStatus()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting network status: %v\n", err)
		os.Exit(1)
	}

	currentBlock := status.CurrentBlockIdentifier.Index
	fmt.Printf("Current block: %d\n", currentBlock)

	// Monitor mempool and blocks
	inMempool := false
	txConfirmed := false
	startTime := time.Now()
	lastCheckedBlock := currentBlock

	for {
		// Check mempool
		found, err := CheckMempool(txID)
		if err != nil {
			fmt.Printf("Error checking mempool: %v\n", err)
		} else if found && !inMempool {
			inMempool = true
			fmt.Println("✅ Transaction found in mempool!")
		} else if !found && inMempool {
			fmt.Println("Transaction left mempool - checking blocks...")

			// Try direct check when transaction leaves mempool
			directCheck, _ := DirectlyCheckTransaction(txID)
			if directCheck {
				txConfirmed = true
				fmt.Println("✅ Transaction confirmed via direct check!")

				// Move CSV file to success folder
				successDir := "correctly-send"

				// Create directory if it doesn't exist
				if _, err := os.Stat(successDir); os.IsNotExist(err) {
					if err := os.Mkdir(successDir, 0755); err != nil {
						fmt.Printf("Warning: Failed to create directory %s: %v\n", successDir, err)
					}
				}

				// Get base filename without path
				baseFileName := *csvFile
				if lastSlash := strings.LastIndex(baseFileName, "/"); lastSlash != -1 {
					baseFileName = baseFileName[lastSlash+1:]
				}

				// Move file to success directory
				destFile := fmt.Sprintf("%s/%s", successDir, baseFileName)
				if err := os.Rename(*csvFile, destFile); err != nil {
					fmt.Printf("Warning: Failed to move CSV file to %s: %v\n", destFile, err)
				} else {
					fmt.Printf("CSV file moved to %s\n", destFile)
				}

				break
			}
		}

		// Check if block has changed - using our more reliable function
		blockChanged, newBlock, blockHash, err := IsBlockChanged(lastCheckedBlock)
		if err != nil {
			fmt.Printf("Error checking block status: %v\n", err)
		} else if blockChanged {
			lastCheckedBlock = newBlock
			fmt.Printf("Block changed to %d. Checking for transaction...\n", newBlock)

			// Try both verification methods
			verified, verifyErr := VerifyTransactionInBlock(blockHash, txID)
			if verifyErr != nil {
				fmt.Printf("Error verifying transaction in block: %v\n", verifyErr)

				// Try direct check as backup
				directCheck, _ := DirectlyCheckTransaction(txID)
				if directCheck {
					verified = true
				}
			}

			if verified {
				txConfirmed = true
				fmt.Println("✅ Transaction confirmed in block!")

				// Move the CSV file to correctly-send/ folder
				successDir := "correctly-send"

				// Create directory if it doesn't exist
				if _, err := os.Stat(successDir); os.IsNotExist(err) {
					if err := os.Mkdir(successDir, 0755); err != nil {
						fmt.Printf("Warning: Failed to create directory %s: %v\n", successDir, err)
					}
				}

				// Get base filename without path
				baseFileName := *csvFile
				if lastSlash := strings.LastIndex(baseFileName, "/"); lastSlash != -1 {
					baseFileName = baseFileName[lastSlash+1:]
				}

				// Move file to success directory
				destFile := fmt.Sprintf("%s/%s", successDir, baseFileName)
				if err := os.Rename(*csvFile, destFile); err != nil {
					fmt.Printf("Warning: Failed to move CSV file to %s: %v\n", destFile, err)
				} else {
					fmt.Printf("CSV file moved to %s\n", destFile)
				}

				break
			} else {
				fmt.Println("Transaction not found in block. Will continue monitoring...")
			}
		}

		// Check if we've been in the mempool for a long time
		if inMempool && time.Since(startTime) > 5*time.Minute {
			fmt.Println("Transaction has been in mempool for over 5 minutes.")
			fmt.Println("This may indicate issues with the transaction or network congestion.")
		}

		// Timeout after 10 minutes
		if time.Since(startTime) > 10*time.Minute {
			fmt.Println("⚠️ Monitoring timed out after 10 minutes.")
			if inMempool {
				fmt.Println("Transaction is still in the mempool. Check later for confirmation.")
			} else {
				fmt.Println("Transaction was not found in mempool or blocks. Please check manually.")
			}
			break
		}

		time.Sleep(CHECK_MEMPOOL_INTERVAL * time.Second)
	}

	if txConfirmed {
		fmt.Println("Transaction processing completed successfully!")
	} else {
		fmt.Println("Transaction processing completed but confirmation status is uncertain.")
	}
}
