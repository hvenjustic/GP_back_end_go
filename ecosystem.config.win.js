module.exports = {
  apps: [
    {
      name: 'GP_back_end_go',
      cwd: __dirname,
      script: 'powershell.exe',
      args: [
        '-NoProfile',
        '-ExecutionPolicy',
        'Bypass',
        '-Command',
        'if (!(Test-Path logs)) { New-Item -ItemType Directory logs | Out-Null }; if (!(Test-Path bin)) { New-Item -ItemType Directory bin | Out-Null }; go build -ldflags="-w -s" -o bin\\back_end_go.exe .; .\\bin\\back_end_go.exe -f config\\config-local.yaml',
      ],
      instances: 1,
      exec_mode: 'fork',
      autorestart: true,
      watch: false,
      max_memory_restart: '500M',
      error_file: './logs/err.log',
      out_file: './logs/out.log',
      log_date_format: 'YYYY-MM-DD HH:mm:ss Z',
      merge_logs: true,
      env: {
        NODE_ENV: 'production',
      },
      min_uptime: '10s',
      max_restarts: 10,
      restart_delay: 4000,
    },
  ],
};
