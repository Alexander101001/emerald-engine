module.exports = {
  apps: [
    {
      name: 'emerald-brain',
      script: 'src/agi/brain.js',
      cwd: __dirname,
      node_args: '--experimental-modules',
      env: {
        NODE_ENV: 'production',
      },
      env_file: '.env',
      watch: false,
      autorestart: true,
      max_restarts: 10,
      restart_delay: 5000,
      log_date_format: 'YYYY-MM-DD HH:mm:ss Z',
      error_file: 'logs/brain-error.log',
      out_file: 'logs/brain-out.log',
      merge_logs: true,
      time: true,
    },
  ],
};
