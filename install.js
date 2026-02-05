#!/usr/bin/env node

const os = require('os');
const path = require('path');
const fs = require('fs');
const https = require('https');
const { spawnSync } = require('child_process');

const platformMap = {
    darwin: 'darwin',
    linux: 'linux',
    win32: 'windows'
};

const archMap = {
    x64: 'amd64',
    arm64: 'arm64'
};

const platform = os.platform();
const arch = os.arch();
const ext = platform === 'win32' ? '.exe' : '';
const platformName = platformMap[platform];
const archName = archMap[arch];

if (!platformName || !archName) {
    console.error('Unsupported platform or architecture:', platform, arch);
    process.exit(1);
}

const pkg = require('./package.json');
const version = pkg.version;
const githubVersion = `v${version}`;
const binName = `ssh-x-term-${platformName}-${archName}${ext}`;
const releaseUrl = `https://github.com/eugeniofciuvasile/ssh-x-term/releases/download/${githubVersion}/${binName}`;

// Directory to store the binary
const installDir = path.join(os.homedir(), '.ssh-x-term');
if (!fs.existsSync(installDir)) {
    fs.mkdirSync(installDir, { recursive: true });
}
const binaryPath = path.join(installDir, binName);

function downloadFile(url, dest, cb, redirectCount = 0) {
    if (redirectCount > 5) {
        cb(new Error('Too many redirects'));
        return;
    }
    const file = fs.createWriteStream(dest, { mode: 0o755 });
    https.get(url, response => {
        if (response.statusCode === 200) {
            response.pipe(file);
            file.on('finish', () => file.close(cb));
        } else if (response.statusCode === 302 || response.statusCode === 301) {
            // Follow redirect
            const redirectUrl = response.headers.location;
            file.close();
            fs.unlinkSync(dest);
            downloadFile(redirectUrl, dest, cb, redirectCount + 1);
        } else {
            file.close();
            fs.unlinkSync(dest);
            cb(new Error(`Failed to get '${url}' (${response.statusCode})`));
        }
    }).on('error', err => {
        fs.unlink(dest, () => cb(err));
    });
}

function checkDeps() {
    const deps = [
        { name: 'tmux', check: 'tmux' },
        { name: 'bw (bitwarden-cli)', check: 'bw' }
    ];

    function checkDep(cmd) {
        const which = platform === 'win32' ? 'where' : 'which';
        const res = spawnSync(which, [cmd]);
        return res.status === 0;
    }

    let missing = [];
    deps.forEach(dep => {
        if (!checkDep(dep.check)) {
            missing.push(dep.name);
        }
    });

    if (missing.length > 0) {
        console.warn('\nMissing system dependencies:');
        missing.forEach(dep => console.warn(`  - ${dep}`));
        console.warn('\nPlease install these manually following the instructions in the README before using ssh-x-term.\n');
    } else {
        console.log('All system dependencies found.');
    }
}

console.log(`Downloading ${binName} from ${releaseUrl} ...`);
downloadFile(releaseUrl, binaryPath, err => {
    if (err) {
        console.error('Failed to download the binary:', err.message);
        process.exit(1);
    }
    if (platform !== 'win32') {
        fs.chmodSync(binaryPath, 0o755);
    }
    console.log('ssh-x-term binary installed at', binaryPath);
    // Dependency check after binary install
    checkDeps();
});
