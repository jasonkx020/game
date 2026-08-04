# EMQX / MQTT paths on tenant EDGE :8081 (included by proxy/tenant.conf)



    location /api/ {
        proxy_pass http://ycwl_platform_api;
    }

    location = /api/v1/health {
        proxy_pass http://ycwl_platform_api;
        access_log off;
    }

    location / {
        proxy_pass http://ycwl_platform_web;
    }