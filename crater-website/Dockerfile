# Copyright 2025 Crater
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

FROM nginx:stable

# 暴露端口 80
EXPOSE 80

# 清除默认配置
RUN rm /etc/nginx/conf.d/default.conf

# 复制定制的 nginx 配置
COPY ./deploy/nginx.conf /etc/nginx/nginx.conf

# 复制构建产物到指定路径（注意末尾斜杠）
COPY ./out/ /usr/share/nginx/html/crater/

# 设置权限
RUN chmod -R 755 /usr/share/nginx/html

CMD ["nginx", "-g", "daemon off;"]