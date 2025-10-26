#!/usr/bin/env node

/**
 * Generate Java bindings from contract ABIs
 * Requires: web3j CLI tool
 * Install: curl -L https://get.web3j.io | sh
 */

const fs = require('fs');
const path = require('path');
const { execSync, execFileSync } = require('child_process');

// Color codes for console output
const colors = {
  reset: "\x1b[0m",
  bright: "\x1b[1m",
  green: "\x1b[32m",
  yellow: "\x1b[33m",
  blue: "\x1b[34m",
  red: "\x1b[31m",
  cyan: "\x1b[36m"
};

function log(message, color = "reset") {
  console.log(colors[color] + message + colors.reset);
}

async function generateJavaBindings() {
  try {
    log("\n☕ Generating Java Bindings for Smart Contracts", "cyan");
    log("=" .repeat(50), "bright");

    // Check if web3j is installed
    try {
      execSync('which web3j', { stdio: 'ignore' });
    } catch (error) {
      log("\n web3j not found!", "red");
      log("\nPlease install web3j first:", "yellow");
      log("  macOS: brew tap web3j/web3j && brew install web3j", "cyan");
      log("  Linux/Windows: curl -L https://get.web3j.io | sh", "cyan");
      log("\nOr download from: https://github.com/web3j/web3j-cli/releases", "yellow");
      process.exit(1);
    }

    // Create Java bindings directory
    const javaDir = path.join(__dirname, '..', 'bindings', 'java');
    const srcDir = path.join(javaDir, 'src', 'main', 'java', 'io', 'sage', 'contracts');
    
    if (!fs.existsSync(srcDir)) {
      fs.mkdirSync(srcDir, { recursive: true });
      log(" Created bindings/java directory structure", "green");
    }

    // Contracts to generate bindings for
    const contracts = [
      {
        name: 'SageRegistryV2',
        abi: 'abi/SageRegistryV2.abi.json',
        bin: 'artifacts/contracts/SageRegistryV2.sol/SageRegistryV2.json'
      },
      {
        name: 'SageVerificationHook',
        abi: 'abi/SageVerificationHook.abi.json',
        bin: 'artifacts/contracts/SageVerificationHook.sol/SageVerificationHook.json'
      }
    ];

    // Generate Java bindings
    for (const contract of contracts) {
      log(`\n Processing ${contract.name}...`, "blue");

      const abiPath = path.join(__dirname, '..', contract.abi);
      const binPath = path.join(__dirname, '..', contract.bin);

      // Check if files exist
      if (!fs.existsSync(abiPath)) {
        log(`    ABI not found: ${contract.abi}`, "yellow");
        log(`  Run 'npm run extract-abi' first`, "yellow");
        continue;
      }

      // Get bytecode from artifact
      let bytecode = '';
      if (fs.existsSync(binPath)) {
        const artifact = JSON.parse(fs.readFileSync(binPath, 'utf8'));
        bytecode = artifact.bytecode;
      }

      // Create temporary bin file
      const tempBinPath = path.join(__dirname, '..', 'temp.bin');
      fs.writeFileSync(tempBinPath, bytecode.replace('0x', ''));

      // Generate Java binding using web3j
      // Use execFileSync to prevent shell injection
      try {
        execFileSync('web3j', [
          'generate',
          'solidity',
          `-a=${abiPath}`,
          `-b=${tempBinPath}`,
          `-o=${javaDir}/src/main/java`,
          `-p=io.sage.contracts`
        ], { stdio: 'pipe' });
        log(`   Generated ${contract.name}.java`, "green");
        
        // Clean up temp file
        fs.unlinkSync(tempBinPath);
        
        // Show file info
        const javaFile = path.join(srcDir, `${contract.name}.java`);
        if (fs.existsSync(javaFile)) {
          const stats = fs.statSync(javaFile);
          log(`  📄 ${javaFile}`, "cyan");
          log(`   Size: ${(stats.size / 1024).toFixed(2)} KB`, "cyan");
        }
      } catch (error) {
        log(`   Failed to generate binding for ${contract.name}`, "red");
        console.error(error.message);
        // Clean up temp file if exists
        if (fs.existsSync(tempBinPath)) {
          fs.unlinkSync(tempBinPath);
        }
      }
    }

    // Create pom.xml for Maven
    const pomContent = `<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0"
         xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
         xsi:schemaLocation="http://maven.apache.org/POM/4.0.0 
         http://maven.apache.org/xsd/maven-4.0.0.xsd">
    
    <modelVersion>4.0.0</modelVersion>
    <groupId>io.sage</groupId>
    <artifactId>sage-contracts</artifactId>
    <version>1.0.0</version>
    <packaging>jar</packaging>
    
    <properties>
        <maven.compiler.source>11</maven.compiler.source>
        <maven.compiler.target>11</maven.compiler.target>
        <project.build.sourceEncoding>UTF-8</project.build.sourceEncoding>
        <web3j.version>4.10.3</web3j.version>
    </properties>
    
    <dependencies>
        <dependency>
            <groupId>org.web3j</groupId>
            <artifactId>core</artifactId>
            <version>\${web3j.version}</version>
        </dependency>
        <dependency>
            <groupId>org.web3j</groupId>
            <artifactId>contracts</artifactId>
            <version>\${web3j.version}</version>
        </dependency>
    </dependencies>
</project>`;

    fs.writeFileSync(path.join(javaDir, 'pom.xml'), pomContent);
    log("\n Created pom.xml", "green");

    // Create build.gradle for Gradle
    const gradleContent = `plugins {
    id 'java'
}

group = 'io.sage'
version = '1.0.0'
sourceCompatibility = '11'

repositories {
    mavenCentral()
}

dependencies {
    implementation 'org.web3j:core:4.10.3'
    implementation 'org.web3j:contracts:4.10.3'
}`;

    fs.writeFileSync(path.join(javaDir, 'build.gradle'), gradleContent);
    log(" Created build.gradle", "green");

    // Create example usage file
    const exampleContent = `package io.sage.example;

import io.sage.contracts.SageRegistryV2;
import org.web3j.crypto.Credentials;
import org.web3j.protocol.Web3j;
import org.web3j.protocol.http.HttpService;
import org.web3j.tx.gas.DefaultGasProvider;

public class Example {
    public static void main(String[] args) throws Exception {
        // Connect to Kaia network
        Web3j web3j = Web3j.build(new HttpService("https://public-en-kairos.node.kaia.io"));
        
        // Load credentials (for write operations)
        // Credentials credentials = Credentials.create("PRIVATE_KEY");
        
        // Contract address (replace with actual deployed address)
        String contractAddress = "0x...";
        
        // Load contract (read-only)
        SageRegistryV2 contract = SageRegistryV2.load(
            contractAddress,
            web3j,
            Credentials.create("0x0"), // dummy credentials for read-only
            new DefaultGasProvider()
        );
        
        // Get owner
        String owner = contract.owner().send();
        System.out.println("Contract owner: " + owner);
        
        // Get agent by DID
        String did = "did:sage:example";
        var agent = contract.getAgentByDID(did).send();
        System.out.println("Agent name: " + agent.component2()); // name
        System.out.println("Agent active: " + agent.component8()); // active
    }
}`;

    const exampleDir = path.join(javaDir, 'src', 'main', 'java', 'io', 'sage', 'example');
    if (!fs.existsSync(exampleDir)) {
      fs.mkdirSync(exampleDir, { recursive: true });
    }
    fs.writeFileSync(path.join(exampleDir, 'Example.java'), exampleContent);
    log(" Created Example.java", "green");

    // Create README
    const readmeContent = `# Java Bindings for SAGE Contracts

## Setup

### Maven
\`\`\`xml
<dependency>
    <groupId>org.web3j</groupId>
    <artifactId>core</artifactId>
    <version>4.10.3</version>
</dependency>
\`\`\`

### Gradle
\`\`\`gradle
implementation 'org.web3j:core:4.10.3'
\`\`\`

## Usage

\`\`\`java
import io.sage.contracts.SageRegistryV2;
import org.web3j.protocol.Web3j;
import org.web3j.protocol.http.HttpService;

// Connect to network
Web3j web3j = Web3j.build(new HttpService("https://public-en-kairos.node.kaia.io"));

// Load contract
SageRegistryV2 contract = SageRegistryV2.load(address, web3j, credentials, gasProvider);

// Read data
var agent = contract.getAgentByDID("did:sage:example").send();

// Write data (requires credentials)
var receipt = contract.registerAgent(did, name, desc, endpoint, pubKey, capabilities, signature).send();
\`\`\`

## Build

\`\`\`bash
# Maven
mvn clean compile

# Gradle
gradle build
\`\`\`

## Regenerate Bindings

\`\`\`bash
npm run generate:java
\`\`\`
`;

    fs.writeFileSync(path.join(javaDir, 'README.md'), readmeContent);
    log(" Created README.md", "green");

    log("\n" + "=".repeat(50), "bright");
    log(" Java binding generation complete!", "green");
    log("\nGenerated files:", "yellow");
    log(`  📁 ${javaDir}/`, "cyan");
    log("  📄 SageRegistryV2.java", "cyan");
    log("  📄 SageVerificationHook.java", "cyan");
    log("  📄 pom.xml - Maven configuration", "cyan");
    log("  📄 build.gradle - Gradle configuration", "cyan");
    log("  📄 Example.java - Usage example", "cyan");

  } catch (error) {
    log("\n Error generating Java bindings:", "red");
    console.error(error);
    process.exit(1);
  }
}

// Run generation
generateJavaBindings();