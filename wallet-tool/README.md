# Mochimo Wallet Tool

This tool allows you to send MCM to multiple destinations in a single transaction. It reads a list of addresses and amounts from a CSV file, validates them, and submits a transaction to the Mochimo network.

## Features

- Validates Mochimo addresses (base58 format with CRC16 checksum)
- Checks balances of addresses before sending
- Supports transaction memos for messages or references
- Manages wallet keys securely using WOTS+ signatures
- Automatically tracks the correct WOTS+ index in the wallet chain
- Monitors transaction status until confirmation
- Handles multiple recipients in a single transaction
- Supports multiple confirmation monitoring
- Can automatically retry broadcasts for failed transactions

## Requirements

- Go 1.16 or higher
- Internet connection to access Mochimo Mesh API
- Dependencies (automaticallly installed by Go):
  - github.com/btcsuite/btcutil/base58
  - github.com/NickP005/WOTS-Go
  - github.com/NickP005/go_mcminterface
  - github.com/sigurn/crc16

## Installation

### Prerequisites

- Go 1.18 or higher (recommended Go 1.20+)
- Git

### Building from Source

1. Clone the repository:
   ```
   git clone https://github.com/NickP005/Vindax-MCM-tools.git
   cd Vindax-MCM-tools
   ```

2. Install dependencies:
   ```
   go mod init github.com/NickP005/Vindax-MCM-tools
   go get github.com/btcsuite/btcutil/base58
   go get github.com/NickP005/WOTS-Go
   go get github.com/NickP005/go_mcminterface
   go get github.com/sigurn/crc16
   ```

3. Build the wallet tool:
   ```
   cd wallet-tool
   go build -o wallet-tool
   ```
   
   Note: Do not use the `-g` flag with `go build` as it's not supported.

### Command Line Flags

The wallet tool supports the following flags:

- `-wallet string`: Path to the wallet cache file (default "wallet-cache.json")
- `-csv string`: Path to the CSV file with addresses and amounts (default "entries.csv")
- `-fee uint`: Transaction fee in nanoMCM (default 500)
- `-api string`: Mesh API URL (default "http://35.208.202.76:8080")
- `-confirmations int`: Number of blocks to confirm transaction (default 1)
- `-keeptrying`: Keep trying to broadcast transaction if not confirmed
- `-timeout int`: Timeout in minutes for transaction monitoring (default 10)

## CSV Format

The CSV file should contain one line for each payment with:
- Mochimo address (base58 format)
- Amount in nMCM (integer)
- Optional memo/reference (in quotes)

Example:
```
5pj2oX9nJFFt3mdHa2wAN73p6QhAYr 1000000 "PAYMENT-2-JUNE"
```

Note: Fields are separated by spaces, memo is optional and must be in quotes if it contains spaces.

## Usage Examples

Send MCM to multiple recipients using a wallet cache and a CSV file:
```
./wallet-tool -wallet wallet-cache.json -csv entries.csv
```

Specify a custom transaction fee (in nanoMCM):
```
./wallet-tool -wallet wallet-cache.json -csv entries.csv -fee 1000
```

Wait for multiple confirmations:
```
./wallet-tool -wallet wallet-cache.json -csv entries.csv -confirmations 5
```

Automatically retry broadcasting if transaction is not confirmed:
```
./wallet-tool -wallet wallet-cache.json -csv entries.csv -keeptrying
```

Use a different Mochimo Mesh API endpoint:
```
./wallet-tool -wallet wallet-cache.json -csv entries.csv -api http://custom-api.example.com:8080
```

Increase monitoring timeout for large confirmations:
```
./wallet-tool -wallet wallet-cache.json -csv entries.csv -confirmations 10 -timeout 30
```

## Troubleshooting

If you see the error "flag provided but not defined", make sure you're only using the flags listed above.

When monitoring transactions that require multiple confirmations, the tool will adjust its timeout period accordingly, adding 2 minutes per confirmation beyond the first. You can override this with the `-timeout` flag.

If a transaction disappears from the blockchain (due to a chain reorganization) and you used the `-keeptrying` flag, the tool will automatically rebroadcast the transaction.
