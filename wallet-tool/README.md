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

## Requirements

- Go 1.16 or higher
- Internet connection to access Mochimo Mesh API
- Dependencies:
  - github.com/btcsuite/btcutil/base58
  - github.com/NickP005/WOTS-Go
  - github.com/NickP005/go_mcminterface
  - github.com/sigurn/crc16

## Installation

