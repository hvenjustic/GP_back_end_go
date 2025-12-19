module.exports = {
  apps: [
    {
      name: 'GP_back_end_go',
      cwd: __dirname,
      script: 'bash',
      args: ['-lc', 'mkdir -p logs && make install && ./bin/back_end_go -f config/config-local.yaml'],
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

