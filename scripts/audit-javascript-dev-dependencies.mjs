#!/usr/bin/env node
import { execFileSync, spawnSync } from 'node:child_process';
import { createHash, randomUUID } from 'node:crypto';
import { copyFile, mkdir, mkdtemp, readdir, rename, rm, stat, writeFile } from 'node:fs/promises';
import { existsSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { dirname, join, resolve, relative, isAbsolute } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const expectedNode = 'v22.20.0';
const expectedNpm = '10.9.3';
const packages = [
  { name: 'pi-extension', relative: 'pi-extension' },
  { name: 'vscode-extension', relative: 'ide/vscode-extension' },
];
const npmCliCandidates = [
  join(dirname(process.execPath), 'node_modules', 'npm', 'bin', 'npm-cli.js'),
  resolve(dirname(process.execPath), '..', 'lib', 'node_modules', 'npm', 'bin', 'npm-cli.js'),
  resolve(dirname(process.execPath), '..', 'node_modules', 'npm', 'bin', 'npm-cli.js'),
  ...(process.platform === 'win32' ? [
    join(process.env.ProgramFiles ?? 'C:\\Program Files', 'nodejs', 'node_modules', 'npm', 'bin', 'npm-cli.js'),
  ] : []),
];
const npmCliPath = npmCliCandidates.find(existsSync);
const gitCandidates = process.platform === 'win32'
  ? [join(process.env.ProgramFiles ?? 'C:\\Program Files', 'Git', 'cmd', 'git.exe'), join(process.env['ProgramFiles(x86)'] ?? 'C:\\Program Files (x86)', 'Git', 'cmd', 'git.exe')]
  : ['/usr/bin/git', '/usr/local/bin/git'];
const gitPath = gitCandidates.find(existsSync);
let stage = 'arguments';
let work = '';
let publishStage = '';

function fail() {
  throw new Error(`QA04_FAILURE:${stage}`);
}

function safeEnvironment(home) {
  const env = {
    PATH: [dirname(process.execPath), '/usr/bin', '/bin'].join(process.platform === 'win32' ? ';' : ':'),
    HOME: home,
    USERPROFILE: home,
    TMPDIR: home,
    TEMP: home,
    TMP: home,
    CI: '1',
    GIT_TERMINAL_PROMPT: '0',
    GIT_CONFIG_NOSYSTEM: '1',
    GIT_CONFIG_GLOBAL: join(home, 'empty.gitconfig'),
    npm_config_registry: 'https://registry.npmjs.org/',
    npm_config_cache: join(home, 'npm-cache'),
    npm_config_userconfig: join(home, 'empty.npmrc'),
    npm_config_globalconfig: join(home, 'empty-global.npmrc'),
    npm_config_ignore_scripts: 'true',
    npm_config_fund: 'false',
    npm_config_update_notifier: 'false',
  };
  for (const key of ['SystemRoot', 'SYSTEMROOT', 'WINDIR', 'COMSPEC', 'PATHEXT']) {
    if (process.env[key]) env[key] = process.env[key];
  }
  return env;
}

function command(file, args, cwd, env, encoding = 'utf8') {
  const result = spawnSync(file, args, {
    cwd, env, encoding, maxBuffer: 64 * 1024 * 1024,
    windowsHide: true, shell: false,
  });
  if (result.error || result.signal || result.status === null) fail();
  return { status: result.status, stdout: result.stdout ?? Buffer.alloc(0) };
}

function gitBlob(commit, path, env) {
  const treeLine = execFileSync(gitPath, ['-C', root, 'ls-tree', '-z', commit, '--', path], { env });
  const row = treeLine.toString('utf8').replace(/\0$/, '');
  if (!row.endsWith(`\t${path}`) || !row.startsWith('100644 blob ')) fail();
  return execFileSync(gitPath, ['-C', root, 'cat-file', 'blob', `${commit}:${path}`], { env, maxBuffer: 16 * 1024 * 1024 });
}

const sha256 = (bytes) => createHash('sha256').update(bytes).digest('hex');

function validateAudit(stdout, status) {
  let report;
  try {
    report = JSON.parse(stdout.toString('utf8'));
  } catch {
    fail();
  }
  if (!report || report.error || report.auditReportVersion < 2 ||
      !report.vulnerabilities || typeof report.vulnerabilities !== 'object' ||
      !report.metadata?.vulnerabilities || !report.metadata?.dependencies) fail();
  const counts = { info: 0, low: 0, moderate: 0, high: 0, critical: 0 };
  for (const item of Object.values(report.vulnerabilities)) {
    if (!item || !(item.severity in counts) || !Array.isArray(item.via)) fail();
    counts[item.severity] += 1;
  }
  const aggregate = report.metadata.vulnerabilities;
  if (aggregate.total !== Object.keys(report.vulnerabilities).length ||
      Object.entries(counts).some(([severity, count]) => aggregate[severity] !== count) ||
      ![0, 1].includes(status) || status !== (aggregate.total > 0 ? 1 : 0)) fail();
  return { report, counts, total: aggregate.total };
}

async function exists(path) {
  try { await stat(path); return true; } catch (error) {
    if (error.code === 'ENOENT') return false;
    throw error;
  }
}

async function main(outputArg) {
  if (!outputArg) fail();
  const outputDir = resolve(outputArg);
  const outputRelative = relative(root, outputDir).replaceAll(String.fromCharCode(92), '/');
  if (outputRelative === '' || (!outputRelative.startsWith('..') && !isAbsolute(outputRelative) &&
      packages.some((item) => outputRelative === item.relative || outputRelative.startsWith(`${item.relative}/`)))) fail();
  if (await exists(outputDir)) fail();

  stage = 'temporary-workspace';
  work = await mkdtemp(join(tmpdir(), 'qa04-javascript-audit-'));
  const env = safeEnvironment(work);
  await writeFile(env.npm_config_userconfig, '');
  await writeFile(env.npm_config_globalconfig, '');
  await writeFile(env.GIT_CONFIG_GLOBAL, '');

  stage = 'tool-version';
  if (process.version !== expectedNode || !npmCliPath) fail();
  const version = command(process.execPath, [npmCliPath, '--version'], work, env);
  if (version.status !== 0 || version.stdout.toString('utf8').trim() !== expectedNpm) fail();

  stage = 'source-commit';
  if (!gitPath) fail();
  const commit = execFileSync(gitPath, ['-C', root, 'rev-parse', '--verify', 'HEAD^{commit}'], { env, encoding: 'utf8' }).trim();
  if (!/^[0-9a-f]{40,64}$/.test(commit)) fail();
  const metadata = {
    auditDateUtc: new Date().toISOString(),
    sourceCommit: commit,
    nodeVersion: process.version,
    npmVersion: expectedNpm,
    registry: 'https://registry.npmjs.org/',
    installFlags: ['ci', '--ignore-scripts', '--no-audit', '--no-fund'],
    lifecycleScriptsEnabled: false,
    packages: [],
  };
  const stagedReports = [];
  const reportTemp = join(work, 'validated-reports');
  await mkdir(reportTemp);

  for (const item of packages) {
    stage = `${item.name}-source-blobs`;
    const manifestPath = `${item.relative}/package.json`;
    const lockPath = `${item.relative}/package-lock.json`;
    const manifestBytes = gitBlob(commit, manifestPath, env);
    const lockBytes = gitBlob(commit, lockPath, env);
    let manifest;
    let lock;
    try {
      manifest = JSON.parse(manifestBytes.toString('utf8'));
      lock = JSON.parse(lockBytes.toString('utf8'));
    } catch {
      fail();
    }
    if (!manifest.name || !manifest.version || lock.name !== manifest.name ||
        lock.version !== manifest.version || lock.lockfileVersion < 3 ||
        lock.packages?.['']?.name !== manifest.name || lock.packages?.['']?.version !== manifest.version) fail();

    const pkgDir = join(work, item.name);
    await mkdir(pkgDir);
    await writeFile(join(pkgDir, 'package.json'), manifestBytes);
    await writeFile(join(pkgDir, 'package-lock.json'), lockBytes);
    const inputFiles = (await readdir(pkgDir)).sort();
    if (inputFiles.join(',') !== 'package-lock.json,package.json') fail();

    stage = `${item.name}-npm-ci`;
    const install = command(process.execPath, [npmCliPath, 'ci', '--ignore-scripts', '--no-audit', '--no-fund'], pkgDir, env);
    if (install.status !== 0) fail();

    const packageMetadata = {
      name: item.name,
      package: manifest.name,
      packageVersion: manifest.version,
      manifestPath,
      manifestRawGitBlobSha256: sha256(manifestBytes),
      lockfilePath: lockPath,
      lockfileRawGitBlobSha256: sha256(lockBytes),
      npmCiExitCode: install.status,
    };
    for (const audit of [
      { kind: 'full', args: ['audit', '--json'] },
      { kind: 'omitDev', args: ['audit', '--omit=dev', '--json'] },
    ]) {
      stage = `${item.name}-${audit.kind}-audit`;
      const result = command(process.execPath, [npmCliPath, ...audit.args], pkgDir, env);
      const validated = validateAudit(result.stdout, result.status);
      const filename = `${item.name}-${audit.kind === 'full' ? 'full' : 'omit-dev'}.json`;
      await writeFile(join(reportTemp, filename), result.stdout);
      packageMetadata[`${audit.kind}AuditExitCode`] = result.status;
      packageMetadata[`${audit.kind}Findings`] = validated.total;
      packageMetadata[`${audit.kind}SeverityCounts`] = validated.counts;
      stagedReports.push(filename);
    }
    metadata.packages.push(packageMetadata);
  }

  stage = 'metadata-validation';
  const serialized = `${JSON.stringify(metadata, null, 2)}\n`;
  if (serialized.includes(work) || serialized.includes(root)) fail();
  await writeFile(join(reportTemp, 'metadata.json'), serialized);
  stagedReports.push('metadata.json');

  stage = 'publish';
  if (await exists(outputDir)) fail();
  await mkdir(dirname(outputDir), { recursive: true });
  publishStage = `${outputDir}.tmp-${randomUUID()}`;
  await mkdir(publishStage);
  for (const filename of stagedReports) await copyFile(join(reportTemp, filename), join(publishStage, filename));
  await rename(publishStage, outputDir);
  publishStage = '';

  console.log(`QA04_OK ${metadata.auditDateUtc} ${commit} pi=${metadata.packages[0].fullFindings}/${metadata.packages[0].omitDevFindings} vscode=${metadata.packages[1].fullFindings}/${metadata.packages[1].omitDevFindings}`);
}

try {
  await main(process.argv[2]);
} catch (error) {
  if (publishStage) await rm(publishStage, { recursive: true, force: true }).catch(() => {});
  if (work) await rm(work, { recursive: true, force: true }).catch(() => {});
  const stageName = typeof error?.message === 'string' && error.message.startsWith('QA04_FAILURE:')
    ? error.message.slice('QA04_FAILURE:'.length) : 'internal';
  process.stderr.write(`QA04_FAILED stage=${stageName}; no report published\n`);
  process.exitCode = 1;
}
if (work) await rm(work, { recursive: true, force: true }).catch(() => {});
