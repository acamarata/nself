const { Queue, Worker } = require('bullmq');

const queueName = 'pdf-processor';
const connection = {
  host: process.env.REDIS_HOST || 'redis',
  port: process.env.REDIS_PORT || 6379,
};

// Create queue
const queue = new Queue(queueName, { connection });

// Create worker
const worker = new Worker(
  queueName,
  async (job) => {
    console.log(`Processing job ${job.id}:`, job.data);
    // Add your job processing logic here
    return { processed: true, timestamp: new Date() };
  },
  { connection }
);

// Event listeners
worker.on('completed', (job, result) => {
  console.log(`Job ${job.id} completed:`, result);
});

worker.on('failed', (job, err) => {
  console.error(`Job ${job.id} failed:`, err.message);
});

console.log(`pdf-processor worker started, waiting for jobs...`);

// Graceful shutdown
process.on('SIGTERM', async () => {
  await worker.close();
  process.exit(0);
});
