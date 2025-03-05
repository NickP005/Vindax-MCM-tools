package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/btcsuite/btcutil/base58"
	"github.com/sigurn/crc16"
)

func AddrTagToBase58(tag []byte) (string, error) {
	if len(tag) != 20 {
		return "", fmt.Errorf("invalid address tag length")
	}

	combined := make([]byte, 22)
	copy(combined, tag)

	// Calculate CRC using XMODEM
	table := crc16.MakeTable(crc16.CRC16_XMODEM)
	crc := crc16.Checksum(tag, table)

	// Append in little-endian
	combined[20] = byte(crc & 0xFF)
	combined[21] = byte((crc >> 8) & 0xFF)

	return base58.Encode(combined), nil
}

func ValidateBase58Tag(addr string) bool {
	decoded := base58.Decode(addr)
	if len(decoded) != 22 {
		return false
	}

	// Get stored checksum (little-endian)
	storedCsum := uint16(decoded[21])<<8 | uint16(decoded[20])

	// Calculate CRC on tag portion using XMODEM
	table := crc16.MakeTable(crc16.CRC16_XMODEM)
	actualCrc := crc16.Checksum(decoded[:20], table)

	return storedCsum == actualCrc
}

func Base58ToAddrTag(addr string) ([]byte, error) {
	decoded := base58.Decode(addr)
	if len(decoded) != 22 {
		return nil, fmt.Errorf("invalid base58 tag length")
	}
	return decoded[:20], nil
}

func main() {
	base58Addr := flag.String("base58", "", "Base58 address to convert to hex")
	hexAddr := flag.String("hex", "", "Hex address (40 characters) to convert to base58")
	flag.Parse()

	// Check that exactly one option is provided
	if (*base58Addr == "" && *hexAddr == "") || (*base58Addr != "" && *hexAddr != "") {
		fmt.Println("Error: Provide either -base58 OR -hex, but not both or neither")
		flag.Usage()
		os.Exit(1)
	}

	// Convert base58 to hex
	if *base58Addr != "" {
		// Validate the base58 address
		if !ValidateBase58Tag(*base58Addr) {
			fmt.Println("Error: Invalid base58 address (wrong length or invalid checksum)")
			os.Exit(1)
		}

		// Convert to hex
		tag, err := Base58ToAddrTag(*base58Addr)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Println(hex.EncodeToString(tag))
	}

	// Convert hex to base58
	if *hexAddr != "" {
		// Remove 0x prefix if present
		*hexAddr = strings.TrimPrefix(*hexAddr, "0x")

		// Validate hex format
		if len(*hexAddr) != 40 {
			fmt.Printf("Error: Hex address must be 40 characters (20 bytes), got %d\n", len(*hexAddr))
			os.Exit(1)
		}

		// Decode hex
		tag, err := hex.DecodeString(*hexAddr)
		if err != nil {
			fmt.Printf("Error: Invalid hex format: %v\n", err)
			os.Exit(1)
		}

		// Convert to base58
		base58Addr, err := AddrTagToBase58(tag)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Println(base58Addr)
	}
}
