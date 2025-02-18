# Description

`path-template` is a small parser and validator written in golang for envoy's path template extension

A path template can have the following operators

- a path glob (matches a path segment): `/v1/user/*`
- a text glob (matches zero or more path segments): `/v1/redirectTo/**`
- a named variable (matches a path segment): `/v1/user/{userId}`
- a named variable with a pattern (matches according to the pattern): `/v1/redirectTo/{subpath=**}`
