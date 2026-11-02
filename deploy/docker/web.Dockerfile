# Multi-stage Dockerfile for ClientPulse Next.js Web Frontend
FROM node:24.22.2-alpine3.24 AS builder

WORKDIR /app

COPY apps/web/package*.json ./
RUN npm ci

COPY apps/web/ ./
ARG NEXT_PUBLIC_API_BASE_URL=https://mindu.ir
ENV NEXT_PUBLIC_API_BASE_URL=${NEXT_PUBLIC_API_BASE_URL}
RUN npm run build

# Production Runner
FROM node:24.22.2-alpine3.24 AS runner

WORKDIR /app

ENV NODE_ENV=production
ENV PORT=3000
ENV HOSTNAME="0.0.0.0"

RUN addgroup --system --gid 1001 nodejs && \
    adduser --system --uid 1001 nextjs

# Copy standalone output from builder
COPY --from=builder /app/public ./public
COPY --from=builder --chown=nextjs:nodejs /app/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /app/.next/static ./.next/static

USER nextjs

EXPOSE 3000

CMD ["node", "server.js"]
