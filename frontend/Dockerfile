# 使用 Nginx 稳定版的 Alpine 版本作为基础镜像
FROM nginx:stable-alpine

# 复制定制的 nginx 配置
COPY ./deploy/nginx.conf /etc/nginx/nginx.conf

# 从本地复制构建好的 dist 目录
COPY ./dist /usr/share/nginx/html

# 暴露端口 80
EXPOSE 80

# 默认命令启动 Nginx
CMD ["nginx", "-g", "daemon off;"]