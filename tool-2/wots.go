package main

import (
	"crypto/sha256"
	"encoding/binary"
)

const (
	PARAMSN     = 32
	WOTSW       = 16
	WOTSLOGW    = 4
	WOTSLEN1    = 64
	WOTSLEN2    = 3
	WOTSLEN     = WOTSLEN1 + WOTSLEN2
	WOTSSIGSIZE = WOTSLEN * PARAMSN

	XMSS_HASH_PADDING_F   = 0
	XMSS_HASH_PADDING_PRF = 3
)

type WOTSAddress [8]uint32

func (addr *WOTSAddress) setKeyAndMask(keyAndMask uint32) {
	addr[7] = keyAndMask
}

func (addr *WOTSAddress) setChainAddr(chain uint32) {
	addr[5] = chain
}

func (addr *WOTSAddress) setHashAddr(hash uint32) {
	addr[6] = hash
}

func ullToBytes(outlen int, in uint64) []byte {
	out := make([]byte, outlen)
	binary.BigEndian.PutUint64(out[outlen-8:], in)
	return out
}

func addrToBytes(addr WOTSAddress) []byte {
	bytes := make([]byte, 32)
	for i := 0; i < 8; i++ {
		binary.BigEndian.PutUint32(bytes[i*4:], addr[i])
	}
	return bytes
}

func prf(in [32]byte, key []byte) []byte {
	buf := make([]byte, 2*PARAMSN+32)
	copy(buf[0:PARAMSN], ullToBytes(PARAMSN, XMSS_HASH_PADDING_PRF))
	copy(buf[PARAMSN:], key[:PARAMSN])
	copy(buf[2*PARAMSN:], in[:])

	hash := sha256.Sum256(buf)
	return hash[:]
}

func thashF(in []byte, pubSeed []byte, addr *WOTSAddress) []byte {
	buf := make([]byte, 3*PARAMSN)
	copy(buf[0:PARAMSN], ullToBytes(PARAMSN, XMSS_HASH_PADDING_F))

	addr.setKeyAndMask(0)
	addrBytes := addrToBytes(*addr)
	var inHash [32]byte
	copy(inHash[:], addrBytes)
	key := prf(inHash, pubSeed)
	copy(buf[PARAMSN:], key)

	addr.setKeyAndMask(1)
	addrBytes = addrToBytes(*addr)
	copy(inHash[:], addrBytes)
	bitmask := prf(inHash, pubSeed)

	for i := 0; i < PARAMSN; i++ {
		buf[2*PARAMSN+i] = in[i] ^ bitmask[i]
	}

	hash := sha256.Sum256(buf)
	return hash[:]
}

func expandSeed(seed []byte) [][]byte {
	outseeds := make([][]byte, WOTSLEN)
	var ctr [32]byte
	for i := 0; i < WOTSLEN; i++ {
		binary.BigEndian.PutUint64(ctr[24:], uint64(i))
		outseeds[i] = prf(ctr, seed)
	}
	return outseeds
}

func genChain(in []byte, start, steps uint32, pubSeed []byte, addr *WOTSAddress) []byte {
	out := make([]byte, PARAMSN)
	copy(out, in)

	for i := start; i < start+steps && i < WOTSW; i++ {
		addr.setHashAddr(i)
		out = thashF(out, pubSeed, addr)
	}
	return out
}

func baseW(msg []byte) []int {
	output := make([]int, WOTSLEN)
	var in, out, bits int
	var total byte

	for consumed := 0; consumed < WOTSLEN; consumed++ {
		if bits == 0 {
			total = msg[in]
			in++
			bits += 8
		}
		bits -= WOTSLOGW
		output[out] = int((total >> bits) & (WOTSW - 1))
		out++
	}
	return output
}

func wotsChecksum(msgBaseW []int) []int {
	csum := 0
	csumBytes := make([]byte, (WOTSLEN2*WOTSLOGW+7)/8)

	// Compute checksum
	for i := 0; i < WOTSLEN1; i++ {
		csum += WOTSW - 1 - msgBaseW[i]
	}

	// Convert checksum to base_w
	csum = csum << (8 - ((WOTSLEN2 * WOTSLOGW) % 8))
	binary.BigEndian.PutUint64(csumBytes, uint64(csum))

	csumBaseW := make([]int, WOTSLEN2)
	copy(csumBaseW, baseW(csumBytes)[:WOTSLEN2])

	return csumBaseW
}

func chainLengths(msg []byte) []int {
	lengths := make([]int, WOTSLEN)
	msgBaseW := baseW(msg[:WOTSLEN1])
	copy(lengths, msgBaseW)

	csumBaseW := wotsChecksum(msgBaseW)
	copy(lengths[WOTSLEN1:], csumBaseW)

	return lengths
}

func bytes32ToWOTSAddress(addr [32]byte) WOTSAddress {
	var result WOTSAddress
	for i := 0; i < 8; i++ {
		result[i] = binary.BigEndian.Uint32(addr[i*4 : (i+1)*4])
	}
	return result
}

func WOTSSign(msg, seed, pubSeed []byte, addr [32]byte) []byte {
	wotsAddr := bytes32ToWOTSAddress(addr)
	sig := make([]byte, WOTSSIGSIZE)
	lengths := chainLengths(msg)
	seeds := expandSeed(seed)

	for i := 0; i < WOTSLEN; i++ {
		wotsAddr.setChainAddr(uint32(i))
		chainResult := genChain(seeds[i], 0, uint32(lengths[i]), pubSeed, &wotsAddr)
		copy(sig[i*PARAMSN:], chainResult)
	}
	return sig
}

func WOTSPkFromSig(sig, msg, pubSeed []byte, addr [32]byte) []byte {
	wotsAddr := bytes32ToWOTSAddress(addr)
	pk := make([]byte, WOTSSIGSIZE)
	lengths := chainLengths(msg)

	for i := 0; i < WOTSLEN; i++ {
		wotsAddr.setChainAddr(uint32(i))
		chainResult := genChain(sig[i*PARAMSN:], uint32(lengths[i]),
			uint32(WOTSW-1-lengths[i]), pubSeed, &wotsAddr)
		copy(pk[i*PARAMSN:], chainResult)
	}
	return pk
}

func WOTSPkGen(seed, pubSeed []byte, addr [32]byte) []byte {
	wotsAddr := bytes32ToWOTSAddress(addr)
	pk := make([]byte, WOTSLEN*PARAMSN)
	seeds := expandSeed(seed)

	for i := 0; i < WOTSLEN; i++ {
		wotsAddr.setChainAddr(uint32(i))
		chainResult := genChain(seeds[i], 0, WOTSW-1, pubSeed, &wotsAddr)
		copy(pk[i*PARAMSN:], chainResult)
	}
	return pk
}
