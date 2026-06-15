FROM oven/bun:1-alpine

WORKDIR /app

COPY package.json bun.lock ./
RUN bun install

COPY . .

EXPOSE 5173

# bun install on start syncs new deps from package.json into the
# node_modules volume, so adding a dep needs only a restart
CMD ["sh", "-c", "bun install && bun run dev --host"]
