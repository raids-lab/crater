FROM harbor.act.buaa.edu.cn/crater/nginx:240302

# 暴露端口 80
EXPOSE 80

# 清除默认配置
RUN rm /etc/nginx/conf.d/default.conf

# 复制定制的 nginx 配置
COPY ./deploy/nginx.conf /etc/nginx/nginx.conf

# 复制构建产物到指定路径（注意末尾斜杠）
COPY ./out/ /usr/share/nginx/html/website/

# 设置权限
RUN chmod -R 755 /usr/share/nginx/html

CMD ["nginx", "-g", "daemon off;"]