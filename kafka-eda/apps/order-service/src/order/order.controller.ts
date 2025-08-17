import { Controller, Post, Body, Logger } from '@nestjs/common';
import { OrderService } from './order.service';
import { type CreateOrderDto } from '../../../order';

@Controller('orders')
export class OrderController {
  constructor(
    private readonly orderService: OrderService,
    private readonly logger: Logger,
  ) {}

  @Post()
  createOrder(@Body() createOrderDto: CreateOrderDto) {
    this.logger.log(
      `Received create order request: ${JSON.stringify(createOrderDto)}`,
      OrderController.name,
    );
    const result = this.orderService.createOrder(createOrderDto);
    return result;
  }
}
