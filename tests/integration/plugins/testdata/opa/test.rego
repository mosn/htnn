package test

import input.request

default allow = false

allow {
    request.method == "GET"
    startswith(request.path, "/echo")
}
