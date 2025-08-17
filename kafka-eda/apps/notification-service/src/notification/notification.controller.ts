import { Controller, Logger } from '@nestjs/common';
import {
  MessagePattern,
  Payload,
  Ctx,
  KafkaContext,
} from '@nestjs/microservices';
import { OrderTopics, type CreatedOrderEvent } from '../../../order';

@Controller()
export class NotificationController {
  constructor(private readonly logger: Logger) {}

  @MessagePattern(OrderTopics.order_created)
  async handleOrderCreated(
    @Payload() payload: CreatedOrderEvent,
    @Ctx() context: KafkaContext,
  ) {
    const { offset } = context.getMessage();
    const topic = context.getTopic();
    const partition = context.getPartition();
    this.logger.log(
      `Received message from topic [${topic}] partition [${partition}] offset [${offset}]`,
      NotificationController.name,
    );

    this.logger.log(
      `Payload: ${JSON.stringify(payload)}`,
      NotificationController.name,
    );

    try {
      this.logger.log(
        `Simulating notification dispatch for Order ID: ${payload.orderId}...`,
        NotificationController.name,
      );
      this.logger.log(
        `Notification for Order ID: ${payload.orderId} processed.`,
        NotificationController.name,
      );
      await context
        .getConsumer()
        .commitOffsets([
          { topic, partition, offset: (parseInt(offset) + 1).toString() },
        ]);
    } catch (error) {
      const err = error as Error;
      this.logger.error(
        `Error processing notification for order ${payload?.orderId}: ${err.message}`,
        err.stack,
        NotificationController.name,
      );
    }
  }
}
