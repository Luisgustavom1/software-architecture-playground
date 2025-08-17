import {
  Injectable,
  Inject,
  OnModuleInit,
  OnModuleDestroy,
  Logger,
} from '@nestjs/common';
import { ClientKafka } from '@nestjs/microservices';
import {
  KAFKA_PRODUCER_SERVICE,
  CreatedOrderEvent,
  CreateOrderDto,
} from '../../../order';

@Injectable()
export class OrderService implements OnModuleInit, OnModuleDestroy {
  constructor(
    @Inject(KAFKA_PRODUCER_SERVICE) private readonly kafkaClient: ClientKafka,
    private readonly logger: Logger,
  ) {}

  async onModuleInit() {
    try {
      await this.kafkaClient.connect();
      this.logger.log(
        'Kafka Producer Client connected successfully',
        OrderService.name,
      );
    } catch (error) {
      this.logger.error(
        'Failed to connect Kafka Producer Client',
        error,
        OrderService.name,
      );
    }
  }

  async onModuleDestroy() {
    await this.kafkaClient.close();
    this.logger.log('Kafka Producer Client disconnected', OrderService.name);
  }

  createOrder(orderData: CreateOrderDto) {
    const topic = 'order_created';
    const eventPayload: CreatedOrderEvent = {
      orderId: `ORD-${Date.now()}`, // Simple unique ID
      timestamp: new Date().toISOString(),
      data: orderData,
    };

    try {
      this.logger.log(
        `Publishing event to topic [${topic}]`,
        OrderService.name,
      );
      this.kafkaClient.emit(topic, JSON.stringify(eventPayload));
      return { success: true, publishedEvent: eventPayload };
    } catch (error) {
      this.logger.error(
        `Failed to publish event to topic [${topic}]`,
        error,
        OrderService.name,
      );
      return { success: false, error: (error as Error).message };
    }
  }
}
