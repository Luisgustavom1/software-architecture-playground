import { Module, Logger } from '@nestjs/common';
import { ClientsModule, Transport } from '@nestjs/microservices';
import { ConfigService } from '@nestjs/config';
import { OrderService } from './order.service';
import { OrderController } from './order.controller';
import { KAFKA_PRODUCER_SERVICE } from '../../../order';

@Module({
  imports: [
    ClientsModule.registerAsync([
      {
        name: KAFKA_PRODUCER_SERVICE,
        useFactory: (configService: ConfigService) => ({
          transport: Transport.KAFKA,
          options: {
            client: {
              clientId: 'order-service-producer',
              brokers: configService
                .getOrThrow<string>('KAFKA_BROKERS')
                .split(','),
            },
            producer: {
              allowAutoTopicCreation: true, // Convenient for development
              idempotent: true, // Prevents the same message from being saved multiple times
              retry: {
                retries: 5,
                maxRetryTime: 30000,
              },
            },
          },
        }),
        inject: [ConfigService],
      },
    ]),
  ],
  providers: [OrderService, Logger],
  controllers: [OrderController],
})
export class OrderModule {}
