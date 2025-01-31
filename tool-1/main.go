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

	"github.com/NickP005/go_mcminterface"
)

func main() {
	wotsAddr := flag.String("wots", "", "WOTS address as hex string (4416 characters)")
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

	mcmAddr := go_mcminterface.WotsAddressFromHex(*wotsAddr)

	fmt.Printf("%x\n", mcmAddr.Address)
}
