const { NodeSDK } = require('@opentelemetry/sdk-node');
const { getNodeAutoInstrumentations } = require('@opentelemetry/auto-instrumentations-node');
const { OTLPTraceExporter } = require('@opentelemetry/exporter-trace-otlp-grpc');
const { OTLPMetricExporter } = require('@opentelemetry/exporter-metrics-otlp-grpc');
const { PeriodicExportingMetricReader } = require('@opentelemetry/sdk-metrics');
const { Resource } = require('@opentelemetry/resources');
const { SemanticResourceAttributes } = require('@opentelemetry/semantic-conventions');
const express = require('express');
const axios = require('axios');
const winston = require('winston');

// Environment variables
const PORT = process.env.PORT || 8080;
const BACKEND_URL = process.env.BACKEND_URL || 'http://backend.default.svc.cluster.local:8080';
const OTEL_ENDPOINT = process.env.OTEL_EXPORTER_OTLP_ENDPOINT || 'otel-collector.observability.svc.cluster.local:4317';

// Logger setup
const logger = winston.createLogger({
  level: 'info',
  format: winston.format.combine(
    winston.format.timestamp(),
    winston.format.json()
  ),
  transports: [
    new winston.transports.Console()
  ]
});

// OpenTelemetry setup
const resource = Resource.default().merge(
  new Resource({
    [SemanticResourceAttributes.SERVICE_NAME]: 'frontend',
    [SemanticResourceAttributes.SERVICE_VERSION]: '1.0.0',
    [SemanticResourceAttributes.DEPLOYMENT_ENVIRONMENT]: 'development',
    application: 'frontend-web'
  })
);

const traceExporter = new OTLPTraceExporter({
  url: `grpc://${OTEL_ENDPOINT}`,
});

const metricExporter = new OTLPMetricExporter({
  url: `grpc://${OTEL_ENDPOINT}`,
});

const sdk = new NodeSDK({
  resource: resource,
  traceExporter: traceExporter,
  metricReader: new PeriodicExportingMetricReader({
    exporter: metricExporter,
    exportIntervalMillis: 10000,
  }),
  instrumentations: [getNodeAutoInstrumentations({
    '@opentelemetry/instrumentation-fs': {
      enabled: false,
    },
  })],
});

// Start OpenTelemetry SDK
try {
  sdk.start();
  logger.info('OpenTelemetry initialized successfully');
} catch (error) {
  logger.error('Error initializing OpenTelemetry', { error: error.message });
}

// Graceful shutdown
process.on('SIGTERM', () => {
  sdk.shutdown()
    .then(() => logger.info('OpenTelemetry terminated'))
    .catch((error) => logger.error('Error terminating OpenTelemetry', { error: error.message }))
    .finally(() => process.exit(0));
});

// Express app
const app = express();

app.use(express.json());
app.use(express.urlencoded({ extended: true }));

// Request logging middleware
app.use((req, res, next) => {
  const start = Date.now();
  res.on('finish', () => {
    const duration = Date.now() - start;
    logger.info('HTTP request', {
      method: req.method,
      path: req.path,
      status: res.statusCode,
      duration: duration,
      ip: req.ip
    });
  });
  next();
});

// Routes
app.get('/', (req, res) => {
  res.json({
    service: 'frontend',
    version: '1.0.0',
    message: 'Welcome to Observability Stack Demo',
    endpoints: {
      health: '/health',
      users: '/api/users',
      user: '/api/users/:id',
      process: '/api/process'
    }
  });
});

app.get('/health', (req, res) => {
  res.json({
    status: 'healthy',
    service: 'frontend',
    timestamp: new Date().toISOString()
  });
});

// Proxy to backend
app.get('/api/users', async (req, res) => {
  try {
    logger.info('Fetching users from backend');
    const response = await axios.get(`${BACKEND_URL}/api/users`, {
      timeout: 5000
    });
    res.json(response.data);
  } catch (error) {
    logger.error('Error fetching users', {
      error: error.message,
      backend: BACKEND_URL
    });
    res.status(500).json({
      error: 'Failed to fetch users',
      message: error.message
    });
  }
});

app.get('/api/users/:id', async (req, res) => {
  const userId = req.params.id;
  try {
    logger.info('Fetching user from backend', { userId });
    const response = await axios.get(`${BACKEND_URL}/api/users/${userId}`, {
      timeout: 5000
    });
    res.json(response.data);
  } catch (error) {
    logger.error('Error fetching user', {
      userId,
      error: error.message
    });
    
    if (error.response && error.response.status === 404) {
      res.status(404).json({ error: 'User not found' });
    } else {
      res.status(500).json({
        error: 'Failed to fetch user',
        message: error.message
      });
    }
  }
});

app.post('/api/process', async (req, res) => {
  try {
    logger.info('Processing request');
    const response = await axios.post(`${BACKEND_URL}/api/process`, req.body, {
      timeout: 10000
    });
    res.json(response.data);
  } catch (error) {
    logger.error('Error processing request', {
      error: error.message
    });
    res.status(500).json({
      error: 'Processing failed',
      message: error.message
    });
  }
});

// Load test endpoint
app.get('/api/load-test', async (req, res) => {
  const requests = parseInt(req.query.requests) || 10;
  logger.info('Starting load test', { requests });
  
  const results = [];
  for (let i = 0; i < requests; i++) {
    try {
      const response = await axios.get(`${BACKEND_URL}/api/users`);
      results.push({ success: true, status: response.status });
    } catch (error) {
      results.push({ success: false, error: error.message });
    }
  }
  
  const successful = results.filter(r => r.success).length;
  res.json({
    total: requests,
    successful,
    failed: requests - successful,
    successRate: (successful / requests * 100).toFixed(2) + '%'
  });
});

// 404 handler
app.use((req, res) => {
  res.status(404).json({
    error: 'Not Found',
    path: req.path
  });
});

// Error handler
app.use((err, req, res, next) => {
  logger.error('Unhandled error', {
    error: err.message,
    stack: err.stack
  });
  res.status(500).json({
    error: 'Internal Server Error',
    message: err.message
  });
});

// Start server
app.listen(PORT, () => {
  logger.info(`Frontend server started`, {
    port: PORT,
    backend: BACKEND_URL,
    otelEndpoint: OTEL_ENDPOINT
  });
});

module.exports = app;
