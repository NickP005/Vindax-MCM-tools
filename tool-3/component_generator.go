package main

import "crypto/sha256"

type Components struct {
	PrivateSeed []byte
	PublicSeed  []byte
	AddrSeed    []byte
}

/*
 * mochimoHash performs a SHA256 hash on input data and returns the result
 *
 * Parameters:
 * - data: byte array to be hashed
 *
 * Returns:
 * - []byte: 32-byte SHA256 hash of the input data
 */
func mochimoHash(data []byte) []byte {
	hash := sha256.Sum256(data)
	return hash[:]
}

/*
 * ComponentsGenerator derives three different seeds from an initial WOTS seed
 * used for transaction signing and key generation.
 *
 * Parameters:
 * - wotsSeed: byte array of 32 bytes used as the initial seed
 *
 * Returns:
 * - Components: struct containing:
 *   1. PrivateSeed: 32 bytes used for WOTS private key generation
 *   2. PublicSeed: 32 bytes used for WOTS chains randomization
 *   3. AddrSeed: 32 bytes used for WOTS address generation
 *
 * The function appends different suffixes to the seed to generate
 * cryptographically distinct components:
 * - "seed" suffix for PrivateSeed
 * - "publ" suffix for PublicSeed
 * - "addr" suffix for AddrSeed
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
