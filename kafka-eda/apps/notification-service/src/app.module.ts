import { Module } from '@nestjs/common';
import { ConfigModule } from '@nestjs/config';
import { NotificationModule } from './notification/notification.module';

@Module({
  imports: [ConfigModule.forRoot({ isGlobal: true }), NotificationModule],
  controllers: [],
  providers: [],
})
export class AppModule {}
