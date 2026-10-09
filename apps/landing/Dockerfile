FROM node:20-alpine AS build
WORKDIR /app
RUN npm install -g pnpm
COPY package.json pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY . .
ARG VITE_APP_URL=https://app.lrsolid.cloud
ARG VITE_GITHUB_URL=https://github.com/OmiLAXR-Ecosystem/LRSolid
ENV VITE_APP_URL=$VITE_APP_URL
ENV VITE_GITHUB_URL=$VITE_GITHUB_URL
RUN pnpm run build

FROM nginx:1.27-alpine
COPY --from=build /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
