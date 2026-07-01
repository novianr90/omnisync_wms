const { execSync, spawn } = require('child_process');
const path = require('path');
const fs = require('fs');
const http = require('http');

const PIDS_FILE = path.join(__dirname, '.e2e-pids.json');
const WMS_BASE_PORT = 9991;
const AUTH_PORT = 8080;

const DB_TYPE = process.env.DB_TYPE || 'sqlite';
const usePostgres = DB_TYPE === 'postgres';
const numWorkers = process.env.CI ? 4 : 1;

const PGHOST = process.env.PGHOST || 'localhost';
const PGPORT = process.env.PGPORT || '5432';
const PGUSER = process.env.PGUSER || 'omnisync';
const PGPASSWORD = process.env.PGPASSWORD || 'test1234';

const JWT_SECRET = process.env.JWT_SECRET_KEY || 'test-signing-key-for-auth-services-unit-tests-12345';
const AUTH_DATABASE_URL =
  process.env.AUTH_DATABASE_URL ||
  `postgres://${PGUSER}:${PGPASSWORD}@${PGHOST}:${PGPORT}/omnisync_test?sslmode=disable`;

const wmsRoot = path.join(__dirname, '..');
const authRoot = path.join(__dirname, '../../auth_services');

const wmsBinName = process.platform === 'win32' ? 'wms_dashboard.exe' : 'wms_dashboard';
const authBinName = process.platform === 'win32' ? 'auth_services.exe' : 'auth_services';

function waitForHealth(url, timeoutMs = 60000) {
  return new Promise((resolve, reject) => {
    const start = Date.now();
    function attempt() {
      const req = http.get(url, (res) => {
        res.resume();
        if (res.statusCode === 200) return resolve();
        schedule();
      });
      req.on('error', schedule);
      req.setTimeout(2000, () => { req.destroy(); schedule(); });
      function schedule() {
        if (Date.now() - start > timeoutMs) {
          return reject(new Error(`[global-setup] ${url} not healthy after ${timeoutMs}ms`));
        }
        setTimeout(attempt, 500);
      }
    }
    attempt();
  });
}

function psql(sql, db) {
  execSync(
    `psql -h ${PGHOST} -p ${PGPORT} -U ${PGUSER} -d "${db || 'omnisync_test'}" -c "${sql}"`,
    { env: { ...process.env, PGPASSWORD }, stdio: 'pipe' }
  );
}

function spawnServer(cmd, args, opts) {
  const proc = spawn(cmd, args, { ...opts, stdio: 'inherit' });
  proc.on('error', (err) => console.error(`[global-setup] spawn error (${cmd}):`, err.message));
  return proc;
}

module.exports = async function globalSetup() {
  // ponytail: skip spawning when services are already running in Docker
  if (process.env.DOCKER_STACK === 'true') {
    console.log('[global-setup] Docker stack detected — waiting for exposed services...');
    await waitForHealth(`http://localhost:${AUTH_PORT}/health`);
    await waitForHealth(`http://localhost:${WMS_BASE_PORT}/health`);
    fs.writeFileSync(PIDS_FILE, JSON.stringify([]));
    console.log('[global-setup] Docker services ready.');
    return;
  }

  const pids = [];

  if (usePostgres) {
    // ponytail: create empty DBs only — each WMS server runs its own tracked migrations on startup
    console.log(`[global-setup] Creating ${numWorkers} worker database(s)...`);
    for (let i = 0; i < numWorkers; i++) {
      const db = `wms_test_${i}`;
      // WITH (FORCE) terminates open connections before drop — requires PG13+
      psql(`DROP DATABASE IF EXISTS ${db} WITH (FORCE);`);
      psql(`CREATE DATABASE ${db};`);
    }
  }

  const wmsOrigins = Array.from({ length: numWorkers }, (_, i) =>
    `http://localhost:${WMS_BASE_PORT + i}`
  ).join(',');

  // Auth service — single shared instance
  const authBinPath = path.join(authRoot, authBinName);
  const useAuthBin = fs.existsSync(authBinPath);
  const [authCmd, authArgs] = useAuthBin ? [authBinPath, []] : ['go', ['run', 'cmd/main.go']];

  console.log(`[global-setup] Starting auth service on :${AUTH_PORT}...`);
  const authProc = spawnServer(authCmd, authArgs, {
    cwd: authRoot,
    env: {
      ...process.env,
      PORT: String(AUTH_PORT),
      JWT_SECRET_KEY: JWT_SECRET,
      DB_TYPE,
      AUTH_DATABASE_URL,
      ALLOWED_ORIGIN: wmsOrigins,
      SEED_ADMIN_EMAIL: process.env.SEED_ADMIN_EMAIL || 'admin@omnisync.com',
      SEED_ADMIN_PASSWORD: process.env.SEED_ADMIN_PASSWORD || 'admin123',
      SEED_OPERATOR_EMAIL: process.env.SEED_OPERATOR_EMAIL || 'operator@omnisync.com',
      SEED_OPERATOR_PASSWORD: process.env.SEED_OPERATOR_PASSWORD || 'operator123',
    },
  });
  pids.push({ name: 'auth', pid: authProc.pid });

  // Per-worker WMS instances
  const wmsBinPath = path.join(wmsRoot, wmsBinName);
  const useWmsBin = fs.existsSync(wmsBinPath);
  const [wmsCmd, wmsArgs] = useWmsBin ? [wmsBinPath, []] : ['go', ['run', 'cmd/main.go']];

  for (let i = 0; i < numWorkers; i++) {
    const port = WMS_BASE_PORT + i;
    const wmsDbUrl = usePostgres
      ? `postgres://${PGUSER}:${PGPASSWORD}@${PGHOST}:${PGPORT}/wms_test_${i}?sslmode=disable`
      : undefined;

    console.log(`[global-setup] Starting WMS instance #${i} on :${port}...`);
    const proc = spawnServer(wmsCmd, wmsArgs, {
      cwd: wmsRoot,
      env: {
        ...process.env,
        PORT: String(port),
        JWT_SECRET_KEY: JWT_SECRET,
        AUTH_API_URL: `http://localhost:${AUTH_PORT}`,
        DB_TYPE,
        ...(wmsDbUrl ? { WMS_DATABASE_URL: wmsDbUrl } : {}),
      },
    });
    pids.push({ name: `wms_${i}`, pid: proc.pid });
  }

  // Wait for all health checks
  console.log('[global-setup] Waiting for all servers to be healthy...');
  await waitForHealth(`http://localhost:${AUTH_PORT}/health`);
  await Promise.all(
    Array.from({ length: numWorkers }, (_, i) =>
      waitForHealth(`http://localhost:${WMS_BASE_PORT + i}/health`)
    )
  );

  fs.writeFileSync(PIDS_FILE, JSON.stringify(pids));
  console.log('[global-setup] All servers ready.');
};
