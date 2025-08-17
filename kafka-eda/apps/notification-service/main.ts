import { NestFactory } from '@nestjs/core';
import { AppModule } from './src/app.module';
import { MicroserviceOptions, Transport } from '@nestjs/microservices';
import { Logger } from '@nestjs/common';
import { Partitioners } from 'kafkajs';

async function bootstrap() {
  const logger = new Logger('KafkaBootstrap');

  const app = await NestFactory.createMicroservice<MicroserviceOptions>(
    AppModule,
    {
      transport: Transport.KAFKA,
      options: {
        client: {
          brokers: process.env.KAFKA_BROKERS?.split(',') ?? ['localhost:9092'],
          retry: {
            initialRetryTime: 1000,
            retries: 8,
            maxRetryTime: 5000,
            factor: 2,
          },
        },
        consumer: {
          groupId:
            process.env.KAFKA_CONSUMER_GROUP_ID ??
            'notification-consumer-group-server',
          allowAutoTopicCreation: true,
          sessionTimeout: 30000,
          rebalanceTimeout: 60000,
          heartbeatInterval: 5000,
        },
        producer: {
          createPartitioner: Partitioners.LegacyPartitioner,
        },
        run: {
          autoCommit: false,
        },
      },
    },
  );

  // Add error handling for Kafka connection
  try {
    await app.listen();
    logger.log('Notification Service successfully connected to Kafka');
  } catch (error) {
    logger.error('Failed to connect to Kafka:', error);
    process.exit(1);
  }
}

bootstrap();
