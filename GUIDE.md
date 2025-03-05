# Guide: How to use the tools correctly

## Quick Links
- [Prerequisites](#prerequisites)
  - [Installing Go](#installing-go)
  - [Building the Tools](#building-the-tools)
- [Address Generation](#address-generation-process)
  - [Generate New Address](#1-generate-a-new-address)
  - [Convert to Base58](#2-convert-to-base58-format)
  - [Usage](#3-usage)
  - [Understanding Addresses](#understanding-address-behavior)
- [Checking Balances](#checking-balances-and-setting-up-endpoints)
  - [Setting Up Endpoint](#setting-up-your-own-endpoint)
  - [Checking Balance](#checking-account-balance)
- [Creating Transactions](#creating-transactions)
  - [Key Components](#understanding-key-components)
  - [Transaction Creation](#creating-a-transaction)
  - [Important Notes](#important-notes)
  - [Security Best Practices](#security-best-practices)
- [Address Format Conversion](#address-format-conversion)
  - [Base58 to Hex](#1-base58-to-hex-conversion)
  - [Hex to Base58](#2-hex-to-base58-conversion)
  - [When to Use Each Format](#when-to-use-each-format)
- [Transaction Queries](#transaction-queries)
  - [Get Transaction Details](#1-get-transaction-details)
  - [Get Address Transactions](#2-get-address-transactions)
  - [Important Notes for Exchange Integration](#important-notes-for-exchange-integration)

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

### Understanding Address Behavior
The MCM 3.0 address system works similarly to the old TAG system, but with one crucial difference:
- Addresses behave like TAGs did in MCM 2.x
- **IMPLICITLY**: When sending to a never-used address, the blockchain automatically associates the WOTS+ public key with that address
- This means the first transaction to an address sets its WOTS+ public key
- Then at every transaction we change the public key part of an address, but the address remains the same

This implicit behavior eliminates the need for explicit TAG resolution while maintaining the same security model.

### Security Notes
- Keep the secret key (`wotsSecretKey`) secure and private
- Back up both public and secret keys
- The public key (`wotsPublicKey`) is needed for sending transactions
- Never share your secret key with anyone

## Checking Balances and Setting Up Endpoints

### Setting Up Your Own Endpoint
1. For production use, it's recommended to run your own MeshAPI endpoint:
   ```bash
   # Clone the repository
   git clone https://github.com/NickP005/mochimo-mesh
   cd mochimo-mesh
   
   # Follow installation instructions in the repository
   ```

2. While testing, you can use the public endpoint at `35.208.202.76:8080`

### Checking Account Balance
You can check an account's balance using the `/account/balance` endpoint:

```bash
curl -X POST http://35.208.202.76:8080/account/balance \
  -H "Content-Type: application/json" \
  -d '{
    "network_identifier": {
      "blockchain": "mochimo",
      "network": "mainnet"
    },
    "account_identifier": { 
      "address": "YOUR_ADDRESS_HERE"  
    }
  }'
```

For example, using address `0x9f810c2447a76e93b17ebff96c0b29952e4355f1`


Example response:
```json
{
  "block_identifier": {
    "index": 660001,
    "hash": "0x33632bf365999af93b8eb5bf4b4c33905b3e202d275a129d9771366a326b5527"
  },
  "balances": [
    {
      "value": "799998501",
      "currency": { "symbol": "MCM", "decimals": 9 }
    }
  ]
}
```

Note:
- The balance is shown in nanoMCM (1 MCM = 1,000,000,000 nanoMCM)
- `decimals: 9` indicates this conversion factor
- Always run your own endpoint for production use
- Public endpoints should only be used for testing

### Security Considerations
- Don't rely on public endpoints for critical operations
- Running your own endpoint ensures data accuracy
- Keep your endpoint secure and properly configured
- Consider using SSL/TLS for endpoint connections

## Creating Transactions

### Understanding Key Components
Before creating a transaction, it's important to understand the difference between:

1. **Source Address** (20 bytes):
   - This is the account identifier/TAG
   - Used to locate the account on the blockchain
   - Example: `81998859591cf1f35fc174a40e14c8138e2a5e03`

2. **Source Public Key** (2208 bytes):
   - The full WOTS public key
   - Required for transaction validation
   - Much longer than the address as it contains the complete cryptographic material

3. **Change Public Key** (2208 bytes):
   - ⚠️ IMPORTANT: Must be different from the source public key
   - Used for receiving change from the transaction
   - Should be a fresh, unused WOTS public key
   - Once used, a WOTS key should never be reused

### Creating a Transaction
Use tool-3 with the following parameters:

```bash
./tool-3 \
  -src <20_bytes_source_address> \
  -source-pk <2208_bytes_source_pubkey> \
  -change-pk <2208_bytes_change_pubkey> \
  -balance <current_balance_in_nanomcm> \
  -dst <20_bytes_destination_address> \
  -amount <amount_in_nanomcm> \
  -secret <32_bytes_secret_key> \
  -memo "Optional memo" \
  -fee 500
```

Example output:
```json
{
  "network_identifier": {
    "blockchain": "mochimo",
    "network": "mainnet"
  },
  "signed_transaction": "000000008199885959..." // Long hex string
}
```

### Important Notes
1. **Key Management**:
   - The source public key can only be used ONCE for sending
   - Always use a fresh change public key for receiving change
   - Never reuse any WOTS key after signing

2. **Balance and Fees**:
   - Ensure source address has sufficient balance for amount + fee
   - Default fee is 500 nanoMCM
   - The change amount is automatically calculated as: balance - amount - fee

3. **Submitting the Transaction**:
   The output JSON can be submitted to the MeshAPI endpoint:
   ```bash
   curl -X POST http://your-meshapi:8080/construction/submit \
     -H "Content-Type: application/json" \
     -d '<output_from_tool_3>'
   ```

### Security Best Practices
- Generate a new change address for each transaction
- Never share or reuse the secret key
- Keep track of which public keys have been used
- Verify all addresses and amounts before submitting

## Address Format Conversion

MCM 3.0 addresses can be represented in two formats:
1. **Hex format**: 40-character string representing 20 bytes (e.g., `9f810c2447a76e93b17ebff96c0b29952e4355f1`)
2. **Base58 format**: Shorter string with checksum for error detection (e.g., `kHtV35ttVpyiH42FePCiHo2iFmcJS3`)

### Converting Between Formats
Use tool-4 to easily convert between these formats:

#### 1. Base58 to Hex Conversion
When you need the raw hex address (e.g., for API calls):
```bash
./tool-4 -base58 <base58Address> -hex
```

Example output:
```
9f810c2447a76e93b17ebff96c0b29952e4355f1
```

#### 2. Hex to Base58 Conversion
When you need a user-friendly address (e.g., for sharing with users):
```bash
./tool-4 -hex <hexAddress> -base58
```

Example output:
```
kHtV35ttVpyiH42FePCiHo2iFmcJS3
```

### When to Use Each Format
- **Hex format**: Use for internal operations, API calls, and when precise control over the address is needed.
- **Base58 format**: Use for user-facing operations, sharing addresses, and when error detection is important.

## Transaction Queries
Transaction queries are essential for tracking deposits and managing exchange operations. The MeshAPI provides several endpoints for retrieving transaction information.

### 1. Get Transaction Details
To retrieve details of a specific transaction:

```bash
curl -X POST http://35.208.202.76:8080/search/transactions \
  -H "Content-Type: application/json" \
  -d '{
    "network_identifier": {
      "blockchain": "mochimo",
      "network": "mainnet"
    },
    "transaction_identifier": {
      "hash": "<TRANSACTION_HASH>"
    }
  }'
```

Example Response:
```json
{
  "transactions": [{
    "block_identifier": {
      "index": 671568,
      "hash": "0x6589babda9737f4ccdd2ed852937f4142aff0c3c810e6f97ae522e8278a97c41"
    },
    "transaction_identifier": {
      "hash": "0x1a7b2b7f7214d385399fa29580eae1d5709cf0955d03339631cbe99ec12c6286"
    },
    "operations": [
      {
        "operation_identifier": {"index": 0},
        "type": "SOURCE_TRANSFER",
        "status": "PENDING",
        "account": {"address": "0x11c4fe438399b40457fc75fb08732459ef44b459"},
        "amount": {
          "value": "-10000",
          "currency": {"symbol": "MCM", "decimals": 9}
        }
      },
      {
        "operation_identifier": {"index": 1},
        "type": "DESTINATION_TRANSFER",
        "status": "PENDING",
        "account": {"address": "0x7ef084f718460de2bda22240c4dca2aef6bacdee"},
        "amount": {
          "value": "9500",
          "currency": {"symbol": "MCM", "decimals": 9}
        },
        "metadata": {"memo": "TEST-4"}
      },
      {
        "operation_identifier": {"index": 2},
        "type": "FEE",
        "status": "PENDING",
        "account": {"address": "0xde6c06720a95122fa23a9d1af64f91bcf3c27b38"},
        "amount": {
          "value": "500",
          "currency": {"symbol": "MCM", "decimals": 9}
        }
      }
    ],
    "metadata": {
      "block_to_live": 0,
      "change_total": 0,
      "fee_total": 500,
      "send_total": 9500
    },
    "timestamp": 1740249206000
  }],
  "total_count": 1
}
```

### 2. Get Address Transactions
To retrieve all transactions for a specific address (useful for scanning deposits):

```bash
curl -X POST http://35.208.202.76:8080/search/transactions \
  -H "Content-Type: application/json" \
  -d '{
    "network_identifier": {
      "blockchain": "mochimo",
      "network": "mainnet"
    },
    "account_identifier": {
      "address": "<ADDRESS>"
    }
  }'
```

The response will include all transactions where the address appears as either source or destination.

### Important Notes for Exchange Integration:
1. **Transaction Monitoring**:
   - Poll `/search/transactions` with your deposit address to monitor incoming transactions
   - Verify transaction status through `status` field in operations
   - Check `amount` values for deposit amounts
   - Verify `block_identifier` to ensure transaction finality

2. **Processing Deposits**:
   - Look for operations of type "DESTINATION_TRANSFER"
   - Verify the destination address matches your deposit address
   - Use the `amount` field to credit user accounts
   - Check memo field for user-specific data if implemented

3. **Best Practices**:
   - Cache transaction results to avoid redundant queries
   - Implement exponential backoff for API requests
   - Consider running your own MeshAPI instance for production use
   - Monitor the `timestamp` field to track transaction age

The MeshAPI provides equivalent functionality to MCM v2's API, making it suitable for exchange integration with improved features and reliability.

