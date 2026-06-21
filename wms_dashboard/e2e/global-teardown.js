const path = require('path');
const fs = require('fs');

const PIDS_FILE = path.join(__dirname, '.e2e-pids.json');

module.exports = async function globalTeardown() {
  if (!fs.existsSync(PIDS_FILE)) return;

  const pids = JSON.parse(fs.readFileSync(PIDS_FILE, 'utf8'));
  for (const { name, pid } of pids) {
    try {
      process.kill(pid, 'SIGTERM');
      console.log(`[global-teardown] Stopped ${name} (pid ${pid})`);
    } catch {
      // already exited
    }
  }
  fs.unlinkSync(PIDS_FILE);
};
