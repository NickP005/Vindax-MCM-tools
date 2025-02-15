# Guide: How to use the tools correctly

## Prerequisites

### Installing Go
1. Visit https://go.dev/doc/install
2. Download and install Go for your operating system
3. Verify installation by running: `go version`

### Building the Tools
All tools are located in their respective directories:
- Tool 1: `tool-1/`
- Tool 2: `tool-2/`
- Tool 3: `tool-3/`

To build each tool:
```bash
cd tool-1   # (or tool-2, tool-3)
go build
```

This will create an executable in each directory. The examples below assume you've:
1. Built all tools successfully
2. Are running commands from the directory containing the executables

## Address Generation Process

### 1. Generate a New Address
First, use tool-2 to generate a new WOTS keypair:

```bash
./tool-2 -n 1
```

This will output something like:
```json
{
  "accounts": [
    {
      "mcmAccountNumber": "00000000000000000000",
      "wotsPublicKey": "aa7c627d2b9f69..." /* 2208 bytes */,
      "wotsSecretKey": "b4d2e12c8a..." /* 32 bytes */
    }
  ]
}
```

⚠️ **IMPORTANT**: 
- The generation process is NOT deterministic
- Save both the public and secret keys securely
- The secret key will be needed for signing transactions
- You cannot recover the keys if lost

### 2. Convert to Base58 Format
Use tool-1 to convert the public key into a more user-friendly base58 format:

```bash
./tool-1 -wots <wotsPublicKey> -base58
```

Example output:
```
kHtV35ttVpyiH42FePCiHo2iFmcJS3
```

### 3. Usage
- The base58 address is what you should share with users for deposits
- This replaces the old TAG system used in MCM 2.x
- The address includes a checksum to prevent typing errors
- Users can send MCM directly to this address without any TAG resolution

### Security Notes
- Keep the secret key (`wotsSecretKey`) secure and private
- Back up both public and secret keys
- The public key (`wotsPublicKey`) is needed for sending transactions
- Never share your secret key with anyone

