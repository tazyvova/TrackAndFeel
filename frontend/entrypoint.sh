#!/bin/sh
# Copy dist to volume if empty
if [ -z "$(ls -A /usr/share/nginx/html)" ]; then
  cp -r /app/dist/* /usr/share/nginx/html/
fi
# Start nginx
exec nginx -g 'daemon off;'