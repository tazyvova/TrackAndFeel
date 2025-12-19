#!/bin/sh
# Create version file
echo "COMMIT=$COMMIT"
echo $COMMIT > /usr/share/nginx/html/version.txt
# Copy dist to volume, overwriting
cp -r /app/dist/* /usr/share/nginx/html/
# Start nginx
exec nginx -g 'daemon off;'
