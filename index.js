#!/usr/bin/env node

const path = require('path');
const os = require('os');
const fs = require('fs');
const { spawnSync } = require('child_process');

// Platform and architecture
const platform = os.platform();
const arch = os.arch();
const ext = platform === 'win32' ? '.exe' : '';

// Map to GitHub release names
const platformMap = {
    darwin: 'darwin',
    linux: 'linux',
    win32: 'windows'
};

const archMap = {
    x64: 'amd64',
    arm64: 'arm64'
};

// Define download paths 
const installDir = path.join(os.homedir(), '.ssh-x-term');
const binName = `ssh-x-term-${platformMap[platform]}-${archMap[arch]}${ext}`;
const binaryPath = path.join(installDir, binName);

// Run the binary
function runBinary() {
    // Check if the binary exists
    if (!fs.existsSync(binaryPath)) {
        console.error(`Binary not found at ${binaryPath}`);
        console.error('Please reinstall the package with: npm install -g ssh-x-term');
        process.exit(1);
    }

    // Make sure it's executable
    if (platform !== 'win32') {
        try {
            fs.chmodSync(binaryPath, '755');
        } catch (err) {
            console.error(`Error making binary executable: ${err.message}`);
        }
    }

    // Run the binary with any passed arguments
    const result = spawnSync(binaryPath, process.argv.slice(2), {
        stdio: 'inherit'
    });

    // Forward the exit code
    process.exit(result.status || 0);
}

// If this script is being executed directly, run the binary
if (require.main === module) {
    runBinary();
}

// Export for programmatic usage
module.exports = {
    run: runBinary
};
