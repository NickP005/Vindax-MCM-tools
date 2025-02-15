# Vindax tools for MCM3.0

## Tool 1
A command-line tool that takes in a MCM 2.X WOTS address as 4416 Hex characters, tagged or untagged, and converts it into a MCM 3.0 address. The output can be either in Hex format (padded to 2x20 bytes) or in Base58 format with checksum.

### Usage
```bash
# Build the tool
cd tool-1
go build

# Run the tool with hex output (default)
./tool-1 -wots <4416_character_hex_string>

# Run the tool with base58 output
./tool-1 -wots <4416_character_hex_string> -base58
```

The base58 output format includes a CRC16-XMODEM checksum and is useful for:
- Human-readable address format
- Error detection through checksum verification
- Shorter representation of addresses
- Compatibility with wallet displays and QR codes

Example outputs for the same address:
```
# Hex format
./tool-1 -wots <address>
> 9f810c2447a76e93b17ebff96c0b29952e4355f1

# Base58 format
./tool-1 -wots <address> -base58
> kHtV35ttVpyiH42FePCiHo2iFmcJS3
```

## Tool 2
A command-line tool that generates WOTS Keypairs and their corresponding MCM 3.0 address, output as a JSON object in the format:
```
{
  "accounts": [
    {
      "mcmAccountNumber": "0000000000000000000000000000000000000000", // 20 bytes, padded hex
      "wotsPublicKey": "0000... (2208 bytes of padded hex)", // 2208 bytes, padded hex
      "wotsSecretKey": "00... (32 bytes of padded hex)" // 32 bytes, padded hex
    },
    {
      "mcmAccountNumber": "0000000000000000000000000000000000000001", // 20 bytes, padded hex
      "wotsPublicKey": "0000... (2208 bytes of padded hex)", // 2208 bytes, padded hex
      "wotsSecretKey": "00... (32 bytes of padded hex)" // 32 bytes, padded hex
    },
    // ... more accounts
  ]
}
```

### Usage
```bash
# Build the tool
cd tool-2
go build

# Generate a single account
./tool-2

# Generate multiple accounts
./tool-2 -n 5  # Generates 5 accounts
```

## Tool 3
A command-line tool that creates and signs Mochimo transactions, outputting them in a format compatible with the MeshAPI /construction/submit endpoint. The tool handles all cryptographic operations locally and produces a JSON output ready for network submission.

### Usage
```bash
# Build the tool
cd tool-3
go build

# Run the tool with required parameters
./tool-3 \ 
  -src <20_bytes_hex>          # Source account address (TAG) \
  -source-pk <2208_bytes_hex>  # Source WOTS public key \
  -change-pk <2208_bytes_hex>  # Change WOTS public key \
  -balance <uint64>            # Source balance in nanoMCM \
  -dst <20_bytes_hex>          # Destination account address \
  -amount <int64>              # Amount to send in nanoMCM \
  -secret <32_bytes_hex>       # Secret key for signing \
  -memo "Optional memo"        # Optional transaction memo \
  -fee 500                     # Optional: Transaction fee in nanoMCM (default: 500)
```

### Example Output
The tool outputs a JSON object ready for submission to the MeshAPI. Here's a sample interaction:

```bash
$ ./tool-3 -src 81998859591cf1f35fc174a40e14c8138e2a5e03 \
          -source-pk <2208_bytes_public_key> \
          -change-pk <2208_bytes_public_key> \
          -balance 10000 \
          -dst f5fc0d11f423e7849bd908dc8bbcabf3002ac0aa \
          -amount 8999 \
          -secret <32_bytes_secret> \
          -memo "TEST"

Resolving TAG 81998859591cf1f35fc174a40e14c8138e2a5e03
Resolved TAG 81998859591cf1f35fc174a40e14c8138e2a5e03 to address 0x81998...652e1 with amount 8999
{
  "network_identifier": {
    "blockchain": "mochimo",
    "network": "mainnet"
  },
  "signed_transaction": "000000008199885959...000000"
}
```

The tool performs several validations:
- Verifies the source has sufficient balance for amount + fee
- Validates that the secret key matches the source public key
- Confirms all input parameters are of correct length
- Checks that addresses are properly formatted

Common errors you might encounter:
```bash
Error: Insufficient balance to send amount and fee
Error: Public key does not match source address
Error: Source account address is required
Failed to resolve TAG: TAG not found
```

# Support & Community

Join our communities for support and discussions:

<div align="center">

[![NickP005 Development Server](https://img.shields.io/discord/709417966881472572?color=7289da&label=NickP005%20Development%20Server&logo=discord&logoColor=white)](https://discord.gg/Q5jM8HJhNT)   
[![Mochimo Official](https://img.shields.io/discord/460867662977695765?color=7289da&label=Mochimo%20Official&logo=discord&logoColor=white)](https://discord.gg/SvdXdr2j3Y)

</div>

- **NickP005 Development Server**: Technical support and development discussions
- **Mochimo Official**: General Mochimo blockchain discussions and community