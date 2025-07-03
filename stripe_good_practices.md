# Stripe Good Practices: Desarrollo y Producción con TypeScript

## 🎯 Objetivo
Esta guía detalla cómo implementar Stripe en una aplicación TypeScript que requiere desarrollo activo mientras mantiene clientes en producción, asegurando una separación adecuada entre entornos y permitiendo un flujo de trabajo eficiente.

## 🔐 1. Configuración de Entornos Separados

### Variables de Entorno
```typescript
// config/stripe.ts
const config = {
  development: {
    publishableKey: process.env.STRIPE_PUBLISHABLE_KEY_TEST!,
    secretKey: process.env.STRIPE_SECRET_KEY_TEST!,
    webhookSecret: process.env.STRIPE_WEBHOOK_SECRET_TEST!,
    apiVersion: '2023-10-16' as const
  },
  production: {
    publishableKey: process.env.STRIPE_PUBLISHABLE_KEY_LIVE!,
    secretKey: process.env.STRIPE_SECRET_KEY_LIVE!,
    webhookSecret: process.env.STRIPE_WEBHOOK_SECRET_LIVE!,
    apiVersion: '2023-10-16' as const
  }
};

export const stripeConfig = config[process.env.NODE_ENV === 'production' ? 'production' : 'development'];
```

### Archivos .env
```bash
# .env.development
STRIPE_PUBLISHABLE_KEY_TEST=pk_test_xxx
STRIPE_SECRET_KEY_TEST=sk_test_xxx
STRIPE_WEBHOOK_SECRET_TEST=whsec_test_xxx

# .env.production
STRIPE_PUBLISHABLE_KEY_LIVE=pk_live_xxx
STRIPE_SECRET_KEY_LIVE=sk_live_xxx
STRIPE_WEBHOOK_SECRET_LIVE=whsec_live_xxx
```

### .gitignore
```bash
# Stripe
.env
.env.local
.env.development.local
.env.test.local
.env.production.local
stripe-cli/
```

## 🏗️ 2. Arquitectura de la Implementación

### Cliente de Stripe Singleton
```typescript
// lib/stripe/client.ts
import Stripe from 'stripe';
import { stripeConfig } from '@/config/stripe';

class StripeClient {
  private static instance: Stripe;
  
  static getInstance(): Stripe {
    if (!this.instance) {
      this.instance = new Stripe(stripeConfig.secretKey, {
        apiVersion: stripeConfig.apiVersion,
        typescript: true,
      });
    }
    return this.instance;
  }
}

export const stripe = StripeClient.getInstance();
```

### Servicio de Abstracción
```typescript
// services/payment.service.ts
import { stripe } from '@/lib/stripe/client';

export class PaymentService {
  async createCustomer(email: string, metadata?: Record<string, string>) {
    try {
      return await stripe.customers.create({
        email,
        metadata: {
          environment: process.env.NODE_ENV,
          ...metadata
        }
      });
    } catch (error) {
      console.error('Error creating customer:', error);
      throw new Error('Failed to create customer');
    }
  }

  async createSubscription(customerId: string, priceId: string) {
    // Validar que el priceId corresponde al entorno correcto
    if (process.env.NODE_ENV === 'production' && priceId.startsWith('price_test_')) {
      throw new Error('Cannot use test price in production');
    }
    
    return await stripe.subscriptions.create({
      customer: customerId,
      items: [{ price: priceId }],
      payment_behavior: 'default_incomplete',
      expand: ['latest_invoice.payment_intent']
    });
  }

  async createCheckoutSession(customerId: string, priceId: string) {
    return await stripe.checkout.sessions.create({
      customer: customerId,
      payment_method_types: ['card'],
      line_items: [{
        price: priceId,
        quantity: 1,
      }],
      mode: 'subscription',
      success_url: `${process.env.NEXT_PUBLIC_APP_URL}/dashboard?success=true`,
      cancel_url: `${process.env.NEXT_PUBLIC_APP_URL}/pricing?canceled=true`,
    });
  }

  async createPortalSession(customerId: string) {
    return await stripe.billingPortal.sessions.create({
      customer: customerId,
      return_url: `${process.env.NEXT_PUBLIC_APP_URL}/dashboard`,
    });
  }
}

export const paymentService = new PaymentService();
```

## 🧪 3. Testing y Webhooks

### Webhooks Handler con Validación de Entorno
```typescript
// app/api/webhooks/stripe/route.ts
import { headers } from 'next/headers';
import { stripe } from '@/lib/stripe/client';
import { stripeConfig } from '@/config/stripe';

export async function POST(req: Request) {
  const body = await req.text();
  const signature = headers().get('stripe-signature')!;
  
  let event: Stripe.Event;
  
  try {
    event = stripe.webhooks.constructEvent(
      body,
      signature,
      stripeConfig.webhookSecret
    );
  } catch (err) {
    console.error(`Webhook signature verification failed.`, err);
    return new Response('Webhook Error', { status: 400 });
  }

  // Log para debugging en desarrollo
  if (process.env.NODE_ENV !== 'production') {
    console.log(`✅ Webhook received: ${event.type}`);
  }

  // Manejar eventos
  switch (event.type) {
    case 'customer.subscription.created':
      await handleSubscriptionCreated(event.data.object);
      break;
    
    case 'customer.subscription.updated':
      await handleSubscriptionUpdated(event.data.object);
      break;
    
    case 'customer.subscription.deleted':
      await handleSubscriptionDeleted(event.data.object);
      break;
    
    case 'invoice.payment_succeeded':
      await handlePaymentSucceeded(event.data.object);
      break;
    
    case 'invoice.payment_failed':
      await handlePaymentFailed(event.data.object);
      break;
    
    default:
      console.log(`Unhandled event type: ${event.type}`);
  }
  
  return new Response('Success', { status: 200 });
}

// Handlers de eventos
async function handleSubscriptionCreated(subscription: Stripe.Subscription) {
  // Actualizar base de datos
  await prisma.stripeSubscription.create({
    data: {
      stripeSubscriptionId: subscription.id,
      stripeCustomerId: subscription.customer as string,
      status: subscription.status,
      priceId: subscription.items.data[0].price.id,
      currentPeriodEnd: new Date(subscription.current_period_end * 1000),
      environment: process.env.NODE_ENV,
    }
  });
}
```

### Stripe CLI para Desarrollo Local
```bash
# Instalar Stripe CLI
brew install stripe/stripe-cli/stripe

# Login
stripe login

# Reenviar webhooks a localhost
stripe listen --forward-to localhost:3000/api/webhooks/stripe

# En otra terminal, trigger eventos de test
stripe trigger payment_intent.succeeded
stripe trigger customer.subscription.created
```

### Testing Unitario
```typescript
// tests/stripe/payment.service.test.ts
import { paymentService } from '@/services/payment.service';
import { stripe } from '@/lib/stripe/client';

jest.mock('@/lib/stripe/client');

describe('PaymentService', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('createCustomer', () => {
    it('should create a customer with metadata', async () => {
      const mockCustomer = { id: 'cus_test123', email: 'test@example.com' };
      (stripe.customers.create as jest.Mock).mockResolvedValue(mockCustomer);

      const customer = await paymentService.createCustomer('test@example.com', {
        userId: 'user_123'
      });

      expect(stripe.customers.create).toHaveBeenCalledWith({
        email: 'test@example.com',
        metadata: {
          environment: 'test',
          userId: 'user_123'
        }
      });
      expect(customer).toEqual(mockCustomer);
    });
  });

  describe('createSubscription', () => {
    it('should throw error when using test price in production', async () => {
      process.env.NODE_ENV = 'production';
      
      await expect(
        paymentService.createSubscription('cus_123', 'price_test_123')
      ).rejects.toThrow('Cannot use test price in production');
    });
  });
});
```

## 📊 4. Base de Datos y Sincronización

### Schema con Tracking de Entorno
```typescript
// prisma/schema.prisma
model User {
  id              String           @id @default(cuid())
  email           String           @unique
  name            String?
  createdAt       DateTime         @default(now())
  updatedAt       DateTime         @updatedAt
  
  stripeCustomer  StripeCustomer?
}

model StripeCustomer {
  id                String   @id @default(cuid())
  userId            String   @unique
  stripeCustomerId  String   @unique
  environment       String   // 'development' | 'production'
  createdAt         DateTime @default(now())
  updatedAt         DateTime @updatedAt
  
  user              User     @relation(fields: [userId], references: [id])
  subscriptions     StripeSubscription[]
  invoices          StripeInvoice[]
}

model StripeSubscription {
  id                    String   @id @default(cuid())
  stripeSubscriptionId  String   @unique
  stripeCustomerId      String
  status                String
  priceId               String
  environment           String
  currentPeriodEnd      DateTime
  cancelAtPeriodEnd     Boolean  @default(false)
  createdAt             DateTime @default(now())
  updatedAt             DateTime @updatedAt
  
  customer              StripeCustomer @relation(fields: [stripeCustomerId], references: [stripeCustomerId])
}

model StripeInvoice {
  id               String   @id @default(cuid())
  stripeInvoiceId  String   @unique
  stripeCustomerId String
  amountPaid       Int
  amountDue        Int
  currency         String
  status           String
  createdAt        DateTime @default(now())
  
  customer         StripeCustomer @relation(fields: [stripeCustomerId], references: [stripeCustomerId])
}
```

### Repositorio de Datos
```typescript
// repositories/stripe.repository.ts
import { PrismaClient } from '@prisma/client';

export class StripeRepository {
  constructor(private prisma: PrismaClient) {}

  async findOrCreateCustomer(userId: string, stripeCustomerId: string) {
    return this.prisma.stripeCustomer.upsert({
      where: { userId },
      update: { stripeCustomerId },
      create: {
        userId,
        stripeCustomerId,
        environment: process.env.NODE_ENV
      }
    });
  }

  async updateSubscription(subscriptionId: string, data: Partial<StripeSubscription>) {
    return this.prisma.stripeSubscription.update({
      where: { stripeSubscriptionId: subscriptionId },
      data
    });
  }

  async getActiveSubscription(userId: string) {
    return this.prisma.stripeSubscription.findFirst({
      where: {
        customer: { userId },
        status: 'active',
        environment: process.env.NODE_ENV
      },
      include: { customer: true }
    });
  }
}
```

## 🚀 5. Procesos de Deployment

### Script de Validación Pre-Deploy
```typescript
// scripts/pre-deploy-check.ts
import { stripe } from '@/lib/stripe/client';

async function preDeployCheck() {
  console.log('🔍 Running pre-deployment checks...');
  
  try {
    // Verificar conexión a Stripe
    const products = await stripe.products.list({ limit: 1 });
    console.log('✅ Stripe connection verified');
    
    // Verificar que existan productos/precios en producción
    if (process.env.NODE_ENV === 'production') {
      const prices = await stripe.prices.list({ 
        active: true,
        limit: 100 
      });
      
      const livePrices = prices.data.filter(p => !p.livemode === false);
      
      if (livePrices.length === 0) {
        throw new Error('No live prices found in Stripe');
      }
      
      console.log(`✅ Found ${livePrices.length} live prices`);
    }
    
    // Verificar variables de entorno
    const requiredEnvVars = [
      'STRIPE_SECRET_KEY_LIVE',
      'STRIPE_PUBLISHABLE_KEY_LIVE',
      'STRIPE_WEBHOOK_SECRET_LIVE',
      'NEXT_PUBLIC_APP_URL'
    ];
    
    const missingVars = requiredEnvVars.filter(v => !process.env[v]);
    
    if (missingVars.length > 0) {
      throw new Error(`Missing environment variables: ${missingVars.join(', ')}`);
    }
    
    console.log('✅ All environment variables present');
    console.log('✅ Pre-deployment checks passed!');
    
  } catch (error) {
    console.error('❌ Pre-deployment check failed:', error);
    process.exit(1);
  }
}

preDeployCheck();
```

### Script de Migración de Datos
```typescript
// scripts/migrate-stripe-data.ts
import { PrismaClient } from '@prisma/client';
import { stripe } from '@/lib/stripe/client';

const prisma = new PrismaClient();

async function migrateTestDataToProduction() {
  // NUNCA ejecutar esto automáticamente
  if (process.env.NODE_ENV !== 'production') {
    throw new Error('This script should only run in production');
  }
  
  console.log('🚀 Starting test data migration to production...');
  
  // Obtener clientes de test que necesitan migración
  const testCustomers = await prisma.stripeCustomer.findMany({
    where: { 
      environment: 'development',
      // Add flags para identificar clientes a migrar
    },
    include: { user: true }
  });
  
  console.log(`Found ${testCustomers.length} customers to migrate`);
  
  for (const customer of testCustomers) {
    try {
      // Crear nuevo cliente en producción
      const prodCustomer = await stripe.customers.create({
        email: customer.user.email,
        metadata: {
          migratedFrom: 'test',
          originalId: customer.stripeCustomerId,
          userId: customer.userId
        }
      });
      
      // Actualizar base de datos
      await prisma.stripeCustomer.update({
        where: { id: customer.id },
        data: {
          stripeCustomerId: prodCustomer.id,
          environment: 'production'
        }
      });
      
      console.log(`✅ Migrated customer: ${customer.user.email}`);
      
    } catch (error) {
      console.error(`❌ Failed to migrate customer ${customer.user.email}:`, error);
    }
  }
  
  console.log('✅ Migration completed');
}

// Solo ejecutar si se pasa el flag --execute
if (process.argv.includes('--execute')) {
  migrateTestDataToProduction()
    .catch(console.error)
    .finally(() => prisma.$disconnect());
}
```

## 🛡️ 6. Mejores Prácticas

### 1. Validación de Productos/Precios
```typescript
// utils/stripe-validators.ts
export function validatePriceId(priceId: string): boolean {
  const isProduction = process.env.NODE_ENV === 'production';
  const isLivePrice = priceId.startsWith('price_') && !priceId.includes('test');
  
  if (isProduction && !isLivePrice) {
    throw new Error('Cannot use test prices in production');
  }
  
  if (!isProduction && isLivePrice) {
    console.warn('⚠️  Using live price in development environment');
  }
  
  return true;
}

export function sanitizeStripeMetadata(metadata: Record<string, any>): Record<string, string> {
  const sanitized: Record<string, string> = {};
  
  for (const [key, value] of Object.entries(metadata)) {
    if (value !== null && value !== undefined) {
      sanitized[key] = String(value);
    }
  }
  
  return sanitized;
}
```

### 2. Feature Flags para Nuevas Funcionalidades
```typescript
// lib/feature-flags.ts
export const features = {
  newCheckoutFlow: process.env.NEXT_PUBLIC_FEATURE_NEW_CHECKOUT === 'true',
  subscriptionUpgrades: process.env.NEXT_PUBLIC_FEATURE_UPGRADES === 'true',
  multipleCurrencies: process.env.NEXT_PUBLIC_FEATURE_MULTI_CURRENCY === 'true',
};

// utils/feature-check.ts
export function isFeatureEnabled(feature: keyof typeof features): boolean {
  return features[feature] || false;
}

// En componentes
import { isFeatureEnabled } from '@/utils/feature-check';

export function PricingPage() {
  if (isFeatureEnabled('newCheckoutFlow')) {
    return <NewCheckoutFlow />;
  }
  
  return <LegacyCheckoutFlow />;
}
```

### 3. Logging y Monitoreo
```typescript
// lib/stripe/logger.ts
import * as Sentry from '@sentry/nextjs';

interface StripeEventLog {
  event: string;
  customerId?: string;
  subscriptionId?: string;
  amount?: number;
  currency?: string;
  error?: any;
}

export function logStripeEvent(event: string, data: StripeEventLog) {
  const logData = {
    ...data,
    timestamp: new Date().toISOString(),
    environment: process.env.NODE_ENV
  };
  
  if (process.env.NODE_ENV === 'production') {
    // Enviar a servicio de logging
    Sentry.captureEvent({
      message: `Stripe: ${event}`,
      level: data.error ? 'error' : 'info',
      extra: logData,
      tags: {
        stripe: true,
        environment: 'production'
      }
    });
  } else {
    // Log en consola para desarrollo
    console.log(`[Stripe ${event}]`, logData);
  }
}

// Uso en webhook handlers
logStripeEvent('subscription.created', {
  event: 'subscription.created',
  customerId: subscription.customer as string,
  subscriptionId: subscription.id
});
```

### 4. Rate Limiting y Retry Logic
```typescript
// lib/stripe/retry.ts
export async function retryStripeOperation<T>(
  operation: () => Promise<T>,
  maxRetries: number = 3,
  delay: number = 1000
): Promise<T> {
  let lastError: Error;
  
  for (let i = 0; i < maxRetries; i++) {
    try {
      return await operation();
    } catch (error: any) {
      lastError = error;
      
      // No reintentar si es un error de validación
      if (error.type === 'StripeInvalidRequestError') {
        throw error;
      }
      
      // Esperar antes de reintentar
      if (i < maxRetries - 1) {
        await new Promise(resolve => setTimeout(resolve, delay * (i + 1)));
      }
    }
  }
  
  throw lastError!;
}

// Uso
const customer = await retryStripeOperation(
  () => stripe.customers.create({ email: 'user@example.com' })
);
```

## 📝 7. Checklist de Deployment

### Pre-Deployment Checklist
```markdown
- [ ] Todos los productos/precios creados en Stripe Dashboard (live)
- [ ] Variables de entorno configuradas en hosting (Vercel, Railway, etc.)
- [ ] Webhook endpoints configurados en Stripe Dashboard
- [ ] Webhook signing secret configurado
- [ ] Tests de integración pasando
- [ ] Base de datos respaldada
- [ ] Feature flags configurados correctamente
- [ ] Logs y monitoreo configurados
- [ ] Rate limits configurados
- [ ] SSL/TLS habilitado para webhooks
```

### Post-Deployment Checklist
```markdown
- [ ] Verificar webhooks recibiendo eventos (Stripe Dashboard)
- [ ] Test de compra con tarjeta real ($1 para verificar)
- [ ] Verificar creación de customer en Stripe
- [ ] Verificar actualización de base de datos
- [ ] Monitoreo de logs activo
- [ ] Alertas configuradas
- [ ] Rollback plan documentado y probado
- [ ] Customer portal accesible
- [ ] Emails transaccionales funcionando
```

## 🔄 8. Flujo de Desarrollo Continuo

### NPM Scripts
```json
{
  "scripts": {
    "dev": "next dev",
    "dev:stripe": "stripe listen --forward-to localhost:3000/api/webhooks/stripe",
    "dev:all": "concurrently \"npm run dev\" \"npm run dev:stripe\"",
    "test": "jest",
    "test:stripe": "jest --testPathPattern=stripe",
    "test:integration": "jest --testPathPattern=integration",
    "db:push": "prisma db push",
    "db:migrate": "prisma migrate dev",
    "db:seed": "ts-node prisma/seed.ts",
    "stripe:fixtures": "stripe fixtures ./test/fixtures/stripe-fixtures.json",
    "deploy:check": "ts-node scripts/pre-deploy-check.ts",
    "deploy:staging": "npm run deploy:check && vercel --env=preview",
    "deploy:prod": "npm run deploy:check && vercel --prod"
  }
}
```

### GitHub Actions CI/CD
```yaml
# .github/workflows/deploy.yml
name: Deploy to Production

on:
  push:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-node@v3
      - run: npm ci
      - run: npm run test
      - run: npm run test:integration

  deploy:
    needs: test
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-node@v3
      - run: npm ci
      - run: npm run deploy:check
        env:
          STRIPE_SECRET_KEY_LIVE: ${{ secrets.STRIPE_SECRET_KEY_LIVE }}
      - run: npm run deploy:prod
```

## 🚨 9. Manejo de Errores y Edge Cases

### Error Handling Middleware
```typescript
// middleware/stripe-error.ts
export function handleStripeError(error: any): {
  message: string;
  code: string;
  statusCode: number;
} {
  if (error.type === 'StripeCardError') {
    return {
      message: 'Tu tarjeta fue rechazada',
      code: error.code,
      statusCode: 400
    };
  }
  
  if (error.type === 'StripeInvalidRequestError') {
    return {
      message: 'Parámetros inválidos',
      code: 'invalid_request',
      statusCode: 400
    };
  }
  
  if (error.type === 'StripeAPIError') {
    return {
      message: 'Error en el servicio de pagos',
      code: 'api_error',
      statusCode: 500
    };
  }
  
  return {
    message: 'Error inesperado en el pago',
    code: 'unknown_error',
    statusCode: 500
  };
}
```

### Componente de Estado de Suscripción
```typescript
// components/subscription-status.tsx
import { useSubscription } from '@/hooks/use-subscription';

export function SubscriptionStatus() {
  const { subscription, loading, error } = useSubscription();
  
  if (loading) return <LoadingSpinner />;
  
  if (error) {
    return (
      <Alert variant="error">
        Error al cargar tu suscripción. Por favor, recarga la página.
      </Alert>
    );
  }
  
  if (!subscription) {
    return (
      <Alert variant="info">
        No tienes una suscripción activa.
        <Link href="/pricing">Ver planes</Link>
      </Alert>
    );
  }
  
  const isExpiringSoon = subscription.currentPeriodEnd < Date.now() + 7 * 24 * 60 * 60 * 1000;
  
  return (
    <div className="subscription-status">
      <h3>Tu suscripción</h3>
      <p>Estado: {subscription.status}</p>
      <p>Próximo pago: {formatDate(subscription.currentPeriodEnd)}</p>
      
      {isExpiringSoon && (
        <Alert variant="warning">
          Tu suscripción expira pronto. 
          <button onClick={handleRenew}>Renovar ahora</button>
        </Alert>
      )}
      
      {subscription.cancelAtPeriodEnd && (
        <Alert variant="info">
          Tu suscripción se cancelará al final del período actual.
          <button onClick={handleReactivate}>Reactivar</button>
        </Alert>
      )}
    </div>
  );
}
```

## 📚 10. Recursos y Referencias

### Enlaces Útiles
- [Stripe API Documentation](https://stripe.com/docs/api)
- [Stripe Testing Guide](https://stripe.com/docs/testing)
- [Stripe Webhooks Best Practices](https://stripe.com/docs/webhooks/best-practices)
- [Stripe CLI Documentation](https://stripe.com/docs/stripe-cli)

### Tarjetas de Test
```typescript
// constants/stripe-test-cards.ts
export const TEST_CARDS = {
  success: '4242424242424242',
  decline: '4000000000000002',
  insufficient: '4000000000009995',
  expired: '4000000000000069',
  cvcFail: '4000000000000127',
  processing: '4000000000000077',
  requires3DS: '4000002500003155',
} as const;
```

### Webhooks Events Reference
```typescript
// types/stripe-events.ts
export const STRIPE_EVENTS = {
  // Customer events
  CUSTOMER_CREATED: 'customer.created',
  CUSTOMER_UPDATED: 'customer.updated',
  CUSTOMER_DELETED: 'customer.deleted',
  
  // Subscription events
  SUBSCRIPTION_CREATED: 'customer.subscription.created',
  SUBSCRIPTION_UPDATED: 'customer.subscription.updated',
  SUBSCRIPTION_DELETED: 'customer.subscription.deleted',
  SUBSCRIPTION_TRIAL_WILL_END: 'customer.subscription.trial_will_end',
  
  // Payment events
  PAYMENT_INTENT_SUCCEEDED: 'payment_intent.succeeded',
  PAYMENT_INTENT_FAILED: 'payment_intent.payment_failed',
  
  // Invoice events
  INVOICE_CREATED: 'invoice.created',
  INVOICE_PAID: 'invoice.paid',
  INVOICE_PAYMENT_FAILED: 'invoice.payment_failed',
  INVOICE_UPCOMING: 'invoice.upcoming',
  
  // Checkout events
  CHECKOUT_SESSION_COMPLETED: 'checkout.session.completed',
  CHECKOUT_SESSION_EXPIRED: 'checkout.session.expired',
} as const;
```

---

## 🎯 Conclusión

Esta arquitectura permite:
- 🔧 Desarrollo seguro con datos de prueba
- 🚀 Despliegues a producción con confianza
- 🔄 Mantenimiento de ambos entornos sincronizados
- 🛡️ Prevención de mezcla entre datos test/producción
- 📊 Monitoreo y debugging efectivo
- 🔒 Seguridad robusta en el manejo de pagos

Recuerda siempre:
1. **Nunca hardcodear API keys**
2. **Siempre validar webhooks**
3. **Mantener logs detallados**
4. **Probar exhaustivamente antes de producción**
5. **Tener un plan de rollback**

¡Happy coding! 🚀
