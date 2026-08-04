# MinIO path prefixes on tenant EDGE :8081 (included by proxy/tenant.conf)

    location = /minio-console {
        return 301 /minio-console/;
    }

    location /minio-console/ {
        rewrite ^/minio-console/(.*) /$1 break;
        proxy_pass http://minio_console;
        proxy_http_version 1.1;
        proxy_set_header Host $http_host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header X-Nginx-Proxy true;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_connect_timeout 300s;
        proxy_read_timeout 300s;
        chunked_transfer_encoding off;
    }
