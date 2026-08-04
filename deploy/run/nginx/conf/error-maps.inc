# YCWL error response format: API paths → JSON (ApiResponse), browser → HTML.
# Include in http { } block only.

map $request_uri $ycwl_error_ext_by_path {
    default .html;
    ~^/api(?:/|$)       .json;
    ~^/minio-api(?:/|$) .json;
    ~^/files(?:/|$)     .json;
    ~^/ws(?:/|$)        .json;
}

map $server_port $ycwl_force_error_ext {
    default "";
    9002    .json;
    9003    .html;
}

map $ycwl_force_error_ext $ycwl_error_ext {
    ""      $ycwl_error_ext_by_path;
    default $ycwl_force_error_ext;
}

map $ycwl_error_ext $ycwl_ep400 {
    default /errors/400.html;
    .json    /errors/400.json;
}

map $ycwl_error_ext $ycwl_ep403 {
    default /errors/403.html;
    .json    /errors/403.json;
}

map $ycwl_error_ext $ycwl_ep404 {
    default /errors/404.html;
    .json    /errors/404.json;
}

map $ycwl_error_ext $ycwl_ep500 {
    default /errors/50x.html;
    .json    /errors/50x.json;
}

map $ycwl_error_ext $ycwl_ep502 {
    default /errors/502.html;
    .json    /errors/502.json;
}

map $ycwl_error_ext $ycwl_ep503 {
    default /errors/503.html;
    .json    /errors/503.json;
}

map $ycwl_error_ext $ycwl_ep504 {
    default /errors/504.html;
    .json    /errors/504.json;
}
