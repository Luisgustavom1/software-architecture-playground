import { Module, Logger } from '@nestjs/common';
import { NotificationController } from './notification.controller';

@Module({
  imports: [],
  controllers: [NotificationController],
  providers: [Logger],
})
export class NotificationModule {}
