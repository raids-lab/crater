FROM nginx:stable-alpine

COPY nginx.conf /etc/nginx/nginx.conf
COPY dist/ /usr/share/nginx/html
CMD ["nginx", "-g", "daemon off;"]