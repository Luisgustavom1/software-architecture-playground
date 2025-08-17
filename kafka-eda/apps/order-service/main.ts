import { NestFactory } from '@nestjs/core';
import { AppModule } from './src/app.module';
import { Logger } from '@nestjs/common';

async function bootstrap() {
  const app = await NestFactory.create(AppModule);
  const port = 3001;
  await app.listen(port);
  Logger.log(
    `Order Service (Producer) is running on: http://localhost:${port}`,
    'Bootstrap',
  );
}
bootstrap();
