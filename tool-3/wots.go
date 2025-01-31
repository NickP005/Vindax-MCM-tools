package main

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
)

const (
	paramSN            = 32
	wotsw              = 16
	wotsLogW           = 4
	wotsLen1           = (8 * paramSN) / wotsLogW // 64
	wotsLen2           = 3
	wotsLen            = wotsLen1 + wotsLen2 // 67
	wotsSigBytes       = wotsLen * paramSN   // 2144
	xmssHashPaddingPRF = 3
	xmssHashPaddingF   = 0
)

// WotsSign generates a WOTS+ signature for the given message using the seed and public seed.
func WotsSign(sig, msg, seed, pubSeed []byte, addr []uint32) error {
	if len(sig) < wotsSigBytes {
		return errors.New("signature buffer too small")
	}
	if len(msg) < paramSN {
		return errors.New("message too short")
	}
	if len(seed) != paramSN {
		return errors.New("seed must be 32 bytes")
	}
	if len(pubSeed) != paramSN {
		return errors.New("pubSeed must be 32 bytes")
	}
	if len(addr) != 8 {
		return errors.New("address must be 8 uint32s")
	}

	lengths := make([]int, wotsLen)
	chainLengths(lengths, msg)

	expandedSeed := make([]byte, wotsLen*paramSN)
	expandSeed(expandedSeed, seed)

	for i := 0; i < wotsLen; i++ {
		chainAddr := withChainAddr(addr, uint32(i))
		genChain(
			sig[i*paramSN:(i+1)*paramSN],
			expandedSeed[i*paramSN:(i+1)*paramSN],
			0,
			uint32(lengths[i]),
			pubSeed,
			chainAddr,
		)
	}

	return nil
}

// WotsPkGen generates a WOTS+ public key from the given seed and public seed.
func WotsPkGen(pk, seed, pubSeed []byte, addr []uint32) error {
	if len(pk) < wotsLen*paramSN {
		return errors.New("public key buffer too small")
	}
	if len(seed) != paramSN {
		return errors.New("seed must be 32 bytes")
	}
	if len(pubSeed) != paramSN {
		return errors.New("pubSeed must be 32 bytes")
	}
	if len(addr) != 8 {
		return errors.New("address must be 8 uint32s")
	}

	expandedSeed := make([]byte, wotsLen*paramSN)
	expandSeed(expandedSeed, seed)

	for i := 0; i < wotsLen; i++ {
		chainAddr := withChainAddr(addr, uint32(i))
		genChain(
			pk[i*paramSN:(i+1)*paramSN],
			expandedSeed[i*paramSN:(i+1)*paramSN],
			0,
			wotsw-1,
			pubSeed,
			chainAddr,
		)
	}

	return nil
}

// WotsPkFromSig computes the WOTS+ public key from a signature and message.
func WotsPkFromSig(pk, sig, msg, pubSeed []byte, addr []uint32) error {
	if len(pk) < wotsLen*paramSN {
		return errors.New("public key buffer too small")
	}
	if len(sig) < wotsSigBytes {
		return errors.New("signature too short")
	}
	if len(msg) < paramSN {
		return errors.New("message too short")
	}
	if len(pubSeed) != paramSN {
		return errors.New("pubSeed must be 32 bytes")
	}
	if len(addr) != 8 {
		return errors.New("address must be 8 uint32s")
	}

	lengths := make([]int, wotsLen)
	chainLengths(lengths, msg)

	for i := 0; i < wotsLen; i++ {
		chainAddr := withChainAddr(addr, uint32(i))
		steps := uint32(wotsw - 1 - lengths[i])
		genChain(
			pk[i*paramSN:(i+1)*paramSN],
			sig[i*paramSN:(i+1)*paramSN],
			uint32(lengths[i]),
			steps,
			pubSeed,
			chainAddr,
		)
	}

	return nil
}

func ullToBytes(out []byte, in uint64) {
	for i := len(out) - 1; i >= 0; i-- {
		out[i] = byte(in & 0xff)
		in >>= 8
	}
}

func addrToBytes(addr []uint32) [32]byte {
	var bytes [32]byte
	for i := 0; i < 8; i++ {
		binary.BigEndian.PutUint32(bytes[i*4:], addr[i])
	}
	return bytes
}

func prf(out []byte, in [32]byte, key []byte) {
	var padding [paramSN]byte
	ullToBytes(padding[:], xmssHashPaddingPRF)

	buf := make([]byte, 0, paramSN+paramSN+32)
	buf = append(buf, padding[:]...)
	buf = append(buf, key...)
	buf = append(buf, in[:]...)

	hash := sha256.Sum256(buf)
	copy(out, hash[:])
}

func thashF(out, in, pubSeed []byte, addr []uint32) {
	var buf [3 * paramSN]byte

	// Set padding
	ullToBytes(buf[:paramSN], xmssHashPaddingF)

	// Generate key
	keyAddr := withKeyAndMask(addr, 0)
	keyAddrBytes := addrToBytes(keyAddr)
	key := make([]byte, paramSN)
	prf(key, keyAddrBytes, pubSeed)
	copy(buf[paramSN:2*paramSN], key)

	// Generate mask
	maskAddr := withKeyAndMask(addr, 1)
	maskAddrBytes := addrToBytes(maskAddr)
	mask := make([]byte, paramSN)
	prf(mask, maskAddrBytes, pubSeed)

	// XOR input with mask
	for i := 0; i < paramSN; i++ {
		buf[2*paramSN+i] = in[i] ^ mask[i]
	}

	hash := sha256.Sum256(buf[:])
	copy(out, hash[:])
}

func genChain(out, in []byte, start, steps uint32, pubSeed []byte, addr []uint32) {
	copy(out, in)
	current := make([]byte, paramSN)
	copy(current, in)

	for i := start; i < start+steps && i < wotsw; i++ {
		hashAddr := withHashAddr(addr, i)
		thashF(current, current, pubSeed, hashAddr)
	}

	copy(out, current)
}

func expandSeed(out, seed []byte) {
	var ctr [32]byte
	for i := 0; i < wotsLen; i++ {
		ullToBytes(ctr[:], uint64(i))
		prf(out[i*paramSN:(i+1)*paramSN], ctr, seed)
	}
}

func baseW(output []int, input []byte) {
	in := 0
	out := 0
	var total byte
	bits := 0

	for out < len(output) {
		if bits == 0 {
			total = input[in]
			in++
			bits = 8
		}
		bits -= wotsLogW
		output[out] = int((total >> bits) & (wotsw - 1))
		out++
	}
}

func wotsChecksum(csumBaseW []int, msgBaseW []int) {
	csum := 0
	for i := 0; i < wotsLen1; i++ {
		csum += wotsw - 1 - msgBaseW[i]
	}

	bitsNeeded := wotsLen2 * wotsLogW
	bytesNeeded := (bitsNeeded + 7) / 8
	shift := uint(8-(bitsNeeded%8)) % 8

	csumBytes := make([]byte, bytesNeeded)
	ullToBytes(csumBytes, uint64(csum)<<shift)
	baseW(csumBaseW, csumBytes)
}

func chainLengths(lengths []int, msg []byte) {
	msgBaseW := make([]int, wotsLen1)
	baseW(msgBaseW, msg[:paramSN])

	csumBaseW := make([]int, wotsLen2)
	wotsChecksum(csumBaseW, msgBaseW)

	copy(lengths[:wotsLen1], msgBaseW)
	copy(lengths[wotsLen1:], csumBaseW)
}

func withChainAddr(addr []uint32, chain uint32) []uint32 {
	newAddr := make([]uint32, 8)
	copy(newAddr, addr)
	newAddr[5] = chain
	return newAddr
}

func withHashAddr(addr []uint32, hash uint32) []uint32 {
	newAddr := make([]uint32, 8)
	copy(newAddr, addr)
	newAddr[6] = hash
	return newAddr
}

func withKeyAndMask(addr []uint32, keyAndMask uint32) []uint32 {
	newAddr := make([]uint32, 8)
	copy(newAddr, addr)
	newAddr[7] = keyAndMask
	return newAddr
}
