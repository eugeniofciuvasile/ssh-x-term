#!/usr/bin/env node

const os = require('os');
const { spawnSync } = require('child_process');

const deps = [
    { name: 'tmux', check: 'tmux' },
    { name: 'sshpass', check: 'sshpass' },
    { name: 'bw (bitwarden-cli)', check: 'bw' }
];

function checkDep(cmd) {
    const which = os.platform() === 'win32' ? 'where' : 'which';
    const res = spawnSync(which, [cmd]);
    return res.status === 0;
}

function checkDeps() {
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

checkDeps();
