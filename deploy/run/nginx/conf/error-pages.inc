# YCWL FreeTV branded error pages (html/errors/)
# Include inside each server { } block.
# Format (HTML vs JSON) is selected by error-maps.inc via $ycwl_ep* variables.

    error_page 400 $ycwl_ep400;
    error_page 403 $ycwl_ep403;
    error_page 404 $ycwl_ep404;
    error_page 500 $ycwl_ep500;
    error_page 502 $ycwl_ep502;
    error_page 503 $ycwl_ep503;
    error_page 504 $ycwl_ep504;

    location ^~ /errors/ {
        alias html/errors/;
        charset utf-8;
        add_header Cache-Control "no-store" always;
    }

    location = /errors/ycwl-error.css {
        alias html/errors/ycwl-error.css;
        default_type text/css;
        charset utf-8;
        add_header Cache-Control "public, max-age=86400" always;
    }
