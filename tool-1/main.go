package main

/*
 * MCM 2.X to MCM 3.0 Address Converter Tool
 *
 * This tool converts MCM 2.X WOTS addresses to MCM 3.0 format using the go_mcminterface library.
 *
 * Command line flags:
 * -wots string: WOTS address in hex format (4416 characters)
 *               Can be tagged or untagged MCM 2.X address
 *
 * Dependencies:
 * - github.com/NickP005/go_mcminterface: Provides MCM address conversion functionality
 *
 * Output:
 * - MCM 3.0 address in hex format, padded to 2x20 bytes
 *
 * Example usage:
 * ./tool-1 -wots <4416_char_hex_string>
 */

import (
	"flag"
	"fmt"
	"os"

	"github.com/btcsuite/btcutil/base58"
	"github.com/sigurn/crc16"

	"github.com/NickP005/go_mcminterface"
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
	wotsAddr := flag.String("wots", "", "WOTS address as hex string (4416 characters)")
	base58Flag := flag.Bool("base58", false, "Output address in base58 format")
	flag.Parse()

	if *wotsAddr == "" {
		fmt.Println("Error: WOTS address is required")
		flag.Usage()
		os.Exit(1)
	}

	if len(*wotsAddr) != 4416 {
		fmt.Printf("Error: WOTS address must be 4416 characters long, got %d\n", len(*wotsAddr))
		os.Exit(1)
	}

	// Remove the last 32 bytes
	*wotsAddr = (*wotsAddr)[:len(*wotsAddr)-64*2]

	mcmAddr := go_mcminterface.WotsAddressFromHex(*wotsAddr)
	addr := mcmAddr.GetAddress()

	if *base58Flag {
		base58Addr, err := AddrTagToBase58(addr)
		if err != nil {
			fmt.Printf("Error converting to base58: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(base58Addr)
	} else {
		fmt.Printf("%x\n", addr)
	}
}
