import { spawn } from 'child_process';

let wailsProcess: ReturnType<typeof spawn> | null = null;

export default async () => {
  console.log('[E2E Setup] Starting wails dev...');
  
  // Kill any existing processes
  await new Promise<void>((resolve) => {
    spawn('pkill', ['-9', '-f', 'wails'], { stdio: 'ignore' });
    spawn('pkill', ['-9', '-f', 'vite'], { stdio: 'ignore' });
    setTimeout(resolve, 2000);
  });

  // Start wails dev
  wailsProcess = spawn('wails', ['dev'], {
    cwd: '/Users/jin/github/light-finger-browser',
    stdio: ['ignore', 'pipe', 'pipe'],
    detached: true,
  });

  wailsProcess.stdout?.on('data', (data: Buffer) => {
    process.stdout.write('[wails] ' + data.toString());
  });

  wailsProcess.stderr?.on('data', (data: Buffer) => {
    process.stderr.write('[wails:err] ' + data.toString());
  });

  // Wait for the frontend to be ready (up to 5 minutes)
  console.log('[E2E Setup] Waiting for frontend to be ready...');
  const startTime = Date.now();
  const maxWait = 300000; // 5 minutes

  while (Date.now() - startTime < maxWait) {
    try {
      const http = require('http');
      await new Promise<void>((resolve, reject) => {
        const req = http.get('http://localhost:5173', (res: any) => {
          resolve();
        });
        req.on('error', reject);
        req.setTimeout(1000, () => {
          req.destroy();
          reject(new Error('timeout'));
        });
      });
      console.log('[E2E Setup] Frontend is ready!');
      return;
    } catch {
      await new Promise(r => setTimeout(r, 5000));
    }
  }

  throw new Error('[E2E Setup] Timeout waiting for frontend');
};

export { wailsProcess };
