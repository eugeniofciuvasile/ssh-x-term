#!/usr/bin/env node

const https = require('https');
const fs = require('fs');
const path = require('path');
const os = require('os');
const { spawn } = require('child_process');

const packageJson = require('./package.json');
const version = packageJson.version;
const name = packageJson.name;

// Determine platform and architecture
const platform = os.platform();
const arch = os.arch();

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

// Validate platform support
if (!platformMap[platform]) {
    console.error(`Unsupported platform: ${platform}. Only macOS, Linux, and Windows are supported.`);
    process.exit(1);
}

if (!archMap[arch]) {
    console.error(`Unsupported architecture: ${arch}. Only x64 and arm64 are supported.`);
    process.exit(1);
}

// Set file extension based on platform
const ext = platform === 'win32' ? '.exe' : '';

// Construct binary name and URL
const binName = `ssh-x-term-${platformMap[platform]}-${archMap[arch]}${ext}`;
const url = `https://github.com/eugeniofciuvasile/ssh-x-term/releases/download/v${version}/${binName}`;

// Define installation directories
const installDir = path.join(os.homedir(), '.ssh-x-term');
const binDir = path.join(__dirname, 'bin');
const destPath = path.join(installDir, binName);
const binPath = path.join(binDir, 'sxt' + ext);

// Download file directly
function downloadFile(url, destPath) {
    return new Promise((resolve, reject) => {
        console.log(`Downloading from ${url} to ${destPath}`);

        // Ensure destination directory exists
        const destDir = path.dirname(destPath);
        if (!fs.existsSync(destDir)) {
            fs.mkdirSync(destDir, { recursive: true });
        }

        const file = fs.createWriteStream(destPath);

        https.get(url, (response) => {
            if (response.statusCode === 302 || response.statusCode === 301) {
                // Handle redirects
                console.log(`Following redirect to ${response.headers.location}`);
                return downloadFile(response.headers.location, destPath)
                    .then(resolve)
                    .catch(reject);
            }

            if (response.statusCode !== 200) {
                reject(new Error(`Failed to download: Server responded with ${response.statusCode}`));
                return;
            }

            response.pipe(file);

            file.on('finish', () => {
                file.close();
                console.log(`Download completed: ${destPath}`);
                resolve(destPath);
            });
        }).on('error', (err) => {
            fs.unlink(destPath, () => { }); // Delete the file on error
            reject(err);
        });
    });
}

// Install dependencies based on platform
async function installDependencies() {
    console.log('Installing dependencies...');

    if (platform === 'darwin') {
        console.log('Installing dependencies with Homebrew...');
        try {
            await runCommand('brew', ['install', 'bitwarden-cli', 'tmux']);
            console.log('Note: sshpass is not available in the default Homebrew. For sshpass installation, see: https://gist.github.com/arunoda/7790979');
        } catch (error) {
            console.warn(`Failed to install dependencies: ${error.message}`);
            console.warn('Please install them manually: bitwarden-cli, sshpass, tmux');
        }
    } else if (platform === 'linux') {
        console.log('Detecting Linux package manager...');

        // Try apt (Debian/Ubuntu)
        if (await commandExists('apt')) {
            console.log('Installing dependencies with apt...');
            console.log('This will prompt for your sudo password.');

            try {
                console.log('Running: sudo apt update');
                await runCommand('sudo', ['apt', 'update']);

                console.log('Running: sudo apt install -y sshpass tmux');
                await runCommand('sudo', ['apt', 'install', '-y', 'sshpass', 'tmux']);

                console.log('Running: npm install -g @bitwarden/cli');
                await runCommand('npm', ['install', '-g', '@bitwarden/cli']);

                console.log('Dependencies installed successfully.');
            } catch (error) {
                console.warn(`Failed to install dependencies: ${error.message}`);
                console.warn('Please install them manually: bitwarden-cli, sshpass, tmux');
            }
        }
        // Try yum/dnf (RHEL/Fedora)
        else if (await commandExists('dnf') || await commandExists('yum')) {
            const pm = await commandExists('dnf') ? 'dnf' : 'yum';
            console.log(`Installing dependencies with ${pm}...`);
            console.log('This will prompt for your sudo password.');

            try {
                console.log(`Running: sudo ${pm} install -y sshpass tmux`);
                await runCommand('sudo', [pm, 'install', '-y', 'sshpass', 'tmux']);

                console.log('Running: npm install -g @bitwarden/cli');
                await runCommand('npm', ['install', '-g', '@bitwarden/cli']);

                console.log('Dependencies installed successfully.');
            } catch (error) {
                console.warn(`Failed to install dependencies: ${error.message}`);
                console.warn('Please install them manually: bitwarden-cli, sshpass, tmux');
            }
        } else {
            console.warn('Unable to detect package manager. Please install dependencies manually: bitwarden-cli, sshpass, tmux');
        }
    } else if (platform === 'win32') {
        console.log('For Windows users:');
        console.log('1. Install tmux via Windows Subsystem for Linux or alternatives like Cygwin');
        console.log('2. Install Bitwarden CLI with: npm install -g @bitwarden/cli');
        console.log('3. For sshpass functionality, you may need alternative approaches, see plink.exe');
    }
}

// Check if command exists
async function commandExists(cmd) {
    try {
        const command = platform === 'win32' ? 'where' : 'which';

        return new Promise((resolve) => {
            const proc = spawn(command, [cmd], { stdio: 'ignore' });

            proc.on('close', (code) => {
                resolve(code === 0);
            });

            proc.on('error', () => {
                resolve(false);
            });
        });
    } catch (error) {
        return false;
    }
}

// Run command and return promise
function runCommand(cmd, args) {
    return new Promise((resolve, reject) => {
        // Use inherit for stdio to ensure password prompts are visible
        const proc = spawn(cmd, args, {
            stdio: 'inherit',
        });

        proc.on('close', code => {
            if (code !== 0) {
                reject(new Error(`Command ${cmd} ${args.join(' ')} failed with code ${code}`));
            } else {
                resolve();
            }
        });

        proc.on('error', reject);
    });
}

// Main installation function
async function install() {
    console.log(`Starting installation of ${name} v${version}...`);
    console.log(`Platform: ${platform}, Architecture: ${arch}`);
    console.log(`Binary name: ${binName}`);
    console.log(`Download URL: ${url}`);

    try {
        // Create download directory if it doesn't exist
        if (!fs.existsSync(installDir)) {
            fs.mkdirSync(installDir, { recursive: true });
        }

        // Download the binary directly
        await downloadFile(url, destPath);

        // Make it executable
        fs.chmodSync(destPath, '755');
        console.log(`Made binary executable: ${destPath}`);

        // Create bin directory if it doesn't exist
        if (!fs.existsSync(binDir)) {
            fs.mkdirSync(binDir, { recursive: true });
        }

        // Create executable with name "sxt"
        console.log(`Creating executable at: ${binPath}`);

        if (fs.existsSync(binPath)) {
            console.log('Removing existing executable...');
            fs.unlinkSync(binPath);
        }

        // On Windows, copy the file instead of creating a symlink
        if (platform === 'win32') {
            console.log('Copying binary to executable path...');
            fs.copyFileSync(destPath, binPath);
        } else {
            // Create a shell script wrapper
            console.log('Creating shell script wrapper...');
            const scriptContent = `#!/bin/sh\n"${destPath}" "$@"\n`;
            fs.writeFileSync(binPath, scriptContent);
            fs.chmodSync(binPath, '755'); // Make executable
        }

        console.log(`Successfully installed ${name} to ${binPath}`);

        // Install dependencies
        await installDependencies();

        console.log('Installation complete! You can now run "sxt" from the command line.');
    } catch (error) {
        console.error('Installation failed:');
        console.error(error);
        process.exit(1);
    }
}

// Run the installation
install();
