# Setting Up SAGE Contracts as Git Submodule

This guide explains how to separate the SAGE smart contracts into their own repository and link them via git submodule.

## Step 1: Create Contracts Repository

First, create a new repository for the contracts:

```bash
# Create new directory for contracts repo
mkdir sage-contracts
cd sage-contracts
git init

# Copy contracts from main repo
cp -r /path/to/sage/contracts/* .

# Initial commit
git add .
git commit -m "Initial commit: SAGE smart contracts"

# Add remote origin
git remote add origin https://github.com/your-org/sage-contracts.git
git push -u origin main
```

## Step 2: Remove Contracts from Main Repository

In the main SAGE repository:

```bash
# Remove contracts directory
git rm -r contracts
git commit -m "Remove contracts (will be added as submodule)"
```

## Step 3: Add Contracts as Submodule

```bash
# Add submodule
git submodule add https://github.com/your-org/sage-contracts.git contracts
git submodule update --init --recursive

# Commit the submodule addition
git add .gitmodules contracts
git commit -m "Add contracts as git submodule"
```

## Step 4: Working with Submodules

### Cloning Repository with Submodules

When someone clones the main repository:

```bash
# Clone with submodules
git clone --recurse-submodules https://github.com/your-org/sage.git

# Or if already cloned
git submodule update --init --recursive
```

### Updating Submodule

To get the latest contracts:

```bash
cd contracts
git fetch
git merge origin/main
cd ..
git add contracts
git commit -m "Update contracts submodule"
```

### Making Changes to Contracts

```bash
cd contracts
# Make your changes
git add .
git commit -m "Update contract feature"
git push origin main

# Update reference in main repo
cd ..
git add contracts
git commit -m "Update contracts submodule reference"
git push
```

## Step 5: CI/CD Configuration

Update your CI/CD to handle submodules:

### GitHub Actions Example

```yaml
name: Build and Test
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
        with:
          submodules: recursive
      
      - name: Setup Node.js
        uses: actions/setup-node@v3
        with:
          node-version: '18'
      
      - name: Install dependencies
        run: |
          npm install
          cd contracts/ethereum && npm install
          cd ../solana && npm install
      
      - name: Run tests
        run: |
          npm test
          cd contracts/ethereum && npx hardhat test
          cd ../solana && anchor test
```

## Step 6: Development Workflow

### For Contract Developers

1. Clone the contracts repository directly
2. Make changes and test
3. Push to contracts repository
4. Create PR for review

### For SAGE Developers

1. Work in main repository
2. If contract changes needed:
   - Make changes in `contracts/` directory
   - Commit and push to contracts repo
   - Update submodule reference in main repo

### Keeping Submodules in Sync

Add to your `.git/config`:

```ini
[submodule "contracts"]
    path = contracts
    url = https://github.com/your-org/sage-contracts.git
    branch = main
    update = rebase
```

Then use:
```bash
git submodule update --remote --rebase
```

## Benefits of This Approach

1. **Separation of Concerns**: Contracts have their own versioning
2. **Independent Development**: Contract and main app teams can work independently
3. **Reusability**: Contracts can be used by other projects
4. **Security Audits**: Easier to audit just the contracts
5. **Gas Optimization**: Contract-specific CI/CD and testing

## Common Issues and Solutions

### Submodule Not Initialized
```bash
git submodule update --init --recursive
```

### Detached HEAD in Submodule
```bash
cd contracts
git checkout main
git pull origin main
```

### Merge Conflicts in Submodule
```bash
cd contracts
git fetch
git checkout main
git merge origin/main
# Resolve conflicts
git add .
git commit
cd ..
git add contracts
git commit -m "Resolve submodule conflicts"
```

## Alternative: NPM Package

Instead of git submodule, you could publish contracts as an NPM package:

```json
{
  "name": "@sage/contracts",
  "version": "1.0.0",
  "main": "index.js",
  "files": [
    "ethereum/artifacts/**/*.json",
    "ethereum/contracts/**/*.sol",
    "solana/target/idl/*.json",
    "solana/target/types/*.ts"
  ]
}
```

Then in main project:
```bash
npm install @sage/contracts
```