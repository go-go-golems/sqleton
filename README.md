# sqleton - a tool to quickly execute SQL commands

[![golangci-lint](https://github.com/wesen/sqleton/actions/workflows/lint.yml/badge.svg)](https://github.com/wesen/sqleton/actions/workflows/lint.yml)
[![golang-pipeline](https://github.com/wesen/sqleton/actions/workflows/push.yml/badge.svg)](https://github.com/wesen/sqleton/actions/workflows/push.yml)

![sqleton logo](doc/logo.png)

I often need to run SQL commands that I've run a thousand times before,
things like `SHOW PROCESSLIST`, getting the last few orders placed,
inspecting `performance_schema`. 

This tool will make it easy to have a single self-contained binary that can be
used to quickly query that data, format it the way I want, and contain self-documentation
because I keep forgetting things.

It makes heavy use of my [glazed](https://github.com/wesen/glazed) library,
and in many ways is a test-driver for its development
