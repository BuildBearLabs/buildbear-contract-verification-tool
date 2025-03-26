# Test Data Directory

This directory contains sample test data for the BuildBear Contract Verification Tool. The data consists of the default foundry broadcast and out directories:

   ```
   test_data/
   ├── broadcast/
   │   └── DeployMarket.s.sol/
   │       └── 1/
   │           └── run-latest.json
   └── out/
       └── Contract.json
   ```

To run the tool with these test directories:
   ```bash
   # In the root directory of the project
   ./buildbear-verify -broadcast ./test_data/broadcast -out ./test_data/out
   ```

This will allow you to test the functionality without needing real contract deployment data.
