httpdump
========

httpdump is a WIP [httpbin](https://httpbin.org) clone written in Go.

## Endpoints

- [/ip](http://httpdump.bruston.uk/ip) returns an origin IP
- [/user-agent](http://httpdump.bruston.uk/user-agent) returns a user-agent string
- [/headers](http://httpdump.bruston.uk/headers) returns a header map
- [/get](http://httpdump.bruston.uk/get) returns GET request information
- [/gzip](http://httpdump.bruston.uk/gzip) returns gzip-encoded data
- [/status/:code](http://httpdump.bruston.uk/status/418) returns a given HTTP status code
- [/stream/:n](http://httpdump.bruston.uk/stream/20) streams n-100 JSON objects
- [/bytes/:n](http://httpdump.bruston.uk/bytes/1024) returns n random bytes of binary data
- [/redirect-to?url=foo](http://httpdump.bruston.uk/redirect-to?url=http://example.com) redirect to URL *foo*
- [/basic-auth/:user/:passwd](http://httpdump.bruston.uk/basic-auth/user/passwd) challenges Basic Auth
- [/hidden-basic-auth/:user/:passwd](http://httpdump.bruston.uk/hidden-basic-auth/user/passwd) 404'd Basic Auth
