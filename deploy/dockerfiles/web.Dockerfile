FROM oven/bun:1 AS dev

WORKDIR /repo/web
COPY web/package.json ./
RUN bun install

CMD ["bun", "run", "dev", "--", "-H", "0.0.0.0", "-p", "3000"]

FROM oven/bun:1 AS builder

WORKDIR /repo/web
COPY web/package.json ./
RUN bun install
COPY web .
RUN bun run build

FROM oven/bun:1 AS prod

RUN addgroup --system --gid 1001 nodejs && \
    adduser --system --uid 1001 nextjs

WORKDIR /app
ENV NODE_ENV=production

COPY --from=builder --chown=nextjs:nodejs /repo/web/.next ./.next
COPY --from=builder --chown=nextjs:nodejs /repo/web/package.json ./
COPY --from=builder --chown=nextjs:nodejs /repo/web/public ./public

USER nextjs

EXPOSE 3000
CMD ["bun", "run", "start", "--", "-H", "0.0.0.0", "-p", "3000"]
