#!/bin/sh
# Copy dist to volume, overwriting
cp -r /app/dist/* /usr/share/nginx/html/
# Start nginx
exec nginx -g 'daemon off;'
