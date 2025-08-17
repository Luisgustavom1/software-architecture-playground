export const KAFKA_PRODUCER_SERVICE = Symbol('KAFKA_PRODUCER_SERVICE');

export type CreatedOrderEvent = {
  orderId: string;
  timestamp: string;
  data: CreateOrderDto;
};

export type CreateOrderDto = {
  productId: string;
  quantity: number;
};

export enum OrderTopics {
  order_created = 'order_created',
}
