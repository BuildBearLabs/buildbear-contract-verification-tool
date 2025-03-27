# BuildBear Contract Verification Tool

A Go application for processing and verifying smart contracts deployed with BuildBear.

## Overview

This tool processes Foundry broadcast artifacts to extract contract information and can send this data to a verification API. It handles:

- Reading contract deployment information from Foundry broadcast files
- Extracting contract artifacts and metadata
- Processing source code and compiler settings
- Grouping contracts by name
- Sending data to a verification API endpoint

## Installation

```bash
# Clone the repository
git clone https://github.com/buildbear/buildbear-contract-verification-tool.git
cd buildbear-contract-verification-tool

# Build the application
go build -o buildbear-verify
```

## Usage

1. Copy the above binary to the root directory of your dAap project.

2. Run the tool:

```bash
# For default paths
./buildbear-verify

# Specify custom paths
./buildbear-verify -broadcast /path/to/broadcast -out /path/to/out

# Send data to verification API
./buildbear-verify -api [base_url]/node/verify/cli/:id

# Example:
./buildbear-verify -broadcast ./broadcast -out ./out -api https://api.buildbear.io/node/verify/cli/:id
```

### Command Line Flags

- `-broadcast`: Path to the broadcast directory (default: "./broadcast")
- `-out`: Path to the output directory (default: "./out")
- `-api`: URL of the verification API (optional)

## Input Structure

The tool expects a Foundry project structure with:

- `broadcast/[chain id]/[script name]` containing deployment information
- `out/` containing contract artifacts

## Output

The tool generates:
- `processed-contracts.json`: A JSON file containing all processed contract information
- Console output showing the processing steps
- If an API URL is provided, it sends the data to the specified endpoint
